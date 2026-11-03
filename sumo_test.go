package main

import (
	"testing"
)

func sumoPads(names ...string) []*Pad {
	var ps []*Pad
	for i, n := range names {
		ps = append(ps, &Pad{ID: i + 1, Name: n, Color: colors[i], Host: i == 0})
	}
	return ps
}

// runSumo fait tourner le jeu comme le salon : une image, puis ce qu'on envoie à la télé.
func runSumo(g *Sumo, secs float64) {
	for t := 0.0; t < secs; t += TVTick.Seconds() {
		g.Step(TVTick.Seconds())
		g.Screen()
	}
}

func TestSumoPushOut(t *testing.T) {
	ps := sumoPads("Ana", "Bob")
	g := newSumo(ps)
	if g.Phase != "count" {
		t.Fatalf("2 lutteurs : la manche démarre, phase %s", g.Phase)
	}
	runSumo(g, 3.1)
	if g.Phase != "play" {
		t.Fatalf("après le compte à rebours : %s", g.Phase)
	}
	a, b := g.get(1), g.get(2)
	// Bob est poussé vers le bord, Ana reste au centre
	b.body.X, b.body.Y = SumoW/2+g.Ring-SumoR, SumoH/2
	b.body.vx = 600
	a.body.X, a.body.Y = SumoW/2, SumoH/2
	runSumo(g, 1)
	if b.Alive || g.Phase != "round" || g.Winner != "Ana" || a.Score != 1 {
		t.Fatalf("Bob sort, Ana marque : alive %v phase %s gagnant %q score %d", b.Alive, g.Phase, g.Winner, a.Score)
	}
	v := g.View(ps[1]).(sumoView)
	if v.Alive || v.Hits == 0 {
		t.Fatalf("la manette de Bob sait qu'il est tombé (et vibre) : %+v", v)
	}
}

func TestSumoDashAndCollision(t *testing.T) {
	ps := sumoPads("Ana", "Bob")
	g := newSumo(ps)
	runSumo(g, 3.1)
	a, b := g.get(1), g.get(2)
	a.body.X, a.body.Y, b.body.X, b.body.Y = 700, 450, 790, 450
	a.body.vx, a.body.vy, b.body.vx, b.body.vy = 0, 0, 0, 0
	g.Input(ps[0], []byte(`{"x":1,"y":0}`))
	g.Input(ps[0], []byte(`{"dash":true}`))
	if a.cool == 0 || a.body.vx < SumoDash {
		t.Fatalf("dash : élan et recharge attendus, vx %.0f", a.body.vx)
	}
	g.Input(ps[0], []byte(`{"dash":true}`))
	if a.body.vx > SumoDash*1.5 {
		t.Fatal("pas de second dash pendant la recharge")
	}
	runSumo(g, 0.2)
	if b.body.vx <= 0 || b.hits == 0 {
		t.Fatalf("Ana percute Bob : il doit reculer (vx %.0f, chocs %d)", b.body.vx, b.hits)
	}
	g.Input(ps[0], []byte(`{"x":"NaN"}`))
	g.Input(ps[0], []byte(`{"x":50,"y":50}`))
	if l := a.in[0]*a.in[0] + a.in[1]*a.in[1]; l > 1.0001 {
		t.Fatalf("joystick borné à 1, obtenu %v", a.in)
	}
}

func TestSumoBotsFinishRounds(t *testing.T) {
	ps := sumoPads("Ana")
	g := newSumo(ps)
	if g.Phase != "wait" {
		t.Fatal("seul : on attend un adversaire")
	}
	g.Input(ps[0], []byte(`{"bot":1}`))
	g.Input(ps[0], []byte(`{"bot":1}`))
	g.Input(ps[0], []byte(`{"bot":1}`))
	if len(g.F) != 4 || g.Phase != "count" {
		t.Fatalf("3 bots ajoutés par le chef, la manche démarre : %d lutteurs, %s", len(g.F), g.Phase)
	}
	// Ana ne bouge pas : les bots doivent finir les manches tout seuls (l'arène rétrécit)
	runSumo(g, 400)
	if g.Round < 5 {
		t.Fatalf("les manches doivent se terminer : %d manches en 400 s", g.Round)
	}
	g.Remove(1)
	if g.F == nil || len(g.F) != 0 {
		t.Fatal("plus d'humain : les bots partent")
	}
}

func TestSumoEveryoneInFirstRound(t *testing.T) {
	ps := sumoPads("Ana", "Bob", "Cy", "Dan")
	g := newSumo(ps[:3])
	g.Add(ps[3]) // arrivé pendant le compte à rebours : il entre dans la manche
	for _, f := range g.F {
		if !f.Alive {
			t.Fatalf("%s doit être dans la première manche", f.Name)
		}
	}
	if g.Round != 1 {
		t.Fatalf("toujours la manche 1, obtenu %d", g.Round)
	}
}

func TestSumoLateJoinerWaits(t *testing.T) {
	ps := sumoPads("Ana", "Bob", "Cy")
	g := newSumo(ps[:2])
	runSumo(g, 3.5)
	g.Add(ps[2])
	if c := g.get(3); c == nil || c.Alive {
		t.Fatal("arrivé en pleine manche : il attend la suivante")
	}
	if v := g.View(ps[2]).(sumoView); v.In {
		t.Fatal("sa manette indique qu'il attend")
	}
}
