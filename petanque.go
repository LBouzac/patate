// Pétanque : de 2 à 6 joueurs, chacun pour soi. Chacun a 3 boules par mène ; on les lance comme au golf
// depuis le cercle. Celui qui n'a pas le point rejoue. À la fin de la mène, celui qui a le point marque une
// boule par boule plus proche du cochonnet que la meilleure des autres. Premier à 7 points.
package main

import (
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
	PW, PH      = 800.0, 500.0
	BouleR      = 14.0
	JackR       = 7.0
	BouleDrag   = 420.0 // le gravier freine fort
	ThrowMax    = 900.0 // vitesse d'un lancer à pleine puissance (px/s)
	CircleX     = 70.0  // cercle de lancer
	CircleY     = PH / 2
	BoulesEach  = 3
	PetMax      = 6
	PetWin      = 7
	ThrowTime   = 15.0
	PetTick     = time.Second / 60
	PetSub      = 4
	PetRooms    = 1000
	PetRestitue = 0.6 // choc entre boules
)

type Boule struct {
	Body
	Owner int `json:"owner"` // id du joueur, 0 pour le cochonnet
}

type Bouliste struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Score int    `json:"score"`
	Left  int    `json:"left"` // boules qu'il lui reste à jouer dans la mène
	send  chan []byte
}

type Pitch struct {
	mu      sync.Mutex
	Pin     string      `json:"pin"`
	Players []*Bouliste `json:"players"`
	Boules  []*Boule    `json:"boules"`
	Jack    *Boule      `json:"jack"`
	Phase   string      `json:"phase"` // wait, aim, roll, mene, end
	Wait    float64     `json:"wait"`
	Turn    int         `json:"turn"`  // id de celui qui lance
	Point   int         `json:"point"` // id de celui qui a le point
	Mene    int         `json:"mene"`
	Winner  string      `json:"winner"`
	Events  []Event     `json:"events"`
	starter int         // qui lance en premier à la prochaine mène
	nextID  int
}

func newPitch(pin string) *Pitch { return &Pitch{Pin: pin, Phase: "wait"} }

func (g *Pitch) event(e Event) { g.Events = append(g.Events, e) }

func (g *Pitch) get(id int) *Bouliste {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (g *Pitch) join(name string) *Bouliste {
	if len(g.Players) >= PetMax {
		return nil
	}
	used := map[string]bool{}
	for _, p := range g.Players {
		used[p.Color] = true
	}
	color := colors[0]
	for _, c := range colors {
		if !used[c] {
			color = c
			break
		}
	}
	g.nextID++
	p := &Bouliste{ID: g.nextID, Name: name, Color: color, send: make(chan []byte, 8)}
	g.Players = append(g.Players, p) // il joue à partir de la prochaine mène
	if g.Phase == "wait" && len(g.Players) >= 2 {
		g.Phase, g.Wait = "mene", 3 // petite attente puis première mène
	}
	return p
}

func (g *Pitch) leave(id int) {
	g.Players = slices.DeleteFunc(g.Players, func(p *Bouliste) bool { return p.ID == id })
	g.Boules = slices.DeleteFunc(g.Boules, func(b *Boule) bool { return b.Owner == id })
	if len(g.Players) < 2 {
		g.Phase, g.Wait, g.Boules, g.Jack = "wait", 0, nil, nil
		for _, p := range g.Players {
			p.Score, p.Left = 0, 0
		}
		return
	}
	g.Point = g.closest()
	if g.Phase == "aim" && g.Turn == id {
		g.next()
	}
}

func (g *Pitch) newMene() {
	if g.get(g.starter) == nil {
		g.starter = g.Players[rand.IntN(len(g.Players))].ID
	}
	g.Mene++
	g.Boules, g.Point = nil, 0
	g.Jack = &Boule{Body: Body{X: 450 + rand.Float64()*270, Y: 80 + rand.Float64()*(PH-160), r: JackR, m: 0.3, drag: BouleDrag}}
	for _, p := range g.Players {
		p.Left = BoulesEach
	}
	g.Turn, g.Phase, g.Wait = g.starter, "aim", ThrowTime
	g.event(Event{Kind: "mene", X: g.Jack.X, Y: g.Jack.Y})
}

func (g *Pitch) throw(p *Bouliste, angle, power float64) {
	if g.Phase != "aim" || g.Turn != p.ID || p.Left <= 0 || !(power > 0) || math.IsNaN(angle) || math.IsInf(angle, 0) {
		return
	}
	b := &Boule{Body: Body{X: CircleX, Y: CircleY, r: BouleR, m: 1, drag: BouleDrag}, Owner: p.ID}
	b.launch(angle, min(power, 1)*ThrowMax)
	g.Boules = append(g.Boules, b)
	p.Left--
	g.Phase = "roll"
	g.event(Event{Kind: "throw", A: p.ID, X: b.X, Y: b.Y})
}

// dist : distance au cochonnet de la meilleure boule du joueur (+Inf s'il n'en a pas sur le terrain).
func (g *Pitch) dist(id int) float64 {
	best := math.Inf(1)
	for _, b := range g.Boules {
		if b.Owner == id {
			best = min(best, math.Hypot(b.X-g.Jack.X, b.Y-g.Jack.Y))
		}
	}
	return best
}

func (g *Pitch) closest() int {
	best, who := math.Inf(1), 0
	if g.Jack == nil {
		return 0
	}
	for _, p := range g.Players {
		if d := g.dist(p.ID); d < best {
			best, who = d, p.ID
		}
	}
	return who
}

// next : celui qui n'a pas le point rejoue (le plus loin d'abord) ; s'il ne reste des boules qu'à celui qui l'a, il joue.
func (g *Pitch) next() {
	g.Point = g.closest()
	var pick *Bouliste
	for _, p := range g.Players {
		if p.Left == 0 || (p.ID == g.Point && g.Point != 0) {
			continue
		}
		if pick == nil || g.dist(p.ID) > g.dist(pick.ID) {
			pick = p
		}
	}
	if pick == nil {
		if pt := g.get(g.Point); pt != nil && pt.Left > 0 {
			pick = pt
		}
	}
	if pick == nil {
		g.score()
		return
	}
	g.Turn, g.Phase, g.Wait = pick.ID, "aim", ThrowTime
}

func (g *Pitch) score() {
	g.Point = g.closest()
	p := g.get(g.Point)
	if p == nil { // personne n'a de boule sur le terrain : mène nulle
		g.Phase, g.Wait = "mene", 3
		return
	}
	other := math.Inf(1)
	for _, o := range g.Players {
		if o.ID != p.ID {
			other = min(other, g.dist(o.ID))
		}
	}
	pts := 0
	for _, b := range g.Boules {
		if b.Owner == p.ID && math.Hypot(b.X-g.Jack.X, b.Y-g.Jack.Y) < other {
			pts++
		}
	}
	p.Score += pts
	g.starter = p.ID
	g.event(Event{Kind: "point", A: p.ID, B: pts, X: g.Jack.X, Y: g.Jack.Y, Text: p.Name})
	if p.Score >= PetWin {
		g.Phase, g.Wait, g.Winner = "end", 8, p.Name
		g.event(Event{Kind: "win", A: p.ID, Text: p.Name})
		return
	}
	g.Phase, g.Wait = "mene", 4
}

func (g *Pitch) step(dt float64) {
	switch g.Phase {
	case "aim":
		if g.Wait -= dt; g.Wait <= 0 { // trop long : la boule est perdue
			if p := g.get(g.Turn); p != nil && p.Left > 0 {
				p.Left--
				g.event(Event{Kind: "lost", A: p.ID})
			}
			g.next()
		}
	case "roll":
		moving := false
		for range PetSub {
			if moving = g.physics(dt / PetSub); g.Phase != "roll" {
				return
			}
		}
		if !moving {
			g.next()
		}
	case "mene":
		if g.Wait -= dt; g.Wait <= 0 {
			g.newMene()
		}
	case "end":
		if g.Wait -= dt; g.Wait <= 0 { // nouvelle partie
			g.Winner = ""
			for _, p := range g.Players {
				p.Score = 0
			}
			g.newMene()
		}
	}
}

// physics : un sous-pas. Une boule qui sort du terrain est morte ; si c'est le cochonnet, la mène est rejouée.
func (g *Pitch) physics(dt float64) bool {
	all := append([]*Boule{g.Jack}, g.Boules...)
	moving := false
	for _, b := range all {
		b.roll(dt)
		moving = moving || b.moving()
	}
	out := func(b *Boule) bool { return b.X < -b.r || b.X > PW+b.r || b.Y < -b.r || b.Y > PH+b.r }
	if out(g.Jack) {
		g.event(Event{Kind: "jackout", X: g.Jack.X, Y: g.Jack.Y})
		g.Phase, g.Wait = "mene", 3 // même lanceur
		return false
	}
	g.Boules = slices.DeleteFunc(g.Boules, func(b *Boule) bool {
		if out(b) {
			g.event(Event{Kind: "out", A: b.Owner, X: b.X, Y: b.Y})
			return true
		}
		return false
	})
	all = append([]*Boule{g.Jack}, g.Boules...)
	for i, a := range all {
		for _, b := range all[i+1:] {
			if v := collide(&a.Body, &b.Body, PetRestitue); v > 40 {
				g.event(Event{Kind: "clack", A: int(min(v, 2000)), X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2})
			}
		}
	}
	return moving
}

// --- salons Pétanque (même principe que les autres jeux, codes séparés) ---

var (
	pitchesMu sync.Mutex
	pitches   = map[string]*Pitch{}
)

func enterPitch(pin, name string) (*Pitch, *Bouliste, string) {
	pitchesMu.Lock()
	defer pitchesMu.Unlock()
	if pin == "" {
		if len(pitches) >= PetRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		g := newPitch(newPin(func(p string) bool { return pitches[p] != nil }))
		pitches[g.Pin] = g
		p := g.join(name) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go g.run()
		return g, p, ""
	}
	g := pitches[pin]
	if g == nil {
		return nil, nil, "Code inconnu"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if p := g.join(name); p != nil {
		return g, p, ""
	}
	return nil, nil, "Partie pleine (6 joueurs)"
}

func (g *Pitch) run() {
	t := time.NewTicker(PetTick)
	defer t.Stop()
	for range t.C {
		if g.closeIfEmpty() {
			return
		}
		g.mu.Lock()
		g.step(PetTick.Seconds())
		b, _ := json.Marshal(g)
		g.Events = nil
		for _, p := range g.Players {
			select {
			case p.send <- b:
			default: // client lent : on saute cette image
			}
		}
		g.mu.Unlock()
	}
}

func (g *Pitch) closeIfEmpty() bool {
	pitchesMu.Lock()
	defer pitchesMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.Players) > 0 {
		return false
	}
	delete(pitches, g.Pin)
	return true
}

func servePetanque(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		g, p, why := enterPitch(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { g.mu.Lock(); g.leave(p.ID); g.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					A float64 `json:"a"`
					P float64 `json:"p"`
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				g.mu.Lock()
				g.throw(p, m.A, m.P)
				g.mu.Unlock()
			},
		}, ""
	})
}
