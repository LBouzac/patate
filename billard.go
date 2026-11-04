// Billard : 2 joueurs (ou contre un bot). Le premier joue les jaunes, le second les rouges. On vise avec la
// blanche comme au golf. Rentrer une de ses billes fait rejouer ; une faute (blanche rentrée, rien touché, ou
// d'abord une bille adverse) passe la main. On gagne en rentrant la noire après toutes ses billes.
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
	TW, TH      = 800.0, 400.0
	PoolR       = 11.0
	PocketR     = 22.0 // centre de la bille à moins de ça d'une poche : elle tombe
	CueMax      = 1300.0
	PoolDrag    = 50.0 // frottement fixe, plus PoolFluid × vitesse : même durée qu'avant, mais l'arrêt est doux
	PoolFluid   = 0.12
	Cushion     = 0.8
	PoolRestit  = 0.95
	PoolTick    = time.Second / 60
	PoolSub     = 4
	PoolTurn    = 25.0
	PoolRooms   = 1000
	PoolBotWait = 1.2
	HeadX       = 200.0 // la blanche repart d'ici
)

// sortes de billes
const (
	Cue = iota
	Yellow
	Red
	Black
)

var pockets = [][2]float64{{0, 0}, {TW / 2, 0}, {TW, 0}, {0, TH}, {TW / 2, TH}, {TW, TH}}

type PoolBall struct {
	Body
	Kind int `json:"k"`
}

type Cueist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Bot  bool   `json:"bot"`
	send chan []byte
}

type Table struct {
	mu      sync.Mutex
	Pin     string      `json:"pin"`
	Players [2]*Cueist  `json:"players"` // 0 : jaunes, 1 : rouges
	Balls   []*PoolBall `json:"balls"`   // la blanche est toujours la première
	Phase   string      `json:"phase"`   // wait, aim, roll, end
	Wait    float64     `json:"wait"`
	Turn    int         `json:"turn"`
	Winner  string      `json:"winner"`
	Last    string      `json:"last"` // ce qui vient de se passer (faute, rejoue…)
	Events  []Event     `json:"events"`
	nextID  int
	// pendant un coup
	firstHit       int
	potted         []int
	cueIn, blackIn bool
}

func newTable(pin string) *Table {
	t := &Table{Pin: pin}
	t.reset()
	return t
}

func (t *Table) event(e Event) { t.Events = append(t.Events, e) }

func ball(kind int, x, y float64) *PoolBall {
	return &PoolBall{Body: Body{X: x, Y: y, r: PoolR, m: 1, drag: PoolDrag, fluid: PoolFluid}, Kind: kind}
}

// rack : triangle de 15 billes, la noire au milieu de la 3e rangée, jaunes et rouges mélangées.
func (t *Table) rack() {
	t.Balls = []*PoolBall{ball(Cue, HeadX, TH/2)}
	kinds := []int{Yellow, Red, Yellow, Yellow, Red, Red, Yellow, Red, Red, Yellow, Red, Yellow, Red, Yellow}
	k := 0
	for row := range 5 {
		for i := 0; i <= row; i++ {
			x := 580 + float64(row)*PoolR*1.75
			y := TH/2 + (float64(i)-float64(row)/2)*PoolR*2.02
			if row == 2 && i == 1 {
				t.Balls = append(t.Balls, ball(Black, x, y))
				continue
			}
			t.Balls = append(t.Balls, ball(kinds[k], x, y))
			k++
		}
	}
}

func (t *Table) reset() {
	t.Winner, t.Last, t.Phase, t.Wait = "", "", "wait", 0
	t.rack()
	if t.Players[0] != nil && t.Players[1] != nil {
		t.Turn, t.Phase, t.Wait = rand.IntN(2), "aim", PoolTurn
	}
}

func (t *Table) join(name string, bot bool) *Cueist {
	slot := -1
	for i, p := range t.Players {
		if p == nil {
			slot = i
			break
		}
	}
	if slot < 0 && !bot { // un humain prend la place du bot
		for i, p := range t.Players {
			if p.Bot {
				slot = i
				break
			}
		}
	}
	if slot < 0 {
		return nil
	}
	t.nextID++
	p := &Cueist{ID: t.nextID, Name: name, Bot: bot}
	if !bot {
		p.send = make(chan []byte, 8)
	}
	t.Players[slot] = p
	t.reset()
	return p
}

func (t *Table) leave(id int) {
	humans := 0
	for i, p := range t.Players {
		if p != nil && p.ID == id {
			t.Players[i] = nil
		} else if p != nil && !p.Bot {
			humans++
		}
	}
	if humans == 0 {
		t.Players = [2]*Cueist{}
	}
	t.reset()
}

func (t *Table) removeBot() {
	for i, p := range t.Players {
		if p != nil && p.Bot {
			t.Players[i] = nil
			t.reset()
		}
	}
}

func (t *Table) team(p *Cueist) int {
	if t.Players[1] == p {
		return 1
	}
	return 0
}

func (t *Table) left(kind int) int {
	n := 0
	for _, b := range t.Balls {
		if b.Kind == kind {
			n++
		}
	}
	return n
}

func (t *Table) shoot(team int, angle, power float64) {
	if t.Phase != "aim" || team != t.Turn || !(power > 0) || math.IsNaN(angle) || math.IsInf(angle, 0) {
		return
	}
	t.Balls[0].launch(angle, min(power, 1)*CueMax)
	t.firstHit, t.potted, t.cueIn, t.blackIn = -1, nil, false, false
	t.Phase, t.Last = "roll", ""
	t.event(Event{Kind: "shot", A: team, X: t.Balls[0].X, Y: t.Balls[0].Y})
}

func (t *Table) step(dt float64) {
	switch t.Phase {
	case "aim":
		if p := t.Players[t.Turn]; p != nil && p.Bot && t.Wait <= PoolTurn-PoolBotWait {
			t.think()
			return
		}
		if t.Wait -= dt; t.Wait <= 0 {
			t.Turn, t.Wait, t.Last = 1-t.Turn, PoolTurn, "Temps écoulé"
		}
	case "roll":
		moving := false
		for range PoolSub {
			moving = t.physics(dt / PoolSub)
		}
		if !moving {
			t.settle()
		}
	case "end":
		if t.Wait -= dt; t.Wait <= 0 {
			t.reset()
		}
	}
}

func (t *Table) physics(dt float64) bool {
	moving := false
	for _, b := range t.Balls {
		b.roll(dt)
		moving = moving || b.moving()
	}
	t.Balls = slices.DeleteFunc(t.Balls, func(b *PoolBall) bool {
		for _, pk := range pockets {
			if math.Hypot(b.X-pk[0], b.Y-pk[1]) < PocketR {
				t.event(Event{Kind: "pot", A: b.Kind, X: pk[0], Y: pk[1]})
				switch b.Kind {
				case Cue:
					t.cueIn = true
					b.stop()
					b.X, b.Y = -100, -100 // remise en jeu à la fin du coup
					return false
				case Black:
					t.blackIn = true
				default:
					t.potted = append(t.potted, b.Kind)
				}
				return true
			}
		}
		if b.Kind != Cue || !t.cueIn {
			if b.bounce(0, 0, TW, TH, Cushion) {
				t.event(Event{Kind: "cushion", X: b.X, Y: b.Y})
			}
		}
		return false
	})
	for i, a := range t.Balls {
		for _, b := range t.Balls[i+1:] {
			if t.cueIn && (a.Kind == Cue || b.Kind == Cue) {
				continue
			}
			if v := collide(&a.Body, &b.Body, PoolRestit); v > 0 {
				if t.firstHit < 0 && (a.Kind == Cue || b.Kind == Cue) {
					t.firstHit = a.Kind + b.Kind - Cue // la sorte de l'autre bille
				}
				t.event(Event{Kind: "clack", A: int(min(v, 2000)), X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2})
			}
		}
	}
	return moving
}

// settle : tout est arrêté, on applique les règles du coup.
func (t *Table) settle() {
	own, opp := Yellow+t.Turn, Red-t.Turn
	shooter := t.Players[t.Turn]
	foul := t.cueIn || t.firstHit < 0 || t.firstHit == opp || (t.firstHit == Black && t.left(own) > 0)
	if t.cueIn {
		t.respot()
	}
	if t.blackIn {
		winner := t.Turn
		if foul || t.left(own) > 0 {
			winner = 1 - t.Turn // noire trop tôt ou avec une faute : perdu
		}
		t.Phase, t.Wait, t.Winner = "end", 6, t.Players[winner].Name
		t.event(Event{Kind: "win", A: winner, Text: t.Winner})
		return
	}
	again := !foul && slices.Contains(t.potted, own)
	switch {
	case foul:
		t.Last = "Faute de " + shooter.Name
	case again:
		t.Last = shooter.Name + " rejoue"
	}
	if !again {
		t.Turn = 1 - t.Turn
	}
	t.Phase, t.Wait = "aim", PoolTurn
}

// respot : la blanche revient au point de départ (un peu décalée si une bille gêne).
func (t *Table) respot() {
	c := t.Balls[0]
	c.stop()
	for dy := 0.0; ; dy += 2 * PoolR {
		for _, y := range []float64{TH/2 + dy, TH/2 - dy} {
			free := true
			for _, b := range t.Balls[1:] {
				if math.Hypot(b.X-HeadX, b.Y-y) < 2*PoolR+1 {
					free = false
				}
			}
			if free {
				c.X, c.Y = HeadX, y
				return
			}
		}
	}
}

// think : le bot cherche une bille à lui (ou la noire) et une poche alignées, sinon il tape sa bille la plus proche.
func (t *Table) think() {
	own := Yellow + t.Turn
	if t.left(own) == 0 {
		own = Black
	}
	c := t.Balls[0]
	bestAngle, bestScore := 0.0, math.Inf(1)
	for _, b := range t.Balls {
		if b.Kind != own {
			continue
		}
		for _, pk := range pockets {
			px, py := pk[0]-b.X, pk[1]-b.Y
			pl := math.Hypot(px, py)
			gx, gy := b.X-px/pl*2*PoolR, b.Y-py/pl*2*PoolR // où la blanche doit toucher
			cx, cy := gx-c.X, gy-c.Y
			cl := math.Hypot(cx, cy)
			cut := math.Acos(max(-1, min(1, (cx*px+cy*py)/(cl*pl))))
			if cut > 1.2 {
				continue
			}
			if score := (cl + pl) * (1 + 2*cut); score < bestScore {
				bestAngle, bestScore = math.Atan2(cy, cx), score
			}
		}
	}
	power := 0.75
	if math.IsInf(bestScore, 1) { // pas de coup propre : on tape la bille la plus proche
		near := math.Inf(1)
		for _, b := range t.Balls {
			if d := math.Hypot(b.X-c.X, b.Y-c.Y); b.Kind == own && d < near {
				near, bestAngle = d, math.Atan2(b.Y-c.Y, b.X-c.X)
			}
		}
	} else {
		power = min(max(bestScore/1400+0.3, 0.35), 1)
	}
	t.shoot(t.Turn, bestAngle+(rand.Float64()*2-1)*0.02, power)
}

// --- salons Billard ---

var (
	tablesMu sync.Mutex
	tables   = map[string]*Table{}
)

func enterTable(pin, name string) (*Table, *Cueist, string) {
	tablesMu.Lock()
	defer tablesMu.Unlock()
	if pin == "" {
		if len(tables) >= PoolRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		t := newTable(newPin(func(p string) bool { return tables[p] != nil }))
		tables[t.Pin] = t
		p := t.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go t.run()
		return t, p, ""
	}
	t := tables[pin]
	if t == nil {
		return nil, nil, "Code inconnu"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if p := t.join(name, false); p != nil {
		return t, p, ""
	}
	return nil, nil, "Partie pleine (2 joueurs)"
}

func (t *Table) run() {
	tk := time.NewTicker(PoolTick)
	defer tk.Stop()
	n := 0
	for range tk.C {
		if t.closeIfEmpty() {
			return
		}
		t.mu.Lock()
		t.step(PoolTick.Seconds())
		if n++; n%2 == 0 { // physique à 60 images/s, envoi à 30 : le navigateur lisse entre deux états
			b, _ := json.Marshal(t)
			t.Events = nil
			for _, p := range t.Players {
				if p != nil && p.send != nil {
					select {
					case p.send <- b:
					default: // client lent : on saute cette image
					}
				}
			}
		}
		t.mu.Unlock()
	}
}

func (t *Table) closeIfEmpty() bool {
	tablesMu.Lock()
	defer tablesMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Players[0] != nil || t.Players[1] != nil {
		return false
	}
	delete(tables, t.Pin)
	return true
}

func serveBillard(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		t, p, why := enterTable(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			Room:  t.Pin,
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { t.mu.Lock(); t.leave(p.ID); t.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Bot int     `json:"bot"`
					A   float64 `json:"a"`
					P   float64 `json:"p"`
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				t.mu.Lock()
				defer t.mu.Unlock()
				switch m.Bot {
				case 1:
					t.join("Bot", true)
				case -1:
					t.removeBot()
				default:
					t.shoot(t.team(p), m.A, m.P)
				}
			},
		}, ""
	})
}
