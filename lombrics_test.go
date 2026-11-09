package main

import (
	"math"
	"testing"
	"time"
)

// worms2 : deux équipes humaines, partie lancée, sans vent, sans mine, baril ni caisse (testés à part).
func worms2(t *testing.T) (*WormGame, *Squad, *Squad) {
	g := newWormGame("0000")
	a, b := g.join("A", false), g.join("B", false)
	for g.Phase != "play" {
		g.step(0.1)
	}
	g.Wind, g.Objs = 0, g.Objs[:0]
	return g, a, b
}

// flatLand : sol plat à partir de la hauteur y.
func flatLand(g *WormGame, y float64) {
	for i := range g.land {
		g.land[i] = float64(i/LGW)*LCell >= y
	}
}

// run : dt secondes de jeu.
func run(g *WormGame, secs float64) {
	for range int(secs / ArtTick.Seconds()) {
		g.step(ArtTick.Seconds())
	}
}

// nextUp : joue jusqu'au tour suivant (ou la fin de partie).
func nextUp(g *WormGame) {
	turn := g.Turn
	for i := 0; i < 5000 && g.Phase != "end" && (g.Phase != "play" || g.Turn == turn); i++ {
		g.step(ArtTick.Seconds())
	}
}

// wormDuel : sol plat à y = 300, le ver qui joue en x = 300, un adversaire en foeX, les autres loin. Sans vent.
func wormDuel(t *testing.T, foeX float64) (*WormGame, *Squad, *Worm, *Worm) {
	g, _, _ := worms2(t)
	flatLand(g, 300)
	me := g.worm(g.Turn)
	var foe *Worm
	far := []float64{40, 80, 700, 740}
	for _, w := range g.Worms {
		w.Y, w.air = 300-WormR-1, false
		switch {
		case w == me:
			w.X, w.Dir = 300, 1
		case foe == nil && w.Team != me.Team:
			foe, w.X = w, foeX
		default:
			w.X, far = far[0], far[1:]
		}
	}
	return g, g.squad(me.Team), me, foe
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
	me := g.worm(g.Turn)
	g.fire(s, Grenade, -math.Pi/2, 0.3, 0) // en l'air : retombe et explose au bout de 3 s
	if g.Phase != "play" || len(g.Projs) != 1 || g.Projs[0].Kind != Grenade || !g.Fired || g.Wait != LRetreat {
		t.Fatal("la grenade doit partir, puis le ver a le temps de se mettre à l'abri")
	}
	x0 := me.X
	s.move = 1
	run(g, 1)
	if me.X < x0+30 {
		t.Fatalf("le ver se met à l'abri pendant que la grenade vole : x=%.0f", me.X)
	}
	g.fire(s, Bazooka, -math.Pi/2, 0.5, 0)
	if len(g.Projs) != 1 {
		t.Fatal("un seul tir par tour")
	}
	nextUp(g)
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
	g.Objs = g.Objs[:0]
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
	if len(g.Projs) != 1 {
		t.Fatalf("le bot aurait dû tirer, phase %s", g.Phase)
	}
	impact := math.Inf(1)
	for i := 0; i < 3000 && len(g.Projs) > 0; i++ {
		g.Events = nil
		g.step(ArtTick.Seconds())
		for _, e := range g.Events {
			if e.Kind == "boom" {
				impact = math.Hypot(e.X-550, e.Y-(300-WormR-1))
			}
		}
	}
	if impact > 80 { // il vise avec un peu de hasard : on vérifie qu'il tombe près, pas qu'il touche
		t.Fatalf("sur terrain plat, le bot doit tomber près de sa cible : %.0f px", impact)
	}
	g.leave(h.ID)
	if len(g.Squads) != 0 {
		t.Fatal("les bots partent avec le dernier humain")
	}
}

func TestWormsFallAlongWall(t *testing.T) {
	g, _, _ := worms2(t)
	for i := range g.land { // une paroi verticale à gauche (x < 400), un sol à y = 400
		r, c := i/LGW, i%LGW
		g.land[i] = float64(c)*LCell < 400 || float64(r)*LCell >= 400
	}
	w := g.Worms[0]
	w.X, w.Y, w.air = 400+WormR+0.5, 200, false // collé à la paroi, rien dessous
	for range 60 {
		g.move(w, 1.0/60)
	}
	if w.Y < 380 || w.air {
		t.Fatalf("un ver contre une paroi doit tomber jusqu'au sol : y=%.0f air=%v", w.Y, w.air)
	}
}

func TestWormsShotgunTwice(t *testing.T) {
	g, s, me, foe := wormDuel(t, 360)
	g.fire(s, Shotgun, 0, 1, 0)
	if foe.HP != 100-int(weapons[Shotgun].dmg) || g.Fired {
		t.Fatalf("premier coup de fusil : %d PV, tour pas fini attendu (fired=%v)", foe.HP, g.Fired)
	}
	g.fire(s, Bazooka, 0, 1, 0) // pas d'autre arme entre les deux tirs
	if len(g.Projs) != 0 {
		t.Fatal("entre les deux tirs du fusil, on ne change pas d'arme")
	}
	s.move = 1
	run(g, 0.5)
	if me.X != 300 {
		t.Fatal("entre les deux tirs du fusil, on ne bouge pas")
	}
	g.fire(s, Shotgun, -math.Pi/2, 1, 0)
	if !g.Fired {
		t.Fatal("après le second tir : temps de repli")
	}
}

func TestWormsBurstWeapons(t *testing.T) {
	for _, weapon := range []int{Uzi, Flamer} {
		g, s, me, foe := wormDuel(t, 340)
		g.fire(s, weapon, 0, 1, 0)
		s.move = 1
		run(g, 0.5)
		if me.X != 300 {
			t.Fatalf("arme %d : on ne bouge pas pendant la rafale", weapon)
		}
		run(g, 2)
		if foe.HP > 80 || !g.Fired {
			t.Fatalf("arme %d : la rafale doit toucher l'adversaire à bout portant (%d PV) puis laisser le repli", weapon, foe.HP)
		}
		if me.HP != 100 {
			t.Fatalf("arme %d : le tireur ne se blesse pas en tirant droit devant (%d PV)", weapon, me.HP)
		}
	}
}

func TestWormsBat(t *testing.T) {
	g, s, _, foe := wormDuel(t, 312)
	g.fire(s, Bat, -0.5, 1, 0)
	if foe.HP != 100-int(weapons[Bat].dmg) || !foe.air || foe.vx < 300 || foe.vy > -200 {
		t.Fatalf("coup de batte : %d PV et envolé vers la droite attendus, obtenu vx=%.0f vy=%.0f", foe.HP, foe.vx, foe.vy)
	}
}

func TestWormsDynamiteAndAirStrike(t *testing.T) {
	g, s, _, foe := wormDuel(t, 600)
	g.fire(s, Dynamite, 0, 1, 0)
	if s.Ammo[Dynamite] != weapons[Dynamite].ammo-1 || len(g.Projs) != 1 {
		t.Fatal("la dynamite est posée et compte une munition")
	}
	nextUp(g)
	if g.solid(300, 302) {
		t.Fatal("la dynamite creuse un gros trou")
	}

	g, s, _, foe = wormDuel(t, 600)
	g.fire(s, AirStrike, 0, 1, foe.X)
	if len(g.Projs) != 5 {
		t.Fatalf("5 missiles attendus, obtenu %d", len(g.Projs))
	}
	nextUp(g)
	if foe.HP >= 100 {
		t.Fatal("la frappe aérienne touche la cible")
	}
	g.fire(s, AirStrike, 0, 1, 0)
	if s.Ammo[AirStrike] != 0 {
		t.Fatal("une seule frappe aérienne par partie")
	}
}

func TestWormsHeli(t *testing.T) {
	g, s, me, _ := wormDuel(t, 600)
	g.fire(s, Heli, 0, 1, 0)
	if me.Heli != HeliTime || g.Fired || s.Ammo[Heli] != weapons[Heli].ammo-1 {
		t.Fatal("l'hélicoptère décolle sans finir le tour")
	}
	s.up, s.move = true, 1
	run(g, 1)
	if me.Y > 250 || me.X < 360 {
		t.Fatalf("en tenant le saut, l'hélicoptère monte et avance : x=%.0f y=%.0f", me.X, me.Y)
	}
	run(g, 2)
	if me.Y < WormR {
		t.Fatalf("l'hélicoptère ne sort pas par le haut de l'écran : y=%.0f", me.Y)
	}
	s.up, s.move = false, 0
	run(g, 6)
	if me.air || me.Heli != 0 || me.HP != 100 {
		t.Fatalf("après 5 s, le ver redescend sans se blesser : air=%v hp=%d", me.air, me.HP)
	}
	g.fire(s, Bazooka, -math.Pi/2, 0.5, 0)
	if len(g.Projs) != 1 {
		t.Fatal("après l'hélicoptère, on peut encore tirer")
	}
}

func TestWormsHurtEndsTurnCratesFlood(t *testing.T) {
	g, s, me, _ := wormDuel(t, 600)
	g.Objs = append(g.Objs[:0], &Obj{Kind: "heal", X: 300, Y: 280, Fuse: -1}) // sans la caisse tombée au hasard en début de tour
	run(g, 0.2)
	if me.HP != 100+HealHP || len(g.Objs) != 0 {
		t.Fatalf("la caisse soigne de %d PV : %d", HealHP, me.HP)
	}
	g.hurt(me, 10)
	if g.Phase != "settle" {
		t.Fatal("le ver qui joue et se blesse finit son tour")
	}
	g.fire(s, Bazooka, 0, 1, 0)
	if len(g.Projs) != 0 {
		t.Fatal("tour fini : plus de tir")
	}
	g.turns = SuddenRounds * len(g.Squads)
	nextUp(g)
	if g.Water != WaterY-FloodStep {
		t.Fatalf("mort subite : l'eau monte de %v, obtenu %.0f", FloodStep, g.Water)
	}
}

func TestWormsNamesAndObjects(t *testing.T) {
	g := newWormGame("0000")
	g.join("A", false)
	g.join("B", false)
	for g.Phase != "play" {
		g.step(0.1)
	}
	kinds := map[string]int{}
	for _, o := range g.Objs {
		if !o.crate() && !g.solid(o.X, o.Y+7) {
			t.Fatalf("%s posé(e) sur la terre attendu : %+v", o.Kind, *o)
		}
		kinds[o.Kind]++
	}
	if kinds["mine"] != 5 || kinds["barrel"] != 3 {
		t.Fatalf("5 mines et 3 barils sur l'île attendus : %v", kinds)
	}
	seen := map[string]bool{}
	for _, w := range g.Worms {
		if w.Name == "" || seen[w.Name] {
			t.Fatalf("chaque ver a un nom à lui : %q", w.Name)
		}
		seen[w.Name] = true
	}
}

func TestWormsRope(t *testing.T) {
	g, s, me, _ := wormDuel(t, 600)
	for i := range g.land { // un plafond au-dessus (y < 150) et le sol à y = 300
		if float64(i/LGW)*LCell < 150 {
			g.land[i] = true
		}
	}
	g.fire(s, Rope, -math.Pi/2, 1, 0)
	if me.Rope == nil || math.Abs(me.Rope[1]-150) > 3 || g.Fired {
		t.Fatalf("la corde s'accroche au plafond sans finir le tour : %v", me.Rope)
	}
	s.up = true
	run(g, 1)
	if me.Y > 250 {
		t.Fatalf("en tenant le haut, la corde raccourcit et le ver monte : y=%.0f", me.Y)
	}
	s.up, s.move = false, 1
	maxX := me.X
	for range 45 { // c'est un pendule : il part et revient, on regarde jusqu'où il va
		run(g, ArtTick.Seconds())
		maxX = max(maxX, me.X)
		if math.Hypot(me.X-me.Rope[0], me.Y-me.Rope[1]) > me.ropeL+1 {
			t.Fatal("le ver reste au bout de sa corde")
		}
	}
	if maxX < 315 {
		t.Fatalf("on se balance au bout de la corde : x max=%.0f", maxX)
	}
	s.move = 0
	g.jump(s)
	if me.Rope != nil {
		t.Fatal("sauter lâche la corde")
	}
	run(g, 3)
	if me.air || me.HP > 100 {
		t.Fatalf("on retombe : air=%v", me.air)
	}
}

func TestWormsMinesAndBarrels(t *testing.T) {
	g, s, me, foe := wormDuel(t, 600)
	g.Objs = append(g.Objs, &Obj{Kind: "mine", X: 560, Y: 293, Fuse: -1}, &Obj{Kind: "barrel", X: 400, Y: 293, Fuse: -1})
	foe.X = 545 // passe près de la mine
	run(g, 0.1)
	if g.Objs[0].Fuse < 0 {
		t.Fatal("un ver près d'une mine la déclenche")
	}
	run(g, 2)
	if foe.HP >= 100 || len(g.Objs) != 1 {
		t.Fatalf("la mine saute et blesse : %d PV", foe.HP)
	}
	g.fire(s, Bazooka, 0, 1, 0) // droit sur le baril
	for i := 0; i < 200 && len(g.Objs) > 0; i++ {
		g.step(ArtTick.Seconds())
	}
	if len(g.Objs) != 0 || g.solid(400, 296) {
		t.Fatal("touché, le baril explose et creuse")
	}
	_ = me

	g, s, me, _ = wormDuel(t, 600)
	g.fire(s, Mine, 0, 1, 0)
	if len(g.Objs) != 1 || g.Objs[0].Kind != "mine" || !g.Fired || s.Ammo[Mine] != weapons[Mine].ammo-1 {
		t.Fatal("on pose une mine à ses pieds")
	}
	run(g, 1)
	if g.Objs[0].Fuse >= 0 {
		t.Fatal("une mine posée ne se déclenche pas tout de suite sous son poseur")
	}
	_ = me
}

func TestWormsAmmoCrate(t *testing.T) {
	g, s, _, _ := wormDuel(t, 600)
	g.Objs = append(g.Objs, &Obj{Kind: "ammo", W: AirStrike, X: 300, Y: 290, Fuse: -1})
	run(g, 0.2)
	if s.Ammo[AirStrike] != weapons[AirStrike].ammo+1 || len(g.Objs) != 0 {
		t.Fatalf("la caisse d'armes donne une frappe aérienne de plus : %d", s.Ammo[AirStrike])
	}
}

func TestWormsBotBat(t *testing.T) {
	g := newWormGame("0000")
	h := g.join("Humain", false)
	bot := g.join("Bot", true)
	for g.Phase != "play" {
		g.step(0.1)
	}
	flatLand(g, 300)
	g.Objs = g.Objs[:0]
	var me, foe *Worm
	for i, w := range g.Worms {
		w.X, w.Y, w.air = 40+float64(i)*30, 300-WormR-1, false
		if w.Team == bot.ID && me == nil {
			me, w.X = w, 400
		} else if w.Team == h.ID && foe == nil {
			foe, w.X = w, 412
		}
	}
	g.team = bot.ID
	g.startTurn()
	g.Turn = me.ID
	run(g, LBotWait+0.2)
	if foe.HP != 100-int(weapons[Bat].dmg) || foe.vx <= 0 {
		t.Fatalf("collé à un adversaire, le bot le frappe à la batte : %d PV", foe.HP)
	}
}
