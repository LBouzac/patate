package main

import (
	"math"
	"testing"
)

// match2 : deux humains, dispositions choisies, coup d'envoi prêt.
func match2(t *testing.T) (*Match, *Footer, *Footer) {
	m := newMatch("0000")
	a, b := m.join("A", false), m.join("B", false)
	if m.Phase != "setup" {
		t.Fatalf("choix des dispositions attendu, obtenu %s", m.Phase)
	}
	m.choose(a, "attack")
	m.choose(b, "defense")
	if m.Phase != "aim" || m.Turn != m.Kick {
		t.Fatalf("coup d'envoi attendu, obtenu %s (tour %d, engagement %d)", m.Phase, m.Turn, m.Kick)
	}
	return m, a, b
}

func settleFoot(m *Match) {
	for i := 0; i < 2000 && m.Phase == "roll"; i++ {
		m.step(FootTick.Seconds())
	}
}

func TestFootFormations(t *testing.T) {
	for name, f := range formations {
		for i, a := range f {
			if a[0] < DiscR || a[0] > FW/2-DiscR-FBallR || a[1] < DiscR || a[1] > FH-DiscR {
				t.Errorf("%s : pion %d hors de sa moitié de terrain %v", name, i, a)
			}
			for _, b := range f[i+1:] {
				if math.Hypot(a[0]-b[0], a[1]-b[1]) < 2*DiscR+4 {
					t.Errorf("%s : pions qui se chevauchent %v %v", name, a, b)
				}
			}
		}
	}
}

func TestFootTurns(t *testing.T) {
	m, _, _ := match2(t)
	other := 1 - m.Turn
	m.shoot(other, 6*other, 0, 0.5) // pas son tour : ignoré
	if m.Phase != "aim" {
		t.Fatal("seule l'équipe dont c'est le tour peut jouer")
	}
	m.shoot(m.Turn, 6*other, 0, 0.5) // pion de l'adversaire : ignoré
	if m.Phase != "aim" {
		t.Fatal("on ne lance que ses propres pions")
	}
	first := m.Turn
	m.shoot(first, 6*first, math.Pi/2, 0.3)
	if m.Phase != "roll" {
		t.Fatal("le coup doit partir")
	}
	settleFoot(m)
	if m.Phase != "aim" || m.Turn != 1-first {
		t.Fatalf("tour de l'adversaire attendu, obtenu %s / %d", m.Phase, m.Turn)
	}
	for range int(TurnTime/FootTick.Seconds()) + 2 { // personne ne joue : le tour passe
		m.step(FootTick.Seconds())
	}
	if m.Turn != first {
		t.Fatal("temps écoulé : le tour doit passer")
	}
}

func TestFootHeavyDisc(t *testing.T) {
	m, _, _ := match2(t)
	d := m.Discs[6*m.Turn]
	d.X, d.Y = 300, 100
	m.Ball.X, m.Ball.Y = 360, 100
	m.shoot(m.Turn, 6*m.Turn, 0, 0.5)
	for range 10 {
		m.step(FootTick.Seconds())
	}
	if m.Ball.vx <= d.vx || d.vx < 0 {
		t.Fatalf("le pion, plus lourd, doit envoyer le ballon plus vite que lui : pion %.0f, ballon %.0f", d.vx, m.Ball.vx)
	}
}

func TestFootGoalAndWin(t *testing.T) {
	m, a, _ := match2(t)
	m.setGoals(0)  // hors limites : ignoré
	m.setGoals(11) // idem
	m.setGoals(2)
	if m.Goals != 2 {
		t.Fatalf("score à atteindre réglé à 2 attendu, obtenu %d", m.Goals)
	}
	for i := 1; i <= m.Goals; i++ {
		for m.Phase != "aim" {
			m.step(0.1)
		}
		m.Turn = 0
		d := m.Discs[0]
		d.X, d.Y = FW-150, FH/2
		m.Ball.X, m.Ball.Y = FW-90, FH/2
		for k, o := range m.Discs[6:] { // la défense s'écarte
			o.X, o.Y = FW/2+50, 30+float64(k)*48
		}
		m.shoot(0, 0, 0, 1)
		settleFoot(m)
		if a.Score != i {
			t.Fatalf("but %d attendu pour A, score %d (phase %s, ballon %.0f,%.0f)", i, a.Score, m.Phase, m.Ball.X, m.Ball.Y)
		}
		if i == 1 {
			m.setGoals(1) // déjà atteint par A : refusé, sinon le match finirait sans but
			if m.Goals != 2 {
				t.Fatalf("score à atteindre sous le score actuel : refus attendu, obtenu %d", m.Goals)
			}
		}
		if i < m.Goals && (m.Phase != "goal" || m.Kick != 1) {
			t.Fatalf("pause puis engagement de B attendus, obtenu %s / %d", m.Phase, m.Kick)
		}
	}
	if m.Phase != "end" || m.Winner != "A" {
		t.Fatalf("victoire de A attendue, obtenu %s / %q", m.Phase, m.Winner)
	}
}

func TestFootDiscEntersGoal(t *testing.T) {
	m, _, _ := match2(t)
	m.Turn = 0
	d := m.Discs[0]
	d.X, d.Y = 60, FH/2
	m.Ball.X, m.Ball.Y = FW/2, 40
	m.shoot(0, 0, math.Pi, 1) // droit dans son propre but, à pleine puissance
	settleFoot(m)
	if d.X >= 0 || d.X < DiscR-CageD-0.01 || !inMouth(d.Y) {
		t.Fatalf("le pion doit finir dans la cage, retenu par le filet : %.1f,%.1f", d.X, d.Y)
	}
}

func TestFootCageSideNet(t *testing.T) {
	m, _, _ := match2(t)
	d := m.Discs[0]
	d.X, d.Y, d.vx, d.vy = -25, FH/2, 0, -900 // dans la cage, lancé vers le filet du côté
	for range 200 {
		m.physics(1.0 / 240)
		if math.Abs(d.Y-FH/2) > MouthH/2-DiscR+0.01 || d.X >= 0 && !inMouth(d.Y) {
			t.Fatalf("le pion doit rester dans la cage : %.1f,%.1f", d.X, d.Y)
		}
	}
}

func TestFootBot(t *testing.T) {
	m := newMatch("0000")
	h := m.join("Humain", false)
	m.join("Bot", true)
	m.choose(h, "balanced")
	if m.Phase != "aim" {
		t.Fatalf("le bot choisit tout de suite sa disposition, phase %s", m.Phase)
	}
	if m.Turn == 0 { // l'humain joue d'abord un petit coup
		m.shoot(0, 0, math.Pi/2, 0.1)
		settleFoot(m)
	}
	for range int(BotThink/FootTick.Seconds()) + 2 {
		m.step(FootTick.Seconds())
	}
	if m.Phase != "roll" {
		t.Fatalf("le bot aurait dû jouer, phase %s", m.Phase)
	}
	m.leave(h.ID)
	if m.Players[0] != nil || m.Players[1] != nil {
		t.Fatal("le bot doit partir avec le dernier humain")
	}
}
