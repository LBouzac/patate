// Golf : mini-golf à plusieurs, chacun joue sa balle en même temps sur le même trou.
// Chaque trou est généré au hasard (murs, bunkers, étangs) puis vérifié jouable. 9 trous, le plus bas par rapport au par gagne.
package main

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	GW, GH     = 800.0, 500.0
	GBallR     = 6.0
	CupR       = 11.0
	ShotMax    = 900.0 // vitesse d'un coup à pleine puissance (px/s)
	CupSpeed   = 420.0 // plus vite, la balle saute par-dessus le trou
	Roll       = 170.0 // décélération sur l'herbe (px/s²)
	SandRoll   = 1100.0
	Bounce     = 0.75
	MaxStrokes = 10 // atteint sans finir le trou : balle ramassée, compte 10
	Holes      = 9
	HoleTime   = 90.0
	MaxGolfers = 8
	GolfTick   = time.Second / 60
	RollSub    = 4 // sous-pas par image : moins de GBallR par sous-pas, une balle ne traverse ni mur ni autre balle
)

type Circle struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	R float64 `json:"r"`
}

type Hole struct {
	Num   int      `json:"num"`
	Par   int      `json:"par"`
	TeeX  float64  `json:"teeX"`
	TeeY  float64  `json:"teeY"`
	CupX  float64  `json:"cupX"`
	CupY  float64  `json:"cupY"`
	Walls []Wall   `json:"walls"`
	Sand  []Circle `json:"sand"`
	Water []Circle `json:"water"`
}

type Golfer struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Color   string  `json:"color"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Moving  bool    `json:"moving"`
	Strokes int     `json:"strokes"` // trou en cours
	Done    bool    `json:"done"`
	ToPar   int     `json:"toPar"` // total de la partie par rapport au par
	Card    []int   `json:"card"`  // coups par trou, 0 = trou pas joué (arrivé en cours de partie)
	vx, vy  float64
	knocked bool    // mise en mouvement par une autre balle : pas de pénalité si elle finit dans l'eau
	fromX   float64 // position avant le coup : on y revient après un plongeon dans l'eau
	fromY   float64
	send    chan []byte
	seen    int // dernier trou (holeID) reçu par ce joueur
}

type Course struct {
	mu      sync.Mutex
	Pin     string    `json:"pin"`
	Hole    Hole      `json:"hole"`
	Golfers []*Golfer `json:"golfers"`
	Phase   string    `json:"phase"` // play, hole (carte de score), end
	Wait    float64   `json:"wait"`  // play : temps restant sur le trou
	Winner  string    `json:"winner"`
	Events  []Event   `json:"events"`
	nextID  int
	rng     *rand.Rand
	holeID  int // change à chaque nouveau trou
}

func inAny(cs []Circle, x, y, r float64) bool {
	for _, c := range cs {
		if math.Hypot(x-c.X, y-c.Y) < c.R+r {
			return true
		}
	}
	return false
}

func hitsWall(ws []Wall, x, y, r float64) bool {
	for _, w := range ws {
		if math.Hypot(x-min(max(x, w.X), w.X+w.W), y-min(max(y, w.Y), w.Y+w.H)) < r {
			return true
		}
	}
	return false
}

// genHole tire des trous au hasard jusqu'à en trouver un jouable mais difficile : beaucoup de murs, du sable
// et de l'eau sur chaque trou, un obstacle qui garde le drapeau, et jamais de coup direct possible.
func genHole(r *rand.Rand, num int) Hole {
	between := func(a, b float64) float64 { return a + r.Float64()*(b-a) }
	for {
		// listes jamais nil : en JSON, null au lieu de [] fait planter le dessin du terrain dans le navigateur
		h := Hole{Num: num, TeeX: between(50, 110), TeeY: between(50, GH-50), CupX: between(GW-150, GW-50), CupY: between(50, GH-50),
			Walls: []Wall{}, Sand: []Circle{}, Water: []Circle{}}
		if r.IntN(2) == 0 { // départ à droite une fois sur deux
			h.TeeX, h.CupX = GW-h.TeeX, GW-h.CupX
		}
		// point au hasard le long du chemin départ → trou (un peu à côté) : c'est là que les obstacles gênent
		onPath := func(off float64) (float64, float64) {
			t, o := between(0.2, 0.8), between(-off, off)
			dx, dy := h.CupX-h.TeeX, h.CupY-h.TeeY
			l := math.Hypot(dx, dy)
			return h.TeeX + dx*t - dy/l*o, h.TeeY + dy*t + dx/l*o
		}
		for i := range 4 + r.IntN(4) {
			var w Wall
			switch r.IntN(3) {
			case 0: // barre horizontale
				w.W, w.H = between(120, 300), 16
			case 1: // barre verticale
				w.W, w.H = 16, between(120, 300)
			default: // bloc
				w.W, w.H = between(36, 90), between(36, 90)
			}
			w.X, w.Y = between(150, GW-150-w.W), between(0, GH-w.H)
			if i%2 == 0 { // un mur sur deux en travers du chemin
				x, y := onPath(110)
				w.X, w.Y = min(max(x-w.W/2, 150), GW-150-w.W), min(max(y-w.H/2, 0), GH-w.H)
			}
			h.Walls = append(h.Walls, w)
		}
		blob := func(x, y float64) []Circle { // quelques cercles qui se chevauchent : une forme plus naturelle
			cs := make([]Circle, 2+r.IntN(3))
			for i := range cs {
				cs[i] = Circle{x + between(-40, 40), y + between(-30, 30), between(22, 45)}
			}
			return cs
		}
		anywhere := func() (float64, float64) { return between(170, GW-170), between(40, GH-40) }
		for range 1 + r.IntN(2) {
			h.Sand = append(h.Sand, blob(anywhere())...)
		}
		h.Water = append(h.Water, blob(onPath(90))...) // un étang sur le chemin
		for range r.IntN(2) {
			h.Water = append(h.Water, blob(anywhere())...)
		}
		// garde du drapeau : un bunker ou un bloc juste à côté, du côté d'où vient la balle une fois sur deux
		a := math.Atan2(h.TeeY-h.CupY, h.TeeX-h.CupX) + between(-0.8, 0.8)
		if r.IntN(2) == 0 {
			a = between(0, 2*math.Pi)
		}
		if r.IntN(2) == 0 {
			for range 2 {
				d, rr := between(46, 58), between(16, 24)
				b := a + between(-0.5, 0.5)
				h.Sand = append(h.Sand, Circle{h.CupX + math.Cos(b)*d, h.CupY + math.Sin(b)*d, rr})
			}
		} else {
			w := Wall{W: 16, H: 70}
			if math.Abs(math.Cos(a)) < 0.7 { // drapeau gardé par-dessus ou par-dessous : barre horizontale
				w.W, w.H = 70, 16
			}
			w.X, w.Y = h.CupX+math.Cos(a)*48-w.W/2, h.CupY+math.Sin(a)*48-w.H/2
			h.Walls = append(h.Walls, w)
		}
		if h.direct() {
			continue
		}
		if par, ok := h.route(); ok {
			h.Par = par
			return h
		}
	}
}

// direct : la balle peut-elle aller droit du départ au trou sans toucher mur ni eau ?
func (h *Hole) direct() bool {
	dx, dy := h.CupX-h.TeeX, h.CupY-h.TeeY
	n := int(math.Hypot(dx, dy) / 4)
	for i := 0; i <= n; i++ {
		x, y := h.TeeX+dx*float64(i)/float64(n), h.TeeY+dy*float64(i)/float64(n)
		if hitsWall(h.Walls, x, y, GBallR) || inAny(h.Water, x, y, GBallR) {
			return false
		}
	}
	return true
}

// route vérifie qu'une balle peut aller du départ au trou sans traverser mur ni eau (parcours en largeur
// sur une grille) et en déduit le par selon la longueur du chemin.
func (h *Hole) route() (int, bool) {
	clear := func(x, y, m float64) bool {
		return !hitsWall(h.Walls, x, y, m) && !inAny(h.Water, x, y, m) && x > m && x < GW-m && y > m && y < GH-m
	}
	// départ et trou bien dégagés, et pas dans le sable
	if !clear(h.TeeX, h.TeeY, 24) || !clear(h.CupX, h.CupY, 24) || inAny(h.Sand, h.TeeX, h.TeeY, 16) || inAny(h.Sand, h.CupX, h.CupY, 16) {
		return 0, false
	}
	const step = 8.0
	type pt struct{ x, y int }
	cell := func(x, y float64) pt { return pt{int(math.Round(x / step)), int(math.Round(y / step))} }
	start, goal := cell(h.TeeX, h.TeeY), cell(h.CupX, h.CupY)
	dist, queue := map[pt]int{start: 0}, []pt{start}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == goal {
			// 4-voisins : longueur en « distance de Manhattan », un peu plus longue que le vrai chemin
			return min(max(2+int(float64(dist[p])*step/380), 2), 5), true
		}
		for _, n := range []pt{{p.x + 1, p.y}, {p.x - 1, p.y}, {p.x, p.y + 1}, {p.x, p.y - 1}} {
			if _, seen := dist[n]; !seen && clear(float64(n.x)*step, float64(n.y)*step, GBallR+2) {
				dist[n] = dist[p] + 1
				queue = append(queue, n)
			}
		}
	}
	return 0, false
}

func newCourse(pin string) *Course {
	c := &Course{Pin: pin, rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
	c.newGame()
	return c
}

func (c *Course) event(e Event) { c.Events = append(c.Events, e) }

func (c *Course) newGame() {
	c.Winner = ""
	for _, g := range c.Golfers {
		g.ToPar, g.Card = 0, nil
	}
	c.nextHole(1)
}

func (c *Course) nextHole(num int) {
	c.Hole = genHole(c.rng, num)
	c.holeID++
	c.Phase, c.Wait = "play", HoleTime
	for _, g := range c.Golfers {
		c.tee(g)
	}
	c.event(Event{Kind: "hole", X: c.Hole.TeeX, Y: c.Hole.TeeY})
}

func (c *Course) tee(g *Golfer) {
	g.X, g.Y, g.vx, g.vy, g.Moving, g.Strokes, g.Done = c.Hole.TeeX, c.Hole.TeeY, 0, 0, false, 0, false
}

func (c *Course) join(name string) *Golfer {
	if len(c.Golfers) >= MaxGolfers {
		return nil
	}
	used := map[string]bool{}
	for _, g := range c.Golfers {
		used[g.Color] = true
	}
	color := colors[0]
	for _, col := range colors {
		if !used[col] {
			color = col
			break
		}
	}
	c.nextID++
	g := &Golfer{ID: c.nextID, Name: name, Color: color, send: make(chan []byte, 8)}
	c.tee(g)
	played := c.Hole.Num - 1 // trous déjà joués : marqués « pas joué » sur sa carte
	if c.Phase != "play" {
		played, g.Done = c.Hole.Num, true // trou déjà fini : il attend le suivant
	}
	g.Card = make([]int, played)
	c.Golfers = append(c.Golfers, g)
	return g
}

func (c *Course) leave(id int) {
	for i, g := range c.Golfers {
		if g.ID == id {
			c.Golfers = append(c.Golfers[:i], c.Golfers[i+1:]...)
			return
		}
	}
}

// shoot : angle en radians, power entre 0 et 1. Ignoré si la balle roule encore ou si le trou est fini.
func (c *Course) shoot(g *Golfer, angle, power float64) {
	if c.Phase != "play" || g.Done || g.Moving || !(power > 0) || math.IsInf(angle, 0) || math.IsNaN(angle) {
		return
	}
	v := min(power, 1) * ShotMax
	g.fromX, g.fromY = g.X, g.Y
	g.vx, g.vy, g.Moving, g.knocked = v*math.Cos(angle), v*math.Sin(angle), true, false
	g.Strokes++
	c.event(Event{Kind: "shot", A: g.ID, X: g.X, Y: g.Y})
}

func (c *Course) step(dt float64) {
	switch c.Phase {
	case "play":
		c.Wait -= dt
		for range RollSub {
			for _, g := range c.Golfers {
				if g.Moving {
					c.roll(g, dt/RollSub)
				}
			}
			c.collide()
		}
		finished := true
		for _, g := range c.Golfers {
			if !g.Moving && !g.Done && g.Strokes >= MaxStrokes {
				g.Done = true
				c.event(Event{Kind: "pickup", A: g.ID, X: g.X, Y: g.Y})
			}
			finished = finished && g.Done
		}
		if (finished && len(c.Golfers) > 0) || c.Wait <= 0 {
			c.endHole()
		}
	case "hole", "end":
		if c.Wait -= dt; c.Wait <= 0 {
			if c.Phase == "end" {
				c.newGame()
			} else {
				c.nextHole(c.Hole.Num + 1)
			}
		}
	}
}

func (c *Course) endHole() {
	for _, g := range c.Golfers {
		if !g.Done { // temps écoulé
			g.Strokes = MaxStrokes
		}
		g.Done, g.Moving = true, false
		g.ToPar += g.Strokes - c.Hole.Par
		g.Card = append(g.Card, g.Strokes)
	}
	c.Phase, c.Wait = "hole", 5
	if c.Hole.Num < Holes || len(c.Golfers) == 0 {
		return
	}
	best := c.Golfers[0]
	for _, g := range c.Golfers {
		if g.ToPar < best.ToPar {
			best = g
		}
	}
	c.Phase, c.Wait, c.Winner = "end", 10, best.Name
	c.event(Event{Kind: "win", A: best.ID, Text: best.Name})
}

// roll fait avancer une balle d'un sous-pas.
func (c *Course) roll(g *Golfer, dt float64) {
	h := &c.Hole
	sp := math.Hypot(g.vx, g.vy)
	dec := Roll
	if inAny(h.Sand, g.X, g.Y, 0) {
		dec = SandRoll
	}
	if ns := sp - dec*dt; ns > 0 {
		g.vx, g.vy = g.vx*ns/sp, g.vy*ns/sp
	} else {
		g.vx, g.vy, g.Moving = 0, 0, false
	}
	g.X, g.Y = g.X+g.vx*dt, g.Y+g.vy*dt

	// bords du terrain
	if g.X < GBallR || g.X > GW-GBallR {
		lim := min(max(g.X, GBallR), GW-GBallR)
		g.X, g.vx = 2*lim-g.X, -g.vx*Bounce
		c.event(Event{Kind: "wall", X: g.X, Y: g.Y})
	}
	if g.Y < GBallR || g.Y > GH-GBallR {
		lim := min(max(g.Y, GBallR), GH-GBallR)
		g.Y, g.vy = 2*lim-g.Y, -g.vy*Bounce
		c.event(Event{Kind: "wall", X: g.X, Y: g.Y})
	}
	for _, w := range h.Walls {
		cx, cy := min(max(g.X, w.X), w.X+w.W), min(max(g.Y, w.Y), w.Y+w.H)
		dx, dy := g.X-cx, g.Y-cy
		d := math.Hypot(dx, dy)
		if d >= GBallR || d == 0 {
			continue
		}
		nx, ny := dx/d, dy/d
		g.X, g.Y = cx+nx*GBallR, cy+ny*GBallR
		if vn := g.vx*nx + g.vy*ny; vn < 0 {
			g.vx, g.vy = g.vx-(1+Bounce)*vn*nx, g.vy-(1+Bounce)*vn*ny
			c.event(Event{Kind: "wall", X: cx, Y: cy})
		}
	}

	if math.Hypot(g.X-h.CupX, g.Y-h.CupY) < CupR && sp < CupSpeed {
		g.X, g.Y, g.vx, g.vy, g.Moving, g.Done = h.CupX, h.CupY, 0, 0, false, true
		c.event(Event{Kind: "cup", A: g.ID, X: h.CupX, Y: h.CupY})
		return
	}
	if inAny(h.Water, g.X, g.Y, 0) {
		c.event(Event{Kind: "splash", A: g.ID, X: g.X, Y: g.Y})
		g.X, g.Y, g.vx, g.vy, g.Moving = g.fromX, g.fromY, 0, 0, false
		if !g.knocked {
			g.Strokes++ // pénalité
		}
		return
	}
}

// collide : choc entre deux balles de même masse, elles échangent leur vitesse le long de l'axe des centres.
// Une balle à l'arrêt qui est touchée se met à rouler.
func (c *Course) collide() {
	for i, a := range c.Golfers {
		for _, b := range c.Golfers[i+1:] {
			if a.Done || b.Done || (!a.Moving && !b.Moving) {
				continue
			}
			dx, dy := b.X-a.X, b.Y-a.Y
			d := math.Hypot(dx, dy)
			if d >= 2*GBallR || d == 0 {
				continue
			}
			nx, ny := dx/d, dy/d
			vn := (a.vx-b.vx)*nx + (a.vy-b.vy)*ny
			if vn <= 0 {
				continue // elles s'écartent déjà (balles serrées au départ)
			}
			for _, g := range []*Golfer{a, b} {
				if !g.Moving {
					g.Moving, g.knocked, g.fromX, g.fromY = true, true, g.X, g.Y
				}
			}
			a.vx, a.vy, b.vx, b.vy = a.vx-vn*nx, a.vy-vn*ny, b.vx+vn*nx, b.vy+vn*ny
			push := (2*GBallR - d) / 2 // on les décolle
			a.X, a.Y, b.X, b.Y = a.X-nx*push, a.Y-ny*push, b.X+nx*push, b.Y+ny*push
			c.event(Event{Kind: "clack", X: a.X + nx*GBallR, Y: a.Y + ny*GBallR})
		}
	}
}

// --- salons Golf (même principe que les autres jeux, codes séparés) ---

var (
	coursesMu sync.Mutex
	courses   = map[string]*Course{}
)

func enterCourse(pin, name string) (*Course, *Golfer, string) {
	coursesMu.Lock()
	defer coursesMu.Unlock()
	if pin == "" {
		if len(courses) >= 1000 {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		c := newCourse(newPin(func(p string) bool { return courses[p] != nil }))
		courses[c.Pin] = c
		g := c.join(name) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go c.run()
		return c, g, ""
	}
	c := courses[pin]
	if c == nil {
		return nil, nil, "Code inconnu"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if g := c.join(name); g != nil {
		return c, g, ""
	}
	return nil, nil, "Partie pleine (8 joueurs)"
}

// broadcast envoie l'état à chacun. Le trou (murs, sable, eau : ~1,4 Ko) ne part qu'une fois par joueur et par trou,
// pas à chaque image : c'était l'essentiel du débit (90 Ko/s par joueur).
func (c *Course) broadcast() {
	type view struct {
		*Course
		Hole *Hole `json:"hole,omitempty"`
	}
	light, _ := json.Marshal(view{c, nil})
	var full []byte
	c.Events = nil
	for _, g := range c.Golfers {
		b, withHole := light, g.seen != c.holeID
		if withHole {
			if full == nil {
				full, _ = json.Marshal(view{c, &c.Hole})
			}
			b = full
		}
		select {
		case g.send <- b:
			if withHole {
				g.seen = c.holeID // marqué reçu seulement si le message est vraiment parti
			}
		default: // client lent : on saute cette image
		}
	}
}

func (c *Course) run() {
	t := time.NewTicker(GolfTick)
	defer t.Stop()
	tick := 0
	for range t.C {
		if c.closeIfEmpty() {
			return
		}
		c.mu.Lock()
		c.step(GolfTick.Seconds())
		if tick++; tick%2 == 0 { // physique à 60 images/s, envoi à 30 : le navigateur lisse entre deux états
			c.broadcast()
		}
		c.mu.Unlock()
	}
}

func (c *Course) closeIfEmpty() bool {
	coursesMu.Lock()
	defer coursesMu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.Golfers) > 0 {
		return false
	}
	delete(courses, c.Pin)
	return true
}

func serveGolf(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		c, g, why := enterCourse(q.Get("pin"), name)
		if g == nil {
			return nil, why
		}
		return &Seat{
			ID:    g.ID,
			Send:  g.send,
			Leave: func() { c.mu.Lock(); c.leave(g.ID); c.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					A float64 `json:"a"` // angle (radians)
					P float64 `json:"p"` // puissance 0..1
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				c.mu.Lock()
				c.shoot(g, m.A, m.P)
				c.mu.Unlock()
			},
		}, ""
	})
}
