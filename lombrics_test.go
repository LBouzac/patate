package main

import (
	"math"
	"testing"
	"time"
)

// worms2 : deux équipes humaines, partie lancée, sans vent.
func worms2(t *testing.T) (*WormGame, *Squad, *Squad) {
	g := newWormGame("0000")
	a, b := g.join("A", false), g.join("B", false)
	for g.Phase != "play" {
		g.step(0.1)
	}
	g.Wind = 0
	return g, a, b
}

// flatLand : sol plat à partir de la hauteur y.
func flatLand(g *WormGame, y float64) {
	for i := range g.land {
		g.land[i] = float64(i/LGW)*LCell >= y
	}
}

func settleWorms(g *WormGame) {
	for i := 0; i < 3000 && (g.Phase == "fly" || g.Phase == "settle"); i++ {
		g.step(ArtTick.Seconds())
	}
}

func TestWormsSetup(t *testing.T) {
	g, a, _ := worms2(t)
	if len(g.Worms) != 2*WormsPer {
		t.Fatalf("%d vers attendus, obtenu %d", 2*WormsPer, len(g.Worms))
	}
	for _, w := range g.Worms {
		if !w.Alive || w.HP != 100 || w.Y >= WaterY || g.hits(w.X, w.Y) || !g.hits(w.X, w.Y+1) {
			t.Fatalf("ver posé sur la terre, hors de l'eau, avec 100 PV attendu : %+v", w)
		}
	}
	if g.landB64() == "" || len(g.land) != LGW*LGH {
		t.Fatal("terrain vide")
	}
	_ = a
}

func TestWormsWalkJumpDrown(t *testing.T) {
	g, a, b := worms2(t)
	flatLand(g, 300)
	w := g.worm(g.Turn)
	s := g.squad(w.Team)
	w.X, w.Y, w.air = 400, 300-WormR-1, false
	s.move = 1
	for range 30 {
		g.step(ArtTick.Seconds())
	}
	if w.X < 430 || math.Abs(w.Y-(300-WormR-1)) > 0.5 {
		t.Fatalf("le ver doit avancer à plat d'environ %v px en 1 s : x=%.1f y=%.1f", WormSpeed, w.X, w.Y)
	}
	s.move = 0
	g.jump(s)
	if !w.air || w.vy >= 0 {
		t.Fatal("le saut doit décoller le ver")
	}
	for range 60 {
		g.step(ArtTick.Seconds())
	}
	if w.air || w.HP != 100 {
		t.Fatalf("après un saut on retombe sans dégâts : air=%v hp=%d", w.air, w.HP)
	}
	other := a
	if s == a {
		other = b
	}
	g.jump(other)           // pas son tour
	for i := range g.land { // plus de terre : tout le monde tombe à l'eau
		g.land[i] = false
	}
	for range 120 {
		g.step(ArtTick.Seconds())
	}
	for _, w := range g.Worms {
		if w.Alive {
			t.Fatal("un ver qui tombe à l'eau se noie")
		}
	}
	if g.Phase != "end" || g.Winner != "" {
		t.Fatalf("tous noyés : partie finie sans gagnant, obtenu %s %q", g.Phase, g.Winner)
	}
}

func TestWormsExplosion(t *testing.T) {
	g, _, _ := worms2(t)
	flatLand(g, 300)
	w := g.Worms[0]
	w.X, w.Y, w.air = 400, 300-WormR-1, false
	g.explode(400, 300, Bazooka)
	if g.solid(400, 302) {
		t.Fatal("l'explosion doit creuser")
	}
	if w.HP >= 100 || w.HP <= 100-int(weapons[Bazooka].dmg) {
		t.Fatalf("explosion tout près : dégâts entre 0 et %v attendus, PV %d", weapons[Bazooka].dmg, w.HP)
	}
	if !w.air || w.vy >= 0 {
		t.Fatal("l'explosion projette le ver en l'air")
	}
}

func TestWormsTurnsAndWin(t *testing.T) {
	g, a, b := worms2(t)
	flatLand(g, 300)
	for i, w := range g.Worms {
		w.X, w.Y, w.air = 100+float64(i)*100, 300-WormR-1, false
	}
	first := g.worm(g.Turn).Team
	s := g.squad(first)
	g.fire(s, Grenade, -math.Pi/2, 0.3) // en l'air : retombe et explose au bout de 3 s
	if g.Phase != "fly" || g.Proj == nil || g.Proj.Kind != Grenade {
		t.Fatal("la grenade doit partir")
	}
	settleWorms(g)
	if g.Phase != "play" || g.worm(g.Turn).Team == first {
		t.Fatalf("au tour de l'autre équipe attendu, obtenu %s / équipe %d", g.Phase, g.worm(g.Turn).Team)
	}
	for _, w := range g.Worms {
		if w.Team == b.ID {
			w.HP = 1
			g.explode(w.X, w.Y, Shotgun)
		}
	}
	if !g.checkWin() || g.Winner != "A" {
		t.Fatalf("équipe B morte : A gagne, obtenu %q", g.Winner)
	}
	_ = a
}

func TestWormsBotAims(t *testing.T) {
	g := newWormGame("0000")
	h := g.join("Humain", false)
	bot := g.join("Bot", true)
	for g.Phase != "play" {
		g.step(0.1)
	}
	flatLand(g, 300)
	var me, target *Worm
	for _, w := range g.Worms {
		w.X, w.Y, w.air = 50, 300-WormR-1, false
		if w.Team == bot.ID && me == nil {
			me, w.X = w, 250
		} else if w.Team == h.ID && target == nil {
			target, w.X = w, 550
		}
	}
	g.team = bot.ID
	g.startTurn()
	g.Turn, g.Wind = me.ID, 0.5
	start := time.Now()
	for range int(LBotWait/ArtTick.Seconds()) + 2 {
		g.step(ArtTick.Seconds())
	}
	if time.Since(start) > time.Second {
		t.Fatalf("le bot réfléchit trop longtemps : %v", time.Since(start))
	}
	if g.Phase != "fly" {
		t.Fatalf("le bot aurait dû tirer, phase %s", g.Phase)
	}
	impact := math.Inf(1)
	for i := 0; i < 3000 && g.Phase == "fly"; i++ {
		g.Events = nil
		g.step(ArtTick.Seconds())
		for _, e := range g.Events {
			if e.Kind == "boom" {
				impact = math.Hypot(e.X-550, e.Y-(300-WormR-1))
			}
		}
	}
	if impact > 60 {
		t.Fatalf("sur terrain plat, le bot doit tomber près de sa cible : %.0f px", impact)
	}
	g.leave(h.ID)
	if len(g.Squads) != 0 {
		t.Fatal("les bots partent avec le dernier humain")
	}
}
