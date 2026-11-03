package main

import (
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// flat : un parcours avec un seul golfeur, sur un trou vide (départ à gauche, trou à droite).
func flat() (*Course, *Golfer) {
	c := newCourse("0000")
	c.Hole = Hole{Num: 1, Par: 3, TeeX: 100, TeeY: 250, CupX: 700, CupY: 250}
	g := c.join("A")
	c.Events = nil
	return c, g
}

func settle(c *Course, g *Golfer) {
	for i := 0; g.Moving && i < 60*30; i++ {
		c.step(1.0 / 60)
	}
}

func TestHolesPlayable(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	seen := map[[2]float64]bool{}
	for i := range 300 {
		h := genHole(r, 1)
		if h.Par < 2 || h.Par > 5 {
			t.Fatalf("trou %d : par %d", i, h.Par)
		}
		if p, ok := h.route(); !ok || p != h.Par {
			t.Fatalf("trou %d injouable", i)
		}
		seen[[2]float64{h.TeeX, h.CupY}] = true
	}
	if len(seen) < 290 {
		t.Fatalf("trous trop semblables : %d différents sur 300", len(seen))
	}
}

func TestShotRollsAndStops(t *testing.T) {
	c, g := flat()
	c.shoot(g, 0, 0.5)
	c.shoot(g, 0, 1) // ignoré : la balle roule encore
	settle(c, g)
	if g.Moving || g.Strokes != 1 || g.X <= 200 || g.Y != 250 {
		t.Fatalf("balle : x=%.0f y=%.0f coups=%d roule=%v", g.X, g.Y, g.Strokes, g.Moving)
	}
}

func TestWallBounceNoTunnel(t *testing.T) {
	c, g := flat()
	c.Hole.Walls = []Wall{{300, 150, 16, 200}}
	c.shoot(g, 0, 1)
	for g.Moving {
		c.step(1.0 / 60)
		if g.X > 300-GBallR+0.01 {
			t.Fatalf("la balle traverse le mur : x=%.1f", g.X)
		}
	}
	if g.X >= 290 || g.X < GBallR {
		t.Fatalf("rebond attendu, x=%.0f", g.X)
	}
}

func TestCupAndSpeed(t *testing.T) {
	c, g := flat()
	g.X = 600
	c.shoot(g, 0, 1) // trop fort : passe par-dessus
	settle(c, g)
	if g.Done {
		t.Fatal("une balle trop rapide ne doit pas tomber")
	}
	c.tee(g)
	g.X = 650
	c.shoot(g, 0, 0.18)
	settle(c, g)
	if !g.Done || g.X != 700 || c.Phase != "hole" || g.ToPar != 1-3 || len(g.Card) != 1 {
		t.Fatalf("dans le trou attendu : fini=%v x=%.0f phase=%s toPar=%d", g.Done, g.X, c.Phase, g.ToPar)
	}
}

func TestWaterPenalty(t *testing.T) {
	c, g := flat()
	c.Hole.Water = []Circle{{300, 250, 40}}
	c.shoot(g, 0, 0.8)
	settle(c, g)
	if g.X != 100 || g.Strokes != 2 {
		t.Fatalf("retour au point de frappe + 1 attendu : x=%.0f coups=%d", g.X, g.Strokes)
	}
}

func TestSandSlows(t *testing.T) {
	dist := func(sand bool) float64 {
		c, g := flat()
		if sand {
			c.Hole.Sand = []Circle{{300, 250, 400}}
		}
		c.shoot(g, 0, 0.4)
		settle(c, g)
		return g.X
	}
	if s, n := dist(true), dist(false); s >= n {
		t.Fatalf("le sable doit freiner : %.0f contre %.0f", s, n)
	}
}

func TestGameFlow(t *testing.T) {
	c, g := flat()
	late := (*Golfer)(nil)
	for hole := 1; hole <= Holes; hole++ {
		if c.Hole.Num != hole || c.Phase != "play" {
			t.Fatalf("trou %d attendu, %d (%s)", hole, c.Hole.Num, c.Phase)
		}
		if hole == 3 {
			late = c.join("B")
		}
		c.Wait = 0.001 // temps écoulé : balle ramassée, MaxStrokes
		c.step(0.01)
		if hole < Holes {
			for c.Phase == "hole" {
				c.step(0.5)
			}
		}
	}
	if c.Phase != "end" || len(g.Card) != Holes || len(late.Card) != Holes || late.Card[0] != 0 || math.Abs(float64(g.ToPar)) == 0 {
		t.Fatalf("fin : phase=%s cartes %v %v", c.Phase, g.Card, late.Card)
	}
	for c.Phase == "end" {
		c.step(0.5)
	}
	if c.Hole.Num != 1 || g.ToPar != 0 || len(g.Card) != 0 {
		t.Fatal("une nouvelle partie doit repartir à zéro")
	}
}

func TestShotValidation(t *testing.T) {
	c, g := flat()
	for _, p := range [][2]float64{{0, 0}, {0, -1}, {math.NaN(), 1}, {math.Inf(1), 1}} {
		c.shoot(g, p[0], p[1])
	}
	if g.Strokes != 0 || g.Moving {
		t.Fatal("coups invalides acceptés")
	}
	c.shoot(g, 0, 50) // puissance plafonnée
	if v := math.Hypot(g.vx, g.vy); v != ShotMax {
		t.Fatalf("vitesse %.0f", v)
	}
}

func TestGolfRooms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(serveGolf))
	defer srv.Close()
	ctx := context.Background()
	c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/?name=A", nil)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Pin     string
		Golfers []struct{ Strokes int }
	}
	for m.Pin == "" {
		if err := wsjson.Read(ctx, c, &m); err != nil {
			t.Fatal(err)
		}
	}
	wsjson.Write(ctx, c, map[string]float64{"a": 0.3, "p": 0.5}) // même format que golf.html
	for len(m.Golfers) == 0 || m.Golfers[0].Strokes == 0 {
		if err := wsjson.Read(ctx, c, &m); err != nil {
			t.Fatal(err)
		}
	}
	c.CloseNow()
	time.Sleep(200 * time.Millisecond)
	coursesMu.Lock()
	_, still := courses[m.Pin]
	coursesMu.Unlock()
	if still {
		t.Fatal("un salon vide doit être supprimé")
	}
}
