package main

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// duel lance un échange entre deux humains immobiles.
func duel(t *testing.T) (*Duel, *Paddle, *Paddle) {
	d := newDuel("0000")
	a, b := d.join("A", false), d.join("B", false)
	for d.Phase != "play" {
		d.step(0.1)
	}
	d.Events = nil
	return d, a, b
}

func TestDuelWallBounce(t *testing.T) {
	d, _, _ := duel(t)
	d.Ball = Ball{400, BallR + 2, 0, -300}
	d.step(0.02)
	if d.Ball.VY <= 0 || d.Ball.Y < BallR {
		t.Fatalf("rebond sur le haut attendu, obtenu %+v", d.Ball)
	}
}

func TestDuelHitAngleAndSpeed(t *testing.T) {
	d, a, _ := duel(t)
	face := PadX + PadW
	// balle au centre de la raquette gauche : repart tout droit, plus vite
	d.Ball = Ball{face + BallR + 5, a.Y, -400, 0}
	d.step(1.0 / 60)
	if d.Ball.VX <= 0 || math.Abs(d.Ball.VY) > 1e-9 || math.Abs(d.Ball.VX-400*BallAccel) > 1e-6 || d.Rally != 1 {
		t.Fatalf("renvoi droit à %v attendu, obtenu %+v (rally %d)", 400*BallAccel, d.Ball, d.Rally)
	}
	// balle sur le bas de la raquette : repart vers le bas
	d.Ball = Ball{face + BallR + 5, a.Y + PadH/2, -400, 0}
	d.step(1.0 / 60)
	if d.Ball.VX <= 0 || d.Ball.VY <= 0 {
		t.Fatalf("renvoi vers le bas attendu, obtenu %+v", d.Ball)
	}
}

func TestDuelNoTunneling(t *testing.T) {
	d, _, b := duel(t)
	right := DW - PadX - PadW
	// à vitesse max, la balle avance de ~16 px par image : elle doit quand même être renvoyée
	d.Ball = Ball{right - BallR - 3, b.Y, BallMax, 0}
	d.step(1.0 / 60)
	if d.Ball.VX >= 0 {
		t.Fatalf("la balle a traversé la raquette : %+v", d.Ball)
	}
}

func TestDuelScoreAndWin(t *testing.T) {
	d, a, b := duel(t)
	b.Y = PadH / 2 // raquette droite tout en haut, la balle passe en bas
	for i := 1; i <= WinScore; i++ {
		d.Ball = Ball{DW - 40, DH - 40, 600, 0}
		for d.Phase == "play" {
			d.step(1.0 / 60)
		}
		if a.Score != i {
			t.Fatalf("point %d attendu pour A, score %d", i, a.Score)
		}
		if i < WinScore {
			if d.Phase != "point" || d.serveTo != 1 {
				t.Fatalf("pause puis service vers B attendus, obtenu %s / %d", d.Phase, d.serveTo)
			}
			for d.Phase != "play" {
				d.step(0.1)
			}
			if d.Ball.VX <= 0 {
				t.Fatal("le service doit partir vers celui qui a perdu le point")
			}
		}
	}
	if d.Phase != "end" || d.Winner != "A" {
		t.Fatalf("victoire de A attendue, obtenu %s / %q", d.Phase, d.Winner)
	}
	for d.Phase == "end" {
		d.step(0.1)
	}
	if a.Score != 0 || b.Score != 0 {
		t.Fatal("revanche : scores remis à zéro")
	}
}

func TestDuelBot(t *testing.T) {
	d := newDuel("0000")
	h := d.join("Humain", false)
	bot := d.join("Bot", true)
	if d.join("Bot", true) != nil {
		t.Fatal("pas de 3e joueur")
	}
	for d.Phase != "play" {
		d.step(0.1)
	}
	// le bot (à droite) doit renvoyer une balle lente qui arrive droit sur lui, même partant de loin
	bot.Y, bot.aim = 100, 0
	d.Ball = Ball{500, 350, 300, 0}
	for range 120 {
		d.step(1.0 / 60)
	}
	if d.Ball.VX >= 0 || d.Rally != 1 {
		t.Fatalf("le bot aurait dû renvoyer la balle : %+v", d.Ball)
	}
	// un humain qui rejoint remplace le bot ; un 3e humain est refusé
	h2 := d.join("Ami", false)
	if h2 == nil || d.Pads[1] != h2 || d.Phase != "wait" {
		t.Fatal("un humain doit prendre la place du bot")
	}
	if d.join("Encore", false) != nil {
		t.Fatal("partie pleine attendue")
	}
	d.leave(h2.ID)
	d.join("Bot", true)
	d.leave(h.ID)
	if d.Pads[0] != nil || d.Pads[1] != nil {
		t.Fatal("le bot doit partir avec le dernier humain")
	}
}

func TestDuelRooms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(serveDuel))
	defer srv.Close()
	ctx := context.Background()
	dial := func(q string) (*websocket.Conn, error) {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		return c, wsjson.Read(ctx, c, &m)
	}
	pinOf := func(c *websocket.Conn) string {
		var m map[string]any
		for m["pin"] == nil {
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
		}
		return m["pin"].(string)
	}
	a, _ := dial("name=A")
	pin := pinOf(a)
	b, err := dial("name=B&pin=" + pin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dial("name=C&pin=" + pin)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation || !strings.Contains(err.Error(), "pleine") {
		t.Fatalf("3e joueur refusé attendu, obtenu %v", err)
	}
	a.CloseNow()
	b.CloseNow()
	time.Sleep(200 * time.Millisecond)
	duelsMu.Lock()
	_, still := duels[pin]
	duelsMu.Unlock()
	if still {
		t.Fatal("un salon vide doit être supprimé")
	}
}
