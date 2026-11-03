// Sumo (mode télé) : chacun pilote son lutteur avec le joystick de son téléphone et pousse les autres hors
// d'une arène ronde qui rétrécit. Le dash donne un grand élan (et rend plus lourd un court instant).
// Le dernier sur le tatami marque un point ; premier à 3 points. Bots possibles.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
)

const (
	SumoW, SumoH    = 1600.0, 900.0
	SumoR           = 34.0  // rayon d'un lutteur
	SumoRing        = 400.0 // rayon de l'arène au début de chaque manche
	SumoRingMin     = 150.0
	SumoShrinkAfter = 8.0 // secondes de combat avant que l'arène rétrécisse
	SumoShrink      = 7.0 // px/s
	SumoAccel       = 1500.0
	SumoMaxSpeed    = 430.0
	SumoDamp        = 2.4 // freinage (par seconde)
	SumoDash        = 760.0
	SumoDashTime    = 0.35 // plus lourd pendant ce temps
	SumoDashCool    = 2.0
	SumoFall        = 0.6 // durée de la chute hors de l'arène
	SumoWin         = 3
	SumoMax         = 10
)

type Fighter struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Color string  `json:"color"`
	Bot   bool    `json:"bot"`
	Alive bool    `json:"alive"` // en lice dans cette manche (même en train de tomber)
	Fall  float64 `json:"fall"`  // copie arrondie de fall, pour la télé
	Score int     `json:"score"`
	Dash  bool    `json:"dash"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	body  Body
	fall  float64    // 0 debout, puis jusqu'à 1 pendant la chute
	in    [2]float64 // joystick, longueur ≤ 1
	face  [2]float64 // dernière direction voulue (pour un dash sans joystick)
	dashT float64
	cool  float64
	hits  int // chocs reçus, pour faire vibrer le téléphone
	think float64
}

type Sumo struct {
	Phase    string     `json:"phase"` // wait, count, play, round, end
	Wait     float64    `json:"wait"`
	Ring     float64    `json:"ring"`
	Round    int        `json:"round"`
	Winner   string     `json:"winner"`   // vainqueur de la manche ("" : égalité)
	Champion string     `json:"champion"` // vainqueur de la partie
	F        []*Fighter `json:"f"`
	Events   []Event    `json:"events"`
	fight    float64    // secondes de combat dans la manche
	bots     int
}

func newSumo(pads []*Pad) *Sumo {
	g := &Sumo{Phase: "wait", Ring: SumoRing, F: []*Fighter{}}
	for _, p := range pads {
		g.F = append(g.F, &Fighter{ID: p.ID, Name: p.Name, Color: p.Color})
	}
	g.ready() // tout le monde dans la première manche
	return g
}

func (g *Sumo) get(id int) *Fighter {
	for _, f := range g.F {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func (g *Sumo) Add(p *Pad) {
	if len(g.F) >= SumoMax {
		g.removeBot() // un humain passe avant un bot
	}
	g.F = append(g.F, &Fighter{ID: p.ID, Name: p.Name, Color: p.Color})
	g.ready()
}

func (g *Sumo) Remove(id int) {
	g.F = slices.DeleteFunc(g.F, func(f *Fighter) bool { return f.ID == id })
	if !slices.ContainsFunc(g.F, func(f *Fighter) bool { return !f.Bot }) {
		g.F = []*Fighter{} // plus personne : les bots partent aussi
	}
	g.ready()
}

func (g *Sumo) addBot() {
	if len(g.F) >= SumoMax {
		return
	}
	used := map[string]bool{}
	for _, f := range g.F {
		used[f.Color] = true
	}
	g.bots++
	f := &Fighter{ID: -g.bots, Name: fmt.Sprintf("Bot %d", g.bots), Bot: true, Color: colors[0]}
	for _, c := range colors {
		if !used[c] {
			f.Color = c
			break
		}
	}
	g.F = append(g.F, f)
	g.ready()
}

func (g *Sumo) removeBot() {
	for i := len(g.F) - 1; i >= 0; i-- {
		if g.F[i].Bot {
			g.F = slices.Delete(g.F, i, i+1)
			g.ready()
			return
		}
	}
}

// ready : il faut au moins 2 lutteurs. On démarre dès qu'ils sont là, on s'arrête s'il en manque.
func (g *Sumo) ready() {
	switch {
	case len(g.F) < 2:
		g.Phase, g.Wait, g.Champion = "wait", 0, ""
		for _, f := range g.F {
			f.Score, f.Alive = 0, false
		}
	case g.Phase == "wait" || g.Phase == "count":
		if g.Phase == "count" {
			g.Round-- // la manche n'a pas commencé : on la relance avec les nouveaux venus
		}
		g.startRound()
	case g.Phase == "play":
		g.checkRound()
	}
}

// startRound : tout le monde en cercle autour du centre, face au milieu.
func (g *Sumo) startRound() {
	g.Round++
	g.Winner, g.Ring, g.fight = "", SumoRing, 0
	off := rand.Float64() * 2 * math.Pi
	for i, f := range g.F {
		a := off + 2*math.Pi*float64(i)/float64(len(g.F))
		f.body = Body{X: SumoW/2 + math.Cos(a)*SumoRing*0.62, Y: SumoH/2 + math.Sin(a)*SumoRing*0.62, r: SumoR, m: 1}
		f.face = [2]float64{-math.Cos(a), -math.Sin(a)}
		f.Alive, f.fall, f.dashT, f.cool, f.in = true, 0, 0, 0, [2]float64{}
	}
	g.Phase, g.Wait = "count", 3
}

func (g *Sumo) Input(p *Pad, b []byte) {
	var m struct {
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
		Dash bool    `json:"dash"`
		Bot  int     `json:"bot"`
	}
	if json.Unmarshal(b, &m) != nil {
		return
	}
	if m.Bot != 0 {
		if p.Host && m.Bot > 0 {
			g.addBot()
		} else if p.Host {
			g.removeBot()
		}
		return
	}
	f := g.get(p.ID)
	if f == nil {
		return
	}
	if m.Dash {
		g.dash(f)
		return
	}
	if math.IsNaN(m.X) || math.IsNaN(m.Y) {
		return
	}
	if l := math.Hypot(m.X, m.Y); l > 1 {
		m.X, m.Y = m.X/l, m.Y/l
	}
	f.in = [2]float64{m.X, m.Y}
	if math.Hypot(m.X, m.Y) > 0.2 {
		f.face = f.in
	}
}

func (g *Sumo) dash(f *Fighter) {
	if g.Phase != "play" || !f.Alive || f.fall > 0 || f.cool > 0 {
		return
	}
	d := f.face
	if l := math.Hypot(d[0], d[1]); l > 0 {
		d = [2]float64{d[0] / l, d[1] / l}
	}
	f.body.vx += d[0] * SumoDash
	f.body.vy += d[1] * SumoDash
	f.dashT, f.cool = SumoDashTime, SumoDashCool
	g.Events = append(g.Events, Event{Kind: "dash", A: f.ID, X: f.body.X, Y: f.body.Y})
}

func (g *Sumo) Step(dt float64) {
	switch g.Phase {
	case "count", "round":
		if g.Wait -= dt; g.Wait <= 0 {
			if g.Phase == "round" {
				g.startRound()
			} else {
				g.Phase = "play"
			}
		}
	case "end":
		if g.Wait -= dt; g.Wait <= 0 {
			g.Champion = ""
			for _, f := range g.F {
				f.Score = 0
			}
			g.startRound()
		}
	case "play":
		g.physics(dt)
	}
}

func (g *Sumo) physics(dt float64) {
	g.fight += dt
	if g.fight > SumoShrinkAfter {
		g.Ring = max(SumoRingMin, g.Ring-SumoShrink*dt)
	}
	for _, f := range g.F {
		if !f.Alive {
			continue
		}
		f.cool = max(0, f.cool-dt)
		f.dashT = max(0, f.dashT-dt)
		f.body.m = 1
		if f.dashT > 0 {
			f.body.m = 2.2 // le dash pousse fort et résiste aux chocs
		}
		if f.fall == 0 {
			if f.Bot {
				g.brain(f, dt)
			}
			b := &f.body
			sp := b.speed()
			vx, vy := b.vx+f.in[0]*SumoAccel*dt, b.vy+f.in[1]*SumoAccel*dt
			if ns := math.Hypot(vx, vy); ns > SumoMaxSpeed && ns > sp { // le joystick n'accélère pas au-delà de la vitesse max
				k := max(sp, SumoMaxSpeed) / ns
				vx, vy = vx*k, vy*k
			}
			b.vx, b.vy = vx, vy
		}
		k := max(0, 1-SumoDamp*dt)
		f.body.vx, f.body.vy = f.body.vx*k, f.body.vy*k
		f.body.X += f.body.vx * dt
		f.body.Y += f.body.vy * dt
	}
	for i, a := range g.F {
		for _, b := range g.F[i+1:] {
			if !a.Alive || !b.Alive || a.fall > 0 || b.fall > 0 {
				continue
			}
			if v := collide(&a.body, &b.body, 1); v > 180 {
				a.hits++
				b.hits++
				g.Events = append(g.Events, Event{Kind: "hit", X: (a.body.X + b.body.X) / 2, Y: (a.body.Y + b.body.Y) / 2, Text: fmt.Sprint(int(v))})
			}
		}
	}
	for _, f := range g.F {
		if !f.Alive {
			continue
		}
		if f.fall == 0 && math.Hypot(f.body.X-SumoW/2, f.body.Y-SumoH/2) > g.Ring {
			f.fall = 0.001 // il passe le bord : il tombe
			f.hits++
			g.Events = append(g.Events, Event{Kind: "out", A: f.ID, X: f.body.X, Y: f.body.Y, Text: f.Name})
		} else if f.fall > 0 {
			if f.fall += dt / SumoFall; f.fall >= 1 {
				f.Alive, f.fall = false, 1
			}
		}
	}
	g.checkRound()
}

// checkRound : la manche finit quand il reste au plus un lutteur debout et que personne n'est en train de tomber.
func (g *Sumo) checkRound() {
	var standing []*Fighter
	for _, f := range g.F {
		if f.Alive && f.fall == 0 {
			standing = append(standing, f)
		} else if f.Alive {
			return // on attend la fin de la chute (deux chutes en même temps : égalité)
		}
	}
	if len(standing) > 1 {
		return
	}
	g.Phase, g.Wait, g.Winner = "round", 3, ""
	if len(standing) == 1 {
		w := standing[0]
		w.Score++
		g.Winner = w.Name
		g.Events = append(g.Events, Event{Kind: "round", A: w.ID, Text: w.Name})
		if w.Score >= SumoWin {
			g.Phase, g.Wait, g.Champion = "end", 7, w.Name
			g.Events = append(g.Events, Event{Kind: "win", A: w.ID, Text: w.Name})
		}
	}
}

// brain : le bot fonce sur le plus proche en venant du côté du centre (pour le pousser dehors), revient
// vers le milieu s'il est trop près du bord, et dashe quand il est bien aligné.
func (g *Sumo) brain(f *Fighter, dt float64) {
	if f.think -= dt; f.think > 0 {
		return
	}
	f.think = 0.12 + rand.Float64()*0.1 // temps de réaction
	cx, cy := SumoW/2, SumoH/2
	me := &f.body
	var tgt *Fighter
	best := math.Inf(1)
	for _, o := range g.F {
		if o != f && o.Alive && o.fall == 0 {
			if d := math.Hypot(o.body.X-me.X, o.body.Y-me.Y); d < best {
				best, tgt = d, o
			}
		}
	}
	dc := math.Hypot(me.X-cx, me.Y-cy)
	var dx, dy float64
	if tgt != nil {
		// viser un peu derrière la cible, vu du centre
		ox, oy := tgt.body.X-cx, tgt.body.Y-cy
		if l := math.Hypot(ox, oy); l > 0 {
			ox, oy = ox/l, oy/l
		}
		dx, dy = tgt.body.X-ox*20-me.X, tgt.body.Y-oy*20-me.Y
	}
	if dc > g.Ring-110 || tgt == nil { // trop près du bord : on rentre
		dx, dy = cx-me.X+dx*0.2, cy-me.Y+dy*0.2
	}
	if l := math.Hypot(dx, dy); l > 0 {
		dx, dy = dx/l, dy/l
	}
	f.in = [2]float64{dx * 0.85, dy * 0.85}
	f.face = f.in
	if tgt != nil && best < 230 && f.cool == 0 && dc < g.Ring-60 && rand.IntN(3) == 0 {
		g.dash(f)
	}
}

func (g *Sumo) Screen() any {
	for _, f := range g.F {
		f.X, f.Y = math.Round(f.body.X), math.Round(f.body.Y)
		f.Dash = f.dashT > 0
		f.Fall = math.Round(f.fall*100) / 100 // copie : arrondir fall lui-même l'empêcherait d'avancer
	}
	s := *g
	s.Wait = math.Ceil(g.Wait)
	s.Ring = math.Round(g.Ring)
	g.Events = nil // envoyés une seule fois
	return s
}

type sumoView struct {
	Phase    string `json:"phase"`
	In       bool   `json:"in"`    // en lice dans cette manche
	Alive    bool   `json:"alive"` // encore debout
	Score    int    `json:"score"`
	Ready    bool   `json:"ready"` // dash disponible
	Hits     int    `json:"hits"`
	Bots     int    `json:"bots"`
	Count    int    `json:"count"` // lutteurs (joueurs + bots)
	Wait     int    `json:"wait"`
	Winner   string `json:"winner"`
	Champion string `json:"champion"`
}

func (g *Sumo) View(p *Pad) any {
	v := sumoView{Phase: g.Phase, Count: len(g.F), Winner: g.Winner, Champion: g.Champion}
	if g.Phase == "count" {
		v.Wait = int(math.Ceil(g.Wait))
	}
	for _, f := range g.F {
		if f.Bot {
			v.Bots++
		}
	}
	if f := g.get(p.ID); f != nil {
		v.In, v.Alive, v.Score, v.Ready, v.Hits = f.Alive, f.Alive && f.fall == 0, f.Score, f.cool == 0, f.hits
	}
	return v
}
