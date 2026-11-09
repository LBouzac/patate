// Lombrics : de 2 à 4 équipes de 3 vers (joueurs ou bots) sur une île qu'on creuse à coups d'explosions.
// Chacun son tour, un ver de l'équipe se déplace (marche, saut, hélicoptère, corde ninja), choisit une arme et
// tire, puis a quelques secondes pour se mettre à l'abri. L'eau noie (et monte en mort subite), les chutes font
// mal, l'île cache des mines et des barils, des caisses de soin et d'armes tombent du ciel.
// La dernière équipe avec un ver vivant gagne.
package main

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"
)

const (
	LW, LH       = 1600.0, 800.0 // l'île, plus grande que l'écran : la caméra suit l'action
	LCell        = 5.0           // le terrain est une grille de cases de 5 px, qu'on creuse
	LGW, LGH     = int(LW / LCell), int(LH / LCell)
	WaterY       = LH - 22 // sous cette ligne, on se noie (elle monte en mort subite)
	WormR        = 6.0
	WormsPer     = 3
	WormSpeed    = 45.0 // px/s en marchant
	WormClimb    = 6.0  // marche qu'un ver peut monter à chaque pixel
	JumpVX       = 110.0
	JumpVY       = 230.0
	FallSafe     = 330.0 // vitesse d'atterrissage sans dégâts
	LTurn        = 45.0
	LRetreat     = 3.0 // secondes pour se mettre à l'abri après avoir tiré
	HeliTime     = 5.0
	HeliSpeed    = 130.0
	RopeMax      = 260.0 // portée de la corde ninja
	MineRange    = 22.0  // une mine se déclenche quand un ver passe à cette distance
	HealHP       = 25
	SuddenRounds = 8    // tours chacun avant la mort subite
	FloodStep    = 12.0 // montée de l'eau à chaque tour en mort subite
	LMax         = 4
	LSub         = 4
	LBotWait     = 1.5
	LRooms       = 1000
)

// Les armes qu'on choisit (de Bazooka à Rope), puis ce qu'elles lancent ou ce qui explose.
const (
	Bazooka   = iota // explose au contact, poussé par le vent
	Grenade          // rebondit, explose au bout de 3 s
	Shotgun          // 2 tirs droits et instantanés
	Uzi              // mitrailleuse : rafale de balles
	Flamer           // lance-flammes
	Bat              // batte de baseball : envoie valser le ver d'en face
	Dynamite         // posée aux pieds, explose au bout de 5 s
	Mine             // posée aux pieds, s'arme en 3 s, saute quand un ver passe
	AirStrike        // 5 missiles tombent du ciel sur la cible
	Heli             // hélicoptère : 5 s de vol, ne termine pas le tour
	Rope             // corde ninja : s'accroche à la terre, ne termine pas le tour
	Missile
	Flame
	Bullet
	Barrel // baril de pétrole
)

// crater : rayon du trou, blast : rayon des dégâts, dmg : dégâts au centre (ou d'un coup direct), speed : vitesse
// à pleine puissance, shots : tirs par tour, ammo : munitions par partie (-1 : illimité).
var weapons = [...]struct {
	crater, blast, dmg, speed float64
	shots, ammo               int
}{
	Bazooka:   {36, 54, 50, 650, 1, -1},
	Grenade:   {40, 58, 55, 520, 1, -1},
	Shotgun:   {12, 30, 25, 0, 2, -1},
	Uzi:       {0, 0, 0, 0, 1, -1},
	Flamer:    {0, 0, 0, 330, 1, -1},
	Bat:       {0, 0, 30, 520, 1, -1},
	Dynamite:  {55, 80, 75, 0, 1, 2},
	Mine:      {25, 45, 40, 0, 1, 2},
	AirStrike: {0, 0, 0, 0, 1, 1},
	Heli:      {0, 0, 0, 0, 1, 2},
	Rope:      {0, 0, 0, 0, 1, -1},
	Missile:   {24, 40, 30, 0, 0, 0},
	Flame:     {3, 9, 3, 0, 0, 0},
	Bullet:    {4, 9, 4, 0, 0, 0},
	Barrel:    {35, 70, 45, 0, 0, 0},
}

// Les caisses d'armes donnent une munition de plus pour une de ces armes.
var crateWeapons = []int{Dynamite, Mine, AirStrike, Heli}

var wormNames = []string{"Gaston", "Raymond", "Josiane", "Ginette", "Marcel", "Huguette", "Lucien", "Paulette", "Fernand", "Odette",
	"Roger", "Germaine", "Gérard", "Simone", "René", "Yvette", "Maurice", "Colette", "Bernard", "Jacky", "Lulu", "Dédé", "Titi",
	"Momo", "Jojo", "Nono", "Bébert", "Zézette", "Riton", "Fanfan", "Kiki", "Mimile"}

type Worm struct {
	ID     int       `json:"id"`
	Team   int       `json:"team"` // id de l'équipe
	HP     int       `json:"hp"`
	X      float64   `json:"x"` // centre
	Y      float64   `json:"y"`
	Dir    int       `json:"dir"` // regarde à gauche (-1) ou à droite (1)
	Alive  bool      `json:"alive"`
	Name   string    `json:"name"`
	Heli   float64   `json:"heli,omitempty"` // secondes de vol qui restent
	Rope   []float64 `json:"rope,omitempty"` // point d'accroche de la corde ninja
	ropeL  float64   // longueur de la corde
	vx, vy float64
	air    bool
	walk   float64 // pixels de marche en attente
}

type Squad struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Bot   bool   `json:"bot"`
	Ammo  []int  `json:"ammo"` // munitions par arme (-1 : illimité)
	send  chan []byte
	next  int  // prochain ver à jouer
	move  int  // flèche tenue : -1, 0 ou 1
	up    bool // touche du haut tenue : l'hélicoptère monte, la corde raccourcit
	down  bool // touche du bas tenue : la corde s'allonge
}

type Rocket struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Kind   int     `json:"kind"`
	Fuse   float64 `json:"fuse"`
	vx, vy float64
	by     int // ver qui a tiré
	age    float64
}

// Obj : ce qui traîne sur l'île. Les caisses (heal : soin, ammo : une munition de l'arme W) descendent en
// parachute ; les mines et les barils explosent quand on les touche (Fuse : secondes avant l'explosion, -1 : rien).
type Obj struct {
	Kind string  `json:"kind"` // heal, ammo, mine, barrel
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Fall bool    `json:"fall"`
	Fuse float64 `json:"fuse"`
	W    int     `json:"w,omitempty"`
	arm  float64 // une mine posée ne se déclenche pas tout de suite
}

func (o *Obj) crate() bool { return o.Kind == "heal" || o.Kind == "ammo" }

// burst : une rafale en cours (mitrailleuse ou lance-flammes).
type burst struct {
	kind, n int
	a, t    float64
}

type WormGame struct {
	mu     sync.Mutex
	Pin    string    `json:"pin"`
	Squads []*Squad  `json:"squads"`
	Worms  []*Worm   `json:"worms"`
	Land   string    `json:"land,omitempty"` // grille en base64, envoyée quand elle change et chaque seconde
	Projs  []*Rocket `json:"projs"`
	Objs   []*Obj    `json:"objs"`
	Phase  string    `json:"phase"` // wait, play, settle, end
	Wait   float64   `json:"wait"`
	Turn   int       `json:"turn"` // ver qui joue
	Weapon int       `json:"weapon"`
	Fired  bool      `json:"fired"` // a tiré : temps de repli
	Wind   float64   `json:"wind"`
	Water  float64   `json:"water"`
	Winner string    `json:"winner"`
	Events []Event   `json:"events"`
	used   int       // tirs faits ce tour
	burst  burst
	land   []bool
	dirty  bool
	ticks  int
	turns  int
	team   int // équipe qui joue
	nextID int
}

func newWormGame(pin string) *WormGame {
	return &WormGame{Pin: pin, Phase: "wait", Worms: []*Worm{}, Projs: []*Rocket{}, Objs: []*Obj{}, Water: WaterY}
}

func (g *WormGame) event(e Event) { g.Events = append(g.Events, e) }

func (g *WormGame) worm(id int) *Worm {
	for _, w := range g.Worms {
		if w.ID == id {
			return w
		}
	}
	return nil
}

func (g *WormGame) squad(id int) *Squad {
	for _, s := range g.Squads {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// solid : y a-t-il de la terre au point (x, y) ? Hors du terrain, non.
func (g *WormGame) solid(x, y float64) bool {
	if x < 0 || y < 0 || x >= LW || y >= LH || g.land == nil {
		return false
	}
	return g.land[int(y/LCell)*LGW+int(x/LCell)]
}

// hits : le corps d'un ver centré en (x, y) (8 points sur son contour) touche-t-il la terre ?
// Règle tenue à chaque mouvement : aucun point du corps dans la terre. Un ver collé à une paroi droite ne la
// touche donc pas et tombe ; posé sur une bosse ou un rebord, il tient.
func (g *WormGame) hits(x, y float64) bool {
	const d = WormR * 0.7
	return g.solid(x, y+WormR) || g.solid(x-d, y+d) || g.solid(x+d, y+d) || g.solid(x-WormR, y) || g.solid(x+WormR, y) ||
		g.solid(x-d, y-d) || g.solid(x+d, y-d) || g.solid(x, y-WormR)
}

func (g *WormGame) landB64() string {
	b := make([]byte, (len(g.land)+7)/8)
	for i, s := range g.land {
		if s {
			b[i/8] |= 1 << (i % 8)
		}
	}
	return base64.StdEncoding.EncodeToString(b)
}

// newGame : une île au relief au hasard, percée de grottes, et les vers posés en surface.
func (g *WormGame) newGame() {
	g.land = make([]bool, LGW*LGH)
	p1, p2, f := rand.Float64()*7, rand.Float64()*7, 2+rand.Float64()*1.5
	for c := range LGW {
		x := (float64(c) + 0.5) * LCell
		e := math.Min(x, LW-x) // les bords de l'île plongent dans l'eau
		top := LH*0.5 + 110*math.Sin(f*x/LW*2*math.Pi+p1) + 45*math.Sin(4.3*f*x/LW*2*math.Pi+p2) + math.Max(0, 120-e)*2.5
		for r := range LGH {
			g.land[r*LGW+c] = (float64(r)+0.5)*LCell > top
		}
	}
	for range 2 + rand.IntN(2) { // îlots flottants : de quoi accrocher la corde, ou se percher
		g.fill(150+rand.Float64()*(LW-300), LH*0.18+rand.Float64()*LH*0.12, 50+rand.Float64()*60, 14+rand.Float64()*10)
	}
	for range 6 + rand.IntN(4) { // grottes
		g.carve(80+rand.Float64()*(LW-160), LH*0.6+rand.Float64()*LH*0.25, 25+rand.Float64()*28)
	}
	g.Water, g.turns = WaterY, 0
	g.Worms, g.Projs, g.Objs = g.Worms[:0], g.Projs[:0], g.Objs[:0]
	names := slices.Clone(wormNames)
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for _, s := range g.Squads {
		s.next = 0
		s.Ammo = make([]int, Missile)
		for i := range s.Ammo {
			s.Ammo[i] = weapons[i].ammo
		}
		for range WormsPer {
			g.nextID++
			w := &Worm{ID: g.nextID, Team: s.ID, HP: 100, Alive: true, Dir: 1, Name: names[len(g.Worms)%len(names)]}
			g.place(w)
			g.Worms = append(g.Worms, w)
		}
	}
	for range 8 {
		g.placeObj("mine")
	}
	for range 5 {
		g.placeObj("barrel")
	}
	g.dirty, g.Winner = true, ""
	g.team = g.Squads[rand.IntN(len(g.Squads))].ID
	g.startTurn()
}

// place pose un ver en surface, loin des autres.
func (g *WormGame) place(w *Worm) {
	for try := range 300 {
		x := 50 + rand.Float64()*(LW-100)
		y := WormR
		for y < g.Water && !g.hits(x, y+1) {
			y++
		}
		if y >= g.Water-WormR {
			continue
		}
		if try < 250 && slices.ContainsFunc(g.Worms, func(o *Worm) bool { return math.Hypot(o.X-x, o.Y-y) < 40 }) {
			continue
		}
		w.X, w.Y = x, y
		if x > LW/2 {
			w.Dir = -1
		}
		return
	}
	w.X, w.Y, w.air = LW/2, 0, true // shortcut: île sans place libre (jamais vu), le ver tombe du ciel
}

// placeObj pose une mine ou un baril en surface, loin des vers (pour ne rien déclencher au départ).
func (g *WormGame) placeObj(kind string) {
	for range 100 {
		x := 30 + rand.Float64()*(LW-60)
		y := 0.0
		for y < g.Water && !g.solid(x, y+7) {
			y++
		}
		if y >= g.Water-7 || slices.ContainsFunc(g.Worms, func(w *Worm) bool { return math.Hypot(w.X-x, w.Y-y) < 50 }) ||
			slices.ContainsFunc(g.Objs, func(o *Obj) bool { return math.Hypot(o.X-x, o.Y-y) < 25 }) {
			continue
		}
		g.Objs = append(g.Objs, &Obj{Kind: kind, X: x, Y: y, Fuse: -1})
		return
	}
}

// startTurn : l'équipe g.team joue avec son prochain ver vivant. Parfois une caisse tombe, et en mort subite l'eau monte.
func (g *WormGame) startTurn() {
	s := g.squad(g.team)
	var mine []*Worm
	for _, w := range g.Worms {
		w.Heli, w.Rope = 0, nil
		if w.Team == s.ID && w.Alive {
			mine = append(mine, w)
		}
	}
	w := mine[s.next%len(mine)]
	s.next++
	for _, o := range g.Squads {
		o.move, o.up, o.down = 0, false, false
	}
	g.Turn, g.Weapon, g.Phase, g.Wait = w.ID, Bazooka, "play", LTurn
	g.Fired, g.used, g.burst = false, 0, burst{}
	g.Wind = math.Round((rand.Float64()*2-1)*10) / 10
	if g.turns++; g.turns > SuddenRounds*len(g.Squads) {
		g.Water = max(g.Water-FloodStep, LH*0.35)
		g.event(Event{Kind: "flood", Y: g.Water})
	}
	if n := len(slices.DeleteFunc(slices.Clone(g.Objs), func(o *Obj) bool { return !o.crate() })); n < 3 && rand.Float64() < 0.4 {
		c := &Obj{Kind: "heal", X: 40 + rand.Float64()*(LW-80), Y: -10, Fall: true, Fuse: -1}
		if rand.IntN(2) == 0 {
			c.Kind, c.W = "ammo", crateWeapons[rand.IntN(len(crateWeapons))]
		}
		g.Objs = append(g.Objs, c)
	}
}

func (g *WormGame) teamAlive(id int) bool {
	return slices.ContainsFunc(g.Worms, func(w *Worm) bool { return w.Team == id && w.Alive })
}

func (g *WormGame) checkWin() bool {
	var left []*Squad
	for _, s := range g.Squads {
		if g.teamAlive(s.ID) {
			left = append(left, s)
		}
	}
	if len(left) > 1 {
		return false
	}
	g.Phase, g.Wait, g.Projs = "end", 6, g.Projs[:0]
	if len(left) == 1 {
		g.Winner = left[0].Name
		g.event(Event{Kind: "win", A: left[0].ID, Text: left[0].Name})
	}
	return true
}

func (g *WormGame) nextTurn() {
	if g.checkWin() {
		return
	}
	i := slices.IndexFunc(g.Squads, func(s *Squad) bool { return s.ID == g.team })
	for k := 1; k <= len(g.Squads); k++ {
		if s := g.Squads[(i+k+len(g.Squads))%len(g.Squads)]; g.teamAlive(s.ID) {
			g.team = s.ID
			g.startTurn()
			return
		}
	}
}

// endTurn : le ver qui joue ne peut plus rien faire, on attend que tout retombe.
func (g *WormGame) endTurn() {
	g.Phase, g.Wait, g.burst = "settle", 0.5, burst{}
	if w := g.worm(g.Turn); w != nil {
		w.Rope = nil // il lâche la corde
	}
}

func (g *WormGame) join(name string, bot bool) *Squad {
	if len(g.Squads) >= LMax {
		return nil
	}
	used := map[string]bool{}
	for _, s := range g.Squads {
		used[s.Color] = true
	}
	color := colors[0]
	for _, c := range colors {
		if !used[c] {
			color = c
			break
		}
	}
	g.nextID++
	s := &Squad{ID: g.nextID, Name: name, Color: color, Bot: bot}
	if !bot {
		s.send = make(chan []byte, 8)
	}
	g.Squads = append(g.Squads, s) // spectateur jusqu'à la prochaine partie
	if g.Phase == "wait" && len(g.Squads) >= 2 {
		g.Phase, g.Wait = "end", 3
	}
	return s
}

func (g *WormGame) leave(id int) {
	g.Squads = slices.DeleteFunc(g.Squads, func(s *Squad) bool { return s.ID == id })
	g.Worms = slices.DeleteFunc(g.Worms, func(w *Worm) bool { return w.Team == id })
	if !slices.ContainsFunc(g.Squads, func(s *Squad) bool { return !s.Bot }) {
		g.Squads = nil // plus aucun humain : les bots partent aussi
	}
	if len(g.Squads) < 2 {
		g.Phase, g.Wait, g.Projs, g.Worms = "wait", 0, g.Projs[:0], g.Worms[:0]
		return
	}
	if g.Phase == "play" && g.team == id {
		g.nextTurn()
	} else if g.Phase != "end" {
		g.checkWin()
	}
}

func (g *WormGame) removeBot() {
	for i := len(g.Squads) - 1; i >= 0; i-- {
		if g.Squads[i].Bot {
			g.leave(g.Squads[i].ID)
			return
		}
	}
}

// active : le ver qui joue, s'il appartient à l'équipe s et que le tour n'est pas fini.
func (g *WormGame) active(s *Squad) *Worm {
	if w := g.worm(g.Turn); g.Phase == "play" && w != nil && w.Alive && w.Team == s.ID {
		return w
	}
	return nil
}

// canMove : le ver qui joue peut se déplacer, sauf pendant une rafale ou entre les deux tirs du fusil.
func (g *WormGame) canMove(s *Squad) *Worm {
	if w := g.active(s); w != nil && g.burst.n == 0 && (g.used == 0 || g.Fired) {
		return w
	}
	return nil
}

func (g *WormGame) jump(s *Squad) {
	w := g.canMove(s)
	if w != nil && w.Rope != nil { // pendu à la corde : on lâche, avec l'élan
		w.Rope = nil
		g.event(Event{Kind: "release", A: w.ID})
		return
	}
	if w != nil && !w.air && w.Heli == 0 {
		w.vx, w.vy, w.air = float64(w.Dir)*JumpVX, -JumpVY, true
		g.event(Event{Kind: "jump", A: w.ID})
	}
}

func (g *WormGame) retreat() { g.Fired, g.Wait = true, LRetreat }

// fire utilise l'arme : angle et puissance pour viser, tx pour la cible de la frappe aérienne.
func (g *WormGame) fire(s *Squad, weapon int, angle, power, tx float64) {
	w := g.active(s)
	if w == nil || weapon < 0 || weapon >= Missile || g.Fired || g.burst.n > 0 || len(s.Ammo) != Missile || s.Ammo[weapon] == 0 ||
		math.IsNaN(angle) || math.IsInf(angle, 0) || math.IsNaN(tx) || math.IsInf(tx, 0) ||
		((weapon == Bazooka || weapon == Grenade) && !(power > 0)) || (g.used > 0 && weapon != g.Weapon) { // le fusil tire deux fois de suite
		return
	}
	if weapon == Rope { // ne compte pas comme un tir : on s'accroche à la terre dans la direction visée, ou on lâche
		if w.Rope != nil {
			w.Rope = nil
			g.event(Event{Kind: "release", A: w.ID})
			return
		}
		for d := WormR + 2; d < RopeMax; d += 2 {
			px, py := w.X+math.Cos(angle)*d, w.Y+math.Sin(angle)*d
			if px < 0 || px >= LW || py < 0 || py >= LH {
				break
			}
			if g.solid(px, py) {
				w.Rope, w.ropeL, w.air, w.Heli = []float64{px, py}, d, true, 0
				g.event(Event{Kind: "rope", A: w.ID, X: px, Y: py})
				return
			}
		}
		return
	}
	if weapon == Heli { // ne compte pas comme un tir
		if w.Heli > 0 || g.used > 0 {
			return
		}
		s.Ammo[Heli]--
		w.Heli = HeliTime
		g.event(Event{Kind: "heli", A: w.ID})
		return
	}
	if s.Ammo[weapon] > 0 {
		s.Ammo[weapon]--
	}
	g.Weapon, s.move, s.up = weapon, 0, false
	g.used++
	if c := math.Cos(angle); c != 0 && weapon != Dynamite && weapon != Mine && weapon != AirStrike {
		w.Dir = int(math.Copysign(1, c))
	}
	x, y := w.X+math.Cos(angle)*(WormR+4), w.Y+math.Sin(angle)*(WormR+4)
	g.event(Event{Kind: "fire", A: w.ID, B: weapon, X: x, Y: y})
	dir := float64(w.Dir)
	switch weapon {
	case Shotgun:
		g.shoot(w, angle, Shotgun)
	case Uzi:
		g.burst = burst{kind: Uzi, n: 15, a: angle}
		return
	case Flamer:
		g.burst = burst{kind: Flamer, n: 25, a: angle}
		return
	case Bat:
		g.bat(w, angle)
	case Dynamite:
		g.Projs = append(g.Projs, &Rocket{X: w.X + dir*8, Y: w.Y, Kind: Dynamite, Fuse: 5, vx: dir * 30, vy: -40, by: w.ID})
	case Mine:
		g.Objs = append(g.Objs, &Obj{Kind: "mine", X: w.X + dir*10, Y: w.Y, Fuse: -1, arm: 3})
	case AirStrike: // les missiles arrivent du côté où regarde le ver
		tx = max(0, min(LW, tx))
		for i := range 5 {
			g.Projs = append(g.Projs, &Rocket{X: tx + float64(i-2)*22 - dir*70, Y: -20 - float64(i)*12, Kind: Missile, vx: dir * 90, vy: 150})
		}
	default: // bazooka, grenade
		v := min(power, 1) * weapons[weapon].speed
		g.Projs = append(g.Projs, &Rocket{X: x, Y: y, Kind: weapon, Fuse: 3, vx: v * math.Cos(angle), vy: v * math.Sin(angle), by: w.ID})
	}
	if g.used >= weapons[weapon].shots && g.Phase == "play" {
		g.retreat()
	}
}

// shoot : tir en ligne droite jusqu'à la terre ou un ver.
func (g *WormGame) shoot(w *Worm, a float64, kind int) {
	x, y := w.X+math.Cos(a)*(WormR+4), w.Y+math.Sin(a)*(WormR+4)
	px, py := x, y
	for d := 0.0; d < 500; d += 2 {
		px, py = x+math.Cos(a)*d, y+math.Sin(a)*d
		if px < 0 || px > LW || py < 0 || py > LH {
			break
		}
		if g.solid(px, py) || g.wormNear(px, py, w.ID) != nil || g.objHit(px, py) {
			g.event(Event{Kind: "shot", A: int(px), B: int(py), X: x, Y: y})
			g.impact(px, py, kind, w.ID, math.Cos(a), math.Sin(a))
			return
		}
	}
	g.event(Event{Kind: "shot", A: int(px), B: int(py), X: x, Y: y})
}

// impact : une balle ou une flamme touche un ver de plein fouet (dégâts entiers, petite poussée), sinon explose.
func (g *WormGame) impact(x, y float64, kind, by int, vx, vy float64) {
	t := g.wormNear(x, y, by)
	if t == nil {
		g.explode(x, y, kind)
		return
	}
	g.event(Event{Kind: "boom", B: 1, X: x, Y: y})
	ax, ay := 0.0, -1.0
	if d := math.Hypot(vx, vy); d > 0 {
		ax, ay = vx/d, vy/d
	}
	push := weapons[kind].dmg * 12
	g.hurt(t, int(weapons[kind].dmg))
	if t.Alive {
		t.vx, t.vy, t.air = t.vx+ax*push, t.vy+ay*push-30, true
	}
}

// bat : coup de batte, envoie valser les vers juste devant.
func (g *WormGame) bat(w *Worm, a float64) {
	hx, hy := w.X+math.Cos(a)*12, w.Y+math.Sin(a)*12
	g.event(Event{Kind: "bat", A: w.ID, X: hx, Y: hy})
	for _, t := range g.Worms {
		if t != w && t.Alive && math.Hypot(t.X-hx, t.Y-hy) < 14 {
			g.hurt(t, int(weapons[Bat].dmg))
			if t.Alive {
				v := weapons[Bat].speed
				t.vx, t.vy, t.air = math.Cos(a)*v, math.Sin(a)*v-150, true
			}
		}
	}
}

// spray : une balle ou une flamme de la rafale en cours.
func (g *WormGame) spray(w *Worm) {
	a := g.burst.a + (rand.Float64()*2-1)*0.06
	if g.burst.kind == Uzi {
		g.shoot(w, a, Bullet)
		return
	}
	a += (rand.Float64()*2 - 1) * 0.1
	v := weapons[Flamer].speed * (0.85 + rand.Float64()*0.3)
	g.Projs = append(g.Projs, &Rocket{X: w.X + math.Cos(a)*(WormR+4), Y: w.Y + math.Sin(a)*(WormR+4), Kind: Flame, Fuse: 0.6, vx: v * math.Cos(a), vy: v * math.Sin(a), by: w.ID})
}

// objHit : une mine ou un baril au point (x, y) ?
func (g *WormGame) objHit(x, y float64) bool {
	return slices.ContainsFunc(g.Objs, func(o *Obj) bool { return !o.crate() && math.Hypot(o.X-x, o.Y-y) < 8 })
}

func (g *WormGame) wormNear(x, y float64, except int) *Worm {
	for _, w := range g.Worms {
		if w.Alive && w.ID != except && math.Hypot(w.X-x, w.Y-y) < WormR+3 {
			return w
		}
	}
	return nil
}

// flight avance un projectile d'un sous-pas. Renvoie done quand il a fini (boom : il explose ou touche, sinon il a
// coulé, est sorti ou s'est éteint).
func (g *WormGame) flight(p *Rocket, dt, wind float64) (done, boom bool) {
	p.age += dt
	grav := Gravity
	if p.Kind == Flame {
		grav *= 0.35
	}
	p.vy += grav * dt
	if p.Kind == Bazooka || p.Kind == Flame {
		p.vx += wind * WindAcc * dt
	}
	nx, ny := p.X+p.vx*dt, p.Y+p.vy*dt
	if ny > g.Water || nx < -50 || nx > LW+50 {
		p.X, p.Y = nx, ny
		return true, false
	}
	switch p.Kind {
	case Grenade, Dynamite:
		if p.Fuse -= dt; p.Fuse <= 0 {
			return true, true
		}
		if g.solid(nx, ny) { // rebond amorti (la dynamite ne rebondit presque pas)
			k := 0.5
			if p.Kind == Dynamite {
				k = 0.1
			}
			hx, hy := g.solid(nx, p.Y), g.solid(p.X, ny)
			if hx || !hy {
				p.vx *= -k
			}
			if hy || !hx {
				p.vy *= -k
			}
			p.vx *= 0.8
			return false, false
		}
	case Flame:
		if p.Fuse -= dt; p.Fuse <= 0 {
			return true, false
		}
		if g.solid(nx, ny) || g.wormNear(nx, ny, p.by) != nil || g.objHit(nx, ny) { // le lance-flammes ne brûle pas son porteur
			p.X, p.Y = nx, ny
			return true, true
		}
	default: // bazooka, missile
		self := p.by
		if p.age > 0.3 {
			self = 0 // l'obus peut retomber sur son tireur
		}
		if g.solid(nx, ny) || g.wormNear(nx, ny, self) != nil || g.objHit(nx, ny) {
			p.X, p.Y = nx, ny
			return true, true
		}
	}
	p.X, p.Y = nx, ny
	return false, false
}

// advance fait voler tous les projectiles pendant dt.
func (g *WormGame) advance(dt float64) {
	for range LSub {
		for i := 0; i < len(g.Projs); {
			p := g.Projs[i]
			done, boom := g.flight(p, dt/LSub, g.Wind)
			if !done {
				i++
				continue
			}
			g.Projs = slices.Delete(g.Projs, i, i+1)
			switch {
			case boom && p.Kind == Flame:
				g.impact(p.X, p.Y, Flame, p.by, p.vx, p.vy)
			case boom:
				g.explode(p.X, p.Y, p.Kind)
			case p.Kind != Flame:
				g.event(Event{Kind: "splash", X: p.X, Y: p.Y})
			}
		}
	}
}

// objects : tout ce qui traîne tombe quand la terre disparaît dessous ; les caisses se ramassent, les mines se
// déclenchent au passage d'un ver, et ce qui a une mèche allumée explose.
func (g *WormGame) objects(dt float64) {
	var blow []*Obj
	g.Objs = slices.DeleteFunc(g.Objs, func(o *Obj) bool {
		if o.Fall = !g.solid(o.X, o.Y+7); o.Fall {
			if o.crate() {
				o.Y += 60 * dt // parachute
			} else {
				o.Y += 200 * dt
			}
		}
		if o.Y > g.Water {
			g.event(Event{Kind: "splash", X: o.X, Y: o.Y})
			return true
		}
		o.arm = max(0, o.arm-dt)
		for _, w := range g.Worms {
			if !w.Alive {
				continue
			}
			d := math.Hypot(w.X-o.X, w.Y-o.Y)
			switch {
			case o.Kind == "heal" && d < 14:
				w.HP += HealHP
				g.event(Event{Kind: "heal", A: w.ID, B: HealHP, X: w.X, Y: w.Y})
				return true
			case o.Kind == "ammo" && d < 14:
				if s := g.squad(w.Team); s != nil && len(s.Ammo) == Missile {
					s.Ammo[o.W]++
				}
				g.event(Event{Kind: "ammo", A: w.ID, B: o.W, X: w.X, Y: w.Y})
				return true
			case o.Kind == "mine" && o.Fuse < 0 && o.arm == 0 && d < MineRange:
				o.Fuse = 1.5
				g.event(Event{Kind: "beep", X: o.X, Y: o.Y})
			}
		}
		if o.Fuse >= 0 {
			if o.Fuse -= dt; o.Fuse < 0 {
				blow = append(blow, o)
				return true
			}
		}
		return false
	})
	for _, o := range blow {
		g.blow(o)
	}
}

// blow fait sauter une mine ou un baril ; le baril de pétrole répand des flammes.
func (g *WormGame) blow(o *Obj) {
	if o.Kind == "mine" {
		g.explode(o.X, o.Y, Mine)
		return
	}
	g.explode(o.X, o.Y, Barrel)
	for range 8 {
		a := -math.Pi/2 + (rand.Float64()*2-1)*1.2
		v := 120 + rand.Float64()*160
		g.Projs = append(g.Projs, &Rocket{X: o.X, Y: o.Y - 4, Kind: Flame, Fuse: 1.2, vx: v * math.Cos(a), vy: v * math.Sin(a)})
	}
}

// fill ajoute de la terre en ellipse (îlots flottants).
func (g *WormGame) fill(x, y, rx, ry float64) {
	for row := range LGH {
		for c := range LGW {
			dx, dy := ((float64(c)+0.5)*LCell-x)/rx, ((float64(row)+0.5)*LCell-y)/ry
			if dx*dx+dy*dy < 1 {
				g.land[row*LGW+c] = true
			}
		}
	}
}

func (g *WormGame) carve(x, y, r float64) {
	for row := max(0, int((y-r)/LCell)); row <= min(LGH-1, int((y+r)/LCell)); row++ {
		for c := max(0, int((x-r)/LCell)); c <= min(LGW-1, int((x+r)/LCell)); c++ {
			if math.Hypot((float64(c)+0.5)*LCell-x, (float64(row)+0.5)*LCell-y) < r {
				g.land[row*LGW+c] = false
			}
		}
	}
	g.dirty = true
}

// explode creuse, blesse et projette les vers autour.
func (g *WormGame) explode(x, y float64, kind int) {
	wp := weapons[kind]
	g.event(Event{Kind: "boom", B: int(wp.crater), X: x, Y: y})
	g.carve(x, y, wp.crater)
	for _, o := range g.Objs { // les mines et les barils voisins sautent à leur tour, juste après
		if !o.crate() && math.Hypot(o.X-x, o.Y-y) < wp.blast+6 && (o.Fuse < 0 || o.Fuse > 0.25) {
			o.Fuse, o.arm = 0.25, 0
		}
	}
	for _, w := range g.Worms {
		d := math.Hypot(w.X-x, w.Y-y)
		if !w.Alive || d >= wp.blast {
			continue
		}
		k := 1 - d/wp.blast
		g.hurt(w, int(math.Round(wp.dmg*k)))
		ax, ay := 0.0, -1.0
		if d > 1 {
			ax, ay = (w.X-x)/d, (w.Y-y)/d
		}
		push := 380 * k * wp.dmg / 50
		w.vx, w.vy, w.air = ax*push, ay*push-120*k, true
	}
}

// hurt : comme dans le genre, le ver qui joue et se blesse finit son tour.
func (g *WormGame) hurt(w *Worm, dmg int) {
	if dmg <= 0 || !w.Alive {
		return
	}
	w.HP -= dmg
	g.event(Event{Kind: "hit", A: w.ID, B: dmg, X: w.X, Y: w.Y})
	if w.HP <= 0 {
		g.kill(w, "dead")
	} else if w.ID == g.Turn && g.Phase == "play" {
		g.endTurn()
	}
}

func (g *WormGame) kill(w *Worm, kind string) {
	w.HP, w.Alive, w.vx, w.vy, w.Heli, w.Rope = 0, false, 0, 0, 0, nil
	g.event(Event{Kind: kind, A: w.ID, X: w.X, Y: w.Y})
	if w.ID == g.Turn && g.Phase == "play" {
		g.endTurn()
	}
}

// move fait tomber, voler, glisser ou marcher un ver pendant dt.
func (g *WormGame) move(w *Worm, dt float64) {
	if !w.Alive {
		return
	}
	s := g.squad(w.Team)
	ctl := s != nil && g.canMove(s) == w // c'est lui qu'on commande
	flying := false
	if w.Rope != nil && !g.solid(w.Rope[0], w.Rope[1]) { // la terre où elle tenait a sauté
		w.Rope = nil
	}
	roped := w.Rope != nil
	if roped { // corde : ← → pour se balancer, haut et bas pour raccourcir ou allonger
		w.air = true
		if ctl {
			w.vx += float64(s.move) * 300 * dt
			if s.up {
				w.ropeL = max(15, w.ropeL-120*dt)
			}
			if s.down {
				w.ropeL = min(RopeMax, w.ropeL+120*dt)
			}
			if s.move != 0 {
				w.Dir = s.move
			}
		}
		if v := math.Hypot(w.vx, w.vy); v > 500 {
			w.vx, w.vy = w.vx*500/v, w.vy*500/v
		}
	}
	if w.Heli > 0 {
		w.Heli = max(0, w.Heli-dt)
		if flying = ctl && (w.air || s.up); flying { // hélicoptère : on monte en tenant le saut, sinon on descend doucement
			w.air = true
			w.vx += (float64(s.move)*HeliSpeed - w.vx) * min(1, 4*dt)
			if s.up {
				w.vy -= 700 * dt
			} else {
				w.vy += 250 * dt
			}
			w.vy = max(-150, min(150, w.vy))
			if s.move != 0 {
				w.Dir = s.move
			}
		}
	}
	if !w.air && !g.hits(w.X, w.Y+1) {
		w.air = true
	}
	if w.air {
		if !flying {
			w.vy += Gravity * dt
		}
		nx, ny := w.X+w.vx*dt, w.Y+w.vy*dt
		if ny < WormR { // haut de la carte : on ne sort pas de l'écran
			ny, w.vy = WormR, 0
		}
		if g.hits(nx, ny) { // on bute : d'abord de côté, puis de haut en bas
			if g.hits(nx, w.Y) { // mur
				w.vx, nx = -w.vx*0.3, w.X
			}
			if g.hits(nx, ny) {
				if w.vy > 0 { // atterrissage : on descend au ras du sol
					for range 10 {
						if g.hits(nx, w.Y+1) {
							break
						}
						w.Y++
					}
					if w.vy > FallSafe && !roped {
						g.hurt(w, int((w.vy-FallSafe)/8))
					}
					w.air, w.vx, w.walk = false, 0, 0
				}
				w.vy, ny = 0, w.Y // plafond, ou posé
			}
		}
		w.X, w.Y = nx, ny
		if roped { // la corde retient : on revient sur le cercle et on perd la vitesse qui l'étirait
			dx, dy := w.X-w.Rope[0], w.Y-w.Rope[1]
			if d := math.Hypot(dx, dy); d > w.ropeL {
				ux, uy := dx/d, dy/d
				if px, py := w.Rope[0]+ux*w.ropeL, w.Rope[1]+uy*w.ropeL; !g.hits(px, py) {
					w.X, w.Y = px, py
				}
				if rv := w.vx*ux + w.vy*uy; rv > 0 {
					w.vx, w.vy = w.vx-rv*ux, w.vy-rv*uy
				}
			}
		}
	} else if ctl && s.move != 0 {
		w.Dir = s.move
		for w.walk += WormSpeed * dt; w.walk >= 1; w.walk-- { // pixel par pixel, en montant les petites marches
			nx := w.X + float64(s.move)
			up := 0.0
			for up <= WormClimb && g.hits(nx, w.Y-up) {
				up++
			}
			if up > WormClimb || nx < 0 || nx > LW {
				w.walk = 0
				break
			}
			w.X, w.Y = nx, w.Y-up
			for d := 0; d < int(WormClimb) && !g.hits(w.X, w.Y+1); d++ { // en descente, on suit le sol
				w.Y++
			}
		}
	}
	if w.Y > g.Water {
		g.kill(w, "drown")
	}
}

func (g *WormGame) still() bool {
	return len(g.Projs) == 0 && !slices.ContainsFunc(g.Worms, func(w *Worm) bool { return w.Alive && w.air }) &&
		!slices.ContainsFunc(g.Objs, func(o *Obj) bool { return o.Fuse >= 0 })
}

func (g *WormGame) step(dt float64) {
	if g.Phase == "wait" || g.Phase == "end" {
		if g.Phase == "end" {
			if g.Wait -= dt; g.Wait <= 0 {
				g.newGame()
			}
		}
		return
	}
	for range LSub {
		for _, w := range g.Worms {
			g.move(w, dt/LSub)
		}
	}
	g.advance(dt)
	g.objects(dt)
	switch g.Phase {
	case "play":
		w := g.worm(g.Turn)
		if w == nil {
			g.endTurn()
			return
		}
		if g.burst.n > 0 { // rafale : le temps du tour s'arrête
			for g.burst.t -= dt; g.burst.t <= 0 && g.burst.n > 0 && g.Phase == "play"; g.burst.t += 0.06 {
				g.burst.n--
				g.spray(w)
			}
			if g.burst.n == 0 && g.Phase == "play" {
				g.retreat()
			}
			return
		}
		if s := g.squad(w.Team); s != nil && s.Bot && g.used == 0 && g.Wait <= LTurn-LBotWait {
			g.think(s, w)
			return
		}
		if g.Wait -= dt; g.Wait <= 0 {
			g.endTurn()
		}
	case "settle": // on attend que tout le monde soit retombé et que plus rien ne vole
		if g.Wait -= dt; g.Wait <= 0 && g.still() {
			g.nextTurn()
		}
	}
}

// think : collé à un adversaire, le bot sort la batte. Sinon il essaie des tirs au bazooka et à la grenade dans
// toutes les directions (vent et terrain compris) et garde celui qui explose le plus près d'un ver adverse,
// sans trop s'approcher des siens.
func (g *WormGame) think(s *Squad, w *Worm) {
	for _, o := range g.Worms {
		if o.Alive && o.Team != s.ID && math.Abs(o.X-w.X) < 18 && math.Abs(o.Y-w.Y) < 12 {
			g.fire(s, Bat, -math.Pi/2+math.Copysign(1.1, o.X-w.X), 1, 0) // vers lui, un peu vers le haut
			return
		}
	}
	bestK, bestA, bestP, best := Bazooka, -math.Pi/2+float64(w.Dir)*0.7, 0.6, math.Inf(1) // faute de mieux : en cloche devant lui
	for _, kind := range []int{Bazooka, Grenade} {
		for deg := 0.0; deg < 360; deg += 5 {
			a := deg * math.Pi / 180
			for p := 0.3; p <= 1.0; p += 0.07 {
				v := p * weapons[kind].speed
				r := &Rocket{X: w.X + math.Cos(a)*(WormR+4), Y: w.Y + math.Sin(a)*(WormR+4), Kind: kind, Fuse: 3, vx: v * math.Cos(a), vy: v * math.Sin(a), by: w.ID}
				boom := false
				for range 600 { // 10 s de vol au plus
					var done bool
					if done, boom = g.flight(r, 1.0/60, g.Wind); done {
						break
					}
				}
				if !boom {
					continue
				}
				score := math.Inf(1)
				for _, o := range g.Worms {
					if !o.Alive {
						continue
					}
					d := math.Hypot(r.X-o.X, r.Y-o.Y)
					if o.Team != s.ID {
						score = min(score, d)
					} else if d < weapons[kind].blast+8 {
						score = math.Inf(1) // touche un ami
						break
					}
				}
				if score < best-5 { // à peu près pareil : il garde le bazooka, plus direct
					bestK, bestA, bestP, best = kind, a, p, score
				}
			}
		}
	}
	g.fire(s, bestK, bestA+(rand.Float64()*2-1)*0.03, bestP*(1+(rand.Float64()*2-1)*0.03), 0)
}

// --- salons Lombrics ---

var (
	wormGamesMu sync.Mutex
	wormGames   = map[string]*WormGame{}
)

func enterWormGame(pin, name string) (*WormGame, *Squad, string) {
	wormGamesMu.Lock()
	defer wormGamesMu.Unlock()
	if pin == "" {
		if len(wormGames) >= LRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		g := newWormGame(newPin(func(p string) bool { return wormGames[p] != nil }))
		wormGames[g.Pin] = g
		s := g.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go g.run()
		return g, s, ""
	}
	g := wormGames[pin]
	if g == nil {
		return nil, nil, "Code inconnu"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if s := g.join(name, false); s != nil {
		return g, s, ""
	}
	return nil, nil, "Partie pleine (4 équipes)"
}

func (g *WormGame) run() {
	tk := time.NewTicker(ArtTick)
	defer tk.Stop()
	for range tk.C {
		if g.closeIfEmpty() {
			return
		}
		g.mu.Lock()
		g.step(ArtTick.Seconds())
		// au plus 10 fois par seconde quand il change (lance-flammes), et chaque seconde pour rattraper une image perdue
		if g.ticks++; g.land != nil && ((g.dirty && g.ticks%3 == 0) || g.ticks%30 == 0) {
			g.Land, g.dirty = g.landB64(), false
		}
		b, _ := json.Marshal(g)
		g.Events, g.Land = nil, ""
		for _, s := range g.Squads {
			if s.send != nil {
				select {
				case s.send <- b:
				default: // client lent : on saute cette image
				}
			}
		}
		g.mu.Unlock()
	}
}

func (g *WormGame) closeIfEmpty() bool {
	wormGamesMu.Lock()
	defer wormGamesMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.Squads) > 0 {
		return false
	}
	delete(wormGames, g.Pin)
	return true
}

func serveLombrics(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		g, s, why := enterWormGame(q.Get("pin"), name)
		if s == nil {
			return nil, why
		}
		return &Seat{
			Room:  g.Pin,
			ID:    s.ID,
			Send:  s.send,
			Leave: func() { g.mu.Lock(); g.leave(s.ID); g.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					T   string  `json:"t"`
					Bot int     `json:"bot"`
					M   int     `json:"m"`
					W   int     `json:"w"`
					A   float64 `json:"a"`
					P   float64 `json:"p"`
					X   float64 `json:"x"`
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				g.mu.Lock()
				defer g.mu.Unlock()
				switch {
				case m.Bot == 1:
					g.join("Bot", true)
				case m.Bot == -1:
					g.removeBot()
				case m.T == "move":
					if g.active(s) != nil {
						s.move = max(-1, min(1, m.M))
					}
				case m.T == "jump":
					g.jump(s)
				case m.T == "up": // touche du haut tenue ou relâchée : l'hélicoptère monte, la corde raccourcit
					s.up = m.M == 1
				case m.T == "down": // la corde s'allonge
					s.down = m.M == 1
				case m.T == "fire":
					g.fire(s, m.W, m.A, m.P, m.X)
				}
			},
		}, ""
	})
}
