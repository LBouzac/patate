// Patate : la bombe passe de joueur en joueur au contact, celui qui la tient quand elle explose est éliminé.
package main

import (
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"
)

//go:embed index.html ricochet.html dessin.html golf.html foot.html admin.html common.js style.css
var files embed.FS

const (
	W, H         = 800.0, 600.0
	R            = 18.0
	Speed        = 220.0
	Tick         = time.Second / 30
	PassCooldown = 0.6 // évite le ping-pong instantané de la bombe
)

var colors = []string{"#e6194b", "#3cb44b", "#ffe119", "#4363d8", "#f58231", "#911eb4", "#42d4f4", "#f032e6", "#bfef45", "#fabed4"} // une couleur par place : 10 joueurs max

type Input struct {
	U bool `json:"u"`
	D bool `json:"d"`
	L bool `json:"l"`
	R bool `json:"r"`
}

type Player struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Color string  `json:"color"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Alive bool    `json:"alive"`
	Bot   bool    `json:"bot"`
	Score int     `json:"score"`
	Team  int     `json:"team"`  // mode équipes : 1 rouge, 2 bleu
	Emote int     `json:"emote"` // 1-4 pendant 2 s, 0 sinon
	// secondes restantes de chaque effet de bonus
	Fast   float64 `json:"fast"`
	Shield float64 `json:"shield"`
	Slow   float64 `json:"slow"`
	in     Input
	send   chan []byte
	// bots : position au dernier "think" et temps restant en mode décoinçage
	lastX, lastY, unstick float64
	emoteT, emoteCool     float64
	// statistiques de la manche en cours
	passes, picks int
	held          float64
	victims       map[int]int
}

// Bonus : "speed" (vitesse x1.5), "shield" (impossible de recevoir la bombe), "ice" (ralentit tous les autres).
type Bonus struct {
	Kind string  `json:"kind"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

var bonusKinds = []string{"speed", "shield", "ice"}

const (
	BonusTime = 4.0 // durée de chaque effet
	MaxBonus  = 3
)

type Wall struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type Arena struct {
	Name  string
	Walls []Wall
}

// Les arènes tournent à chaque manche, dans cet ordre.
var arenas = []Arena{
	{"Plaine", nil},
	{"Piliers", []Wall{{190, 140, 60, 60}, {550, 140, 60, 60}, {190, 400, 60, 60}, {550, 400, 60, 60}, {370, 270, 60, 60}}},
	{"Croix", []Wall{{250, 280, 300, 40}, {380, 150, 40, 300}}},
	{"Couloirs", []Wall{{120, 180, 560, 24}, {120, 396, 560, 24}, {388, 0, 24, 110}, {388, 490, 24, 110}}},
	{"Labyrinthe", []Wall{{140, 120, 24, 200}, {140, 120, 180, 24}, {636, 280, 24, 200}, {480, 456, 180, 24}, {300, 300, 200, 24}, {480, 120, 24, 120}, {296, 400, 24, 120}}},
}

// Modes : "classic", "hot" (chaque passe à un nouveau joueur recharge la mèche), "teams" (2 équipes, pas de passe entre coéquipiers).
var modes = map[string]bool{"classic": true, "hot": true, "teams": true}

var teamNames = []string{"", "Rouges", "Bleus"}

const (
	MultiBomb = 6   // une 2e bombe dès qu'il reste au moins 6 joueurs en vie
	HotStart  = 6.0 // mode patate chaude : mèche de départ, recharge et plafond
	HotGain   = 1.5
	HotMax    = 8.0
)

type Bomb struct {
	Holder int     `json:"holder"`
	Fuse   float64 `json:"fuse"`
	from   int     // porteur précédent : pas de recharge sur un renvoi (patate chaude)
	cool   float64
}

// Event : ce qui s'est passé depuis le dernier envoi (fil d'actions, sons, effets).
type Event struct {
	Kind string  `json:"kind"` // pass, boom, pick, win, emote
	A    int     `json:"a,omitempty"`
	B    int     `json:"b,omitempty"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Text string  `json:"text,omitempty"`
}

type Stat struct {
	Title string `json:"title"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Game struct {
	mu      sync.Mutex
	Pin     string    `json:"pin"`
	Mode    string    `json:"mode"`
	Bonuses []Bonus   `json:"bonuses"`
	Players []*Player `json:"players"`
	Phase   string    `json:"phase"` // lobby, play, end
	Bombs   []*Bomb   `json:"bombs"`
	Wait    float64   `json:"wait"`
	Winner  string    `json:"winner"`
	WinTeam int       `json:"winTeam"`
	Arena   string    `json:"arena"`
	Walls   []Wall    `json:"walls"`
	Events  []Event   `json:"events"`
	Stats   []Stat    `json:"stats"`
	arena   int
	nextID  int
	spawn   float64 // secondes avant le prochain bonus
}

func newGame(pin, mode string) *Game {
	if !modes[mode] {
		mode = "classic"
	}
	return &Game{Pin: pin, Mode: mode, Phase: "lobby", Wait: 3, Arena: arenas[0].Name, Walls: arenas[0].Walls}
}

func (g *Game) event(e Event) { g.Events = append(g.Events, e) }

// hits dit si un cercle de rayon r en (x, y) touche un mur.
func (g *Game) hits(x, y, r float64) bool {
	for _, w := range g.Walls {
		if math.Hypot(x-min(max(x, w.X), w.X+w.W), y-min(max(y, w.Y), w.Y+w.H)) < r {
			return true
		}
	}
	return false
}

// freeSpot tire une position au hasard hors des murs.
func (g *Game) freeSpot(margin float64) (float64, float64) {
	for {
		x, y := margin+rand.Float64()*(W-2*margin), margin+rand.Float64()*(H-2*margin)
		if !g.hits(x, y, margin+4) {
			return x, y
		}
	}
}

// collide sort le joueur des murs (en glissant le long) et le garde dans l'arène.
func (g *Game) collide(p *Player) {
	for _, w := range g.Walls {
		cx, cy := min(max(p.X, w.X), w.X+w.W), min(max(p.Y, w.Y), w.Y+w.H)
		dx, dy := p.X-cx, p.Y-cy
		d := math.Hypot(dx, dy)
		switch {
		case d >= R:
		case d == 0: // centre dans le mur (vitesse < R par tick, n'arrive pas en jeu normal)
			p.X, p.Y = g.freeSpot(R)
		default:
			p.X, p.Y = p.X+dx/d*(R-d), p.Y+dy/d*(R-d)
		}
	}
	p.X, p.Y = min(max(p.X, R), W-R), min(max(p.Y, R), H-R)
}

func (g *Game) join(name string) *Player {
	used := map[string]bool{}
	for _, p := range g.Players {
		used[p.Color] = true
	}
	for _, c := range colors {
		if !used[c] {
			g.nextID++
			p := &Player{ID: g.nextID, Name: name, Color: c, Alive: g.Phase == "lobby", send: make(chan []byte, 4)}
			p.X, p.Y = g.freeSpot(R)
			g.Players = append(g.Players, p)
			return p
		}
	}
	return nil // partie pleine
}

func (g *Game) leave(id int) {
	for i, p := range g.Players {
		if p.ID == id {
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			break
		}
	}
	for _, p := range g.Players {
		if !p.Bot {
			return
		}
	}
	g.Players, g.Phase, g.Wait, g.Bombs = nil, "lobby", 3, nil // plus aucun humain : on retire les bots
}

func (g *Game) addBot() {
	n := 1
	for _, p := range g.Players {
		if p.Bot {
			n++
		}
	}
	if p := g.join("Bot " + strconv.Itoa(n)); p != nil {
		p.Bot, p.send = true, nil // envoi sur canal nil : le select prend default, rien n'est envoyé
	}
}

func (g *Game) removeBot() {
	for i := len(g.Players) - 1; i >= 0; i-- {
		if g.Players[i].Bot {
			g.leave(g.Players[i].ID)
			return
		}
	}
}

// think choisit les touches d'un bot : le porteur fonce sur le plus proche, les autres fuient le porteur.
func (g *Game) think(b *Player) {
	if b.unstick > 0 {
		return
	}
	moved := math.Hypot(b.X-b.lastX, b.Y-b.lastY)
	b.lastX, b.lastY = b.X, b.Y
	if moved < 1 && b.in != (Input{}) { // bloqué contre un mur : direction au hasard un instant
		dirs := []Input{{U: true}, {D: true}, {L: true}, {R: true}, {U: true, L: true}, {U: true, R: true}, {D: true, L: true}, {D: true, R: true}}
		b.in, b.unstick = dirs[rand.IntN(len(dirs))], 0.4+rand.Float64()*0.4
		return
	}
	if g.Phase != "play" {
		if rand.Float64() < 0.03 {
			b.in = Input{rand.Float64() < 0.3, rand.Float64() < 0.3, rand.Float64() < 0.3, rand.Float64() < 0.3}
		}
		return
	}
	var tx, ty float64
	best := math.Inf(1)
	if g.holds(b) != nil { // porteur : fonce sur la cible valide la plus proche
		for _, p := range g.Players {
			if d := math.Hypot(p.X-b.X, p.Y-b.Y); p != b && g.canReceive(b, p) && d < best {
				best, tx, ty = d, p.X-b.X, p.Y-b.Y
			}
		}
	} else { // sinon : fuit le porteur ennemi le plus proche
		for _, bo := range g.Bombs {
			h := g.get(bo.Holder)
			if h == nil || !g.canReceive(h, b) {
				continue
			}
			if d := math.Hypot(b.X-h.X, b.Y-h.Y); d < best && d > 0 {
				// fuite + attraction vers le centre pour ne pas rester coincé dans un coin
				best = d
				tx, ty = (b.X-h.X)/d+(W/2-b.X)/(W/2)*0.9, (b.Y-h.Y)/d+(H/2-b.Y)/(H/2)*0.9
			}
		}
	}
	d := math.Hypot(tx, ty)
	if d == 0 {
		return
	}
	tx, ty = tx/d, ty/d
	b.in = Input{U: ty < -0.3, D: ty > 0.3, L: tx < -0.3, R: tx > 0.3}
}

func (g *Game) get(id int) *Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (g *Game) alive() []*Player {
	var a []*Player
	for _, p := range g.Players {
		if p.Alive {
			a = append(a, p)
		}
	}
	return a
}

func (g *Game) holds(p *Player) *Bomb {
	for _, b := range g.Bombs {
		if b.Holder == p.ID {
			return b
		}
	}
	return nil
}

// canReceive dit si le porteur h peut passer une bombe à p.
func (g *Game) canReceive(h, p *Player) bool {
	return p != h && p.Alive && p.Shield == 0 && g.holds(p) == nil && (g.Mode != "teams" || p.Team != h.Team)
}

func (g *Game) fuse() float64 {
	if g.Mode == "hot" {
		return HotStart
	}
	return 8 + rand.Float64()*12
}

// over : manche finie quand il reste au plus un survivant (ou une seule équipe).
func (g *Game) over() bool {
	teams := map[int]bool{}
	a := g.alive()
	for _, p := range a {
		teams[p.Team] = true
	}
	return len(a) <= 1 || (g.Mode == "teams" && len(teams) <= 1)
}

// refill complète les bombes : 1, ou 2 s'il reste au moins MultiBomb joueurs en vie.
func (g *Game) refill() {
	a := g.alive()
	want := 1
	if len(a) >= MultiBomb {
		want = 2
	}
	for len(g.Bombs) < want {
		var free []*Player
		for _, p := range a {
			if g.holds(p) == nil {
				free = append(free, p)
			}
		}
		if len(free) == 0 {
			return
		}
		g.Bombs = append(g.Bombs, &Bomb{Holder: free[rand.IntN(len(free))].ID, Fuse: g.fuse(), cool: PassCooldown})
	}
}

func (g *Game) startRound() {
	order := rand.Perm(len(g.Players)) // équipes tirées au hasard, sans changer l'ordre d'affichage
	for i, p := range g.Players {
		p.Alive, p.Fast, p.Shield, p.Slow, p.Team = true, 0, 0, 0, 0
		p.passes, p.picks, p.held, p.victims = 0, 0, 0, nil
		if g.Mode == "teams" {
			p.Team = order[i]%2 + 1
		}
	}
	g.Phase, g.Bonuses, g.spawn, g.Bombs, g.Stats = "play", nil, 3, nil, nil
	g.refill()
}

func (g *Game) endRound() {
	a := g.alive()
	g.Phase, g.Bombs, g.Bonuses, g.Wait, g.Winner, g.WinTeam = "end", nil, nil, 7, "", 0
	switch {
	case g.Mode == "teams" && len(a) > 0:
		g.WinTeam, g.Winner = a[0].Team, teamNames[a[0].Team]
	case len(a) == 1:
		g.Winner = a[0].Name
	}
	for _, p := range a {
		p.Score += 2 // bonus de victoire (tous les survivants de l'équipe gagnante en mode équipes)
	}
	g.Stats = g.stats()
	g.event(Event{Kind: "win", Text: g.Winner})
}

// stats : les records de la manche, affichés à la fin.
func (g *Game) stats() []Stat {
	var out []Stat
	best := func(title string, val func(p *Player) float64, format string) {
		var top *Player
		for _, p := range g.Players {
			if val(p) > 0 && (top == nil || val(p) > val(top)) {
				top = p
			}
		}
		if top != nil {
			out = append(out, Stat{title, top.Name, fmt.Sprintf(format, val(top))})
		}
	}
	best("Le plus de passes", func(p *Player) float64 { return float64(p.passes) }, "%.0f passes")
	best("Accro à la bombe", func(p *Player) float64 { return p.held }, "%.1f s avec la bombe")
	best("Chasseur de bonus", func(p *Player) float64 { return float64(p.picks) }, "%.0f bonus")
	var from, to *Player
	n := 0
	for _, p := range g.Players {
		for id, k := range p.victims {
			if v := g.get(id); v != nil && (k > n || k == n && v.ID < to.ID) { // à égalité : résultat stable
				from, to, n = p, v, k
			}
		}
	}
	if from != nil {
		out = append(out, Stat{"Victime préférée", from.Name + " → " + to.Name, fmt.Sprintf("%d fois", n)})
	}
	return out
}

func (g *Game) emote(p *Player, n int) {
	if n < 1 || n > 4 || p.emoteCool > 0 {
		return
	}
	p.Emote, p.emoteT, p.emoteCool = n, 2, 0.8
	g.event(Event{Kind: "emote", A: p.ID, X: p.X, Y: p.Y, Text: strconv.Itoa(n)})
}

func (g *Game) step(dt float64) {
	for _, p := range g.Players {
		p.unstick = max(p.unstick-dt, 0)
		p.emoteCool = max(p.emoteCool-dt, 0)
		if p.emoteT = max(p.emoteT-dt, 0); p.emoteT == 0 {
			p.Emote = 0
		}
		if p.Bot && rand.Float64() < 0.25 { // ~8 décisions/s : un humain peut les battre
			g.think(p)
		}
		dx, dy := 0.0, 0.0
		if p.in.L {
			dx--
		}
		if p.in.R {
			dx++
		}
		if p.in.U {
			dy--
		}
		if p.in.D {
			dy++
		}
		p.Fast, p.Shield, p.Slow = max(p.Fast-dt, 0), max(p.Shield-dt, 0), max(p.Slow-dt, 0)
		if !p.Alive || (dx == 0 && dy == 0) {
			continue
		}
		s := Speed
		if g.holds(p) != nil {
			s *= 1.1 // petit avantage au porteur pour qu'il puisse rattraper
		}
		if p.Fast > 0 {
			s *= 1.5
		}
		if p.Slow > 0 {
			s *= 0.5
		}
		n := math.Hypot(dx, dy)
		p.X, p.Y = p.X+dx/n*s*dt, p.Y+dy/n*s*dt
		g.collide(p)
	}

	switch g.Phase {
	case "lobby":
		if len(g.Players) < 2 {
			g.Wait = 3
			return
		}
		if g.Wait -= dt; g.Wait <= 0 {
			g.startRound()
		}
	case "play":
		// bombe dont le porteur est parti en cours de manche : elle disparaît, refill en redonne une
		g.Bombs = slices.DeleteFunc(g.Bombs, func(b *Bomb) bool { h := g.get(b.Holder); return h == nil || !h.Alive })
		if g.over() {
			g.endRound()
			return
		}
		g.refill()
		g.bonuses(dt)
		for _, b := range slices.Clone(g.Bombs) {
			h := g.get(b.Holder)
			h.held += dt
			b.cool -= dt
			if b.Fuse -= dt; b.Fuse <= 0 {
				h.Alive = false
				g.Bombs = slices.DeleteFunc(g.Bombs, func(o *Bomb) bool { return o == b })
				g.event(Event{Kind: "boom", A: h.ID, X: h.X, Y: h.Y})
				for _, p := range g.alive() {
					p.Score++ // 1 point par survie à une explosion
				}
				continue
			}
			if b.cool > 0 {
				continue
			}
			for _, p := range g.Players {
				if !g.canReceive(h, p) || math.Hypot(p.X-h.X, p.Y-h.Y) >= 2*R {
					continue
				}
				if g.Mode == "hot" && p.ID != b.from {
					b.Fuse = min(b.Fuse+HotGain, HotMax)
				}
				b.from, b.Holder, b.cool = h.ID, p.ID, PassCooldown
				h.passes++
				if h.victims == nil {
					h.victims = map[int]int{}
				}
				h.victims[p.ID]++
				g.event(Event{Kind: "pass", A: h.ID, B: p.ID, X: p.X, Y: p.Y})
				break
			}
		}
		if g.over() {
			g.endRound()
			return
		}
		g.refill() // nouvelle bombe tout de suite après une explosion
	case "end":
		if g.Wait -= dt; g.Wait <= 0 {
			g.arena = (g.arena + 1) % len(arenas)
			g.Arena, g.Walls = arenas[g.arena].Name, arenas[g.arena].Walls
			for _, p := range g.Players {
				p.Alive, p.Team = true, 0
				p.X, p.Y = g.freeSpot(R)
			}
			g.Phase, g.Wait, g.Stats, g.Winner, g.WinTeam = "lobby", 3, nil, "", 0
		}
	}
}

func (g *Game) bonuses(dt float64) {
	if g.spawn -= dt; g.spawn <= 0 && len(g.Bonuses) < MaxBonus {
		g.spawn = 3 + rand.Float64()*3
		g.Bonuses = append(g.Bonuses, Bonus{bonusKinds[rand.IntN(len(bonusKinds))], 0, 0})
		b := &g.Bonuses[len(g.Bonuses)-1]
		b.X, b.Y = g.freeSpot(40)
	}
	for i := 0; i < len(g.Bonuses); i++ {
		b := g.Bonuses[i]
		for _, p := range g.Players {
			if !p.Alive || math.Hypot(p.X-b.X, p.Y-b.Y) > R+12 {
				continue
			}
			switch b.Kind {
			case "speed":
				p.Fast = BonusTime
			case "shield":
				p.Shield = BonusTime
			case "ice":
				for _, o := range g.Players {
					if o != p {
						o.Slow = BonusTime
					}
				}
			}
			p.picks++
			g.event(Event{Kind: "pick", A: p.ID, X: b.X, Y: b.Y, Text: b.Kind})
			g.Bonuses = append(g.Bonuses[:i], g.Bonuses[i+1:]...)
			i--
			break
		}
	}
}

// Salons : une partie par code PIN.
// Ordre des verrous : roomsMu puis g.mu, jamais l'inverse.
var (
	roomsMu sync.Mutex
	rooms   = map[string]*Game{}
)

// enter crée un salon (pin vide, avec le mode donné) ou rejoint celui du code donné.
func enter(pin, name, mode string) (*Game, *Player, string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	if pin == "" {
		if len(rooms) >= 1000 {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		pin = newPin(func(p string) bool { return rooms[p] != nil })
		g := newGame(pin, mode)
		rooms[pin] = g
		p := g.join(name) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go g.run()
		return g, p, ""
	}
	g := rooms[pin]
	if g == nil {
		return nil, nil, "Code inconnu"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if p := g.join(name); p != nil {
		return g, p, ""
	}
	return nil, nil, "Partie pleine (10 joueurs max)"
}

func (g *Game) run() {
	t := time.NewTicker(Tick)
	defer t.Stop()
	for range t.C {
		if g.closeIfEmpty() {
			return
		}
		g.mu.Lock()
		g.step(Tick.Seconds())
		b, _ := json.Marshal(g)
		// shortcut: un client lent qui saute une image perd aussi ses événements (son/effet manqué) ; acceptable pour un jeu entre amis
		g.Events = nil
		for _, p := range g.Players {
			select {
			case p.send <- b:
			default: // client lent : on saute cette image plutôt que de bloquer tout le monde
			}
		}
		g.mu.Unlock()
	}
}

func (g *Game) closeIfEmpty() bool {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.Players) > 0 {
		return false
	}
	delete(rooms, g.Pin)
	return true
}

func serveWS(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		g, p, why := enter(q.Get("pin"), name, q.Get("mode"))
		if p == nil {
			return nil, why
		}
		return &Seat{
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { g.mu.Lock(); g.leave(p.ID); g.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Input
					Bot   int `json:"bot"`   // +1 ajoute un bot, -1 en retire un
					Emote int `json:"emote"` // 1 à 4
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				g.mu.Lock()
				defer g.mu.Unlock()
				switch m.Bot {
				case 1:
					g.addBot()
				case -1:
					g.removeBot()
				}
				switch {
				case m.Emote != 0:
					g.emote(p, m.Emote)
				case m.Bot == 0:
					p.in = m.Input
				}
			},
		}, ""
	})
}

func main() {
	http.HandleFunc("/ws", serveWS)
	http.HandleFunc("/ws/ricochet", serveDuel)
	http.HandleFunc("/ws/golf", serveGolf)
	http.HandleFunc("/ws/dessin", serveSketch)
	http.HandleFunc("/ws/foot", serveFoot)
	http.HandleFunc("/admin/update", serveUpdate)
	http.HandleFunc("/admin/version", serveVersion)
	http.Handle("/", http.FileServerFS(files))
	addr := ":" + cmp.Or(os.Getenv("PORT"), "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           secure(http.DefaultServeMux),
		ReadHeaderTimeout: 10 * time.Second, // une connexion qui traîne ne bloque pas le serveur
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    16 << 10,
	}
	log.Println("Patate", version(), "sur", addr)
	log.Fatal(srv.ListenAndServe())
}
