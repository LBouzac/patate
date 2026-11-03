package main

import (
	"fmt"
	"testing"
)

func TestImpBank(t *testing.T) {
	cats := map[string]bool{}
	for _, c := range impBank {
		if cats[c.Cat] {
			t.Errorf("thème en double : %s", c.Cat)
		}
		cats[c.Cat] = true
		words := map[string]bool{}
		for _, w := range c.Words {
			if w == "" || words[w] {
				t.Errorf("%s : mot vide ou en double %q", c.Cat, w)
			}
			words[w] = true
		}
	}
}

// impGame : une partie à 4 dont on connaît l'imposteur.
func impGame(t *testing.T) (*Impostor, []*Pad, *Pad) {
	ps := sumoPads("Ana", "Bob", "Cy", "Dan")
	g := newImpostor(ps)
	var imp *Pad
	for _, p := range ps {
		v := g.View(p).(impView)
		if v.Imp {
			imp = p
			if v.Word != "" {
				t.Fatal("l'imposteur ne voit pas le mot")
			}
		} else if v.Word == "" || v.Cat == "" {
			t.Fatal("les autres voient le mot et le thème")
		}
	}
	if imp == nil || g.Order[0] == imp.ID {
		t.Fatal("un imposteur, qui ne commence pas")
	}
	if s := g.Screen().(Impostor); s.Word != "" || s.Imp != 0 {
		t.Fatal("la télé ne dévoile rien")
	}
	for _, p := range ps {
		g.Input(p, []byte(`{"seen":true}`))
	}
	if g.Phase != "clue" {
		t.Fatal("tout le monde a vu son mot : indices")
	}
	for range ps {
		cur := g.Order[g.Turn]
		for _, p := range ps {
			if v := g.View(p).(impView); v.MyTurn != (p.ID == cur) {
				t.Fatal("seul celui qui parle a la main")
			}
		}
		g.Input(ps[cur-1], []byte(`{"done":true}`))
	}
	if g.Phase != "vote" {
		t.Fatalf("après le tour : vote, obtenu %s", g.Phase)
	}
	return g, ps, imp
}

func vote(g *Impostor, from *Pad, to int) { g.Input(from, []byte(fmt.Sprintf(`{"vote":%d}`, to))) }

func TestImpCaughtAndFails(t *testing.T) {
	g, ps, imp := impGame(t)
	for _, p := range ps {
		to := imp.ID
		if p == imp {
			to = ps[(imp.ID)%4].ID
		}
		vote(g, p, p.ID) // voter pour soi : refusé
		vote(g, p, to)
	}
	if g.Phase != "guess" || !g.Caught {
		t.Fatalf("démasqué : dernière chance, obtenu %s", g.Phase)
	}
	v := g.View(imp).(impView)
	if len(v.Words) != 12 {
		t.Fatal("12 mots proposés à l'imposteur")
	}
	wrong := v.Words[0]
	if wrong == g.word {
		wrong = v.Words[1]
	}
	g.Input(imp, []byte(fmt.Sprintf(`{"guess":%q}`, wrong)))
	if g.Phase != "result" || g.Win != "civ" || g.Word == "" {
		t.Fatalf("mauvais mot : les autres gagnent, %s %s", g.Phase, g.Win)
	}
	for _, p := range ps {
		want := 1
		if p == imp {
			want = 0
		}
		if s := g.get(p.ID); s.Score != want {
			t.Fatalf("%s : %d points, %d attendus", s.Name, s.Score, want)
		}
	}
	g.Input(ps[1], []byte(`{"next":true}`))
	if g.Phase != "result" {
		t.Fatal("seul le chef lance la manche suivante")
	}
	g.Input(ps[0], []byte(`{"next":true}`))
	if g.Phase != "see" || g.Round != 2 || g.imp == imp.ID {
		t.Fatal("manche suivante, avec un autre imposteur")
	}
}

func TestImpCaughtButGuesses(t *testing.T) {
	g, ps, imp := impGame(t)
	for _, p := range ps {
		if p != imp {
			vote(g, p, imp.ID)
		}
	}
	vote(g, imp, ps[imp.ID%4].ID)
	g.Input(imp, []byte(fmt.Sprintf(`{"guess":%q}`, g.word)))
	if g.Win != "imp" || g.get(imp.ID).Score != 1 {
		t.Fatal("démasqué mais trouve le mot : +1 pour l'imposteur")
	}
}

func TestImpEscapesOnTie(t *testing.T) {
	g, ps, imp := impGame(t)
	others := []*Pad{}
	for _, p := range ps {
		if p != imp {
			others = append(others, p)
		}
	}
	// 2 votes contre un innocent, 2 contre l'imposteur : égalité
	vote(g, others[0], imp.ID)
	vote(g, others[1], imp.ID)
	vote(g, others[2], others[0].ID)
	vote(g, imp, others[0].ID)
	if g.Phase != "result" || g.Win != "imp" || g.get(imp.ID).Score != 2 {
		t.Fatalf("égalité : l'imposteur s'en sort avec 2 points (%s %s)", g.Phase, g.Win)
	}
}

func TestImpLeaves(t *testing.T) {
	g, ps, imp := impGame(t)
	g.Remove(imp.ID)
	if g.Phase != "result" || g.Win != "" {
		t.Fatal("l'imposteur part : manche annulée")
	}
	g.Input(ps[0], []byte(`{"next":true}`))
	if imp == ps[0] {
		g.Input(ps[1], []byte(`{"next":true}`))
	}
	if g.Phase != "see" {
		t.Fatal("manche suivante à 3")
	}
	for len(g.P) > 2 {
		g.Remove(g.P[0].ID)
	}
	if g.Phase != "result" && g.Phase != "wait" {
		t.Fatalf("moins de 3 : manche annulée, obtenu %s", g.Phase)
	}
}

func TestImpTimers(t *testing.T) {
	g, _, _ := impGame(t)
	g.Step(ImpVote + 0.1)
	if g.Phase != "result" || g.Win != "imp" {
		t.Fatalf("personne n'a voté : l'imposteur s'en sort, obtenu %s %s", g.Phase, g.Win)
	}
}
