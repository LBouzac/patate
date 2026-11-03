package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestPartyHostAndPick(t *testing.T) {
	pt := newParty("1234")
	a, b, c := pt.join("Ana"), pt.join("Bob"), pt.join("Cy")
	if !a.Host || b.Host || c.Host || a.Color == b.Color {
		t.Fatalf("le premier arrivé est le chef, une couleur chacun : %+v %+v", a, b)
	}
	pt.input(b, []byte(`{"play":"quiz"}`))
	if pt.game != nil {
		t.Fatal("seul le chef choisit le jeu")
	}
	pt.input(a, []byte(`{"play":"inconnu"}`))
	if pt.game != nil {
		t.Fatal("jeu inconnu : rien ne se lance")
	}
	pt.input(a, []byte(`{"play":"imposteur"}`))
	if pt.kind == nil || pt.kind.ID != "imposteur" {
		t.Fatal("le chef lance l'Imposteur")
	}
	pt.leave(a.ID)
	if !b.Host {
		t.Fatal("le chef part : le suivant devient chef")
	}
	pt.input(b, []byte(`{"menu":true}`))
	if pt.game != nil || pt.kind != nil {
		t.Fatal("retour au menu")
	}
	pt.leave(c.ID)
	pt.input(b, []byte(`{"play":"chaine"}`))
	if pt.game != nil {
		t.Fatal("Dessin en chaîne demande 3 joueurs")
	}
	pt.input(b, []byte(`{"play":"sumo"}`))
	if pt.kind.ID != "sumo" {
		t.Fatal("Sumo se joue même seul (avec des bots)")
	}
	pt.input(b, []byte(`{"again":true}`))
	if pt.kind.ID != "sumo" {
		t.Fatal("rejouer relance le même jeu")
	}
	for i := 0; i < TVMax-1; i++ {
		pt.join("X")
	}
	if pt.join("Trop") != nil {
		t.Fatal("10 joueurs au plus")
	}
}

// Les messages des jeux traversent le salon intacts (le quiz utilise "pick", comme… le choix du jeu autrefois).
func TestPartyForwardsGameInput(t *testing.T) {
	pt := newParty("1234")
	a := pt.join("Ana")
	pt.input(a, []byte(`{"play":"quiz"}`))
	q := pt.game.(*Quiz)
	q.Step(QuizIntro)
	pt.input(a, []byte(fmt.Sprintf(`{"pick":%d}`, q.good)))
	if q.Phase != "reveal" || q.P[0].Score == 0 {
		t.Fatalf("la réponse du quiz doit arriver au jeu : phase %s, score %d", q.Phase, q.P[0].Score)
	}
}

// Rien ne change : rien n'est renvoyé, sauf un signe de vie de temps en temps. Client saturé : on réessaie.
func TestDeliverOnlyChanges(t *testing.T) {
	o := &out{send: make(chan []byte, 1)}
	t0 := time.Now()
	o.deliver([]byte("a"), t0)
	if string(<-o.send) != "a" {
		t.Fatal("premier envoi")
	}
	o.deliver([]byte("a"), t0.Add(100*time.Millisecond))
	if len(o.send) != 0 {
		t.Fatal("rien n'a changé : rien à envoyer")
	}
	o.deliver([]byte("a"), t0.Add(TVBeat+time.Millisecond))
	if string(<-o.send) != string(beat) {
		t.Fatal("signe de vie attendu")
	}
	o.send <- []byte("plein")
	o.deliver([]byte("b"), t0.Add(2*TVBeat))
	<-o.send
	o.deliver([]byte("b"), t0.Add(2*TVBeat+time.Millisecond))
	if string(<-o.send) != "b" {
		t.Fatal("un message perdu (client saturé) doit repartir")
	}
}

// Un dessin à deviner part à part, avant la vue, et une seule fois.
func TestBigPartSentOnce(t *testing.T) {
	pt := newParty("1234")
	var ps []*Pad
	for _, n := range []string{"A", "B", "C"} {
		ps = append(ps, pt.join(n))
	}
	pt.input(ps[0], []byte(`{"play":"chaine"}`))
	g := pt.game.(*Chain)
	for _, p := range ps {
		g.Input(p, []byte(`{"text":"un chat"}`))
	}
	for _, p := range ps {
		g.Input(p, []byte(`{"draw":[[0,2,10,10,500,500]]}`))
	}
	if g.Phase != "guess" {
		t.Fatalf("phase %s", g.Phase)
	}
	drain := func(p *Pad) (msgs []string) {
		for len(p.send) > 0 {
			msgs = append(msgs, string(<-p.send))
		}
		return
	}
	for _, p := range ps {
		drain(p)
	}
	now := time.Now()
	pt.flush(now)
	m := drain(ps[0])
	if len(m) != 2 || !strings.HasPrefix(m[0], `{"big":[[0,2,10,10,500,500]]`) || !strings.Contains(m[1], `"phase":"guess"`) {
		t.Fatalf("dessin puis vue attendus, obtenu %v", m)
	}
	g.Left -= 1.5 // le chrono change : la vue repart, pas le dessin
	pt.flush(now.Add(time.Second))
	if m := drain(ps[0]); len(m) != 1 || strings.Contains(m[0], "big") {
		t.Fatalf("seule la vue doit repartir, obtenu %v", m)
	}
}

func TestTeleOverWebSocket(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/tele", serveScreen)
	mux.HandleFunc("/ws/manette", servePad)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(path string) *websocket.Conn {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		c.SetReadLimit(1 << 20)
		return c
	}
	// lit jusqu'à un message qui vérifie ok
	until := func(c *websocket.Conn, ok func(map[string]any) bool) map[string]any {
		for {
			var m map[string]any
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
			if ok(m) {
				return m
			}
		}
	}
	tv := dial("/ws/tele?name=TV&pin=")
	defer tv.CloseNow()
	m := until(tv, func(m map[string]any) bool { return m["pin"] != nil })
	pin := m["pin"].(string)
	if len(pin) != 4 {
		t.Fatalf("code à 4 chiffres attendu : %v", m)
	}
	if c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/ws/manette?name=Z&pin=0000", nil); err == nil {
		var m map[string]any
		if err := wsjson.Read(ctx, c, &m); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
			t.Fatal("code inconnu : refus attendu")
		}
	}
	ana := dial("/ws/manette?name=Ana&pin=" + pin)
	defer ana.CloseNow()
	until(ana, func(m map[string]any) bool { return m["you"] != nil })
	until(tv, func(m map[string]any) bool { ps, _ := m["players"].([]any); return len(ps) == 1 })
	wsjson.Write(ctx, ana, map[string]string{"play": "quiz"})
	until(tv, func(m map[string]any) bool { return m["game"] == "quiz" })
	v := until(ana, func(m map[string]any) bool { return m["game"] == "quiz" })
	if me := v["me"].(map[string]any); me["host"] != true || me["name"] != "Ana" {
		t.Fatalf("la manette sait qui elle est : %v", v)
	}

	// la télé se recharge (lien ?pin=) : elle retrouve son salon
	tv.Close(websocket.StatusNormalClosure, "")
	tv2 := dial("/ws/tele?name=TV&pin=" + pin)
	defer tv2.CloseNow()
	until(tv2, func(m map[string]any) bool { return m["pin"] == pin && m["game"] == "quiz" })

	ana.Close(websocket.StatusNormalClosure, "")
	tv2.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		partiesMu.Lock()
		n := len(parties)
		partiesMu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("salon vide : il doit fermer")
}

// Chaque jeu tourne sans paniquer quand les joueurs arrivent et partent n'importe quand.
func TestTVGamesSurviveChurn(t *testing.T) {
	for _, k := range tvGames {
		pt := newParty("1")
		var ps []*Pad
		for i := 0; i < 4; i++ {
			ps = append(ps, pt.join("J"))
		}
		pt.input(ps[0], []byte(`{"play":"`+k.ID+`"}`))
		if pt.game == nil {
			t.Fatalf("%s ne démarre pas", k.ID)
		}
		msgs := []string{`{"x":1,"y":0}`, `{"dash":true}`, `{"bot":1}`, `{"pick":0}`, `{"pick":2}`, `{"seen":true}`,
			`{"done":true}`, `{"vote":2}`, `{"guess":"Chat"}`, `{"next":true}`, `{"text":"bonjour"}`, `{"draw":[[1,1,0,0,9,9]]}`, `nul`}
		for step := 0; step < 3000; step++ {
			p := ps[step%len(ps)]
			pt.input(p, []byte(msgs[step%len(msgs)]))
			pt.game.Step(TVTick.Seconds())
			if step%7 == 0 {
				pt.flush(time.Now())
				for _, p := range ps {
					for len(p.send) > 0 {
						b := <-p.send
						if !json.Valid(b) {
							t.Fatalf("%s : message invalide %s", k.ID, b)
						}
					}
				}
			}
			if step == 1000 {
				pt.leave(ps[1].ID)
				ps = append(ps[:1], ps[2:]...)
			}
			if step == 2000 {
				ps = append(ps, pt.join("Nouveau"))
			}
		}
	}
}
