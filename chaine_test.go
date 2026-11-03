package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChainEachChainVisitsEveryone(t *testing.T) {
	for n := 3; n <= 10; n++ {
		g := newChain(sumoPads(make([]string, n)...))
		for c := 0; c < n; c++ {
			seen := map[int]bool{}
			for k := 0; k < g.Steps; k++ {
				for j := 0; j < n; j++ {
					if g.chainOf(j, k) == c {
						if seen[j] {
							t.Fatalf("%d joueurs : la chaîne %d repasse par le joueur %d", n, c, j)
						}
						seen[j] = true
					}
				}
			}
		}
	}
}

func TestChainFullGame(t *testing.T) {
	ps := sumoPads("Ana", "Bob", "Cy")
	g := newChain(ps)
	if g.Phase != "write" || g.Steps != 3 {
		t.Fatalf("3 joueurs : écrire, dessiner, deviner ; obtenu %s, %d étapes", g.Phase, g.Steps)
	}
	g.Input(ps[0], []byte(`{"text":"  un   chat\nqui skie  "}`))
	g.Input(ps[1], []byte(`{"text":""}`)) // rien écrit : son idée
	if g.links[0][0].Text != "un chat qui skie" || g.links[1][0].Text != g.P[1].idea {
		t.Fatalf("phrases : %q / %q", g.links[0][0].Text, g.links[1][0].Text)
	}
	g.Step(ChainWrite)
	if !g.Late || g.Phase != "write" {
		t.Fatal("chrono écoulé : on laisse aux téléphones le temps d'envoyer")
	}
	if v := g.View(ps[2]).(tvBig).V.(chainView); !v.Due {
		t.Fatal("la manette doit envoyer ce qu'elle a")
	}
	g.Step(ChainLate)
	if g.Phase != "draw" || g.links[2][0].Text != g.P[2].idea {
		t.Fatalf("Cy n'a rien envoyé : son idée sert de départ ; phase %s", g.Phase)
	}
	// Ana dessine la phrase de Cy (chaîne 2)
	v := g.View(ps[0]).(tvBig)
	if v.V.(chainView).Prompt != g.links[2][0].Text || v.Big != nil {
		t.Fatalf("Ana doit dessiner « %s », obtenu %+v", g.links[2][0].Text, v.V)
	}
	for _, p := range ps {
		g.Input(p, []byte(`{"draw":[[0,2,-50,10,2000,900],[3,9,1,1,2,2],[1,1,5]]}`))
	}
	if g.Phase != "guess" {
		t.Fatal("tout le monde a dessiné : on devine")
	}
	var d [][]int
	json.Unmarshal(g.links[0][1].Draw, &d)
	if len(d) != 2 || d[0][2] != 0 || d[0][4] != DrawW || d[0][5] != DrawH || d[1][1] != 3 {
		t.Fatalf("dessin remis au propre (bornes, traits invalides retirés) : %v", d)
	}
	if v := g.View(ps[0]).(tvBig); v.Big == nil {
		t.Fatal("pour deviner, la manette reçoit le dessin")
	}
	g.Input(ps[0], []byte(`{"text":"un chien"}`))
	g.Remove(2) // Bob part : son maillon est rempli
	g.Step(ChainGuess)
	g.Step(ChainLate + 0.1)
	if g.Phase != "reveal" || g.RC != 0 || g.RS != 1 {
		t.Fatalf("révélation : %s", g.Phase)
	}
	s := g.Screen().(Chain)
	if s.Owner != "Ana" || len(s.Show) != 1 || s.Show[0].Text != "un chat qui skie" {
		t.Fatalf("la télé montre le début de la chaîne d'Ana : %+v", s.Show)
	}
	g.Input(ps[1], []byte(`{"next":true}`)) // pas le chef
	if g.RS != 1 {
		t.Fatal("seul le chef fait avancer")
	}
	for i := 0; i < 9; i++ {
		g.Input(ps[0], []byte(`{"next":true}`))
	}
	if g.Phase != "done" {
		t.Fatalf("3 chaînes de 3 maillons : terminé, obtenu %s (%d,%d)", g.Phase, g.RC, g.RS)
	}
	skips := 0
	for _, c := range g.links {
		for k, l := range c {
			if l.Kind != kindOf(k) || l.By == "" {
				t.Fatalf("maillon %d mal formé : %+v", k, l)
			}
			if l.Skip {
				skips++
			}
		}
	}
	if skips != 2 { // Bob (parti) et Cy (rien deviné)
		t.Fatalf("2 maillons vides attendus, obtenu %d", skips)
	}
}

func TestChainRejectsHugeDrawing(t *testing.T) {
	big := "[" + strings.Repeat("[0,1,1,1,2,2],", ChainStrokes) + "[0,1,1,1,2,2]]"
	if _, ok := cleanDrawing([]byte(big)); ok {
		t.Fatal("trop de traits : refusé")
	}
	if _, ok := cleanDrawing([]byte(`{"a":1}`)); ok {
		t.Fatal("pas un dessin : refusé")
	}
	if d, ok := cleanDrawing([]byte(`[]`)); !ok || string(d) != "[]" {
		t.Fatal("dessin vide accepté")
	}
}

func TestChainRevealAutoAdvances(t *testing.T) {
	ps := sumoPads("A", "B", "C")
	g := newChain(ps)
	for g.Phase != "reveal" {
		g.Step(1)
	}
	g.Step(ChainShow + 0.1)
	if g.RS != 2 {
		t.Fatal("le chef ne fait rien : on avance tout seul")
	}
}
