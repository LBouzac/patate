// Imposteur (mode télé) : chacun reçoit le même mot secret sur son téléphone, sauf l'imposteur qui ne connaît
// que le thème. Chacun son tour dit un mot à voix haute en rapport avec le mot secret, puis tout le monde vote
// sur son téléphone. Si l'imposteur est démasqué, il a encore une chance : retrouver le mot parmi 12.
//
// Points : imposteur jamais démasqué +2 ; démasqué mais trouve le mot +1 ; sinon +1 pour tous les autres.
package main

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"slices"
)

const (
	ImpMin   = 3
	ImpSee   = 25.0 // pour lire son mot
	ImpClue  = 20.0 // par joueur
	ImpVote  = 60.0
	ImpGuess = 25.0
)

type Suspect struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Score int    `json:"score"`
	In    bool   `json:"in"` // joue cette manche (sinon : arrivé en cours, il attend la suivante)
	Seen  bool   `json:"seen"`
	Voted bool   `json:"voted"`
	Votes int    `json:"votes"` // votes reçus (affichés au résultat)
	vote  int
}

type Impostor struct {
	Phase  string     `json:"phase"` // wait, see, clue, vote, guess, result
	Round  int        `json:"round"`
	Cat    string     `json:"cat"` // le thème est public
	P      []*Suspect `json:"p"`
	Order  []int      `json:"order"` // ordre de passage des indices
	Turn   int        `json:"turn"`  // index dans Order
	Left   float64    `json:"left"`
	Imp    int        `json:"imp"`    // révélé au moment voulu seulement (0 sinon)
	Word   string     `json:"word"`   // idem
	Caught bool       `json:"caught"` // l'imposteur a été démasqué
	Guess  string     `json:"guess"`  // mot proposé par l'imposteur démasqué
	Win    string     `json:"win"`    // "imp", "civ", ou "" (manche annulée)
	Note   string     `json:"note"`
	imp    int
	word   string
	words  []string // les 12 mots du thème, mélangés : proposés à l'imposteur démasqué
	prev   int      // imposteur précédent : on évite qu'il le soit deux fois de suite
}

func newImpostor(pads []*Pad) *Impostor {
	g := &Impostor{Phase: "wait"}
	for _, p := range pads {
		g.P = append(g.P, &Suspect{ID: p.ID, Name: p.Name, Color: p.Color})
	}
	g.newRound()
	return g
}

func (g *Impostor) get(id int) *Suspect {
	for _, s := range g.P {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (g *Impostor) in() []*Suspect {
	var r []*Suspect
	for _, s := range g.P {
		if s.In {
			r = append(r, s)
		}
	}
	return r
}

func (g *Impostor) newRound() {
	if len(g.P) < ImpMin {
		g.Phase = "wait"
		return
	}
	g.Round++
	c := impBank[rand.IntN(len(impBank))]
	g.Cat, g.word = c.Cat, c.Words[rand.IntN(len(c.Words))]
	g.words = slices.Clone(c.Words[:])
	rand.Shuffle(len(g.words), func(i, j int) { g.words[i], g.words[j] = g.words[j], g.words[i] })
	pool := []int{}
	for _, s := range g.P {
		*s = Suspect{ID: s.ID, Name: s.Name, Color: s.Color, Score: s.Score, In: true}
		if s.ID != g.prev || len(g.P) == 1 {
			pool = append(pool, s.ID)
		}
	}
	g.imp = pool[rand.IntN(len(pool))]
	g.prev = g.imp
	g.Order = nil
	for _, i := range rand.Perm(len(g.P)) {
		g.Order = append(g.Order, g.P[i].ID)
	}
	if g.Order[0] == g.imp { // l'imposteur ne commence pas : il a besoin d'entendre les autres
		g.Order = append(g.Order[1:], g.imp)
	}
	g.Phase, g.Left, g.Turn = "see", ImpSee, 0
	g.Imp, g.Word, g.Caught, g.Guess, g.Win, g.Note = 0, "", false, "", "", ""
}

func (g *Impostor) Add(p *Pad) {
	g.P = append(g.P, &Suspect{ID: p.ID, Name: p.Name, Color: p.Color})
	if g.Phase == "wait" {
		g.newRound()
	}
}

func (g *Impostor) Remove(id int) {
	s := g.get(id)
	if s == nil {
		return
	}
	g.P = slices.DeleteFunc(g.P, func(o *Suspect) bool { return o.ID == id })
	if !s.In || g.Phase == "result" || g.Phase == "wait" {
		if len(g.P) < ImpMin {
			g.Phase = "wait"
		}
		return
	}
	switch {
	case id == g.imp:
		g.end("", "L'imposteur a quitté la partie : manche annulée")
		return
	case len(g.in()) < ImpMin:
		g.end("", "Plus assez de joueurs : manche annulée")
		return
	}
	if i := slices.Index(g.Order, id); i >= 0 {
		g.Order = slices.Delete(g.Order, i, i+1)
		if g.Phase == "clue" && i < g.Turn {
			g.Turn--
		} else if g.Phase == "clue" && i == g.Turn {
			g.Left = ImpClue
			if g.Turn >= len(g.Order) {
				g.startVote()
			}
		}
	}
	for _, o := range g.P { // ses votes et les votes contre lui ne comptent plus
		if o.vote == id {
			o.vote, o.Voted = 0, false
		}
	}
	g.check()
}

func (g *Impostor) startVote() { g.Phase, g.Left = "vote", ImpVote }

// check avance quand tout le monde a fait ce qu'il fallait.
func (g *Impostor) check() {
	in := g.in()
	switch g.Phase {
	case "see":
		if !slices.ContainsFunc(in, func(s *Suspect) bool { return !s.Seen }) {
			g.Phase, g.Left = "clue", ImpClue
		}
	case "vote":
		if !slices.ContainsFunc(in, func(s *Suspect) bool { return !s.Voted }) {
			g.count()
		}
	}
}

// count dépouille les votes : le plus désigné est démasqué s'il est seul en tête et que c'est l'imposteur.
func (g *Impostor) count() {
	for _, s := range g.P {
		s.Votes = 0
	}
	for _, s := range g.P {
		if t := g.get(s.vote); s.In && s.Voted && t != nil {
			t.Votes++
		}
	}
	top, tie := 0, false
	var who *Suspect
	for _, s := range g.P {
		switch {
		case s.Votes > top:
			top, who, tie = s.Votes, s, false
		case s.Votes == top && top > 0:
			tie = true
		}
	}
	g.Imp = g.imp
	switch {
	case who == nil || tie:
		g.end("imp", "Égalité des votes : l'imposteur s'en sort")
	case who.ID != g.imp:
		g.end("imp", who.Name+" était innocent")
	default:
		g.Caught = true
		g.Phase, g.Left = "guess", ImpGuess
	}
}

func (g *Impostor) end(win, note string) {
	g.Phase, g.Left, g.Win, g.Note = "result", 0, win, note
	g.Imp, g.Word = g.imp, g.word
	switch {
	case win == "imp" && !g.Caught:
		if s := g.get(g.imp); s != nil {
			s.Score += 2
		}
	case win == "imp":
		if s := g.get(g.imp); s != nil {
			s.Score++
		}
	case win == "civ":
		for _, s := range g.in() {
			if s.ID != g.imp {
				s.Score++
			}
		}
	}
}

func (g *Impostor) Input(p *Pad, b []byte) {
	var m struct {
		Seen  bool   `json:"seen"`
		Done  bool   `json:"done"`
		Vote  int    `json:"vote"`
		Guess string `json:"guess"`
		Next  bool   `json:"next"`
	}
	if json.Unmarshal(b, &m) != nil {
		return
	}
	if m.Next && p.Host && (g.Phase == "result" || g.Phase == "wait") {
		g.newRound()
		return
	}
	s := g.get(p.ID)
	if s == nil || !s.In {
		return
	}
	switch {
	case m.Seen && g.Phase == "see":
		s.Seen = true
		g.check()
	case m.Done && g.Phase == "clue" && g.Order[g.Turn] == s.ID:
		g.nextTurn()
	case m.Vote != 0 && g.Phase == "vote" && m.Vote != s.ID:
		if t := g.get(m.Vote); t != nil && t.In {
			s.vote, s.Voted = m.Vote, true // on peut changer d'avis tant que tout le monde n'a pas voté
			g.check()
		}
	case m.Guess != "" && g.Phase == "guess" && s.ID == g.imp && slices.Contains(g.words, m.Guess):
		g.Guess = m.Guess
		if m.Guess == g.word {
			g.end("imp", "Démasqué, mais il a trouvé le mot !")
		} else {
			g.end("civ", "Démasqué, et il ne trouve pas le mot")
		}
	}
}

func (g *Impostor) nextTurn() {
	g.Turn++
	g.Left = ImpClue
	if g.Turn >= len(g.Order) {
		g.startVote()
	}
}

func (g *Impostor) Step(dt float64) {
	if g.Phase == "wait" || g.Phase == "result" {
		return
	}
	if g.Left -= dt; g.Left > 0 {
		return
	}
	switch g.Phase {
	case "see":
		g.Phase, g.Left = "clue", ImpClue
	case "clue":
		g.nextTurn()
	case "vote":
		g.count()
	case "guess":
		g.end("civ", "Démasqué, et il n'a pas trouvé le mot à temps")
	}
}

func (g *Impostor) Screen() any {
	s := *g
	s.Left = math.Round(max(0, g.Left)*10) / 10
	return s
}

type impView struct {
	Phase  string   `json:"phase"`
	In     bool     `json:"in"`
	Cat    string   `json:"cat"`
	Word   string   `json:"word"` // vide pour l'imposteur
	Imp    bool     `json:"imp"`  // tu es l'imposteur
	Seen   bool     `json:"seen"`
	MyTurn bool     `json:"myTurn"`
	Turn   string   `json:"turn"` // qui parle
	Vote   int      `json:"vote"`
	Opts   []int    `json:"opts"`  // pour qui on peut voter
	Words  []string `json:"words"` // dernière chance de l'imposteur
	Left   int      `json:"left"`
	Score  int      `json:"score"`
	Win    string   `json:"win"`
	Note   string   `json:"note"`
	Secret string   `json:"secret"` // le mot, révélé à tous au résultat
	ImpID  int      `json:"impId"`
	Count  int      `json:"count"` // joueurs dans le jeu
}

func (g *Impostor) View(p *Pad) any {
	v := impView{Phase: g.Phase, Cat: g.Cat, Left: int(math.Ceil(max(0, g.Left))), Win: g.Win, Note: g.Note, Count: len(g.P)}
	s := g.get(p.ID)
	if s == nil {
		return v
	}
	v.In, v.Score, v.Seen = s.In, s.Score, s.Seen
	if g.Phase == "result" {
		v.Secret, v.ImpID = g.word, g.imp
	}
	if !s.In || g.Phase == "wait" {
		return v
	}
	v.Imp = s.ID == g.imp
	if !v.Imp {
		v.Word = g.word
	}
	if g.Phase == "clue" && g.Turn < len(g.Order) {
		v.MyTurn = g.Order[g.Turn] == s.ID
		if t := g.get(g.Order[g.Turn]); t != nil {
			v.Turn = t.Name
		}
	}
	if g.Phase == "vote" {
		v.Vote = s.vote
		for _, o := range g.in() {
			if o.ID != s.ID {
				v.Opts = append(v.Opts, o.ID)
			}
		}
	}
	if g.Phase == "guess" && v.Imp {
		v.Words = g.words
	}
	return v
}
