package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Salon : seul le chef choisit le jeu ; tout le monde y part avec le groupe, puis revient au même salon.
func TestSalonFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/salon", serveSalon)
	mux.HandleFunc("/ws/foot", serveFoot)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(path, q string) *websocket.Conn {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+path+"?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		var hello map[string]any
		if err := wsjson.Read(ctx, c, &hello); err != nil {
			t.Fatal(err)
		}
		return c
	}
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
	members := func(n int) func(map[string]any) bool {
		return func(m map[string]any) bool { l, _ := m["members"].([]any); return len(l) == n }
	}
	hasGo := func(m map[string]any) bool { return m["go"] != nil }

	a := dial("/ws/salon", "name=A&pin=")
	pin := until(a, members(1))["pin"].(string)
	b := dial("/ws/salon", "name=B&pin="+pin)
	st := until(b, members(2))
	if chef := st["members"].([]any)[0].(map[string]any)["name"]; chef != "A" {
		t.Fatalf("le premier arrivé est le chef, obtenu %v", chef)
	}
	wsjson.Write(ctx, b, map[string]string{"pick": "/foot.html"}) // pas le chef : ignoré
	wsjson.Write(ctx, a, map[string]string{"pick": "/foot.html"})
	ga, gb := until(a, hasGo), until(b, hasGo)
	group, _ := ga["group"].(string)
	if ga["go"] != "/foot.html" || group == "" || gb["group"] != group {
		t.Fatalf("tout le salon part au foot avec le même groupe : %v / %v", ga, gb)
	}
	a.Close(websocket.StatusNormalClosure, "")
	b.Close(websocket.StatusNormalClosure, "")

	q := "&pin=&group=" + url.QueryEscape(group)
	fa := dial("/ws/foot", "name=A"+q)
	fb := dial("/ws/foot", "name=B"+q)
	wsjson.Write(ctx, fb, map[string]string{"go": "/salon.html"}) // en partie, n'importe qui ramène le groupe au salon
	if m := until(fa, hasGo); m["go"] != "/salon.html" || m["group"] != group {
		t.Fatalf("retour au salon attendu pour tous, obtenu %v", m)
	}
	fa.Close(websocket.StatusNormalClosure, "")
	fb.Close(websocket.StatusNormalClosure, "")

	sb := dial("/ws/salon", "name=B"+q)
	defer sb.CloseNow()
	sa := dial("/ws/salon", "name=A"+q)
	defer sa.CloseNow()
	st = until(sa, members(2))
	if chef := st["members"].([]any)[0].(map[string]any)["name"]; chef != "B" {
		t.Fatalf("même salon pour tout le groupe, premier revenu chef : %v", st)
	}
}
