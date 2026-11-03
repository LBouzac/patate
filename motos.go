// Motos : de 2 à 8 (joueurs ou bots) en temps réel sur une grille. Chaque moto avance toute seule et laisse un mur
// derrière elle ; on tourne à gauche ou à droite. Toucher un mur, un bord ou une autre moto élimine.
// Le dernier en piste marque un point ; premier à 5 points.
package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"
)

const (
	MotoCols  = 100
	MotoRows  = 60
	MotoTick  = time.Second / 12 // une case par tick
	MotoMax   = 8
	MotoWin   = 5
	MotoRooms = 1000
)

// directions : 0 droite, 1 bas, 2 gauche, 3 haut
var dirs = [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

type Rider struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Color string   `json:"color"`
	Bot   bool     `json:"bot"`
	Alive bool     `json:"alive"`
	Score int      `json:"score"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Dir   int      `json:"dir"`
	Pts   [][2]int `json:"pts"` // départ puis chaque virage : avec la tête, ça dessine le mur
	queue []int    // virages demandés, appliqués un par tick
	send  chan []byte
}

type Track struct {
	mu       sync.Mutex
	Pin      string                  `json:"pin"`
	Riders   []*Rider                `json:"riders"`
	Phase    string                  `json:"phase"` // wait, count, play, round, end
	Wait     float64                 `json:"wait"`
	Winner   string                  `json:"winner"`   // vainqueur de la manche
	Champion string                  `json:"champion"` // vainqueur de la partie
	Round    int                     `json:"round"`
	Events   []Event                 `json:"events"`
	grid     [MotoCols][MotoRows]int // id de la moto qui occupe la case, 0 si libre
	nextID   int
}

func newTrack(pin string) *Track { return &Track{Pin: pin, Phase: "wait"} }

func (g *Track) event(e Event) { g.Events = append(g.Events, e) }

func (g *Track) join(name string, bot bool) *Rider {
	if len(g.Riders) >= MotoMax {
		return nil
	}
	used := map[string]bool{}
	for _, r := range g.Riders {
		used[r.Color] = true
	}
	color := colors[0]
	for _, c := range colors {
		if !used[c] {
			color = c
			break
		}
	}
	g.nextID++
	r := &Rider{ID: g.nextID, Name: name, Color: color, Bot: bot, Pts: [][2]int{}}
	if !bot {
		r.send = make(chan []byte, 8)
	}
	g.Riders = append(g.Riders, r) // il entre en piste à la prochaine manche
	if g.Phase == "wait" && len(g.Riders) >= 2 {
		g.startRound()
	}
	return r
}

func (g *Track) leave(id int) {
	g.Riders = slices.DeleteFunc(g.Riders, func(r *Rider) bool { return r.ID == id })
	if !slices.ContainsFunc(g.Riders, func(r *Rider) bool { return !r.Bot }) {
		g.Riders = nil // plus aucun humain : les bots partent aussi
	}
	if len(g.Riders) < 2 {
		g.Phase, g.Wait = "wait", 0
		for _, r := range g.Riders {
			r.Score, r.Alive = 0, false
		}
		return
	}
	if g.Phase == "play" {
		g.checkRound()
	}
}

func (g *Track) removeBot() {
	for i := len(g.Riders) - 1; i >= 0; i-- {
		if g.Riders[i].Bot {
			g.leave(g.Riders[i].ID)
			return
		}
	}
}

// startRound : grille vide, motos réparties sur les bords gauche et droit, face au centre.
func (g *Track) startRound() {
	g.grid = [MotoCols][MotoRows]int{}
	g.Round++
	g.Winner = ""
	n := len(g.Riders)
	perSide := (n + 1) / 2
	for i, r := range g.Riders {
		// lignes intercalées : gauche sur les lignes impaires, droite sur les paires, jamais face à face
		side, k := i%2, i/2
		r.Y = (2*k + 1 + side) * MotoRows / (2*perSide + 1)
		r.X, r.Dir = 8, 0
		if side == 1 {
			r.X, r.Dir = MotoCols-9, 2
		}
		r.Alive, r.queue = true, nil
		r.Pts = [][2]int{{r.X, r.Y}}
		g.grid[r.X][r.Y] = r.ID
	}
	g.Phase, g.Wait = "count", 3
}

// steer : virage absolu (dir 0 à 3) ou relatif (turn -1 gauche, +1 droite). Demi-tour interdit.
func (r *Rider) steer(dir, turn int) {
	if len(r.queue) >= 2 {
		return
	}
	last := r.Dir
	if len(r.queue) > 0 {
		last = r.queue[len(r.queue)-1]
	}
	if turn != 0 {
		dir = (last + turn + 4) % 4
	}
	if dir < 0 || dir > 3 || dir == last || dir == (last+2)%4 {
		return
	}
	r.queue = append(r.queue, dir)
}

func (g *Track) free(x, y int) bool {
	return x >= 0 && x < MotoCols && y >= 0 && y < MotoRows && g.grid[x][y] == 0
}

func (g *Track) move() {
	type head struct{ x, y int }
	next := map[*Rider]head{}
	for _, r := range g.Riders {
		if !r.Alive {
			continue
		}
		if r.Bot {
			g.think(r)
		}
		if len(r.queue) > 0 {
			r.Dir, r.queue = r.queue[0], r.queue[1:]
			r.Pts = append(r.Pts, [2]int{r.X, r.Y})
		}
		next[r] = head{r.X + dirs[r.Dir][0], r.Y + dirs[r.Dir][1]}
	}
	for r, h := range next {
		crash := !g.free(h.x, h.y)
		for o, oh := range next {
			if o != r && oh == h { // deux motos sur la même case
				crash = true
			}
		}
		if crash {
			r.Alive = false
			g.event(Event{Kind: "crash", A: r.ID, X: float64(r.X), Y: float64(r.Y), Text: r.Name})
		}
	}
	for r, h := range next {
		if r.Alive {
			r.X, r.Y = h.x, h.y
			g.grid[h.x][h.y] = r.ID
		}
	}
	g.checkRound()
}

func (g *Track) checkRound() {
	var alive []*Rider
	for _, r := range g.Riders {
		if r.Alive {
			alive = append(alive, r)
		}
	}
	if len(alive) > 1 {
		return
	}
	g.Phase, g.Wait = "round", 3
	if len(alive) == 1 {
		w := alive[0]
		w.Score++
		w.Alive = false
		g.Winner = w.Name
		g.event(Event{Kind: "round", A: w.ID, Text: w.Name})
		if w.Score >= MotoWin {
			g.Phase, g.Wait, g.Champion = "end", 6, w.Name
			g.event(Event{Kind: "win", A: w.ID, Text: w.Name})
		}
	}
}

// think : le bot garde le cap tant que c'est libre (avec parfois un virage au hasard), sinon il prend la direction la plus dégagée.
func (g *Track) think(r *Rider) {
	ahead := func(d int) bool { return g.free(r.X+dirs[d][0], r.Y+dirs[d][1]) }
	if ahead(r.Dir) && g.free(r.X+2*dirs[r.Dir][0], r.Y+2*dirs[r.Dir][1]) && rand.IntN(30) > 0 {
		return
	}
	best, bestN := r.Dir, -1
	for _, d := range []int{r.Dir, (r.Dir + 1) % 4, (r.Dir + 3) % 4} {
		if !ahead(d) {
			continue
		}
		if n := g.space(r.X+dirs[d][0], r.Y+dirs[d][1], 150); n > bestN {
			best, bestN = d, n
		}
	}
	if best != r.Dir {
		r.queue = []int{best}
	}
}

// space compte les cases libres atteignables depuis (x, y), jusqu'à limit.
func (g *Track) space(x, y, limit int) int {
	seen := map[[2]int]bool{{x, y}: true}
	queue := [][2]int{{x, y}}
	for len(queue) > 0 && len(seen) < limit {
		p := queue[0]
		queue = queue[1:]
		for _, d := range dirs {
			n := [2]int{p[0] + d[0], p[1] + d[1]}
			if !seen[n] && g.free(n[0], n[1]) {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return len(seen)
}

func (g *Track) step(dt float64) {
	switch g.Phase {
	case "count", "round":
		if g.Wait -= dt; g.Wait <= 0 {
			if g.Phase == "round" {
				g.startRound()
			} else {
				g.Phase = "play"
			}
		}
	case "play":
		g.move()
	case "end":
		if g.Wait -= dt; g.Wait <= 0 {
			g.Champion = ""
			for _, r := range g.Riders {
				r.Score = 0
			}
			g.startRound()
		}
	}
}

// --- salons Motos ---

var (
	tracksMu sync.Mutex
	tracks   = map[string]*Track{}
)

func enterTrack(pin, name string) (*Track, *Rider, string) {
	tracksMu.Lock()
	defer tracksMu.Unlock()
	if pin == "" {
		if len(tracks) >= MotoRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		g := newTrack(newPin(func(p string) bool { return tracks[p] != nil }))
		tracks[g.Pin] = g
		r := g.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go g.run()
		return g, r, ""
	}
	g := tracks[pin]
	if g == nil {
		return nil, nil, "Code inconnu"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if r := g.join(name, false); r != nil {
		return g, r, ""
	}
	return nil, nil, "Partie pleine (8 joueurs)"
}

func (g *Track) run() {
	tk := time.NewTicker(MotoTick)
	defer tk.Stop()
	for range tk.C {
		if g.closeIfEmpty() {
			return
		}
		g.mu.Lock()
		g.step(MotoTick.Seconds())
		b, _ := json.Marshal(g)
		g.Events = nil
		for _, r := range g.Riders {
			if r.send != nil {
				select {
				case r.send <- b:
				default: // client lent : on saute cette image
				}
			}
		}
		g.mu.Unlock()
	}
}

func (g *Track) closeIfEmpty() bool {
	tracksMu.Lock()
	defer tracksMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.Riders) > 0 {
		return false
	}
	delete(tracks, g.Pin)
	return true
}

func serveMotos(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		g, rd, why := enterTrack(q.Get("pin"), name)
		if rd == nil {
			return nil, why
		}
		return &Seat{
			ID:    rd.ID,
			Send:  rd.send,
			Leave: func() { g.mu.Lock(); g.leave(rd.ID); g.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Bot  int  `json:"bot"`
					Dir  *int `json:"dir"`
					Turn int  `json:"turn"`
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
				case g.Phase == "play" && rd.Alive && m.Dir != nil:
					rd.steer(*m.Dir, 0)
				case g.Phase == "play" && rd.Alive && (m.Turn == 1 || m.Turn == -1):
					rd.steer(0, m.Turn)
				}
			},
		}, ""
	})
}
