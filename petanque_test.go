package main

import (
	"math"
	"testing"
)

// pitch2 : deux joueurs, première mène lancée, cochonnet posé à un endroit connu.
func pitch2(t *testing.T) (*Pitch, *Bouliste, *Bouliste) {
	g := newPitch("0000")
	a, b := g.join("A"), g.join("B")
	for g.Phase != "aim" {
		g.step(0.1)
	}
	g.Jack.X, g.Jack.Y = 500, 250
	return g, a, b
}

func settlePet(g *Pitch) {
	for i := 0; i < 3000 && g.Phase == "roll"; i++ {
		g.step(PetTick.Seconds())
	}
}

// place pose directement une boule du joueur (sans la lancer) et passe au tour suivant.
func place(g *Pitch, p *Bouliste, x, y float64) {
	g.Boules = append(g.Boules, &Boule{Body: Body{X: x, Y: y, r: BouleR, m: 1, drag: BouleDrag}, Owner: p.ID})
	p.Left--
	g.next()
}

func TestPetanqueThrow(t *testing.T) {
	g, a, b := pitch2(t)
	first := g.get(g.Turn)
	other := a
	if first == a {
		other = b
	}
	g.throw(other, 0, 0.5) // pas son tour
	if len(g.Boules) != 0 {
		t.Fatal("seul celui dont c'est le tour lance")
	}
	g.throw(first, 0, 0.6)
	if len(g.Boules) != 1 || first.Left != BoulesEach-1 || g.Phase != "roll" {
		t.Fatalf("une boule lancée attendue, obtenu %d boules, %d restantes, %s", len(g.Boules), first.Left, g.Phase)
	}
	settlePet(g)
	bl := g.Boules[0]
	if bl.moving() || bl.X <= CircleX+100 {
		t.Fatalf("la boule doit avoir roulé puis s'être arrêtée : %+v", bl.Body)
	}
	if g.Turn != other.ID {
		t.Fatal("celui qui n'a pas le point doit jouer ensuite")
	}
}

func TestPetanqueTurnOrderAndScore(t *testing.T) {
	g, a, b := pitch2(t)
	g.Turn = a.ID
	place(g, a, 510, 250) // A à 10 px : il a le point
	if g.Point != a.ID || g.Turn != b.ID {
		t.Fatalf("A a le point, B doit jouer : point %d tour %d", g.Point, g.Turn)
	}
	place(g, b, 560, 250) // B à 60 px : toujours pas le point, il rejoue
	if g.Turn != b.ID {
		t.Fatal("B n'a toujours pas le point : il rejoue")
	}
	place(g, b, 540, 250)
	place(g, b, 600, 250) // B n'a plus de boules : A finit
	if g.Turn != a.ID {
		t.Fatalf("B n'a plus de boules, A doit jouer, tour %d", g.Turn)
	}
	place(g, a, 520, 250)
	place(g, a, 700, 250) // fin de mène
	if a.Score != 2 || g.Phase != "mene" {
		t.Fatalf("A marque 2 (10 et 20 px, devant 40 px de B), obtenu %d / %s", a.Score, g.Phase)
	}
	for g.Phase != "aim" {
		g.step(0.1)
	}
	if g.Turn != a.ID || a.Left != BoulesEach || len(g.Boules) != 0 {
		t.Fatal("nouvelle mène : celui qui a marqué commence, boules remises")
	}
}

func TestPetanqueShootOut(t *testing.T) {
	g, a, b := pitch2(t)
	g.Turn = a.ID
	place(g, a, 500, 300)
	// B tire fort droit sur la boule de A : elle doit bouger
	g.Turn = b.ID
	ax := g.Boules[0].X
	g.throw(b, math.Atan2(300-CircleY, ax-CircleX), 1)
	settlePet(g)
	if g.Boules[0].X == ax && len(g.Boules) == 2 {
		t.Fatal("la boule de A aurait dû être frappée")
	}
}

func TestPetanqueJackOutReplays(t *testing.T) {
	g, a, _ := pitch2(t)
	g.Turn = a.ID
	mene := g.Mene
	g.Jack.X = PW - 10
	g.Jack.launch(0, 500) // le cochonnet sort
	g.Phase = "roll"
	settlePet(g)
	if g.Phase != "mene" {
		t.Fatalf("cochonnet sorti : mène rejouée, obtenu %s", g.Phase)
	}
	for g.Phase != "aim" {
		g.step(0.1)
	}
	if g.Mene != mene+1 || a.Left != BoulesEach {
		t.Fatal("nouvelle mène attendue, boules remises")
	}
}

func TestPetanqueTimeoutAndLeave(t *testing.T) {
	g, a, b := pitch2(t)
	g.Turn = a.ID
	for range int(ThrowTime/PetTick.Seconds()) + 2 {
		g.step(PetTick.Seconds())
	}
	if a.Left != BoulesEach-1 {
		t.Fatal("trop long : la boule est perdue")
	}
	g.leave(b.ID)
	if g.Phase != "wait" || a.Score != 0 {
		t.Fatal("moins de 2 joueurs : attente")
	}
}
