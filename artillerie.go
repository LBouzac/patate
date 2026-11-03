// Artillerie : de 2 à 4 chars (joueurs ou bots) sur un terrain vallonné, chacun son tour. On règle l'angle et
// la puissance comme au golf (on tire en arrière). Le vent change à chaque tour, les obus creusent le sol.
// Le dernier char debout gagne.
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
	AW, AH     = 800.0, 450.0
	Cols       = 200 // le terrain est découpé en colonnes de AW/Cols px
	TankR      = 12.0
	ShellMax   = 700.0 // vitesse d'un obus à pleine puissance (px/s)
	Gravity    = 400.0
	WindAcc    = 90.0 // accélération horizontale pour un vent de 1
	CraterR    = 32.0
	BlastR     = 55.0
	BlastDmg   = 70.0 // dégâts d'un tir direct
	ArtTurn    = 20.0
	ArtMax     = 4
	ArtTick    = time.Second / 30
	ArtSub     = 4
	ArtRooms   = 1000
	ArtBotWait = 1.5
)

type Tank struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Color string  `json:"color"`
	Bot   bool    `json:"bot"`
	HP    int     `json:"hp"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Alive bool    `json:"alive"`
	send  chan []byte
}

type Shell struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	vx, vy float64
	by     int     // char qui a tiré
	age    float64 // s depuis le tir : l'obus ne touche pas son propre char au départ
}

type Battle struct {
	mu     sync.Mutex
	Pin    string  `json:"pin"`
	Tanks  []*Tank `json:"tanks"`
	Ground []int   `json:"ground"` // hauteur du sol (depuis le bas) par colonne
	Shell  *Shell  `json:"shell"`
	Phase  string  `json:"phase"` // wait, aim, fly, boom, end
	Wait   float64 `json:"wait"`
	Turn   int     `json:"turn"` // id du char qui tire
	Wind   float64 `json:"wind"` // -1 (vers la gauche) à 1 (vers la droite)
	Winner string  `json:"winner"`
	Events []Event `json:"events"`
	height []float64
	nextID int
}

func newBattle(pin string) *Battle { return &Battle{Pin: pin, Phase: "wait", Ground: []int{}} }

func (g *Battle) event(e Event) { g.Events = append(g.Events, e) }

func (g *Battle) get(id int) *Tank {
	for _, t := range g.Tanks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// groundY : ordonnée (depuis le haut) de la surface à l'abscisse x.
func (g *Battle) groundY(x float64) float64 {
	i := min(max(int(x/(AW/Cols)), 0), Cols-1)
	return AH - g.height[i]
}

func (g *Battle) syncGround() {
	g.Ground = g.Ground[:0]
	for _, h := range g.height {
		g.Ground = append(g.Ground, int(math.Round(h)))
	}
}

// newGame : nouveau terrain (somme de sinus au hasard), chars répartis, vie pleine.
func (g *Battle) newGame() {
	g.height = make([]float64, Cols)
	type wave struct{ a, f, p float64 }
	ws := []wave{{70, 1 + rand.Float64(), rand.Float64() * 7}, {35, 2 + rand.Float64()*2, rand.Float64() * 7}, {15, 5 + rand.Float64()*4, rand.Float64() * 7}}
	for i := range g.height {
		x := float64(i) / Cols * 2 * math.Pi
		h := 150.0
		for _, w := range ws {
			h += w.a * math.Sin(w.f*x+w.p)
		}
		g.height[i] = min(max(h, 40), AH-120)
	}
	for i, t := range g.Tanks {
		t.HP, t.Alive = 100, true
		t.X = AW * float64(i+1) / float64(len(g.Tanks)+1)
		t.X += (rand.Float64()*2 - 1) * 30
		t.Y = g.groundY(t.X)
	}
	g.syncGround()
	g.Winner, g.Shell = "", nil
	g.Turn = g.Tanks[rand.IntN(len(g.Tanks))].ID
	g.newTurn(g.Turn)
}

func (g *Battle) newTurn(id int) {
	g.Turn, g.Phase, g.Wait = id, "aim", ArtTurn
	g.Wind = math.Round((rand.Float64()*2-1)*10) / 10
}

func (g *Battle) join(name string, bot bool) *Tank {
	if len(g.Tanks) >= ArtMax {
		return nil
	}
	used := map[string]bool{}
	for _, t := range g.Tanks {
		used[t.Color] = true
	}
	color := colors[0]
	for _, c := range colors {
		if !used[c] {
			color = c
			break
		}
	}
	g.nextID++
	t := &Tank{ID: g.nextID, Name: name, Color: color, Bot: bot}
	if !bot {
		t.send = make(chan []byte, 8)
	}
	g.Tanks = append(g.Tanks, t) // spectateur jusqu'à la prochaine partie
	if g.Phase == "wait" && len(g.Tanks) >= 2 {
		g.Phase, g.Wait = "end", 3 // petite attente puis première partie
	}
	return t
}

func (g *Battle) leave(id int) {
	g.Tanks = slices.DeleteFunc(g.Tanks, func(t *Tank) bool { return t.ID == id })
	if !slices.ContainsFunc(g.Tanks, func(t *Tank) bool { return !t.Bot }) {
		g.Tanks = nil // plus aucun humain : les bots partent aussi
	}
	if len(g.Tanks) < 2 {
		g.Phase, g.Wait, g.Shell = "wait", 0, nil
		return
	}
	if g.Phase == "aim" && g.Turn == id {
		g.nextTurn()
	} else if g.Phase != "end" {
		g.checkWin()
	}
}

func (g *Battle) removeBot() {
	for i := len(g.Tanks) - 1; i >= 0; i-- {
		if g.Tanks[i].Bot {
			g.leave(g.Tanks[i].ID)
			return
		}
	}
}

func (g *Battle) fire(t *Tank, angle, power float64) {
	if g.Phase != "aim" || g.Turn != t.ID || !t.Alive || !(power > 0) || math.IsNaN(angle) || math.IsInf(angle, 0) {
		return
	}
	v := min(power, 1) * ShellMax
	g.Shell = &Shell{X: t.X + math.Cos(angle)*16, Y: t.Y - 10 + math.Sin(angle)*16, vx: v * math.Cos(angle), vy: v * math.Sin(angle), by: t.ID}
	g.Phase = "fly"
	g.event(Event{Kind: "fire", A: t.ID, X: g.Shell.X, Y: g.Shell.Y})
}

// flight avance l'obus d'un sous-pas. Renvoie true s'il a fini sa course (explosion ou sortie).
func (g *Battle) flight(s *Shell, dt float64, wind float64) (done, boom bool) {
	s.age += dt
	s.vy += Gravity * dt
	s.vx += wind * WindAcc * dt
	s.X, s.Y = s.X+s.vx*dt, s.Y+s.vy*dt
	if s.X < 0 || s.X > AW || s.Y > AH {
		return true, false
	}
	if s.Y >= g.groundY(s.X) {
		return true, true
	}
	for _, t := range g.Tanks {
		if t.Alive && (t.ID != s.by || s.age > 0.3) && math.Hypot(s.X-t.X, s.Y-(t.Y-6)) < TankR+3 {
			return true, true
		}
	}
	return false, false
}

func (g *Battle) explode(x, y float64) {
	g.event(Event{Kind: "boom", X: x, Y: y})
	w := AW / Cols
	for i := range g.height { // cratère
		dx := (float64(i)+0.5)*w - x
		if math.Abs(dx) < CraterR {
			bottom := y + math.Sqrt(CraterR*CraterR-dx*dx)
			g.height[i] = max(min(g.height[i], AH-bottom), 5)
		}
	}
	g.syncGround()
	for _, t := range g.Tanks {
		if !t.Alive {
			continue
		}
		if d := math.Hypot(t.X-x, t.Y-6-y); d < BlastR {
			dmg := int(math.Round(BlastDmg * (1 - d/BlastR)))
			t.HP -= dmg
			g.event(Event{Kind: "hit", A: t.ID, B: dmg, X: t.X, Y: t.Y})
			if t.HP <= 0 {
				t.HP, t.Alive = 0, false
				g.event(Event{Kind: "dead", A: t.ID, X: t.X, Y: t.Y, Text: t.Name})
			}
		}
		t.Y = g.groundY(t.X) // le char retombe au fond du cratère
	}
}

func (g *Battle) alive() []*Tank {
	return slices.DeleteFunc(slices.Clone(g.Tanks), func(t *Tank) bool { return !t.Alive })
}

func (g *Battle) checkWin() bool {
	al := g.alive()
	if len(al) > 1 {
		return false
	}
	g.Phase, g.Wait, g.Shell = "end", 6, nil
	if len(al) == 1 {
		g.Winner = al[0].Name
		g.event(Event{Kind: "win", A: al[0].ID, Text: al[0].Name})
	}
	return true
}

func (g *Battle) nextTurn() {
	if g.checkWin() {
		return
	}
	i := slices.IndexFunc(g.Tanks, func(t *Tank) bool { return t.ID == g.Turn })
	for k := 1; k <= len(g.Tanks); k++ {
		if t := g.Tanks[(i+k+len(g.Tanks))%len(g.Tanks)]; t.Alive {
			g.newTurn(t.ID)
			return
		}
	}
}

func (g *Battle) step(dt float64) {
	switch g.Phase {
	case "aim":
		if t := g.get(g.Turn); t != nil && t.Bot && g.Wait <= ArtTurn-ArtBotWait {
			g.think(t)
			return
		}
		if g.Wait -= dt; g.Wait <= 0 {
			g.nextTurn()
		}
	case "fly":
		for range ArtSub {
			if done, boom := g.flight(g.Shell, dt/ArtSub, g.Wind); done {
				if boom {
					g.explode(g.Shell.X, g.Shell.Y)
				} else {
					g.event(Event{Kind: "miss", X: g.Shell.X, Y: g.Shell.Y})
				}
				g.Shell, g.Phase, g.Wait = nil, "boom", 1.2
				return
			}
		}
	case "boom":
		if g.Wait -= dt; g.Wait <= 0 {
			g.nextTurn()
		}
	case "end":
		if g.Wait -= dt; g.Wait <= 0 {
			g.newGame()
		}
	}
}

// think : le bot essaie des angles et des puissances (vent compris) et garde le tir qui tombe le plus près d'un adversaire.
func (g *Battle) think(t *Tank) {
	var target *Tank
	for _, o := range g.alive() {
		if o != t && (target == nil || math.Abs(o.X-t.X) < math.Abs(target.X-t.X)) {
			target = o
		}
	}
	if target == nil {
		return
	}
	bestA, bestP, bestD := -math.Pi/2, 0.5, math.Inf(1)
	for deg := 200.0; deg <= 340; deg += 4 { // vers le haut, de la gauche à la droite
		a := deg * math.Pi / 180
		for p := 0.3; p <= 1.0; p += 0.05 {
			s := &Shell{X: t.X + math.Cos(a)*16, Y: t.Y - 10 + math.Sin(a)*16, vx: p * ShellMax * math.Cos(a), vy: p * ShellMax * math.Sin(a), by: t.ID}
			for range 1200 { // 20 s de vol au plus
				if done, _ := g.flight(s, 1.0/60, g.Wind); done {
					break
				}
			}
			if d := math.Hypot(s.X-target.X, s.Y-target.Y); d < bestD {
				bestA, bestP, bestD = a, p, d
			}
		}
	}
	g.fire(t, bestA+(rand.Float64()*2-1)*0.03, bestP*(1+(rand.Float64()*2-1)*0.03))
}

// --- salons Artillerie ---

var (
	battlesMu sync.Mutex
	battles   = map[string]*Battle{}
)

func enterBattle(pin, name string) (*Battle, *Tank, string) {
	battlesMu.Lock()
	defer battlesMu.Unlock()
	if pin == "" {
		if len(battles) >= ArtRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		g := newBattle(newPin(func(p string) bool { return battles[p] != nil }))
		battles[g.Pin] = g
		t := g.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go g.run()
		return g, t, ""
	}
	g := battles[pin]
	if g == nil {
		return nil, nil, "Code inconnu"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if t := g.join(name, false); t != nil {
		return g, t, ""
	}
	return nil, nil, "Partie pleine (4 joueurs)"
}

func (g *Battle) run() {
	tk := time.NewTicker(ArtTick)
	defer tk.Stop()
	for range tk.C {
		if g.closeIfEmpty() {
			return
		}
		g.mu.Lock()
		g.step(ArtTick.Seconds())
		b, _ := json.Marshal(g)
		g.Events = nil
		for _, t := range g.Tanks {
			if t.send != nil {
				select {
				case t.send <- b:
				default: // client lent : on saute cette image
				}
			}
		}
		g.mu.Unlock()
	}
}

func (g *Battle) closeIfEmpty() bool {
	battlesMu.Lock()
	defer battlesMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.Tanks) > 0 {
		return false
	}
	delete(battles, g.Pin)
	return true
}

func serveArtillerie(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		g, t, why := enterBattle(q.Get("pin"), name)
		if t == nil {
			return nil, why
		}
		return &Seat{
			ID:    t.ID,
			Send:  t.send,
			Leave: func() { g.mu.Lock(); g.leave(t.ID); g.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Bot int     `json:"bot"`
					A   float64 `json:"a"`
					P   float64 `json:"p"`
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				g.mu.Lock()
				defer g.mu.Unlock()
				switch m.Bot {
				case 1:
					g.join("Bot", true)
				case -1:
					g.removeBot()
				default:
					g.fire(t, m.A, m.P)
				}
			},
		}, ""
	})
}
