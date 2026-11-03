package main

import "testing"

// track2 : deux humains, manche en cours.
func track2(t *testing.T) (*Track, *Rider, *Rider) {
	g := newTrack("0000")
	a, b := g.join("A", false), g.join("B", false)
	for g.Phase != "play" {
		g.step(MotoTick.Seconds())
	}
	return g, a, b
}

func TestMotosMoveAndTurn(t *testing.T) {
	g, a, _ := track2(t)
	x, y := a.X, a.Y
	g.step(MotoTick.Seconds())
	if a.X != x+1 || a.Y != y || g.grid[x][y] != a.ID {
		t.Fatalf("la moto avance d'une case et laisse un mur : (%d,%d)", a.X, a.Y)
	}
	a.steer(2, 0) // demi-tour : ignoré
	a.steer(0, 1) // à droite : vers le bas
	g.step(MotoTick.Seconds())
	if a.Dir != 1 || a.Y != y+1 || len(a.Pts) != 2 {
		t.Fatalf("virage vers le bas attendu, dir %d y %d virages %d", a.Dir, a.Y, len(a.Pts))
	}
}

func TestMotosCrashAndScore(t *testing.T) {
	g, a, b := track2(t)
	a.Y, b.Y = 0, 30
	a.steer(3, 0) // vers le haut : dans le bord
	g.step(MotoTick.Seconds())
	if a.Alive || b.Score != 1 || g.Phase != "round" || g.Winner != "B" {
		t.Fatalf("A dans le bord : B marque, obtenu A vivant=%v B=%d %s %q", a.Alive, b.Score, g.Phase, g.Winner)
	}
	for g.Phase != "play" {
		g.step(MotoTick.Seconds())
	}
	if !a.Alive || !b.Alive || len(a.Pts) != 1 {
		t.Fatal("nouvelle manche : tout le monde repart")
	}
}

func TestMotosHeadOnAndTrail(t *testing.T) {
	g, a, b := track2(t)
	// face à face sur la même ligne, une case d'écart : même case visée, les deux tombent
	a.X, a.Y, a.Dir = 50, 10, 0
	b.X, b.Y, b.Dir = 52, 10, 2
	g.step(MotoTick.Seconds())
	if a.Alive || b.Alive || g.Phase != "round" || g.Winner != "" {
		t.Fatal("choc frontal : les deux sont éliminés, personne ne marque")
	}
	for g.Phase != "play" {
		g.step(MotoTick.Seconds())
	}
	g.grid[a.X+1][a.Y] = b.ID // un mur juste devant A
	g.step(MotoTick.Seconds())
	if a.Alive {
		t.Fatal("A doit s'écraser sur le mur")
	}
}

func TestMotosMatchAndBot(t *testing.T) {
	g := newTrack("0000")
	h := g.join("Humain", false)
	if g.Phase != "wait" {
		t.Fatal("seul : en attente")
	}
	bot := g.join("Bot", true)
	for g.Phase != "play" {
		g.step(MotoTick.Seconds())
	}
	h.steer(3, 0) // l'humain fonce dans le bord du haut
	for i := 0; i < 200 && g.Phase == "play"; i++ {
		g.step(MotoTick.Seconds())
	}
	if bot.Score != 1 {
		t.Fatalf("le bot doit survivre et marquer, score %d", bot.Score)
	}
	bot.Score = MotoWin - 1
	for g.Phase != "play" {
		g.step(MotoTick.Seconds())
	}
	h.steer(3, 0)
	for i := 0; i < 200 && g.Phase == "play"; i++ {
		g.step(MotoTick.Seconds())
	}
	if g.Phase != "end" || g.Champion != "Bot" {
		t.Fatalf("5 points : le bot gagne la partie, obtenu %s / %q", g.Phase, g.Champion)
	}
	g.leave(h.ID)
	if len(g.Riders) != 0 {
		t.Fatal("les bots partent avec le dernier humain")
	}
}

// Au départ, aucune moto n'est sur la même ligne qu'une autre : pas de choc frontal garanti si personne ne tourne.
func TestMotosStartRows(t *testing.T) {
	for n := 2; n <= MotoMax; n++ {
		g := newTrack("0000")
		g.join("Humain", false)
		for range n - 1 {
			g.Riders = append(g.Riders, &Rider{ID: len(g.Riders) + 1, Bot: true})
		}
		g.startRound()
		rows := map[int]bool{}
		for _, r := range g.Riders {
			if rows[r.Y] || r.Y <= 0 || r.Y >= MotoRows-1 {
				t.Fatalf("%d motos : ligne de départ %d en double ou au bord", n, r.Y)
			}
			rows[r.Y] = true
		}
	}
}
