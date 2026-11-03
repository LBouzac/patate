// Quiz rapide (mode télé) : la question et les 4 réponses s'affichent sur la télé, chacun répond sur son
// téléphone (A, B, C ou D). Une bonne réponse rapporte de 500 à 1000 points selon la rapidité. 10 questions.
package main

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"slices"
)

const (
	QuizCount  = 10
	QuizTime   = 15.0 // secondes pour répondre
	QuizReveal = 5.0  // la bonne réponse reste affichée
	QuizIntro  = 3.0
)

type Quizzer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Score  int    `json:"score"`
	Gain   int    `json:"gain"` // points de la dernière question
	Done   bool   `json:"done"` // a répondu à la question en cours
	Pick   int    `json:"pick"` // réponse choisie (-1 : aucune), montrée sur la télé seulement à la révélation
	Streak int    `json:"streak"`
}

type Quiz struct {
	Phase  string     `json:"phase"` // intro, q, reveal, end
	N      int        `json:"n"`     // numéro de la question (1 à QuizCount)
	Total  int        `json:"total"`
	Q      string     `json:"q"`
	A      [4]string  `json:"a"`
	Good   int        `json:"good"` // bonne réponse (révélée à la fin de la question, -1 avant)
	Counts [4]int     `json:"counts"`
	Left   float64    `json:"left"`
	Time   float64    `json:"time"`
	P      []*Quizzer `json:"p"`
	qs     []quizQ
	good   int
}

func newQuiz(pads []*Pad) *Quiz {
	g := &Quiz{Phase: "intro", Left: QuizIntro, Total: QuizCount, Time: QuizTime, Good: -1}
	order := rand.Perm(len(quizBank))[:QuizCount]
	for _, i := range order {
		g.qs = append(g.qs, quizBank[i])
	}
	for _, p := range pads {
		g.Add(p)
	}
	return g
}

func (g *Quiz) get(id int) *Quizzer {
	for _, q := range g.P {
		if q.ID == id {
			return q
		}
	}
	return nil
}

func (g *Quiz) Add(p *Pad) {
	g.P = append(g.P, &Quizzer{ID: p.ID, Name: p.Name, Color: p.Color, Pick: -1})
}

func (g *Quiz) Remove(id int) {
	g.P = slices.DeleteFunc(g.P, func(q *Quizzer) bool { return q.ID == id })
	g.checkAll()
}

// ask passe à la question suivante, réponses mélangées.
func (g *Quiz) ask() {
	q := g.qs[g.N]
	g.N++
	g.Q = q.Q
	perm := rand.Perm(4)
	for i, j := range perm {
		g.A[i] = q.A[j]
		if j == 0 { // la première réponse de la banque est la bonne
			g.good = i
		}
	}
	g.Phase, g.Left, g.Good, g.Counts = "q", QuizTime, -1, [4]int{}
	for _, p := range g.P {
		p.Done, p.Pick, p.Gain = false, -1, 0
	}
}

func (g *Quiz) reveal() {
	g.Phase, g.Left, g.Good = "reveal", QuizReveal, g.good
	for _, p := range g.P {
		if p.Pick >= 0 {
			g.Counts[p.Pick]++
		}
		if p.Pick == g.good {
			p.Streak++
		} else {
			p.Streak = 0
		}
		p.Score += p.Gain
	}
}

func (g *Quiz) checkAll() {
	if g.Phase != "q" || len(g.P) == 0 {
		return
	}
	for _, p := range g.P {
		if !p.Done {
			return
		}
	}
	g.reveal() // tout le monde a répondu : pas la peine d'attendre
}

func (g *Quiz) Input(p *Pad, b []byte) {
	var m struct {
		Pick *int `json:"pick"`
	}
	if json.Unmarshal(b, &m) != nil || m.Pick == nil || *m.Pick < 0 || *m.Pick > 3 || g.Phase != "q" {
		return
	}
	q := g.get(p.ID)
	if q == nil || q.Done {
		return
	}
	q.Done, q.Pick = true, *m.Pick
	if q.Pick == g.good {
		q.Gain = 500 + int(math.Round(500*g.Left/QuizTime/10))*10 // 500 points + jusqu'à 500 pour la rapidité
	}
	g.checkAll()
}

func (g *Quiz) Step(dt float64) {
	if g.Phase == "end" {
		return
	}
	if g.Left -= dt; g.Left > 0 {
		return
	}
	switch {
	case g.Phase == "q":
		g.reveal()
	case g.Phase == "intro" || g.N < len(g.qs):
		g.ask()
	default:
		g.Phase, g.Left = "end", 0
	}
}

func (g *Quiz) Screen() any {
	s := *g
	s.Left = math.Round(max(0, g.Left)*10) / 10
	if g.Phase == "q" || g.Phase == "intro" { // les choix restent secrets jusqu'à la révélation
		s.P = make([]*Quizzer, len(g.P))
		for i, p := range g.P {
			c := *p
			c.Pick, c.Gain = -1, 0
			s.P[i] = &c
		}
	}
	return s
}

type quizView struct {
	Phase  string    `json:"phase"`
	N      int       `json:"n"`
	Total  int       `json:"total"`
	A      [4]string `json:"a"`
	Left   int       `json:"left"`
	Pick   int       `json:"pick"`
	Done   bool      `json:"done"`
	Good   int       `json:"good"`
	Gain   int       `json:"gain"`
	Score  int       `json:"score"`
	Rank   int       `json:"rank"`
	Of     int       `json:"of"`
	Streak int       `json:"streak"`
	Late   bool      `json:"late"` // arrivé en cours de question
}

func (g *Quiz) View(p *Pad) any {
	v := quizView{Phase: g.Phase, N: g.N, Total: g.Total, Left: int(math.Ceil(max(0, g.Left))), Good: g.Good, Pick: -1, Of: len(g.P)}
	if g.Phase == "q" || g.Phase == "reveal" {
		v.A = g.A
	}
	q := g.get(p.ID)
	if q == nil {
		v.Late = true
		return v
	}
	v.Pick, v.Done, v.Gain, v.Score, v.Streak = q.Pick, q.Done, q.Gain, q.Score, q.Streak
	if g.Phase == "q" {
		v.Gain = 0 // on ne sait pas encore si c'est juste
	}
	v.Rank = 1
	for _, o := range g.P {
		if o.Score > q.Score {
			v.Rank++
		}
	}
	return v
}
