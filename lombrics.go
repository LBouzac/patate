// Lombrics : de 2 à 4 équipes de 3 vers (joueurs ou bots) sur une île qu'on creuse à coups d'explosions.
// Chacun son tour, un ver de l'équipe se déplace, saute, choisit son arme (bazooka, grenade, fusil) et tire.
// L'eau noie, les chutes font mal. La dernière équipe avec un ver vivant gagne.
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
	LCell     = 5.0 // le terrain est une grille de cases de 5 px, qu'on creuse
	LGW, LGH  = int(AW / LCell), int(AH / LCell)
	WaterY    = AH - 22 // sous cette ligne, on se noie
	WormR     = 6.0
	WormsPer  = 3
	WormSpeed = 45.0 // px/s en marchant
	WormClimb = 6.0  // marche qu'un ver peut monter à chaque pixel
	JumpVX    = 110.0
	JumpVY    = 230.0
	FallSafe  = 330.0 // vitesse d'atterrissage sans dégâts
	LTurn     = 30.0
	LMax      = 4
	LSub      = 4
	LBotWait  = 1.5
	LRooms    = 1000
)

// Armes : rayon du cratère, dégâts au centre, vitesse à pleine puissance.
const (
	Bazooka = iota // explose au contact, poussé par le vent
	Grenade        // rebondit et explose au bout de 3 s
	Shotgun        // tir instantané en ligne droite
)

var weapons = [...]struct{ crater, dmg, speed float64 }{{36, 50, 650}, {40, 55, 520}, {12, 25, 0}}

type Worm struct {
	ID     int     `json:"id"`
	Team   int     `json:"team"` // id de l'équipe
	HP     int     `json:"hp"`
	X      float64 `json:"x"` // centre
	Y      float64 `json:"y"`
	Dir    int     `json:"dir"` // regarde à gauche (-1) ou à droite (1)
	Alive  bool    `json:"alive"`
	vx, vy float64
	air    bool
	walk   float64 // pixels de marche en attente
}

type Squad struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Bot   bool   `json:"bot"`
	send  chan []byte
	next  int // prochain ver à jouer
	move  int // flèche tenue : -1, 0 ou 1
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

type WormGame struct {
	mu     sync.Mutex
	Pin    string   `json:"pin"`
	Squads []*Squad `json:"squads"`
	Worms  []*Worm  `json:"worms"`
	Land   string   `json:"land,omitempty"` // grille en base64, envoyée quand elle change et chaque seconde
	Proj   *Rocket  `json:"proj"`
	Phase  string   `json:"phase"` // wait, play, fly, settle, end
	Wait   float64  `json:"wait"`
	Turn   int      `json:"turn"` // ver qui joue
	Weapon int      `json:"weapon"`
	Wind   float64  `json:"wind"`
	Winner string   `json:"winner"`
	Events []Event  `json:"events"`
	land   []bool
	dirty  bool
	ticks  int
	team   int // équipe qui joue
	nextID int
}

func newWormGame(pin string) *WormGame { return &WormGame{Pin: pin, Phase: "wait", Worms: []*Worm{}} }

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
	if x < 0 || y < 0 || x >= AW || y >= AH || g.land == nil {
		return false
	}
	return g.land[int(y/LCell)*LGW+int(x/LCell)]
}

// hits : le corps d'un ver centré en (x, y) touche-t-il la terre ?
func (g *WormGame) hits(x, y float64) bool {
	const d = WormR * 0.7
	return g.solid(x, y+WormR) || g.solid(x-d, y+d) || g.solid(x+d, y+d) || g.solid(x-WormR, y) || g.solid(x+WormR, y) || g.solid(x, y-WormR)
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
	p1, p2, f := rand.Float64()*7, rand.Float64()*7, 1.5+rand.Float64()
	for c := range LGW {
		x := (float64(c) + 0.5) * LCell
		e := math.Min(x, AW-x) // les bords de l'île plongent dans l'eau
		top := 210 + 60*math.Sin(f*x/AW*2*math.Pi+p1) + 25*math.Sin(4.3*x/AW*2*math.Pi+p2) + math.Max(0, 80-e)*2.5
		for r := range LGH {
			g.land[r*LGW+c] = (float64(r)+0.5)*LCell > top
		}
	}
	for range 3 + rand.IntN(3) { // grottes
		g.carve(60+rand.Float64()*(AW-120), 260+rand.Float64()*130, 22+rand.Float64()*22)
	}
	g.Worms = g.Worms[:0]
	for _, s := range g.Squads {
		s.next = 0
		for range WormsPer {
			g.nextID++
			w := &Worm{ID: g.nextID, Team: s.ID, HP: 100, Alive: true, Dir: 1}
			g.place(w)
			g.Worms = append(g.Worms, w)
		}
	}
	g.dirty, g.Winner, g.Proj = true, "", nil
	g.team = g.Squads[rand.IntN(len(g.Squads))].ID
	g.startTurn()
}

// place pose un ver en surface, loin des autres.
func (g *WormGame) place(w *Worm) {
	for try := range 300 {
		x := 50 + rand.Float64()*(AW-100)
		y := WormR
		for y < WaterY && !g.hits(x, y+1) {
			y++
		}
		if y >= WaterY-WormR {
			continue
		}
		if try < 250 && slices.ContainsFunc(g.Worms, func(o *Worm) bool { return math.Hypot(o.X-x, o.Y-y) < 40 }) {
			continue
		}
		w.X, w.Y = x, y
		if x > AW/2 {
			w.Dir = -1
		}
		return
	}
	w.X, w.Y, w.air = AW/2, 0, true // shortcut: île sans place libre (jamais vu), le ver tombe du ciel
}

// startTurn : l'équipe g.team joue avec son prochain ver vivant.
func (g *WormGame) startTurn() {
	s := g.squad(g.team)
	var mine []*Worm
	for _, w := range g.Worms {
		if w.Team == s.ID && w.Alive {
			mine = append(mine, w)
		}
	}
	w := mine[s.next%len(mine)]
	s.next++
	for _, o := range g.Squads {
		o.move = 0
	}
	g.Turn, g.Weapon, g.Phase, g.Wait = w.ID, Bazooka, "play", LTurn
	g.Wind = math.Round((rand.Float64()*2-1)*10) / 10
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
	g.Phase, g.Wait, g.Proj = "end", 6, nil
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
		g.Phase, g.Wait, g.Proj, g.Worms = "wait", 0, nil, g.Worms[:0]
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

// active : le ver qui joue, s'il appartient à l'équipe s et peut agir.
func (g *WormGame) active(s *Squad) *Worm {
	if w := g.worm(g.Turn); g.Phase == "play" && w != nil && w.Alive && w.Team == s.ID {
		return w
	}
	return nil
}

func (g *WormGame) jump(s *Squad) {
	if w := g.active(s); w != nil && !w.air {
		w.vx, w.vy, w.air = float64(w.Dir)*JumpVX, -JumpVY, true
		g.event(Event{Kind: "jump", A: w.ID})
	}
}

func (g *WormGame) fire(s *Squad, weapon int, angle, power float64) {
	w := g.active(s)
	if w == nil || weapon < 0 || weapon >= len(weapons) || !(power > 0) || math.IsNaN(angle) || math.IsInf(angle, 0) {
		return
	}
	g.Weapon = weapon
	if c := math.Cos(angle); c != 0 {
		w.Dir = int(math.Copysign(1, c))
	}
	s.move = 0
	x, y := w.X+math.Cos(angle)*(WormR+4), w.Y+math.Sin(angle)*(WormR+4)
	g.event(Event{Kind: "fire", A: w.ID, B: weapon, X: x, Y: y})
	if weapon == Shotgun { // en ligne droite jusqu'à la terre ou un ver
		for d := 0.0; d < 500; d += 2 {
			px, py := x+math.Cos(angle)*d, y+math.Sin(angle)*d
			if px < 0 || px > AW || py < 0 || py > AH {
				break
			}
			if g.solid(px, py) || g.wormAt(px, py, w.ID) {
				g.explode(px, py, Shotgun)
				break
			}
		}
		g.Phase, g.Wait = "settle", 1
		return
	}
	v := min(power, 1) * weapons[weapon].speed
	g.Proj = &Rocket{X: x, Y: y, Kind: weapon, Fuse: 3, vx: v * math.Cos(angle), vy: v * math.Sin(angle), by: w.ID}
	g.Phase = "fly"
}

func (g *WormGame) wormAt(x, y float64, except int) bool {
	return slices.ContainsFunc(g.Worms, func(w *Worm) bool { return w.Alive && w.ID != except && math.Hypot(w.X-x, w.Y-y) < WormR+3 })
}

// flight avance le projectile d'un sous-pas. Renvoie done quand il a fini (boom : il explose, sinon il a coulé ou est sorti).
func (g *WormGame) flight(p *Rocket, dt, wind float64) (done, boom bool) {
	p.age += dt
	p.vy += Gravity * dt
	if p.Kind == Bazooka {
		p.vx += wind * WindAcc * dt
	}
	nx, ny := p.X+p.vx*dt, p.Y+p.vy*dt
	if ny > WaterY || nx < -50 || nx > AW+50 {
		p.X, p.Y = nx, ny
		return true, false
	}
	if p.Kind == Grenade {
		if p.Fuse -= dt; p.Fuse <= 0 {
			return true, true
		}
		if g.solid(nx, ny) { // rebond amorti
			hx, hy := g.solid(nx, p.Y), g.solid(p.X, ny)
			if hx || !hy {
				p.vx *= -0.5
			}
			if hy || !hx {
				p.vy *= -0.5
			}
			p.vx *= 0.8
			return false, false
		}
	} else {
		self := p.by
		if p.age > 0.3 {
			self = 0 // l'obus peut retomber sur son tireur
		}
		if g.solid(nx, ny) || g.wormAt(nx, ny, self) {
			p.X, p.Y = nx, ny
			return true, true
		}
	}
	p.X, p.Y = nx, ny
	return false, false
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
func (g *WormGame) explode(x, y float64, weapon int) {
	wp := weapons[weapon]
	g.event(Event{Kind: "boom", B: int(wp.crater), X: x, Y: y})
	g.carve(x, y, wp.crater)
	blast := wp.crater + 18
	for _, w := range g.Worms {
		d := math.Hypot(w.X-x, w.Y-y)
		if !w.Alive || d >= blast {
			continue
		}
		k := 1 - d/blast
		dmg := int(math.Round(wp.dmg * k))
		g.hurt(w, dmg)
		ax, ay := 0.0, -1.0
		if d > 1 {
			ax, ay = (w.X-x)/d, (w.Y-y)/d
		}
		push := 380 * k * wp.dmg / 50
		w.vx, w.vy, w.air = ax*push, ay*push-120*k, true
	}
}

func (g *WormGame) hurt(w *Worm, dmg int) {
	if dmg <= 0 || !w.Alive {
		return
	}
	w.HP -= dmg
	g.event(Event{Kind: "hit", A: w.ID, B: dmg, X: w.X, Y: w.Y})
	if w.HP <= 0 {
		g.kill(w, "dead")
	}
}

func (g *WormGame) kill(w *Worm, kind string) {
	w.HP, w.Alive, w.vx, w.vy = 0, false, 0, 0
	g.event(Event{Kind: kind, A: w.ID, X: w.X, Y: w.Y})
	if w.ID == g.Turn && g.Phase == "play" { // le ver qui joue meurt : fin du tour
		g.Phase, g.Wait = "settle", 1
	}
}

// move fait tomber, glisser ou marcher un ver pendant dt.
func (g *WormGame) move(w *Worm, dt float64) {
	if !w.Alive {
		return
	}
	if !w.air && !g.hits(w.X, w.Y+1) {
		w.air = true
	}
	if w.air {
		w.vy += Gravity * dt
		nx, ny := w.X+w.vx*dt, w.Y+w.vy*dt
		if g.hits(nx, w.Y) {
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
				if w.vy > FallSafe {
					g.hurt(w, int((w.vy-FallSafe)/8))
				}
				w.air, w.vx, w.walk = false, 0, 0
			}
			w.vy, ny = 0, w.Y
		}
		w.X, w.Y = nx, ny
	} else if s := g.squad(w.Team); s != nil && s.move != 0 && g.active(s) == w {
		w.Dir = s.move
		for w.walk += WormSpeed * dt; w.walk >= 1; w.walk-- { // pixel par pixel, en montant les petites marches
			nx := w.X + float64(s.move)
			up := 0.0
			for up <= WormClimb && g.hits(nx, w.Y-up) {
				up++
			}
			if up > WormClimb || nx < 0 || nx > AW {
				w.walk = 0
				break
			}
			w.X, w.Y = nx, w.Y-up
			for d := 0; d < int(WormClimb) && !g.hits(w.X, w.Y+1); d++ { // en descente, on suit le sol
				w.Y++
			}
		}
	}
	if w.Y > WaterY {
		g.kill(w, "drown")
	}
}

func (g *WormGame) still() bool {
	return !slices.ContainsFunc(g.Worms, func(w *Worm) bool { return w.Alive && w.air })
}

func (g *WormGame) step(dt float64) {
	if g.Phase != "wait" && g.Phase != "end" {
		for range LSub {
			for _, w := range g.Worms {
				g.move(w, dt/LSub)
			}
		}
	}
	switch g.Phase {
	case "play":
		if w := g.worm(g.Turn); w != nil {
			if s := g.squad(w.Team); s != nil && s.Bot && g.Wait <= LTurn-LBotWait {
				g.think(s, w)
				return
			}
		}
		if g.Wait -= dt; g.Wait <= 0 {
			g.Phase, g.Wait = "settle", 0.5
		}
	case "fly":
		for range LSub {
			if done, boom := g.flight(g.Proj, dt/LSub, g.Wind); done {
				if boom {
					g.explode(g.Proj.X, g.Proj.Y, g.Proj.Kind)
				} else {
					g.event(Event{Kind: "splash", X: g.Proj.X, Y: g.Proj.Y})
				}
				g.Proj, g.Phase, g.Wait = nil, "settle", 1.2
				return
			}
		}
	case "settle": // on attend que tout le monde soit retombé
		if g.Wait -= dt; g.Wait <= 0 && g.still() {
			g.nextTurn()
		}
	case "end":
		if g.Wait -= dt; g.Wait <= 0 {
			g.newGame()
		}
	}
}

// think : le bot essaie des tirs au bazooka dans toutes les directions (vent compris) et garde celui qui explose
// le plus près d'un ver adverse, sans trop s'approcher des siens.
func (g *WormGame) think(s *Squad, w *Worm) {
	bestA, bestP, best := -math.Pi/2, 0.5, math.Inf(1)
	for deg := 0.0; deg < 360; deg += 5 {
		a := deg * math.Pi / 180
		for p := 0.3; p <= 1.0; p += 0.07 {
			r := &Rocket{X: w.X + math.Cos(a)*(WormR+4), Y: w.Y + math.Sin(a)*(WormR+4), vx: p * weapons[Bazooka].speed * math.Cos(a), vy: p * weapons[Bazooka].speed * math.Sin(a), by: w.ID}
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
				} else if d < weapons[Bazooka].crater+25 {
					score = math.Inf(1) // touche un ami
					break
				}
			}
			if score < best {
				bestA, bestP, best = a, p, score
			}
		}
	}
	g.fire(s, Bazooka, bestA+(rand.Float64()*2-1)*0.03, bestP*(1+(rand.Float64()*2-1)*0.03))
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
		if g.ticks++; g.land != nil && (g.dirty || g.ticks%30 == 0) { // chaque seconde aussi : rattrape une image perdue
			g.Land = g.landB64()
		}
		b, _ := json.Marshal(g)
		g.Events, g.Land, g.dirty = nil, "", false
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
				case m.T == "fire":
					g.fire(s, m.W, m.A, m.P)
				}
			},
		}, ""
	})
}
