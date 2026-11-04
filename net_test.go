package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestResumeSeat(t *testing.T) {
	old := Grace
	Grace = 300 * time.Millisecond
	defer func() { Grace = old }()

	var leaves, resumes, joins atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
			id := int(joins.Add(1))
			return &Seat{ID: id, Send: make(chan []byte, 8), OnMsg: func([]byte) {},
				Leave: func() { leaves.Add(1) }, Resume: func() { resumes.Add(1) }}, ""
		})
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(q string) (*websocket.Conn, map[string]any, error) {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/jeu?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		return c, m, wsjson.Read(ctx, c, &m)
	}
	wait := func(d time.Duration) { time.Sleep(d) }

	c, hello, _ := dial("name=A")
	token, _ := hello["token"].(string)
	if len(token) != 32 || hello["you"] != 1.0 {
		t.Fatalf("jeton et identifiant attendus à l'arrivée, obtenu %v", hello)
	}
	c.CloseNow() // coupure réseau : pas d'au revoir
	wait(100 * time.Millisecond)
	if leaves.Load() != 0 {
		t.Fatal("coupure : la place doit rester réservée")
	}

	c, hello, _ = dial("resume=" + token)
	if hello["you"] != 1.0 || hello["token"] != token || resumes.Load() != 1 || joins.Load() != 1 {
		t.Fatalf("reconnexion : même place attendue, obtenu %v (reprises %d, arrivées %d)", hello, resumes.Load(), joins.Load())
	}
	c.CloseNow()
	wait(Grace + 200*time.Millisecond)
	if leaves.Load() != 1 {
		t.Fatalf("sans retour avant le délai, le joueur quitte : départs %d", leaves.Load())
	}
	_, _, err := dial("resume=" + token)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation || !strings.Contains(err.Error(), "trop longtemps") {
		t.Fatalf("place expirée : refus attendu, obtenu %v", err)
	}

	c, _, _ = dial("name=B")
	c.Close(websocket.StatusGoingAway, "") // départ volontaire (Quitter, onglet fermé)
	wait(100 * time.Millisecond)
	if leaves.Load() != 2 {
		t.Fatalf("départ volontaire : la place est libérée tout de suite, départs %d", leaves.Load())
	}
}

// Le téléphone se réveille : la nouvelle connexion arrive avant que le serveur ait vu mourir l'ancienne.
// Elle doit reprendre la place (et l'ancienne être coupée), sans que le joueur quitte la partie.
func TestResumeTakesOverLiveConnection(t *testing.T) {
	var leaves, joins atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
			return &Seat{ID: int(joins.Add(1)), Send: make(chan []byte, 8), OnMsg: func([]byte) {}, Leave: func() { leaves.Add(1) }}, ""
		})
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(q string) (*websocket.Conn, map[string]any) {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/jeu?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := wsjson.Read(ctx, c, &m); err != nil {
			t.Fatal(err)
		}
		return c, m
	}
	old, hello := dial("name=A")
	token := hello["token"].(string)
	c, again := dial("resume=" + token) // l'ancienne connexion est toujours ouverte
	if again["you"] != 1.0 {
		t.Fatalf("la nouvelle connexion doit reprendre la place 1, obtenu %v", again)
	}
	if _, _, err := old.Read(ctx); err == nil {
		t.Fatal("l'ancienne connexion doit être coupée")
	}
	time.Sleep(100 * time.Millisecond)
	if leaves.Load() != 0 {
		t.Fatal("reprise : le joueur ne doit pas quitter la partie")
	}
	c.Close(websocket.StatusGoingAway, "")
	time.Sleep(100 * time.Millisecond)
	if leaves.Load() != 1 {
		t.Fatalf("départ volontaire après reprise : départs %d", leaves.Load())
	}
}

// Dans un vrai jeu : un duel contre le bot continue après une coupure, sans repartir de zéro.
func TestResumeRealGame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(serveDuel))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(q string) (*websocket.Conn, map[string]any) {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/ws/ricochet?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := wsjson.Read(ctx, c, &m); err != nil {
			t.Fatal(err)
		}
		return c, m
	}
	state := func(c *websocket.Conn) map[string]any {
		var m map[string]any
		for m["phase"] == nil {
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
		}
		return m
	}
	c, hello := dial("name=A")
	wsjson.Write(ctx, c, map[string]int{"bot": 1})
	m := state(c)
	for m["phase"] == "wait" {
		m = state(c)
	}
	pin := m["pin"].(string)
	c.CloseNow() // coupure
	time.Sleep(150 * time.Millisecond)
	c, again := dial("resume=" + hello["token"].(string))
	m = state(c)
	if again["you"] != hello["you"] || m["phase"] == "wait" || m["pin"] != pin {
		t.Fatalf("le duel doit continuer avec le même joueur : %v / phase %v", again, m["phase"])
	}
	c.Close(websocket.StatusGoingAway, "")
}

// Chat : relayé à tout le salon (pas aux autres salons), tronqué, anti-spam ; jamais transmis au jeu.
func TestChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(serveDuel))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(q string) *websocket.Conn {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/ws/ricochet?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	// next lit jusqu'au prochain message de chat (en sautant les états du jeu)
	next := func(c *websocket.Conn) map[string]any {
		for {
			var m map[string]any
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
			if ch, ok := m["chat"].(map[string]any); ok {
				return ch
			}
		}
	}
	pinOf := func(c *websocket.Conn) string {
		for {
			var m map[string]any
			wsjson.Read(ctx, c, &m)
			if p, ok := m["pin"].(string); ok {
				return p
			}
		}
	}
	a := dial("name=A")
	pin := pinOf(a)
	b := dial("name=B&pin=" + pin)
	other := dial("name=Autre") // un autre salon
	defer a.CloseNow()
	defer b.CloseNow()
	defer other.CloseNow()

	long := strings.Repeat("é", 300)
	wsjson.Write(ctx, a, map[string]string{"chat": "  salut \n tout   le monde  "})
	if m := next(b); m["from"] != "A" || m["text"] != "salut tout le monde" {
		t.Fatalf("B doit recevoir le message de A, nettoyé : %v", m)
	}
	if m := next(a); m["text"] != "salut tout le monde" {
		t.Fatalf("l'auteur reçoit aussi son message : %v", m)
	}
	wsjson.Write(ctx, a, map[string]string{"chat": "spam"}) // trop tôt : ignoré
	time.Sleep(ChatEvery)
	wsjson.Write(ctx, a, map[string]string{"chat": long})
	if m := next(b); len([]rune(m["text"].(string))) != ChatMax {
		t.Fatalf("le spam est ignoré et un message trop long est coupé à %d caractères : %v", ChatMax, m)
	}
	ctxShort, cancelShort := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancelShort()
	for {
		var m map[string]any
		if wsjson.Read(ctxShort, other, &m) != nil {
			break // plus rien : l'autre salon n'a jamais reçu de chat
		}
		if m["chat"] != nil {
			t.Fatal("un autre salon ne doit pas recevoir le chat")
		}
	}
}

func TestChatPaused(t *testing.T) {
	s := &Seat{ID: 1, Send: make(chan []byte, 2), name: "A", CanChat: func() bool { return false }}
	chatJoin("test", s, "")
	defer chatLeave("test", s)
	chatSay("test", s, "la réponse c'est chat")
	if b := string(<-s.Send); !strings.Contains(b, "pause") || strings.Contains(b, "réponse") {
		t.Fatalf("chat en pause : seul l'auteur est prévenu, le message ne part pas : %s", b)
	}
}

// Changer de jeu : tout le salon reçoit l'ordre, puis se retrouve dans un même salon du nouveau jeu, sans code.
// Le salon est plein (Foot : 2) : le suivant du groupe en ouvre un autre.
func TestGroupSwitchGame(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", serveWS)
	mux.HandleFunc("/ws/foot", serveFoot)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(path, q string) (*websocket.Conn, map[string]any) {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+path+"?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var hello map[string]any
		if err := wsjson.Read(ctx, c, &hello); err != nil {
			t.Fatal(err)
		}
		return c, hello
	}
	until := func(c *websocket.Conn, key string) map[string]any { // saute les états jusqu'au message voulu
		for {
			var m map[string]any
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
			if m[key] != nil {
				return m
			}
		}
	}
	a, _ := dial("/ws", "name=A&pin=")
	pin := until(a, "pin")["pin"].(string)
	b, _ := dial("/ws", "name=B&pin="+pin)
	wsjson.Write(ctx, a, map[string]string{"go": "/admin.html"}) // pas un jeu : ignoré
	wsjson.Write(ctx, a, map[string]string{"go": "/foot.html"})
	ga, gb := until(a, "go"), until(b, "go")
	group, _ := ga["group"].(string)
	if ga["go"] != "/foot.html" || group == "" || gb["group"] != group {
		t.Fatalf("même ordre et même groupe attendus pour tout le salon : %v / %v", ga, gb)
	}
	a.Close(websocket.StatusNormalClosure, "")
	b.Close(websocket.StatusNormalClosure, "")

	q := "&pin=&group=" + url.QueryEscape(group)
	fa, ha := dial("/ws/foot", "name=A"+q)
	defer fa.CloseNow()
	fb, hb := dial("/ws/foot", "name=B"+q)
	defer fb.CloseNow()
	pa, pb := until(fa, "pin")["pin"], until(fb, "pin")["pin"]
	if pa != pb || ha["group"] != group || hb["group"] != group {
		t.Fatalf("le groupe doit se retrouver dans le même salon : %v / %v", pa, pb)
	}
	fc, _ := dial("/ws/foot", "name=C"+q)
	defer fc.CloseNow()
	if pc := until(fc, "pin")["pin"]; pc == pa {
		t.Fatal("salon plein : le suivant doit en ouvrir un autre")
	}
}
