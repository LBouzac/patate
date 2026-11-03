// Dessine : chacun son tour, un joueur envoie une photo ; un autre, tiré au hasard, la dessine en 30 s
// pendant que les autres devinent. L'auteur de la photo valide chaque proposition ; plus on trouve vite,
// plus on marque.
package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	DrawTime    = 30.0
	JudgeTime   = 5.0 // après le chrono, pour juger les propositions restantes
	RevealTime  = 6.0
	PhotoTime   = 60.0 // sans photo passé ce délai, on passe au joueur suivant
	SketchMin   = 3
	SketchMax   = 10
	SketchTick  = time.Second / 5
	CW, CH      = 1000, 750 // coordonnées du dessin
	PhotoMax    = 300_000   // taille max de la photo (data URL)
	DrawerBonus = 25        // pour le dessinateur, par joueur qui trouve
	MaxGuesses  = 150       // par manche
	MaxPoints   = 40_000    // coordonnées par manche (~320 Ko) : borne la mémoire d'un salon
	MaxSketches = 50        // salons simultanés : une photo de 300 Ko chacun au plus
)

type Sketcher struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
	Gain  int    `json:"gain"` // points de la manche en cours
	Found bool   `json:"found"`
	send  chan []byte
}

type Guess struct {
	ID   int    `json:"id"`
	By   int    `json:"by"`
	Text string `json:"text"`
	Pts  int    `json:"pts"` // fixés à l'envoi : la lenteur du juge ne coûte rien
	OK   *bool  `json:"ok"`  // nil = pas encore jugée
}

type Sketch struct {
	mu        sync.Mutex
	Pin       string      `json:"pin"`
	Players   []*Sketcher `json:"players"`
	Phase     string      `json:"phase"` // wait, photo, draw, judge, reveal, end
	Wait      float64     `json:"wait"`
	Owner     int         `json:"owner"`  // auteur de la photo
	Drawer    int         `json:"drawer"` // dessinateur
	Turn      int         `json:"turn"`   // tours déjà commencés
	Guesses   []Guess     `json:"guesses"`
	order     []int       // qui envoie une photo, dans l'ordre
	photo     string
	lines     [][]int
	points    int // coordonnées dans lines
	nextID    int
	nextGuess int
}

func newSketch(pin string) *Sketch { return &Sketch{Pin: pin, Phase: "wait"} }

func (s *Sketch) get(id int) *Sketcher {
	for _, p := range s.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func push(p *Sketcher, b []byte) {
	select {
	case p.send <- b:
	default: // shortcut: client saturé, le message est perdu (un trait peut manquer) ; renvoyer le dessin entier si ça gêne
	}
}

func (s *Sketch) broadcast(m any, to func(*Sketcher) bool) {
	b, _ := json.Marshal(m)
	for _, p := range s.Players {
		if to(p) {
			push(p, b)
		}
	}
}

func (s *Sketch) join(name string) *Sketcher {
	if len(s.Players) >= SketchMax {
		return nil
	}
	s.nextID++
	p := &Sketcher{ID: s.nextID, Name: name, send: make(chan []byte, 256)}
	s.Players = append(s.Players, p)
	if s.Phase != "wait" && s.Phase != "end" {
		s.order = append(s.order, p.ID) // il aura son tour
	}
	if len(s.lines) > 0 {
		b, _ := json.Marshal(map[string]any{"lines": s.lines})
		push(p, b)
	}
	return p
}

func (s *Sketch) leave(id int) {
	s.Players = slices.DeleteFunc(s.Players, func(p *Sketcher) bool { return p.ID == id })
	switch s.Phase {
	case "photo", "draw", "judge":
		if id == s.Owner || id == s.Drawer || len(s.Players) < SketchMin {
			s.nextTurn() // manche annulée
		}
	case "reveal":
		if len(s.Players) < SketchMin {
			s.nextTurn()
		}
	}
}

func (s *Sketch) start() {
	if (s.Phase != "wait" && s.Phase != "end") || len(s.Players) < SketchMin {
		return
	}
	s.order, s.Turn = nil, 0
	for _, p := range s.Players {
		p.Score = 0
		s.order = append(s.order, p.ID)
	}
	s.nextTurn()
}

func (s *Sketch) nextTurn() {
	s.Guesses, s.lines, s.points, s.photo, s.Owner, s.Drawer, s.Wait = nil, nil, 0, "", 0, 0, 0
	for _, p := range s.Players {
		p.Gain, p.Found = 0, false
	}
	if len(s.Players) < SketchMin {
		s.Phase = "wait"
		return
	}
	for s.Turn < len(s.order) {
		id := s.order[s.Turn]
		s.Turn++
		if s.get(id) != nil {
			s.Phase, s.Wait, s.Owner = "photo", PhotoTime, id
			return
		}
	}
	s.Phase = "end"
}

func (s *Sketch) setPhoto(p *Sketcher, data string) {
	if s.Phase != "photo" || p.ID != s.Owner || len(data) > PhotoMax || !strings.HasPrefix(data, "data:image/jpeg;base64,") {
		return
	}
	others := slices.DeleteFunc(slices.Clone(s.Players), func(o *Sketcher) bool { return o.ID == s.Owner })
	s.Drawer = others[rand.IntN(len(others))].ID
	s.photo, s.Phase, s.Wait = data, "draw", DrawTime
	s.broadcast(map[string]string{"photo": data}, func(o *Sketcher) bool { return o.ID == s.Owner || o.ID == s.Drawer })
}

func (s *Sketch) addLine(p *Sketcher, pts []int) {
	if s.Phase != "draw" || p.ID != s.Drawer || len(pts) < 4 || len(pts) > 400 || len(pts)%2 != 0 || s.points+len(pts) > MaxPoints {
		return
	}
	for i := range pts {
		pts[i] = min(max(pts[i], 0), []int{CW, CH}[i%2])
	}
	s.lines = append(s.lines, pts)
	s.points += len(pts)
	s.broadcast(map[string][]int{"line": pts}, func(o *Sketcher) bool { return o.ID != s.Drawer })
}

func (s *Sketch) clear(p *Sketcher) {
	if s.Phase != "draw" || p.ID != s.Drawer {
		return
	}
	s.lines, s.points = nil, 0
	s.broadcast(map[string]bool{"clear": true}, func(o *Sketcher) bool { return o.ID != s.Drawer })
}

func (s *Sketch) guess(p *Sketcher, text string) {
	text = strings.TrimSpace(text)
	if r := []rune(text); len(r) > 40 {
		text = string(r[:40])
	}
	if s.Phase != "draw" || p.ID == s.Owner || p.ID == s.Drawer || p.Found || text == "" || len(s.Guesses) >= MaxGuesses {
		return
	}
	s.nextGuess++
	pts := 10 + int(90*s.Wait/DrawTime+0.5) // 100 tout de suite, 10 à la fin
	s.Guesses = append(s.Guesses, Guess{ID: s.nextGuess, By: p.ID, Text: text, Pts: pts})
}

func (s *Sketch) judge(p *Sketcher, id int, ok bool) {
	if (s.Phase != "draw" && s.Phase != "judge") || p.ID != s.Owner {
		return
	}
	i := slices.IndexFunc(s.Guesses, func(g Guess) bool { return g.ID == id && g.OK == nil })
	if i < 0 {
		return
	}
	g := &s.Guesses[i]
	g.OK = &ok
	if by := s.get(g.By); ok && by != nil && !by.Found {
		by.Found, by.Score, by.Gain = true, by.Score+g.Pts, by.Gain+g.Pts
		if d := s.get(s.Drawer); d != nil {
			d.Score, d.Gain = d.Score+DrawerBonus, d.Gain+DrawerBonus
		}
	}
	allFound := !slices.ContainsFunc(s.Players, func(o *Sketcher) bool { return o.ID != s.Owner && o.ID != s.Drawer && !o.Found })
	if allFound || (s.Phase == "judge" && !s.pending()) {
		s.reveal()
	}
}

func (s *Sketch) pending() bool {
	return slices.ContainsFunc(s.Guesses, func(g Guess) bool { return g.OK == nil })
}

func (s *Sketch) reveal() {
	s.Phase, s.Wait = "reveal", RevealTime
	s.broadcast(map[string]string{"photo": s.photo}, func(*Sketcher) bool { return true })
}

func (s *Sketch) step(dt float64) {
	if s.Wait = max(s.Wait-dt, 0); s.Wait > 0 {
		return
	}
	switch s.Phase {
	case "photo", "reveal":
		s.nextTurn()
	case "draw":
		if s.pending() {
			s.Phase, s.Wait = "judge", JudgeTime
		} else {
			s.reveal()
		}
	case "judge":
		s.reveal()
	}
}

// stateFor : ceux qui devinent ne voient pas le texte des bonnes réponses des autres (avant la révélation).
func (s *Sketch) stateFor(p *Sketcher) []byte {
	gs := s.Guesses
	if p.ID != s.Owner && p.ID != s.Drawer && s.Phase != "reveal" {
		gs = slices.Clone(gs)
		for i, g := range gs {
			if g.By != p.ID && (g.OK == nil || *g.OK) {
				gs[i].Text = ""
			}
		}
	}
	b, _ := json.Marshal(struct {
		*Sketch
		Guesses []Guess `json:"guesses"`
		Turns   int     `json:"turns"`
	}{s, gs, len(s.order)})
	return b
}

// --- salons Dessine (même principe que Ricochet, codes séparés) ---

var (
	sketchesMu sync.Mutex
	sketches   = map[string]*Sketch{}
)

func enterSketch(pin, name string) (*Sketch, *Sketcher, string) {
	sketchesMu.Lock()
	defer sketchesMu.Unlock()
	if pin == "" {
		if len(sketches) >= MaxSketches {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		s := newSketch(newPin(func(p string) bool { return sketches[p] != nil }))
		sketches[s.Pin] = s
		p := s.join(name) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go s.run()
		return s, p, ""
	}
	s := sketches[pin]
	if s == nil {
		return nil, nil, "Code inconnu"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.join(name); p != nil {
		return s, p, ""
	}
	return nil, nil, "Partie pleine (10 joueurs)"
}

func (s *Sketch) run() {
	t := time.NewTicker(SketchTick)
	defer t.Stop()
	for range t.C {
		if s.closeIfEmpty() {
			return
		}
		s.mu.Lock()
		s.step(SketchTick.Seconds())
		for _, p := range s.Players {
			push(p, s.stateFor(p))
		}
		s.mu.Unlock()
	}
}

func (s *Sketch) closeIfEmpty() bool {
	sketchesMu.Lock()
	defer sketchesMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Players) > 0 {
		return false
	}
	delete(sketches, s.Pin)
	return true
}

func serveSketch(w http.ResponseWriter, r *http.Request) {
	session(w, r, PhotoMax+1024, func(name string, q url.Values) (*Seat, string) {
		s, p, why := enterSketch(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { s.mu.Lock(); s.leave(p.ID); s.mu.Unlock() },
			OnMsg: func(b []byte) {
				var m struct {
					Start bool   `json:"start"`
					Photo string `json:"photo"`
					Line  []int  `json:"line"`
					Clear bool   `json:"clear"`
					Guess string `json:"guess"`
					Judge int    `json:"judge"`
					OK    bool   `json:"ok"`
				}
				if json.Unmarshal(b, &m) != nil {
					return
				}
				s.mu.Lock()
				defer s.mu.Unlock()
				switch {
				case m.Start:
					s.start()
				case m.Photo != "":
					s.setPhoto(p, m.Photo)
				case m.Line != nil:
					s.addLine(p, m.Line)
				case m.Clear:
					s.clear(p)
				case m.Guess != "":
					s.guess(p, m.Guess)
				case m.Judge != 0:
					s.judge(p, m.Judge, m.OK)
				}
			},
		}, ""
	})
}
