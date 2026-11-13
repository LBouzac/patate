package main

import (
	"slices"
	"testing"
)

// bar2 : deux joueurs, partie lancée, A commence (en bas, vise la ligne 0), B en haut.
func bar2(t *testing.T) (*Barrage, *Pawn, *Pawn) {
	g := newBarrage("0000")
	a, b := g.join("A", false), g.join("B", false)
	for g.Phase != "play" {
		g.step(0.1)
	}
	if g.Turn != a.ID || a.X != 4 || a.Y != 8 || a.Goal != 0 || b.Y != 0 || b.Goal != 8 || a.Walls != 10 {
		t.Fatalf("départ inattendu : tour %d, A %+v, B %+v", g.Turn, a, b)
	}
	return g, a, b
}

func TestBarrageMoveAndTurn(t *testing.T) {
	g, a, b := bar2(t)
	g.move(b, 4, 1) // pas son tour
	if b.Y != 0 {
		t.Fatal("seul celui dont c'est le tour joue")
	}
	g.move(a, 4, 6) // deux cases d'un coup : interdit
	if a.Y != 8 {
		t.Fatal("on n'avance que d'une case")
	}
	g.move(a, 4, 7)
	if a.Y != 7 || g.Turn != b.ID {
		t.Fatalf("A avance et la main passe à B : A %+v, tour %d", a, g.Turn)
	}
}

func TestBarrageWallBlocksAndCounts(t *testing.T) {
	g, a, b := bar2(t)
	g.wall(a, Fence{X: 3, Y: 0}) // barre horizontale sous B, colonnes 3 et 4
	if len(g.Fences) != 1 || a.Walls != 9 || g.Turn != b.ID {
		t.Fatalf("barre posée attendue : %d barres, %d restantes, tour %d", len(g.Fences), a.Walls, g.Turn)
	}
	if slices.Contains(g.Moves, [2]int{4, 1}) {
		t.Fatal("la barre empêche B de descendre tout droit")
	}
	// chevauchement et croisement refusés, et la barre ne sort pas du plateau
	for _, f := range []Fence{{X: 4, Y: 0}, {X: 3, Y: 0, V: true}, {X: 8, Y: 0}, {X: -1, Y: 3}} {
		if g.fits(f) {
			t.Fatalf("barre %+v acceptée à tort", f)
		}
	}
	if !g.fits(Fence{X: 5, Y: 0}) || !g.fits(Fence{X: 3, Y: 1}) {
		t.Fatal("une barre à côté doit être possible")
	}
}

// Une barre qui enfermerait quelqu'un est refusée, et le joueur garde la main.
func TestBarrageNeverCloseThePath(t *testing.T) {
	g, a, b := bar2(t)
	g.Fences = []Fence{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 4, Y: 0}, {X: 6, Y: 0}} // ligne 0 fermée sauf la colonne 8
	g.Turn = a.ID
	g.wall(a, Fence{X: 7, Y: 0, V: true}) // couperait B de la colonne 8 : il serait enfermé
	if len(g.Fences) != 4 || a.Walls != 10 || g.Turn != a.ID {
		t.Fatal("la barre qui enferme B doit être refusée sans coûter le tour")
	}
	if d := dist(g.Fences, b, b.X, b.Y); d < 0 {
		t.Fatal("B a toujours un chemin")
	}
}

func TestBarrageJump(t *testing.T) {
	g, a, b := bar2(t)
	b.X, b.Y = 4, 7 // B juste devant A
	g.Moves = g.moves(a)
	if !slices.Contains(g.Moves, [2]int{4, 6}) || slices.Contains(g.Moves, [2]int{4, 7}) {
		t.Fatalf("A saute par-dessus B : %v", g.Moves)
	}
	g.Fences = []Fence{{X: 3, Y: 6}} // barre derrière B : on passe à côté en diagonale
	g.Moves = g.moves(a)
	if slices.Contains(g.Moves, [2]int{4, 6}) || !slices.Contains(g.Moves, [2]int{3, 7}) || !slices.Contains(g.Moves, [2]int{5, 7}) {
		t.Fatalf("barre derrière B : diagonales attendues, obtenu %v", g.Moves)
	}
}

func TestBarrageWinTwoPlayers(t *testing.T) {
	g, a, _ := bar2(t)
	a.X, a.Y = 2, 1
	g.Moves = g.moves(a)
	g.move(a, 2, 0)
	if g.Phase != "end" || g.Winner != "A" || a.Score != 1 {
		t.Fatalf("A arrive en face et gagne : %s %q %d", g.Phase, g.Winner, a.Score)
	}
	for g.Phase == "end" {
		g.step(0.1)
	}
	if g.Phase != "play" || a.Y != 8 || len(g.Fences) != 0 || a.Walls != 10 {
		t.Fatal("revanche : tout est remis en place")
	}
}

// À 3 ou 4, chacun part d'un coin et le premier sur le trophée du centre gagne.
func TestBarrageCenterMode(t *testing.T) {
	g := newBarrage("0000")
	ps := []*Pawn{g.join("A", false), g.join("B", false), g.join("C", false), g.join("D", false)}
	if g.join("E", false) != nil {
		t.Fatal("4 joueurs au plus")
	}
	for g.Phase != "play" {
		g.step(0.1)
	}
	if !g.Center || ps[0].Walls != 5 {
		t.Fatalf("4 joueurs : trophée au centre et 5 barres chacun, obtenu %v / %d", g.Center, ps[0].Walls)
	}
	for _, p := range ps {
		if d := dist(g.Fences, p, p.X, p.Y); d != 8 {
			t.Fatalf("chaque coin est à 8 pas du centre, %s est à %d", p.Name, d)
		}
	}
	p := g.get(g.Turn)
	p.X, p.Y = BarCenter, BarCenter+1
	g.Moves = g.moves(p)
	g.move(p, BarCenter, BarCenter)
	if g.Phase != "end" || g.Winner != p.Name {
		t.Fatal("toucher le trophée fait gagner")
	}
}

func TestBarrageTimeoutAndLeave(t *testing.T) {
	g, a, b := bar2(t)
	for range int(BarTurn/BarTick.Seconds()) + 2 {
		g.step(BarTick.Seconds())
	}
	if a.Y != 7 || g.Turn != b.ID {
		t.Fatalf("trop long : le pion avance tout seul vers son but, A %+v", a)
	}
	g.leave(b.ID)
	if g.Phase != "wait" || a.In {
		t.Fatal("moins de 2 joueurs : attente")
	}
	// à 3, un départ en cours de partie laisse jouer les deux autres
	g = newBarrage("0000")
	x, y, z := g.join("X", false), g.join("Y", false), g.join("Z", false)
	for g.Phase != "play" {
		g.step(0.1)
	}
	g.leave(g.Turn)
	if g.Phase != "play" || g.get(g.Turn) == nil || len(g.playing()) != 2 {
		t.Fatal("la partie continue à 2")
	}
	_, _, _ = x, y, z
}

// Le bot avance vers son but, et pose une barre quand l'adversaire est plus près que lui.
func TestBarrageBot(t *testing.T) {
	g := newBarrage("0000")
	h := g.join("Humain", false)
	bot := g.join("Bot", true)
	for g.Phase != "play" {
		g.step(0.1)
	}
	g.Turn, g.Moves = bot.ID, g.moves(bot)
	for range int(BarBotWait/BarTick.Seconds()) + 2 {
		g.step(BarTick.Seconds())
	}
	if bot.Y != 1 || g.Turn != h.ID {
		t.Fatalf("à égalité, le bot avance : %+v", bot)
	}
	h.X, h.Y = 4, 2 // l'humain à 2 pas du but : le bot doit le bloquer
	walled := false
	for range 20 { // le bot rate parfois la barre exprès : on lui laisse plusieurs essais
		g.Fences, bot.Walls, bot.Y = []Fence{}, 10, 1
		g.Turn, g.Wait, g.Moves = bot.ID, BarTurn, g.moves(bot)
		for g.Turn == bot.ID {
			g.step(BarTick.Seconds())
		}
		if len(g.Fences) == 1 {
			walled = true
			break
		}
	}
	if !walled || dist(g.Fences, h, h.X, h.Y) <= 2 {
		t.Fatalf("le bot aurait dû retarder l'humain : barres %v", g.Fences)
	}
	g.leave(h.ID)
	if len(g.Players) != 0 {
		t.Fatal("les bots partent avec le dernier humain")
	}
}
