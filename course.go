// Course (mode télé) : course de petites voitures vue de dessus, sur la télé. La voiture accélère toute seule ;
// on tourne avec les boutons gauche/droite du téléphone (ou en l'inclinant). Des boîtes sur la piste donnent un
// objet : turbo, ou flaque d'huile à lâcher derrière soi. 3 tours, puis le circuit suivant ; les points s'additionnent.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
)

const (
	CarR        = 16.0  // rayon de choc d'une voiture
	CarMax      = 330.0 // vitesse max sur la piste (px/s)
	CarGrass    = 130.0 // vitesse max dans l'herbe
	CarAccel    = 260.0
	CarTurn     = 3.3 // rad/s
	CarGrip     = 7.0 // adhérence : plus c'est bas, plus ça glisse
	CarBoost    = 1.55
	BoostTime   = 1.3
	SpinTime    = 0.9
	RaceLaps    = 3
	RaceAfter   = 20.0 // après la première arrivée, les autres ont 20 s
	RaceResult  = 10.0
	RaceRespawn = 2.5 // hors piste trop longtemps : on revient sur la piste
	BoxBack     = 4.0 // une boîte réapparaît après 4 s
	OilMax      = 8
	KartMax     = 10
)

var racePoints = []int{10, 8, 6, 5, 4, 3, 2, 1}

// ---------- Circuits : une ligne centrale fermée (points de passage lissés) et une largeur ----------

type Circuit struct {
	Name  string
	W     float64
	Ctrl  [][2]float64
	pts   [][2]float64 // ligne centrale échantillonnée (un point tous les ~8 px)
	cum   []float64    // longueur parcourue jusqu'à chaque point
	L     float64
	boxes [][2]float64
	big   json.RawMessage // géométrie envoyée une seule fois à la télé
}

var circuits = []*Circuit{
	{Name: "L'ovale", W: 120, Ctrl: [][2]float64{{300, 170}, {800, 120}, {1300, 170}, {1480, 450}, {1300, 730}, {800, 780}, {300, 730}, {120, 450}}},
	{Name: "Le haricot", W: 120, Ctrl: [][2]float64{{250, 190}, {700, 140}, {1000, 320}, {1350, 160}, {1500, 430}, {1300, 760}, {800, 700}, {480, 780}, {160, 620}, {130, 380}}},
	{Name: "Le serpentin", W: 115, Ctrl: [][2]float64{{220, 150}, {600, 150}, {750, 360}, {900, 150}, {1380, 150}, {1490, 450}, {1380, 760}, {1000, 760}, {850, 580}, {700, 760}, {220, 760}, {110, 450}}},
}

func init() {
	for _, c := range circuits {
		c.build()
	}
}

// build lisse les points de passage (Catmull-Rom) et place les boîtes à objets.
func (c *Circuit) build() {
	n := len(c.Ctrl)
	var raw [][2]float64
	for i := range n {
		p0, p1, p2, p3 := c.Ctrl[(i-1+n)%n], c.Ctrl[i], c.Ctrl[(i+1)%n], c.Ctrl[(i+2)%n]
		for k := range 60 {
			t := float64(k) / 60
			t2, t3 := t*t, t*t*t
			var p [2]float64
			for d := range 2 {
				p[d] = 0.5 * (2*p1[d] + (-p0[d]+p2[d])*t + (2*p0[d]-5*p1[d]+4*p2[d]-p3[d])*t2 + (-p0[d]+3*p1[d]-3*p2[d]+p3[d])*t3)
			}
			raw = append(raw, p)
		}
	}
	// rééchantillonnage régulier
	c.pts = [][2]float64{raw[0]}
	acc := 0.0
	for i := 1; i <= len(raw); i++ {
		a, b := raw[i-1], raw[i%len(raw)]
		acc += math.Hypot(b[0]-a[0], b[1]-a[1])
		if acc >= 8 && i < len(raw) {
			c.pts = append(c.pts, b)
			acc = 0
		}
	}
	c.cum = make([]float64, len(c.pts))
	for i := 1; i < len(c.pts); i++ {
		c.cum[i] = c.cum[i-1] + dist2(c.pts[i-1], c.pts[i])
	}
	c.L = c.cum[len(c.pts)-1] + dist2(c.pts[len(c.pts)-1], c.pts[0])
	for _, f := range []float64{0.27, 0.56, 0.82} { // 3 rangées de 3 boîtes
		i := int(f * float64(len(c.pts)))
		nx, ny := c.normal(i)
		for _, o := range []float64{-c.W * 0.3, 0, c.W * 0.3} {
			c.boxes = append(c.boxes, [2]float64{c.pts[i][0] + nx*o, c.pts[i][1] + ny*o})
		}
	}
	flat := make([]int, 0, 2*len(c.pts))
	for _, p := range c.pts {
		flat = append(flat, int(math.Round(p[0])), int(math.Round(p[1])))
	}
	c.big, _ = json.Marshal(map[string]any{"name": c.Name, "w": c.W, "pts": flat, "boxes": c.boxes})
}

func dist2(a, b [2]float64) float64 { return math.Hypot(b[0]-a[0], b[1]-a[1]) }

func (c *Circuit) at(i int) [2]float64 { n := len(c.pts); return c.pts[((i%n)+n)%n] }

// tangent : direction de la piste au point i.
func (c *Circuit) tangent(i int) (float64, float64) {
	a, b := c.at(i-1), c.at(i+1)
	l := dist2(a, b)
	return (b[0] - a[0]) / l, (b[1] - a[1]) / l
}

func (c *Circuit) normal(i int) (float64, float64) { tx, ty := c.tangent(i); return -ty, tx }

// nearest : point de la ligne centrale le plus proche, cherché autour du dernier connu (jamais sur la ligne droite
// d'à côté), et sa distance.
func (c *Circuit) nearest(x, y float64, from int) (int, float64) {
	n := len(c.pts)
	best, bd := from, math.Inf(1)
	for k := -8; k <= 12; k++ { // une voiture avance de 2 ou 3 points par image au plus
		i := ((from+k)%n + n) % n
		if d := math.Hypot(c.pts[i][0]-x, c.pts[i][1]-y); d < bd {
			best, bd = i, d
		}
	}
	return best, bd
}

// ---------- Voitures ----------

type Car struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Color  string  `json:"color"`
	Bot    bool    `json:"bot"`
	In     bool    `json:"in"` // dans la course en cours (sinon : arrivé en route, il attend la suivante)
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	A      float64 `json:"a"` // cap (radians)
	Lap    int     `json:"lap"`
	Rank   int     `json:"rank"`
	Item   string  `json:"item"` // "", turbo, huile
	Spin   bool    `json:"spin"`
	Boost  bool    `json:"boost"`
	Done   bool    `json:"done"`
	Place  int     `json:"place"` // ordre d'arrivée (0 : pas arrivé)
	Time   float64 `json:"time"`
	Pts    int     `json:"pts"` // points du championnat
	Gain   int     `json:"gain"`
	body   Body
	head   float64
	speed  float64
	steer  float64
	idx    int     // point de la piste le plus proche
	dist   float64 // distance parcourue depuis le départ (négative sur la grille)
	off    float64 // temps passé loin de la piste
	spin   float64
	boost  float64
	think  float64
	lane   float64 // décalage préféré des bots
	pokes  int     // à faire vibrer : chocs, tête-à-queue
	gotBox int     // objets ramassés
}

type Oil struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	t float64
}

type Kart struct {
	Phase  string  `json:"phase"` // count, race, result
	Wait   float64 `json:"wait"`
	Race   int     `json:"race"`
	Laps   int     `json:"laps"`
	Name   string  `json:"name"`
	Cars   []*Car  `json:"cars"`
	Boxes  []bool  `json:"boxes"`
	Oils   []*Oil  `json:"oils"`
	Events []Event `json:"events"`
	Clock  float64 `json:"clock"`
	Finish float64 `json:"finish"` // compte à rebours après la première arrivée (0 : pas encore)
	c      *Circuit
	boxT   []float64
	bots   int
	placed int
}

func newKart(pads []*Pad) *Kart {
	g := &Kart{Laps: RaceLaps, Cars: []*Car{}, Oils: []*Oil{}}
	for _, p := range pads {
		g.Cars = append(g.Cars, &Car{ID: p.ID, Name: p.Name, Color: p.Color})
	}
	g.start()
	return g
}

func (g *Kart) get(id int) *Car {
	for _, c := range g.Cars {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// start : nouvelle course sur le circuit suivant. Grille de départ : le leader du championnat part dernier.
func (g *Kart) start() {
	g.c = circuits[g.Race%len(circuits)]
	g.Race++
	g.Name, g.Phase, g.Wait, g.Clock, g.Finish, g.placed = g.c.Name, "count", 3, 0, 0, 0
	g.Boxes, g.boxT = make([]bool, len(g.c.boxes)), make([]float64, len(g.c.boxes))
	for i := range g.Boxes {
		g.Boxes[i] = true
	}
	g.Oils = []*Oil{}
	grid := slices.Clone(g.Cars)
	rand.Shuffle(len(grid), func(i, j int) { grid[i], grid[j] = grid[j], grid[i] })
	sort.SliceStable(grid, func(i, j int) bool { return grid[i].Pts > grid[j].Pts }) // leader derrière
	slices.Reverse(grid)
	for k, car := range grid {
		back := 40 + float64(k/2)*52
		i := len(g.c.pts) - int(back/8)
		nx, ny := g.c.normal(i)
		side := -1.0
		if k%2 == 1 {
			side = 1
		}
		p := g.c.at(i)
		tx, ty := g.c.tangent(i)
		*car = Car{ID: car.ID, Name: car.Name, Color: car.Color, Bot: car.Bot, Pts: car.Pts, In: true,
			body: Body{X: p[0] + nx*side*g.c.W*0.22, Y: p[1] + ny*side*g.c.W*0.22, r: CarR, m: 1},
			head: math.Atan2(ty, tx), idx: ((i % len(g.c.pts)) + len(g.c.pts)) % len(g.c.pts), dist: -back,
			lane: (rand.Float64() - 0.5) * g.c.W * 0.4}
	}
}

func (g *Kart) Add(p *Pad) {
	if len(g.Cars) >= KartMax {
		g.removeBot()
	}
	g.Cars = append(g.Cars, &Car{ID: p.ID, Name: p.Name, Color: p.Color})
	if g.Phase == "count" { // pas encore parti : il prend place sur la grille
		g.Race--
		g.start()
	}
}

func (g *Kart) Remove(id int) {
	g.Cars = slices.DeleteFunc(g.Cars, func(c *Car) bool { return c.ID == id })
	if !slices.ContainsFunc(g.Cars, func(c *Car) bool { return !c.Bot }) {
		g.Cars = []*Car{} // plus personne : les bots partent aussi
	}
}

func (g *Kart) addBot() {
	if len(g.Cars) >= KartMax {
		return
	}
	used := map[string]bool{}
	for _, c := range g.Cars {
		used[c.Color] = true
	}
	g.bots++
	car := &Car{ID: -g.bots, Name: fmt.Sprintf("Bot %d", g.bots), Bot: true, Color: colors[0]}
	for _, col := range colors {
		if !used[col] {
			car.Color = col
			break
		}
	}
	g.Cars = append(g.Cars, car)
	if g.Phase == "count" {
		g.Race--
		g.start()
	}
}

func (g *Kart) removeBot() {
	for i := len(g.Cars) - 1; i >= 0; i-- {
		if g.Cars[i].Bot {
			g.Cars = slices.Delete(g.Cars, i, i+1)
			return
		}
	}
}

func (g *Kart) Input(p *Pad, b []byte) {
	var m struct {
		S    *float64 `json:"s"` // direction : -1 gauche, +1 droite
		Item bool     `json:"item"`
		Bot  int      `json:"bot"`
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
	car := g.get(p.ID)
	if car == nil {
		return
	}
	if m.S != nil && !math.IsNaN(*m.S) {
		car.steer = min(max(*m.S, -1), 1)
	}
	if m.Item {
		g.use(car)
	}
}

// use : utiliser son objet.
func (g *Kart) use(car *Car) {
	if g.Phase != "race" || !car.In || car.Done || car.Item == "" {
		return
	}
	switch car.Item {
	case "turbo":
		car.boost = BoostTime
		g.Events = append(g.Events, Event{Kind: "turbo", A: car.ID, X: car.body.X, Y: car.body.Y})
	case "huile":
		if len(g.Oils) >= OilMax {
			g.Oils = g.Oils[1:]
		}
		g.Oils = append(g.Oils, &Oil{X: car.body.X - math.Cos(car.head)*34, Y: car.body.Y - math.Sin(car.head)*34, t: 20})
		g.Events = append(g.Events, Event{Kind: "oil", A: car.ID})
	}
	car.Item = ""
}

func (g *Kart) Step(dt float64) {
	switch g.Phase {
	case "count":
		if g.Wait -= dt; g.Wait <= 0 {
			g.Phase = "race"
			g.Events = append(g.Events, Event{Kind: "go"})
		}
	case "result":
		if g.Wait -= dt; g.Wait <= 0 {
			g.start()
		}
	case "race":
		g.Clock += dt
		g.drive(dt)
		if g.Finish > 0 {
			if g.Finish -= dt; g.Finish <= 0 {
				g.end()
			}
		}
	}
}

func angleDiff(a, b float64) float64 { return math.Remainder(a-b, 2*math.Pi) }

func (g *Kart) drive(dt float64) {
	c := g.c
	for _, car := range g.Cars {
		if !car.In {
			continue
		}
		if car.Bot && !car.Done {
			g.brain(car, dt)
		}
		b := &car.body
		_, d := c.nearest(b.X, b.Y, car.idx)
		top := CarMax
		if d > c.W/2 {
			top = CarGrass
		}
		if car.boost > 0 {
			car.boost -= dt
			top *= CarBoost
			car.speed = max(car.speed, top*0.9)
		}
		if car.Done {
			top = 120 // tour d'honneur au ralenti
		}
		if car.spin > 0 { // tête-à-queue : on ne contrôle plus rien
			car.spin -= dt
			car.head += 10 * dt
			car.speed *= 1 - 1.8*dt
		} else {
			if car.speed < top {
				car.speed = min(top, car.speed+CarAccel*dt)
			} else {
				car.speed = max(top, car.speed-420*dt)
			}
			grip := min(1, car.speed/140)
			car.head += car.steer * CarTurn * dt * (0.35 + 0.65*grip)
		}
		// la vitesse suit le cap avec un peu de glisse
		wx, wy := math.Cos(car.head)*car.speed, math.Sin(car.head)*car.speed
		k := min(1, CarGrip*dt)
		b.vx += (wx - b.vx) * k
		b.vy += (wy - b.vy) * k
		b.X += b.vx * dt
		b.Y += b.vy * dt
		b.bounce(0, 0, SumoW, SumoH, 0.3)
	}
	for i, a := range g.Cars {
		for _, o := range g.Cars[i+1:] {
			if a.In && o.In {
				if v := collide(&a.body, &o.body, 0.4); v > 120 {
					a.pokes++
					o.pokes++
					g.Events = append(g.Events, Event{Kind: "bump", X: (a.body.X + o.body.X) / 2, Y: (a.body.Y + o.body.Y) / 2})
				}
			}
		}
	}
	for _, car := range g.Cars {
		if !car.In {
			continue
		}
		b := &car.body
		// après un choc, la vitesse repart de ce qu'il reste dans l'axe de la voiture
		car.speed = max(0, b.vx*math.Cos(car.head)+b.vy*math.Sin(car.head))
		i, d := c.nearest(b.X, b.Y, car.idx)
		if d <= c.W/2+40 { // loin de la piste, la progression est gelée : couper par l'herbe ne rapporte rien
			ds := c.cum[i] - c.cum[car.idx]
			if ds < -c.L/2 {
				ds += c.L
			} else if ds > c.L/2 {
				ds -= c.L
			}
			car.idx = i
			car.dist += ds
		}
		if d > c.W/2+24 {
			if car.off += dt; car.off > RaceRespawn { // perdu dans l'herbe : retour sur la piste
				p := c.at(i)
				tx, ty := c.tangent(i)
				b.X, b.Y, b.vx, b.vy = p[0], p[1], 0, 0
				car.head, car.speed, car.off = math.Atan2(ty, tx), 0, 0
				g.Events = append(g.Events, Event{Kind: "respawn", A: car.ID})
			}
		} else {
			car.off = 0
		}
		if !car.Done && car.dist >= float64(g.Laps)*c.L {
			g.placed++
			car.Done, car.Place, car.Time = true, g.placed, math.Round(g.Clock*10)/10
			car.Item, car.boost = "", 0
			g.Events = append(g.Events, Event{Kind: "finish", A: car.ID, Text: car.Name})
			if g.Finish == 0 {
				g.Finish = RaceAfter
			}
		}
		for j, at := range c.boxes {
			if g.Boxes[j] && !car.Done && math.Hypot(at[0]-b.X, at[1]-b.Y) < CarR+14 {
				g.Boxes[j], g.boxT[j] = false, BoxBack
				if car.Item == "" {
					car.Item = "turbo"
					if rand.IntN(5) < 2 {
						car.Item = "huile"
					}
					car.gotBox++
					g.Events = append(g.Events, Event{Kind: "box", A: car.ID})
				}
			}
		}
		for j := 0; j < len(g.Oils); j++ {
			o := g.Oils[j]
			if car.spin <= 0 && car.boost <= 0 && math.Hypot(o.X-b.X, o.Y-b.Y) < CarR+12 {
				car.spin = SpinTime
				car.pokes++
				g.Oils = slices.Delete(g.Oils, j, j+1)
				j--
				g.Events = append(g.Events, Event{Kind: "spin", A: car.ID, X: b.X, Y: b.Y})
			}
		}
	}
	for j := range g.boxT {
		if !g.Boxes[j] {
			if g.boxT[j] -= dt; g.boxT[j] <= 0 {
				g.Boxes[j] = true
			}
		}
	}
	g.Oils = slices.DeleteFunc(g.Oils, func(o *Oil) bool { o.t -= dt; return o.t <= 0 })
	g.rank()
	if g.Phase == "race" && !slices.ContainsFunc(g.Cars, func(c *Car) bool { return c.In && !c.Done }) && len(g.Cars) > 0 {
		g.end()
	}
}

// rank : les arrivés dans l'ordre d'arrivée, puis les autres selon la distance parcourue.
func (g *Kart) rank() {
	order := slices.Clone(g.Cars)
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.In != b.In {
			return a.In
		}
		if a.Done != b.Done {
			return a.Done
		}
		if a.Done {
			return a.Place < b.Place
		}
		return a.dist > b.dist
	})
	for i, c := range order {
		c.Rank = i + 1
		c.Lap = min(g.Laps, max(1, int(math.Floor(c.dist/g.c.L))+1))
	}
}

// end : fin de la course, points du championnat.
func (g *Kart) end() {
	g.rank()
	for _, c := range g.Cars {
		c.Gain = 0
		if c.In && c.Rank-1 < len(racePoints) {
			c.Gain = racePoints[c.Rank-1]
			c.Pts += c.Gain
		}
		c.steer = 0
	}
	g.Phase, g.Wait = "result", RaceResult
}

// brain : le bot vise un point un peu plus loin sur la piste (avec son couloir préféré) et utilise ses objets.
func (g *Kart) brain(car *Car, dt float64) {
	if car.think -= dt; car.think > 0 {
		return
	}
	car.think = 0.08 + rand.Float64()*0.06
	c := g.c
	i := car.idx + 11
	p := c.at(i)
	nx, ny := c.normal(i)
	tx, ty := p[0]+nx*car.lane-car.body.X, p[1]+ny*car.lane-car.body.Y
	diff := angleDiff(math.Atan2(ty, tx), car.head)
	car.steer = min(max(diff*2.2, -1), 1)
	switch car.Item {
	case "turbo":
		if math.Abs(diff) < 0.15 && rand.IntN(4) == 0 {
			g.use(car)
		}
	case "huile":
		for _, o := range g.Cars {
			if o != car && o.In && math.Hypot(o.body.X-car.body.X, o.body.Y-car.body.Y) < 140 && o.dist < car.dist {
				g.use(car)
				break
			}
		}
	}
}

func (g *Kart) Screen() any {
	for _, c := range g.Cars {
		c.X, c.Y = math.Round(c.body.X), math.Round(c.body.Y)
		c.A = math.Round(c.head*100) / 100
		c.Spin, c.Boost = c.spin > 0, c.boost > 0
	}
	s := *g
	s.Wait = math.Ceil(g.Wait)
	s.Finish = math.Ceil(g.Finish)
	s.Clock = math.Round(g.Clock*10) / 10
	g.Events = nil // envoyés une seule fois
	return tvBig{V: s, Big: g.c.big}
}

type kartView struct {
	Phase  string `json:"phase"`
	Wait   int    `json:"wait"`
	In     bool   `json:"in"`
	Rank   int    `json:"rank"`
	Of     int    `json:"of"`
	Lap    int    `json:"lap"`
	Laps   int    `json:"laps"`
	Item   string `json:"item"`
	Done   bool   `json:"done"`
	Place  int    `json:"place"`
	Pts    int    `json:"pts"`
	Gain   int    `json:"gain"`
	Pokes  int    `json:"pokes"`
	Boxes  int    `json:"boxes"`
	Bots   int    `json:"bots"`
	Name   string `json:"name"`
	Race   int    `json:"race"`
	Finish int    `json:"finish"`
}

func (g *Kart) View(p *Pad) any {
	v := kartView{Phase: g.Phase, Laps: g.Laps, Name: g.Name, Race: g.Race, Of: len(g.Cars), Finish: int(math.Ceil(g.Finish))}
	if g.Phase == "count" {
		v.Wait = int(math.Ceil(g.Wait))
	}
	for _, c := range g.Cars {
		if c.Bot {
			v.Bots++
		}
	}
	if c := g.get(p.ID); c != nil {
		v.In, v.Rank, v.Lap, v.Item, v.Done, v.Place, v.Pts, v.Gain, v.Pokes, v.Boxes = c.In, c.Rank, c.Lap, c.Item, c.Done, c.Place, c.Pts, c.Gain, c.pokes, c.gotBox
	}
	return v
}
