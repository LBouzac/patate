// Barrage : de 2 à 4 joueurs (bots possibles) sur un plateau de 9×9. À son tour, on avance son pion d'une case ou on
// pose une barre de 2 cases de long entre les cases, sans jamais fermer tout passage à quelqu'un. À 2, le premier
// arrivé dans le camp d'en face gagne ; à 3 ou 4, chacun part d'un coin et le premier qui touche le trophée au
// centre gagne.
package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"
)

const (
	BarN       = 9 // cases par côté
	BarMax     = 4
	BarTurn    = 30.0 // secondes par coup ; trop long : le pion avance tout seul
	BarBotWait = 1.0
	BarEnd     = 6.0
	BarTick    = time.Second / 10
	BarRooms   = 1000
	BarCenter  = BarN / 2
)

// barWalls : barres de chacun selon le nombre de joueurs
var barWalls = map[int]int{2: 10, 3: 7, 4: 5}

// Fence : une barre posée sur le croisement (X, Y) des rainures, entre les cases X/X+1 et Y/Y+1.
// Horizontale (V faux) : bloque le passage entre les lignes Y et Y+1, sur les colonnes X et X+1.
// Verticale : bloque le passage entre les colonnes X et X+1, sur les lignes Y et Y+1.
type Fence struct {
	X  int  `json:"x"`
	Y  int  `json:"y"`
	V  bool `json:"v"`
	By int  `json:"by"`
}

type Pawn struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Score int    `json:"score"` // parties gagnées
	Walls int    `json:"walls"` // barres restantes
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Goal  int    `json:"goal"` // ligne à atteindre, -1 : la case du centre
	In    bool   `json:"in"`   // joue la partie en cours (sinon il attend la suivante)
	Bot   bool   `json:"bot"`
	send  chan []byte
}

type Barrage struct {
	mu      sync.Mutex
	Pin     string   `json:"pin"`
	Players []*Pawn  `json:"players"`
	Fences  []Fence  `json:"fences"`
	Phase   string   `json:"phase"` // wait, play, end
	Wait    float64  `json:"wait"`
	Turn    int      `json:"turn"`
	Moves   [][2]int `json:"moves"` // cases où peut aller celui dont c'est le tour
	Center  bool     `json:"center"`
	Winner  string   `json:"winner"`
	Events  []Event  `json:"events"`
	starter int      // qui commence la prochaine partie (tourne à chaque partie)
	nextID  int
}

func newBarrage(pin string) *Barrage {
	return &Barrage{Pin: pin, Phase: "wait", Fences: []Fence{}, Moves: [][2]int{}}
}

func (g *Barrage) event(e Event) { g.Events = append(g.Events, e) }

func (g *Barrage) get(id int) *Pawn {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (g *Barrage) playing() []*Pawn {
	var in []*Pawn
	for _, p := range g.Players {
		if p.In {
			in = append(in, p)
		}
	}
	return in
}

func (g *Barrage) join(name string, bot bool) *Pawn {
	if len(g.Players) >= BarMax {
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
	p := &Pawn{ID: g.nextID, Name: name, Color: color, Bot: bot, Goal: -1}
	if !bot {
		p.send = make(chan []byte, 8)
	}
	g.Players = append(g.Players, p) // il joue à partir de la prochaine partie
	if g.Phase == "wait" {
		g.Wait = 3
	}
	return p
}

func (g *Barrage) leave(id int) {
	if g.Phase == "play" && g.Turn == id {
		g.next()
	}
	g.Players = slices.DeleteFunc(g.Players, func(p *Pawn) bool { return p.ID == id })
	if !slices.ContainsFunc(g.Players, func(p *Pawn) bool { return !p.Bot }) {
		g.Players = nil // plus aucun humain : les bots partent aussi
	}
	if len(g.Players) < 2 {
		g.Phase, g.Wait, g.Fences, g.Moves, g.Winner = "wait", 3, []Fence{}, [][2]int{}, ""
		for _, p := range g.Players {
			p.In = false
		}
		return
	}
	if g.Phase != "play" {
		return
	}
	if in := g.playing(); len(in) == 1 { // tous les autres sont partis : le dernier gagne
		g.win(in[0])
		return
	}
	g.Moves = g.moves(g.get(g.Turn)) // un pion parti libère sa case
}

func (g *Barrage) removeBot() {
	for i := len(g.Players) - 1; i >= 0; i-- {
		if g.Players[i].Bot {
			g.leave(g.Players[i].ID)
			return
		}
	}
}

// start : nouvelle partie avec tous les présents. À 2, face à face ; à 3 ou 4, dans les coins, trophée au centre.
func (g *Barrage) start() {
	n := len(g.Players)
	g.Center, g.Fences, g.Winner = n > 2, []Fence{}, ""
	sides := [][3]int{{BarCenter, BarN - 1, 0}, {BarCenter, 0, BarN - 1}}
	corners := [][3]int{{0, BarN - 1, -1}, {BarN - 1, 0, -1}, {0, 0, -1}, {BarN - 1, BarN - 1, -1}}
	for i, p := range g.Players {
		s := sides
		if g.Center {
			s = corners
		}
		p.X, p.Y, p.Goal, p.Walls, p.In = s[i][0], s[i][1], s[i][2], barWalls[n], true
	}
	first := g.Players[g.starter%n]
	g.starter++
	g.Phase, g.Turn, g.Wait, g.Moves = "play", first.ID, BarTurn, g.moves(first)
	g.event(Event{Kind: "start"})
}

func (g *Barrage) next() {
	in := g.playing()
	i := slices.IndexFunc(in, func(p *Pawn) bool { return p.ID == g.Turn })
	p := in[(i+1)%len(in)]
	g.Turn, g.Wait, g.Moves = p.ID, BarTurn, g.moves(p)
}

func (g *Barrage) win(p *Pawn) {
	p.Score++
	g.Phase, g.Wait, g.Winner, g.Moves = "end", BarEnd, p.Name, [][2]int{}
	g.event(Event{Kind: "win", A: p.ID, Text: p.Name})
}

func reached(p *Pawn, x, y int) bool {
	if p.Goal < 0 {
		return x == BarCenter && y == BarCenter
	}
	return y == p.Goal
}

func inBoard(x, y int) bool { return x >= 0 && x < BarN && y >= 0 && y < BarN }

// blocked : une barre sépare-t-elle ces deux cases voisines ?
func blocked(fs []Fence, x1, y1, x2, y2 int) bool {
	for _, f := range fs {
		if x1 == x2 && !f.V && f.Y == min(y1, y2) && (f.X == x1 || f.X == x1-1) {
			return true
		}
		if y1 == y2 && f.V && f.X == min(x1, x2) && (f.Y == y1 || f.Y == y1-1) {
			return true
		}
	}
	return false
}

var barDirs = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// dist : nombre de pas jusqu'au but de p en partant de (x, y), sans compter les pions ; -1 si c'est fermé.
func dist(fs []Fence, p *Pawn, x, y int) int {
	var seen [BarN][BarN]bool
	seen[x][y] = true
	q, d := [][2]int{{x, y}}, 0
	for len(q) > 0 {
		var nq [][2]int
		for _, c := range q {
			if reached(p, c[0], c[1]) {
				return d
			}
			for _, dd := range barDirs {
				nx, ny := c[0]+dd[0], c[1]+dd[1]
				if inBoard(nx, ny) && !seen[nx][ny] && !blocked(fs, c[0], c[1], nx, ny) {
					seen[nx][ny] = true
					nq = append(nq, [2]int{nx, ny})
				}
			}
		}
		q, d = nq, d+1
	}
	return -1
}

func (g *Barrage) occupied(x, y int) bool {
	return slices.ContainsFunc(g.playing(), func(p *Pawn) bool { return p.X == x && p.Y == y })
}

// moves : une case dans chaque direction. Un pion juste devant se saute ; si une barre ou un pion empêche de sauter
// tout droit, on passe à côté de lui en diagonale.
func (g *Barrage) moves(p *Pawn) [][2]int {
	out := [][2]int{}
	if p == nil {
		return out
	}
	free := func(x, y int) bool { return inBoard(x, y) && !g.occupied(x, y) }
	for _, d := range barDirs {
		nx, ny := p.X+d[0], p.Y+d[1]
		if !inBoard(nx, ny) || blocked(g.Fences, p.X, p.Y, nx, ny) {
			continue
		}
		if !g.occupied(nx, ny) {
			out = append(out, [2]int{nx, ny})
			continue
		}
		if jx, jy := nx+d[0], ny+d[1]; free(jx, jy) && !blocked(g.Fences, nx, ny, jx, jy) {
			out = append(out, [2]int{jx, jy})
			continue
		}
		for _, s := range [2][2]int{{d[1], d[0]}, {-d[1], -d[0]}} { // les deux côtés
			if sx, sy := nx+s[0], ny+s[1]; free(sx, sy) && !blocked(g.Fences, nx, ny, sx, sy) && !slices.Contains(out, [2]int{sx, sy}) {
				out = append(out, [2]int{sx, sy})
			}
		}
	}
	return out
}

// fits : la barre tient sur le plateau sans chevaucher ni croiser une autre.
func (g *Barrage) fits(f Fence) bool {
	if f.X < 0 || f.X >= BarN-1 || f.Y < 0 || f.Y >= BarN-1 {
		return false
	}
	for _, e := range g.Fences {
		if e.X == f.X && e.Y == f.Y || e.V == f.V && (!f.V && e.Y == f.Y && abs(e.X-f.X) == 1 || f.V && e.X == f.X && abs(e.Y-f.Y) == 1) {
			return false
		}
	}
	return true
}

func abs(n int) int { return max(n, -n) }

// dists : la distance au but de chaque joueur avec ces barres, ok faux si l'un d'eux est enfermé.
func (g *Barrage) dists(fs []Fence) (map[int]int, bool) {
	out := map[int]int{}
	for _, p := range g.playing() {
		if out[p.ID] = dist(fs, p, p.X, p.Y); out[p.ID] < 0 {
			return out, false
		}
	}
	return out, true
}

func (g *Barrage) move(p *Pawn, x, y int) {
	if g.Phase != "play" || g.Turn != p.ID || !slices.Contains(g.Moves, [2]int{x, y}) {
		return
	}
	p.X, p.Y = x, y
	g.event(Event{Kind: "move", A: p.ID})
	if reached(p, x, y) {
		g.win(p)
		return
	}
	g.next()
}

func (g *Barrage) wall(p *Pawn, f Fence) {
	if g.Phase != "play" || g.Turn != p.ID || p.Walls <= 0 || !g.fits(f) {
		return
	}
	f.By = p.ID
	if _, ok := g.dists(append(slices.Clone(g.Fences), f)); !ok {
		g.event(Event{Kind: "bad", A: p.ID})
		return
	}
	g.Fences = append(g.Fences, f)
	p.Walls--
	g.event(Event{Kind: "wall", A: p.ID})
	g.next()
}

// stepToGoal : le pas qui rapproche le plus du but (au hasard entre les égalités).
func (g *Barrage) stepToGoal(p *Pawn) {
	best, pick := -1, [][2]int{}
	for _, m := range g.Moves {
		d := dist(g.Fences, p, m[0], m[1])
		switch {
		case d < 0:
		case best < 0 || d < best:
			best, pick = d, [][2]int{m}
		case d == best:
			pick = append(pick, m)
		}
	}
	if len(pick) == 0 { // aucun pas possible (encerclé par des pions) : il passe son tour
		g.next()
		return
	}
	m := pick[rand.IntN(len(pick))]
	g.move(p, m[0], m[1])
}

// think : si un adversaire est plus près du but, le bot cherche la barre qui le retarde le plus sans trop le gêner
// lui-même ; sinon il avance. Il lui arrive de ne pas voir la bonne barre : un humain peut le battre.
func (g *Barrage) think(b *Pawn) {
	now, _ := g.dists(g.Fences)
	rival := func(d map[int]int) int {
		r := -1
		for id, v := range d {
			if id != b.ID && (r < 0 || v < r) {
				r = v
			}
		}
		return r
	}
	if b.Walls > 0 && rival(now) < now[b.ID] && rand.Float64() < 0.8 {
		var best Fence
		gain := 0
		for x := range BarN - 1 {
			for y := range BarN - 1 {
				for _, v := range []bool{false, true} {
					f := Fence{X: x, Y: y, V: v}
					if !g.fits(f) {
						continue
					}
					d, ok := g.dists(append(slices.Clone(g.Fences), f))
					if gn := (rival(d) - rival(now)) - (d[b.ID] - now[b.ID]); ok && gn > gain {
						best, gain = f, gn
					}
				}
			}
		}
		if gain > 0 {
			g.wall(b, best)
			return
		}
	}
	g.stepToGoal(b)
}

func (g *Barrage) step(dt float64) {
	switch g.Phase {
	case "wait":
		if len(g.Players) < 2 {
			g.Wait = 3
			return
		}
		if g.Wait -= dt; g.Wait <= 0 {
			g.start()
		}
	case "play":
		p := g.get(g.Turn)
		if p.Bot && g.Wait <= BarTurn-BarBotWait {
			g.think(p)
			return
		}
		if g.Wait -= dt; g.Wait <= 0 {
			g.stepToGoal(p)
		}
	case "end":
		if g.Wait -= dt; g.Wait <= 0 {
			g.start() // revanche avec tous les présents, nouveaux venus compris
		}
	}
}

// --- salons Barrage (même principe que les autres jeux, codes séparés) ---

var (
	barragesMu sync.Mutex
	barrages   = map[string]*Barrage{}
)

func enterBarrage(pin, name string) (*Barrage, *Pawn, string) {
	barragesMu.Lock()
	defer barragesMu.Unlock()
	if pin == "" {
		if len(barrages) >= BarRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		g := newBarrage(newPin(func(p string) bool { return barrages[p] != nil }))
		barrages[g.Pin] = g
		p := g.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go g.run()
		return g, p, ""
	}
	g := barrages[pin]
	if g == nil {
		return nil, nil, "Code inconnu"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if p := g.join(name, false); p != nil {
		return g, p, ""
	}
	return nil, nil, "Partie pleine (4 joueurs)"
}

func (g *Barrage) run() {
	t := time.NewTicker(BarTick)
	defer t.Stop()
	for range t.C {
		if g.closeIfEmpty() {
			return
		}
		g.mu.Lock()
		g.step(BarTick.Seconds())
		b, _ := json.Marshal(g) // envoi continu : le client sait ainsi que la connexion est vivante
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

func (g *Barrage) closeIfEmpty() bool {
	barragesMu.Lock()
	defer barragesMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.Players) > 0 {
		return false
	}
	delete(barrages, g.Pin)
	return true
}

func serveBarrage(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		g, p, why := enterBarrage(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			Room:  g.Pin,
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { g.mu.Lock(); g.leave(p.ID); g.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Bot  int     `json:"bot"` // +1 ajoute un bot, -1 en retire un
					Move *[2]int `json:"move"`
					Wall *Fence  `json:"wall"`
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				g.mu.Lock()
				defer g.mu.Unlock()
				switch {
				case m.Bot == 1:
					g.join("Bot "+strconv.Itoa(len(g.Players)), true) // numérotés : chacun son initiale sur le pion
				case m.Bot == -1:
					g.removeBot()
				case m.Move != nil:
					g.move(p, m.Move[0], m.Move[1])
				case m.Wall != nil:
					g.wall(p, *m.Wall)
				}
			},
		}, ""
	})
}
