package main

import (
	"math"
	"testing"
)

// table2 : deux humains, table vide sauf la blanche ; le joueur 0 (jaunes) a la main.
func table2(t *testing.T) (*Table, *Cueist, *Cueist) {
	tb := newTable("0000")
	a, b := tb.join("A", false), tb.join("B", false)
	if tb.Phase != "aim" || len(tb.Balls) != 16 {
		t.Fatalf("partie lancée avec 16 billes attendue, obtenu %s / %d", tb.Phase, len(tb.Balls))
	}
	tb.Balls = tb.Balls[:1]
	tb.Balls[0].X, tb.Balls[0].Y = HeadX, TH/2
	tb.Turn = 0
	return tb, a, b
}

func settlePool(tb *Table) {
	for i := 0; i < 5000 && tb.Phase == "roll"; i++ {
		tb.step(PoolTick.Seconds())
	}
}

// shootAt tire la blanche droit sur (x, y).
func shootAt(tb *Table, x, y, power float64) {
	c := tb.Balls[0]
	tb.shoot(tb.Turn, math.Atan2(y-c.Y, x-c.X), power)
	settlePool(tb)
}

func TestBillardPotOwnPlaysAgain(t *testing.T) {
	tb, _, _ := table2(t)
	// une jaune sur la diagonale du coin bas droit, la blanche derrière : on la pousse dans la poche
	tb.Balls = append(tb.Balls, ball(Yellow, TW-60, TH-60), ball(Yellow, 400, 60), ball(Red, 400, 340))
	tb.Balls[0].X, tb.Balls[0].Y = TW-200, TH-200
	shootAt(tb, TW-60-PoolR*0.7, TH-60-PoolR*0.7, 0.5)
	if tb.left(Yellow) != 1 || tb.Turn != 0 || tb.Phase != "aim" {
		t.Fatalf("jaune rentrée : le joueur des jaunes rejoue, obtenu %d jaunes, tour %d, %s", tb.left(Yellow), tb.Turn, tb.Phase)
	}
}

func TestBillardFoulCueIn(t *testing.T) {
	tb, _, _ := table2(t)
	tb.Balls = append(tb.Balls, ball(Yellow, 400, 60))
	tb.Balls[0].X, tb.Balls[0].Y = TW-100, TH-100
	shootAt(tb, TW, TH, 0.4) // droit dans le coin
	if tb.Turn != 1 || tb.Balls[0].X != HeadX {
		t.Fatalf("blanche rentrée : faute, main à l'adversaire et blanche replacée, obtenu tour %d, x=%.0f", tb.Turn, tb.Balls[0].X)
	}
}

func TestBillardFoulOpponentFirst(t *testing.T) {
	tb, _, _ := table2(t)
	tb.Balls = append(tb.Balls, ball(Red, 400, TH/2), ball(Yellow, 400, 60))
	shootAt(tb, 400, TH/2, 0.3)
	if tb.Turn != 1 || tb.Last == "" {
		t.Fatal("toucher d'abord une bille adverse : faute")
	}
}

func TestBillardBlack(t *testing.T) {
	// noire rentrée alors qu'il reste des jaunes : perdu
	tb, _, b := table2(t)
	tb.Balls = append(tb.Balls, ball(Black, TW-60, TH-60), ball(Yellow, 400, 60))
	tb.Balls[0].X, tb.Balls[0].Y = TW-200, TH-200
	shootAt(tb, TW-60-PoolR*0.7, TH-60-PoolR*0.7, 0.5)
	if tb.Phase != "end" || tb.Winner != b.Name {
		t.Fatalf("noire trop tôt : B gagne, obtenu %s / %q", tb.Phase, tb.Winner)
	}
	// plus aucune jaune : la noire fait gagner
	tb, a, _ := table2(t)
	tb.Balls = append(tb.Balls, ball(Black, TW-60, TH-60), ball(Red, 400, 60))
	tb.Balls[0].X, tb.Balls[0].Y = TW-200, TH-200
	shootAt(tb, TW-60-PoolR*0.7, TH-60-PoolR*0.7, 0.5)
	if tb.Phase != "end" || tb.Winner != a.Name {
		t.Fatalf("noire à la fin : A gagne, obtenu %s / %q", tb.Phase, tb.Winner)
	}
}

func TestBillardBot(t *testing.T) {
	tb := newTable("0000")
	h := tb.join("Humain", false)
	tb.join("Bot", true)
	tb.Turn = 1
	for range int(PoolBotWait/PoolTick.Seconds()) + 2 {
		tb.step(PoolTick.Seconds())
	}
	if tb.Phase != "roll" {
		t.Fatalf("le bot aurait dû jouer, phase %s", tb.Phase)
	}
	settlePool(tb)
	if tb.Phase != "aim" && tb.Phase != "end" {
		t.Fatalf("après le coup du bot, phase %s", tb.Phase)
	}
	tb.leave(h.ID)
	if tb.Players[0] != nil || tb.Players[1] != nil {
		t.Fatal("le bot doit partir avec le dernier humain")
	}
}
