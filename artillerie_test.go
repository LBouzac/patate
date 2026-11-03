package main

import (
	"math"
	"testing"
	"time"
)

// battle2 : deux humains, partie lancée, sans vent.
func battle2(t *testing.T) (*Battle, *Tank, *Tank) {
	g := newBattle("0000")
	a, b := g.join("A", false), g.join("B", false)
	for g.Phase != "aim" {
		g.step(0.1)
	}
	g.Wind = 0
	return g, a, b
}

func settleArt(g *Battle) {
	for i := 0; i < 3000 && (g.Phase == "fly" || g.Phase == "boom"); i++ {
		g.step(ArtTick.Seconds())
	}
}

func TestArtTerrainAndTanks(t *testing.T) {
	g, a, b := battle2(t)
	if len(g.Ground) != Cols {
		t.Fatalf("%d colonnes attendues, obtenu %d", Cols, len(g.Ground))
	}
	for _, h := range g.Ground {
		if h < 5 || h > int(AH) {
			t.Fatalf("hauteur hors limites : %d", h)
		}
	}
	for _, tk := range []*Tank{a, b} {
		if !tk.Alive || tk.HP != 100 || math.Abs(tk.Y-g.groundY(tk.X)) > 0.01 {
			t.Fatalf("char posé sur le sol avec 100 PV attendu : %+v", tk)
		}
	}
}

func TestArtCraterAndDamage(t *testing.T) {
	g, a, b := battle2(t)
	b.X = 600
	b.Y = g.groundY(b.X)
	before := g.groundY(600)
	g.explode(600, b.Y-6) // tir direct
	if b.HP != 100-int(BlastDmg) {
		t.Fatalf("tir direct : %d PV attendus, obtenu %d", 100-int(BlastDmg), b.HP)
	}
	if g.groundY(600) <= before {
		t.Fatal("l'explosion doit creuser le sol")
	}
	if math.Abs(b.Y-g.groundY(b.X)) > 0.01 {
		t.Fatal("le char retombe au fond du cratère")
	}
	a.HP = 100
}

func TestArtTurnAndWin(t *testing.T) {
	g, a, b := battle2(t)
	g.newTurn(a.ID)
	g.Wind = 0
	g.fire(b, -math.Pi/2, 0.5) // pas son tour
	if g.Phase != "aim" {
		t.Fatal("seul le char dont c'est le tour tire")
	}
	g.fire(a, -math.Pi/3, 0.4)
	if g.Phase != "fly" {
		t.Fatal("l'obus doit partir")
	}
	settleArt(g)
	if g.Phase != "aim" || g.Turn != b.ID {
		t.Fatalf("au tour de B attendu, obtenu %s / %d", g.Phase, g.Turn)
	}
	b.HP = 1
	g.explode(b.X, b.Y-6)
	if b.Alive || !g.checkWin() || g.Winner != "A" {
		t.Fatalf("B détruit : A gagne, obtenu %q", g.Winner)
	}
}

func TestArtBotAims(t *testing.T) {
	g := newBattle("0000")
	h := g.join("Humain", false)
	bot := g.join("Bot", true)
	for g.Phase != "aim" {
		g.step(0.1)
	}
	// terrain plat pour juger la visée
	for i := range g.height {
		g.height[i] = 100
	}
	h.X, bot.X = 200, 600
	h.Y, bot.Y = g.groundY(200), g.groundY(600)
	g.newTurn(bot.ID)
	g.Wind = 0.5
	start := time.Now()
	for range int(ArtBotWait/ArtTick.Seconds()) + 2 {
		g.step(ArtTick.Seconds())
	}
	if time.Since(start) > time.Second {
		t.Fatalf("le bot réfléchit trop longtemps : %v", time.Since(start))
	}
	if g.Phase != "fly" {
		t.Fatalf("le bot aurait dû tirer, phase %s", g.Phase)
	}
	settleArt(g)
	// le bot vise avec un peu de hasard (il touche ~9 fois sur 10) : on vérifie qu'il tombe près, pas qu'il touche
	impact := math.Inf(1)
	for _, e := range g.Events {
		if e.Kind == "boom" {
			impact = math.Hypot(e.X-h.X, e.Y-h.Y)
		}
	}
	if impact > 2*BlastR {
		t.Fatalf("par vent faible sur terrain plat, le bot doit tomber près de sa cible : %.0f px", impact)
	}
	g.leave(h.ID)
	if len(g.Tanks) != 0 {
		t.Fatal("les bots partent avec le dernier humain")
	}
}
