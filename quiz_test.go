package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestQuizBank(t *testing.T) {
	seen := map[string]bool{}
	for _, q := range quizBank {
		if seen[q.Q] {
			t.Errorf("question en double : %s", q.Q)
		}
		seen[q.Q] = true
		ans := map[string]bool{}
		for _, a := range q.A {
			if strings.TrimSpace(a) == "" || ans[a] {
				t.Errorf("%s : réponses vides ou en double %v", q.Q, q.A)
			}
			ans[a] = true
		}
	}
	if len(quizBank) < 100 {
		t.Errorf("au moins 100 questions, il y en a %d", len(quizBank))
	}
}

func TestQuizFlow(t *testing.T) {
	ps := sumoPads("Ana", "Bob")
	g := newQuiz(ps)
	if g.Phase != "intro" {
		t.Fatal("intro d'abord")
	}
	for g.Phase == "intro" {
		g.Step(0.1)
	}
	if g.Phase != "q" || g.N != 1 || g.Good != -1 {
		t.Fatalf("question 1, bonne réponse cachée : %s %d %d", g.Phase, g.N, g.Good)
	}
	if s := g.Screen().(Quiz); s.Good != -1 {
		t.Fatal("la télé ne montre pas la bonne réponse avant la fin")
	}
	good := g.good
	g.Input(ps[0], []byte(fmt.Sprintf(`{"pick":%d}`, good)))
	g.Input(ps[0], []byte(fmt.Sprintf(`{"pick":%d}`, (good+1)%4))) // on ne change pas d'avis
	if s := g.Screen().(Quiz); s.P[0].Pick != -1 || !s.P[0].Done {
		t.Fatal("la télé montre qui a répondu, pas quoi")
	}
	g.Input(ps[1], []byte(fmt.Sprintf(`{"pick":%d}`, (good+1)%4)))
	if g.Phase != "reveal" {
		t.Fatal("tout le monde a répondu : révélation tout de suite")
	}
	a, b := g.get(1), g.get(2)
	if a.Score != 1000 || b.Score != 0 || a.Pick != good || g.Counts[good] != 1 {
		t.Fatalf("réponse immédiate juste = 1000 points : Ana %d, Bob %d", a.Score, b.Score)
	}
	for g.Phase != "end" {
		if g.Phase == "q" {
			g.Step(QuizTime / 2) // Ana répond à mi-temps, juste
			g.Input(ps[0], []byte(fmt.Sprintf(`{"pick":%d}`, g.good)))
		}
		g.Step(0.1)
	}
	if g.N != QuizCount || a.Score != 1000+9*750 || a.Streak != QuizCount {
		t.Fatalf("10 questions, 750 points à mi-temps : %d questions, %d points, série %d", g.N, a.Score, a.Streak)
	}
	if v := g.View(ps[1]).(quizView); v.Rank != 2 || v.Score != 0 {
		t.Fatalf("Bob est 2e avec 0 point : %+v", v)
	}
}

func TestQuizLateAndLeave(t *testing.T) {
	ps := sumoPads("Ana", "Bob", "Cy")
	g := newQuiz(ps[:2])
	g.Step(QuizIntro)
	g.Input(ps[0], []byte(`{"pick":0}`))
	g.Remove(2)
	if g.Phase != "reveal" {
		t.Fatal("Bob part, Ana a répondu : on révèle")
	}
	if v := g.View(ps[2]).(quizView); !v.Late {
		t.Fatal("inconnu du quiz : arrivé en retard")
	}
	g.Add(ps[2])
	g.Step(QuizReveal)
	g.Input(ps[2], []byte(`{"pick":1}`))
	if c := g.get(3); !c.Done {
		t.Fatal("le nouveau joue dès la question suivante")
	}
}
