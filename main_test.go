package main

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// bomb renvoie la seule bombe en jeu (tests à moins de 6 joueurs).
func bomb(g *Game) *Bomb {
	if len(g.Bombs) != 1 {
		return &Bomb{}
	}
	return g.Bombs[0]
}

func TestRound(t *testing.T) {
	g := newGame("0000", "classic")
	a, b := g.join("A"), g.join("B")
	for g.Phase == "lobby" {
		g.step(0.1)
	}
	a.X, a.Y, b.X, b.Y = 100, 100, 600, 500 // loin l'un de l'autre
	holder, other := a, b
	if bomb(g).Holder == b.ID {
		holder, other = b, a
	}

	bomb(g).cool = 0
	other.X, other.Y = holder.X+10, holder.Y // contact
	g.step(0.01)
	if bomb(g).Holder != other.ID {
		t.Fatal("la bombe aurait dû passer au contact")
	}
	g.step(0.01)
	if bomb(g).Holder != other.ID {
		t.Fatal("le cooldown aurait dû empêcher le retour immédiat")
	}

	bomb(g).Fuse = 0.001
	g.step(0.01)
	if g.Phase != "end" || g.Winner != holder.Name || other.Alive {
		t.Fatalf("fin attendue avec %s gagnant, obtenu phase=%s winner=%q", holder.Name, g.Phase, g.Winner)
	}

	for g.Phase == "end" {
		g.step(0.1)
	}
	if g.Phase != "lobby" || !a.Alive || !b.Alive {
		t.Fatal("retour au lobby avec tout le monde vivant attendu")
	}
}

func TestFull(t *testing.T) {
	g := newGame("0000", "classic")
	for range colors {
		g.join("x")
	}
	if g.join("trop") != nil {
		t.Fatal("le 11e joueur aurait dû être refusé")
	}
}

func TestBots(t *testing.T) {
	g := newGame("0000", "classic")
	h := g.join("Humain")
	for range 12 {
		g.addBot()
	}
	if len(g.Players) != 10 {
		t.Fatalf("10 joueurs max bots compris, obtenu %d", len(g.Players))
	}
	g.removeBot()
	if len(g.Players) != 9 || g.Players[8].Name != "Bot 8" {
		t.Fatal("le dernier bot (Bot 9) aurait dû être retiré")
	}

	// un bot porteur doit rattraper un humain immobile
	for g.Phase == "lobby" {
		g.step(0.1)
	}
	b := g.Players[1]
	for _, p := range g.Players {
		p.Alive = p == h || p == b
	}
	g.Bombs = []*Bomb{{Holder: b.ID, Fuse: 999}}
	h.X, h.Y, b.X, b.Y = 700, 500, 100, 100
	for i := 0; i < 30*10 && bomb(g).Holder == b.ID; i++ {
		g.step(1.0 / 30)
	}
	if bomb(g).Holder != h.ID {
		t.Fatal("le bot n'a pas rattrapé l'humain en 10 s")
	}

	g.leave(h.ID)
	if len(g.Players) != 0 || g.Phase != "lobby" {
		t.Fatal("les bots doivent partir avec le dernier humain")
	}
}

func playing(t *testing.T) (*Game, *Player, *Player) {
	g := newGame("0000", "classic")
	a, b := g.join("A"), g.join("B")
	for g.Phase == "lobby" {
		g.step(0.1)
	}
	if bomb(g).Holder == b.ID {
		a, b = b, a
	}
	bomb(g).Fuse, bomb(g).cool, g.spawn = 999, 0, 999 // a porte la bombe, pas de bonus aléatoire
	a.X, a.Y, b.X, b.Y = 100, 100, 600, 500
	return g, a, b
}

func TestShield(t *testing.T) {
	g, a, b := playing(t)
	g.Bonuses = []Bonus{{"shield", b.X, b.Y}}
	g.step(0.01)
	if b.Shield == 0 || len(g.Bonuses) != 0 {
		t.Fatal("b aurait dû ramasser le bouclier")
	}
	b.X, b.Y = a.X+10, a.Y
	g.step(0.01)
	if bomb(g).Holder != a.ID {
		t.Fatal("le bouclier aurait dû bloquer la bombe")
	}
	b.Shield = 0
	g.step(0.01)
	if bomb(g).Holder != b.ID {
		t.Fatal("sans bouclier la bombe passe")
	}
}

func TestSpeedAndIce(t *testing.T) {
	g, a, b := playing(t)
	g.Bonuses = []Bonus{{"ice", a.X, a.Y}, {"speed", b.X, b.Y}}
	g.step(0.01)
	if b.Slow == 0 || a.Slow != 0 || b.Fast == 0 {
		t.Fatal("ice doit geler les autres (pas celui qui le prend) et speed accélérer b")
	}
	a.in, b.in = Input{R: true}, Input{L: true}
	ax, bx := a.X, b.X
	g.step(0.1)
	// a : 220*1.1 (porteur) ; b : 220*1.5*0.5
	if da, db := a.X-ax, bx-b.X; math.Abs(da-24.2) > 0.01 || math.Abs(db-16.5) > 0.01 {
		t.Fatalf("vitesses inattendues : a=%.2f b=%.2f", da, db)
	}
}

func TestRooms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(serveWS))
	defer srv.Close()
	ctx := context.Background()
	dial := func(q string) (*websocket.Conn, map[string]any) {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := wsjson.Read(ctx, c, &m); err != nil {
			return c, map[string]any{"err": websocket.CloseStatus(err), "msg": err.Error()}
		}
		return c, m
	}
	state := func(c *websocket.Conn) map[string]any {
		var m map[string]any
		for m["pin"] == nil {
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
		}
		return m
	}

	a, _ := dial("name=A")
	pin := state(a)["pin"].(string)
	b, _ := dial("name=B")
	if pin2 := state(b)["pin"].(string); pin2 == pin {
		t.Fatal("deux créations doivent donner deux salons")
	}
	c, _ := dial("name=C&pin=" + pin)
	if n := len(state(c)["players"].([]any)); n != 2 {
		t.Fatalf("C devrait rejoindre le salon de A (2 joueurs), obtenu %d", n)
	}
	_, m := dial("name=D&pin=99999")
	if m["err"] != websocket.StatusPolicyViolation || !strings.Contains(m["msg"].(string), "Code inconnu") {
		t.Fatalf("code inconnu attendu, obtenu %v", m)
	}

	a.CloseNow()
	c.CloseNow()
	time.Sleep(200 * time.Millisecond)
	roomsMu.Lock()
	_, still := rooms[pin]
	roomsMu.Unlock()
	if still {
		t.Fatal("un salon vide doit être supprimé")
	}
	b.CloseNow()
}

func TestWall(t *testing.T) {
	g := newGame("0000", "classic")
	g.Walls = []Wall{{400, 0, 40, 600}}
	p := g.join("A")
	p.X, p.Y, p.in = 300, 300, Input{R: true}
	for range 60 {
		g.step(1.0 / 30)
	}
	if math.Abs(p.X-(400-R)) > 0.01 {
		t.Fatalf("le mur aurait dû arrêter le joueur à x=%v, il est à %v", 400-R, p.X)
	}
	for range 200 {
		if x, y := g.freeSpot(R); g.hits(x, y, R) {
			t.Fatal("freeSpot a donné une position dans un mur")
		}
	}
}

func TestScoresAndArenas(t *testing.T) {
	g := newGame("0000", "classic")
	a, b, c := g.join("A"), g.join("B"), g.join("C")
	for g.Phase == "lobby" {
		g.step(0.1)
	}
	for _, p := range []*Player{a, b, c} {
		p.X, p.Y = 100+float64(p.ID)*200, 300 // loin les uns des autres
	}
	g.spawn = 999
	first := g.get(bomb(g).Holder)
	bomb(g).Fuse = 0.001
	g.step(0.01)
	second := g.get(bomb(g).Holder)
	bomb(g).Fuse = 0.001
	g.step(0.01)
	if g.Phase != "end" || first.Score != 0 || second.Score != 1 || g.Winner == "" {
		t.Fatalf("scores attendus 0 / 1 / 4, obtenu %d / %d, phase %s", first.Score, second.Score, g.Phase)
	}
	if w := g.alive()[0]; w.Score != 4 {
		t.Fatalf("le gagnant devrait avoir 1+1+2 = 4 points, il a %d", w.Score)
	}

	for g.Phase == "end" {
		g.step(0.1)
	}
	if g.Arena != arenas[1].Name {
		t.Fatalf("arène suivante attendue (%s), obtenu %s", arenas[1].Name, g.Arena)
	}
	for _, p := range g.Players {
		if g.hits(p.X, p.Y, R) || p.Score == 0 && p != first {
			t.Fatal("joueur dans un mur, ou scores perdus entre les manches")
		}
	}
}

func TestArenasPlayable(t *testing.T) {
	// parcours en largeur sur une grille : toute zone libre doit être atteignable (pas de poche fermée)
	const step = 6.0
	for _, ar := range arenas {
		g := newGame("0000", "classic")
		g.Walls = ar.Walls
		type pt struct{ x, y int }
		free := map[pt]bool{}
		var start pt
		for x := 0; R+float64(x)*step <= W-R; x++ {
			for y := 0; R+float64(y)*step <= H-R; y++ {
				if !g.hits(R+float64(x)*step, R+float64(y)*step, R) {
					free[pt{x, y}], start = true, pt{x, y}
				}
			}
		}
		seen, queue := map[pt]bool{start: true}, []pt{start}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for _, n := range []pt{{p.x + 1, p.y}, {p.x - 1, p.y}, {p.x, p.y + 1}, {p.x, p.y - 1}} {
				if free[n] && !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
		if len(seen) != len(free) {
			t.Errorf("%s : %d positions libres inatteignables", ar.Name, len(free)-len(seen))
		}
	}
}

// round démarre une manche avec n joueurs bien espacés, sans bonus.
func round(t *testing.T, mode string, n int) *Game {
	g := newGame("0000", mode)
	for i := range n {
		g.join(string(rune('A' + i)))
	}
	for g.Phase == "lobby" {
		g.step(0.1)
	}
	for i, p := range g.Players {
		p.X, p.Y = 60+float64(i%5)*160, 100+float64(i/5)*300
	}
	g.spawn = 999
	return g
}

func TestMultiBomb(t *testing.T) {
	g := round(t, "classic", 6)
	if len(g.Bombs) != 2 || g.Bombs[0].Holder == g.Bombs[1].Holder {
		t.Fatalf("6 joueurs : 2 bombes sur 2 porteurs différents attendues, obtenu %d", len(g.Bombs))
	}
	// un porteur ne peut pas recevoir la 2e bombe
	h1, h2 := g.get(g.Bombs[0].Holder), g.get(g.Bombs[1].Holder)
	h2.X, h2.Y = h1.X+10, h1.Y
	g.Bombs[0].cool, g.Bombs[1].cool = 0, 0
	g.step(0.01)
	if g.Bombs[0].Holder == g.Bombs[1].Holder {
		t.Fatal("un joueur ne doit jamais porter deux bombes")
	}
	// une explosion à 6 : il reste 5 joueurs, donc plus de 2e bombe
	g.Bombs[0].Fuse = 0.001
	g.step(0.01)
	if len(g.alive()) != 5 || len(g.Bombs) != 1 {
		t.Fatalf("après une explosion à 6 : 5 vivants et 1 bombe attendus, obtenu %d / %d", len(g.alive()), len(g.Bombs))
	}
}

func TestHotPotato(t *testing.T) {
	g := round(t, "hot", 3)
	b := bomb(g)
	if b.Fuse != HotStart {
		t.Fatalf("mèche de départ %v attendue, obtenu %v", HotStart, b.Fuse)
	}
	h := g.get(b.Holder)
	var other *Player
	for _, p := range g.Players {
		if p != h {
			other = p
			break
		}
	}
	pass := func(from, to *Player) {
		b.cool = 0
		to.X, to.Y = from.X+10, from.Y
		g.step(0.01)
		to.X, to.Y = from.X+300, from.Y // on les sépare
	}
	b.Fuse = 3
	pass(h, other)
	if b.Holder != other.ID || math.Abs(b.Fuse-(3-0.01+HotGain)) > 1e-9 {
		t.Fatalf("passe à un nouveau joueur : +%v s attendu, mèche = %v", HotGain, b.Fuse)
	}
	f := b.Fuse
	other.X, other.Y = 400, 300
	h.X, h.Y = 100, 500
	pass(other, h) // renvoi au porteur précédent : pas de recharge
	if b.Holder != h.ID || b.Fuse >= f {
		t.Fatalf("renvoi au porteur précédent : pas de recharge attendue (avant %v, après %v)", f, b.Fuse)
	}
	b.Fuse = HotMax - 0.1
	b.from = 0
	other.X, other.Y = h.X+10, h.Y
	b.cool = 0
	g.step(0.01)
	if b.Fuse > HotMax {
		t.Fatalf("la mèche ne doit pas dépasser %v, obtenu %v", HotMax, b.Fuse)
	}
}

func TestTeams(t *testing.T) {
	g := round(t, "teams", 4)
	n := map[int]int{}
	for _, p := range g.Players {
		n[p.Team]++
	}
	if n[1] != 2 || n[2] != 2 {
		t.Fatalf("2 équipes de 2 attendues, obtenu %v", n)
	}
	h := g.get(bomb(g).Holder)
	var mate, enemy *Player
	for _, p := range g.Players {
		switch {
		case p == h:
		case p.Team == h.Team:
			mate = p
		default:
			enemy = p
		}
	}
	mate.X, mate.Y = h.X+10, h.Y
	bomb(g).cool = 0
	g.step(0.01)
	if bomb(g).Holder != h.ID {
		t.Fatal("pas de passe entre coéquipiers")
	}
	mate.X, mate.Y = 700, 500
	enemy.X, enemy.Y = h.X+10, h.Y
	g.step(0.01)
	if bomb(g).Holder != enemy.ID {
		t.Fatal("la passe à un adversaire doit marcher")
	}

	// les 2 adversaires sautent : l'équipe de h gagne, ses 2 survivants prennent le bonus
	for _, p := range g.Players {
		if p.Team != h.Team && p != enemy {
			p.Alive = false
		}
	}
	enemy.X, enemy.Y = 400, 300
	bomb(g).Fuse = 0.001
	g.step(0.01)
	if g.Phase != "end" || g.WinTeam != h.Team || g.Winner != teamNames[h.Team] {
		t.Fatalf("victoire de l'équipe %d attendue, obtenu phase=%s équipe=%d", h.Team, g.Phase, g.WinTeam)
	}
	if h.Score != 3 || mate.Score != 3 { // +1 survie + 2 victoire
		t.Fatalf("3 points chacun attendus, obtenu %d et %d", h.Score, mate.Score)
	}
}

func TestStatsAndEvents(t *testing.T) {
	g := round(t, "classic", 2)
	h := g.get(bomb(g).Holder)
	o := g.Players[0]
	if o == h {
		o = g.Players[1]
	}
	g.Events = nil
	g.Bonuses = []Bonus{{"speed", h.X - 15, h.Y}} // à portée de h seulement (o est à 45 px)
	o.X, o.Y = h.X+30, h.Y                        // contact (< 2R) pour la passe
	bomb(g).cool = 0
	g.step(0.01)
	kinds := []string{}
	for _, e := range g.Events {
		kinds = append(kinds, e.Kind)
	}
	if !slices.Contains(kinds, "pick") || !slices.Contains(kinds, "pass") {
		t.Fatalf("événements pick et pass attendus, obtenu %v", kinds)
	}
	bomb(g).Fuse = 0.001
	g.step(0.01)
	if g.Phase != "end" {
		t.Fatal("fin de manche attendue")
	}
	got := map[string]string{}
	for _, s := range g.Stats {
		got[s.Title] = s.Name + " " + s.Value
	}
	if got["Le plus de passes"] != h.Name+" 1 passes" || got["Victime préférée"] != h.Name+" → "+o.Name+" 1 fois" || got["Chasseur de bonus"] != h.Name+" 1 bonus" {
		t.Fatalf("stats inattendues : %v", got)
	}
	if !slices.ContainsFunc(g.Events, func(e Event) bool { return e.Kind == "boom" }) || !slices.ContainsFunc(g.Events, func(e Event) bool { return e.Kind == "win" }) {
		t.Fatal("événements boom et win attendus")
	}
}

func TestEmote(t *testing.T) {
	g := newGame("0000", "classic")
	p := g.join("A")
	g.emote(p, 9)
	if p.Emote != 0 {
		t.Fatal("emote hors 1-4 refusée")
	}
	g.emote(p, 2)
	g.emote(p, 3) // trop tôt : ignorée
	if p.Emote != 2 || len(g.Events) != 1 {
		t.Fatalf("anti-spam : 1 seule emote attendue, obtenu emote=%d événements=%d", p.Emote, len(g.Events))
	}
	for range 70 {
		g.step(1.0 / 30)
	}
	if p.Emote != 0 {
		t.Fatal("l'emote doit disparaître après 2 s")
	}
	g.emote(p, 3)
	if p.Emote != 3 {
		t.Fatal("nouvelle emote possible après le délai")
	}
}

func TestModeValidation(t *testing.T) {
	if newGame("1", "n'importe quoi").Mode != "classic" || newGame("1", "hot").Mode != "hot" {
		t.Fatal("mode inconnu → classique")
	}
}
