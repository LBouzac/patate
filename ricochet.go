// Ricochet : duel de raquettes à 2 joueurs (ou contre un bot). La balle accélère à chaque renvoi,
// l'angle dépend de l'endroit où elle touche la raquette. Premier à WinScore points.
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
	DW, DH    = 800.0, 500.0
	PadW      = 12.0
	PadH      = 90.0
	PadX      = 30.0 // distance entre le bord et la raquette
	PadSpeed  = 430.0
	BotSpeed  = 0.82 // le bot est un peu plus lent qu'un humain
	BallR     = 8.0
	BallStart = 380.0
	BallMax   = 950.0
	BallAccel = 1.06
	MaxAngle  = 1.05 // ~60° quand la balle touche le bout de la raquette
	WinScore  = 7
	DuelTick  = time.Second / 60
)

type Paddle struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Y     float64 `json:"y"` // centre
	Score int     `json:"score"`
	Bot   bool    `json:"bot"`
	up    bool
	down  bool
	aim   float64 // bot : erreur de visée, retirée à chaque renvoi
	send  chan []byte
}

type Ball struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
}

type Duel struct {
	mu      sync.Mutex
	Pin     string     `json:"pin"`
	Pads    [2]*Paddle `json:"pads"` // 0 gauche, 1 droite ; nil = place libre
	Ball    Ball       `json:"ball"`
	Phase   string     `json:"phase"` // wait, count, play, point, end
	Wait    float64    `json:"wait"`
	Winner  string     `json:"winner"`
	Rally   int        `json:"rally"` // renvois dans l'échange en cours
	Events  []Event    `json:"events"`
	nextID  int
	serveTo int // côté qui reçoit le prochain service
}

func newDuel(pin string) *Duel {
	d := &Duel{Pin: pin, Phase: "wait"}
	d.Ball = Ball{X: DW / 2, Y: DH / 2}
	return d
}

func (d *Duel) event(e Event) { d.Events = append(d.Events, e) }

// join place un joueur. Un humain prend la place d'un bot si la partie est pleine.
func (d *Duel) join(name string, bot bool) *Paddle {
	slot := -1
	for i, p := range d.Pads {
		if p == nil {
			slot = i
			break
		}
	}
	if slot < 0 && !bot {
		for i, p := range d.Pads {
			if p.Bot {
				slot = i
				break
			}
		}
	}
	if slot < 0 {
		return nil
	}
	d.nextID++
	p := &Paddle{ID: d.nextID, Name: name, Y: DH / 2, Bot: bot}
	if !bot {
		p.send = make(chan []byte, 8)
	}
	d.Pads[slot] = p
	d.reset()
	return p
}

func (d *Duel) leave(id int) {
	humans := 0
	for i, p := range d.Pads {
		if p != nil && p.ID == id {
			d.Pads[i] = nil
		} else if p != nil && !p.Bot {
			humans++
		}
	}
	if humans == 0 {
		d.Pads = [2]*Paddle{} // plus aucun humain : le bot part aussi, le salon se ferme
	}
	d.reset()
}

func (d *Duel) removeBot() {
	for i, p := range d.Pads {
		if p != nil && p.Bot {
			d.Pads[i] = nil
			d.reset()
		}
	}
}

// reset : nouvelle partie dès qu'un joueur arrive ou part.
func (d *Duel) reset() {
	d.Phase, d.Wait, d.Winner, d.Rally = "wait", 0, "", 0
	d.Ball = Ball{X: DW / 2, Y: DH / 2}
	for _, p := range d.Pads {
		if p != nil {
			p.Score = 0
		}
	}
}

func (d *Duel) serve() {
	angle := (rand.Float64() - 0.5) * 0.9 // ±25°
	dir := -1.0
	if d.serveTo == 1 {
		dir = 1
	}
	d.Ball = Ball{DW / 2, DH / 2, dir * BallStart * math.Cos(angle), BallStart * math.Sin(angle)}
	d.Rally = 0
	d.Phase = "play"
	d.event(Event{Kind: "serve", X: DW / 2, Y: DH / 2})
}

func (d *Duel) step(dt float64) {
	for _, p := range d.Pads {
		if p == nil {
			continue
		}
		speed := PadSpeed
		if p.Bot {
			d.think(p)
			speed *= BotSpeed
		}
		if p.up != p.down {
			if p.up {
				speed = -speed
			}
			p.Y = min(max(p.Y+speed*dt, PadH/2), DH-PadH/2)
		}
	}

	switch d.Phase {
	case "wait":
		if d.Pads[0] != nil && d.Pads[1] != nil {
			d.Phase, d.Wait = "count", 3
			d.serveTo = rand.IntN(2)
		}
	case "count", "point":
		if d.Wait -= dt; d.Wait <= 0 {
			d.serve()
		}
	case "play":
		d.moveBall(dt)
	case "end":
		if d.Wait -= dt; d.Wait <= 0 { // revanche
			d.reset()
		}
	}
}

func (d *Duel) moveBall(dt float64) {
	b := &d.Ball
	ox, oy := b.X, b.Y
	b.X, b.Y = b.X+b.VX*dt, b.Y+b.VY*dt
	if b.Y < BallR || b.Y > DH-BallR { // rebond sur le haut ou le bas
		lim := BallR
		if b.Y > DH-BallR {
			lim = DH - BallR
		}
		b.Y, b.VY = 2*lim-b.Y, -b.VY
		d.event(Event{Kind: "wall", X: b.X, Y: lim})
	}

	// test par segment (et non par position) : à grande vitesse la balle ne traverse jamais une raquette
	if left := PadX + PadW; b.VX < 0 && ox-BallR >= left && b.X-BallR < left {
		t := (ox - BallR - left) / (ox - b.X)
		d.hit(0, oy+(b.Y-oy)*t, left+BallR)
	} else if right := DW - PadX - PadW; b.VX > 0 && ox+BallR <= right && b.X+BallR > right {
		t := (right - ox - BallR) / (b.X - ox)
		d.hit(1, oy+(b.Y-oy)*t, right-BallR)
	}

	switch {
	case b.X < -BallR:
		d.point(1)
	case b.X > DW+BallR:
		d.point(0)
	}
}

// hit : la balle arrive sur la raquette du côté side à la hauteur y.
func (d *Duel) hit(side int, y, x float64) {
	p := d.Pads[side]
	off := (y - p.Y) / (PadH/2 + BallR) // -1 en haut … +1 en bas de la raquette
	if math.Abs(off) > 1 {
		return // ratée : la balle continue
	}
	speed := min(math.Hypot(d.Ball.VX, d.Ball.VY)*BallAccel, BallMax)
	dir := 1.0
	if side == 1 {
		dir = -1
	}
	angle := off * MaxAngle
	d.Ball = Ball{x, y, dir * speed * math.Cos(angle), speed * math.Sin(angle)}
	d.Rally++
	if o := d.Pads[1-side]; o.Bot {
		o.aim = (rand.Float64() - 0.5) * PadH * 1.1 // le bot visera un peu à côté
	}
	d.event(Event{Kind: "hit", A: p.ID, X: x, Y: y, Text: string(rune('0' + side))})
}

func (d *Duel) point(side int) {
	p := d.Pads[side]
	p.Score++
	d.serveTo = 1 - side // service vers celui qui vient de perdre le point
	d.event(Event{Kind: "score", A: p.ID, X: d.Ball.X, Y: d.Ball.Y, Text: string(rune('0' + side))})
	if p.Score >= WinScore {
		d.Phase, d.Wait, d.Winner = "end", 6, p.Name
		d.event(Event{Kind: "win", A: p.ID, Text: p.Name})
		return
	}
	d.Phase, d.Wait = "point", 1.2
	d.Ball = Ball{X: DW / 2, Y: DH / 2}
}

// think : le bot suit la balle quand elle vient vers lui, sinon il revient au centre.
func (d *Duel) think(p *Paddle) {
	target := DH / 2
	coming := (p == d.Pads[0] && d.Ball.VX < 0) || (p == d.Pads[1] && d.Ball.VX > 0)
	if d.Phase == "play" && coming {
		target = d.Ball.Y + p.aim
	}
	p.up, p.down = target < p.Y-8, target > p.Y+8
}

// --- salons Ricochet (même principe que Patate, codes séparés) ---

var (
	duelsMu sync.Mutex
	duels   = map[string]*Duel{}
)

func enterDuel(pin, name string) (*Duel, *Paddle, string) {
	duelsMu.Lock()
	defer duelsMu.Unlock()
	if pin == "" {
		if len(duels) >= 1000 {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		d := newDuel(newPin(func(p string) bool { return duels[p] != nil }))
		duels[d.Pin] = d
		p := d.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go d.run()
		return d, p, ""
	}
	d := duels[pin]
	if d == nil {
		return nil, nil, "Code inconnu"
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if p := d.join(name, false); p != nil {
		return d, p, ""
	}
	return nil, nil, "Partie pleine (2 joueurs)"
}

func (d *Duel) run() {
	t := time.NewTicker(DuelTick)
	defer t.Stop()
	n := 0
	for range t.C {
		if d.closeIfEmpty() {
			return
		}
		d.mu.Lock()
		d.step(DuelTick.Seconds())
		if n++; n%2 == 0 { // physique à 60 images/s, envoi à 30 : le navigateur lisse entre deux états
			b, _ := json.Marshal(d)
			d.Events = nil
			for _, p := range d.Pads {
				if p != nil && p.send != nil {
					select {
					case p.send <- b:
					default: // client lent : on saute cette image
					}
				}
			}
		}
		d.mu.Unlock()
	}
}

func (d *Duel) closeIfEmpty() bool {
	duelsMu.Lock()
	defer duelsMu.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Pads[0] != nil || d.Pads[1] != nil {
		return false
	}
	delete(duels, d.Pin)
	return true
}

func serveDuel(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		d, p, why := enterDuel(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { d.mu.Lock(); d.leave(p.ID); d.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Input
					Bot int `json:"bot"` // +1 : jouer contre un bot, -1 : le retirer
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				d.mu.Lock()
				defer d.mu.Unlock()
				switch m.Bot {
				case 1:
					d.join("Bot", true)
				case -1:
					d.removeBot()
				default:
					p.up, p.down = m.U, m.D
				}
			},
		}, ""
	})
}
