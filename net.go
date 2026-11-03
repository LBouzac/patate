package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/coder/websocket"
)

// Seat : la place d'un joueur dans un salon, quel que soit le jeu.
type Seat struct {
	ID    int
	Send  chan []byte      // états à envoyer au client
	OnMsg func(msg []byte) // chaque message reçu du client
	Leave func()           // à la déconnexion
}

// MaxSessions : connexions simultanées, tous jeux confondus. Au-delà, le petit conteneur manquerait de mémoire.
const MaxSessions = 300

var sessions atomic.Int32

// secure ajoute les en-têtes de sécurité à toutes les réponses.
// unsafe-inline : les pages ont leur script dans le HTML ; tout le reste (scripts externes, cadres, autres sites) est bloqué.
func secure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		hd.Set("X-Content-Type-Options", "nosniff")
		hd.Set("Referrer-Policy", "no-referrer")
		hd.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.ServeHTTP(w, r)
	})
}

// session gère une connexion WebSocket commune à tous les jeux : pseudo nettoyé, join,
// envoi de {"you": id}, puis relais serveur → client et client → OnMsg.
// join renvoie la place, ou nil et la raison du refus (affichée au joueur).
func session(w http.ResponseWriter, r *http.Request, readLimit int64, join func(name string, q url.Values) (*Seat, string)) {
	if sessions.Add(1) > MaxSessions {
		sessions.Add(-1)
		http.Error(w, "Serveur plein, réessaie plus tard", http.StatusServiceUnavailable)
		return
	}
	defer sessions.Add(-1)
	q := r.URL.Query()
	name := strings.TrimSpace(q.Get("name"))
	if n := []rune(name); len(n) > 12 {
		name = string(n[:12])
	}
	if name == "" {
		name = "Joueur"
	}
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(readLimit)

	s, why := join(name, q)
	if s == nil {
		c.Close(websocket.StatusPolicyViolation, why)
		return
	}
	defer s.Leave()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	hello, _ := json.Marshal(map[string]int{"you": s.ID})
	if c.Write(ctx, websocket.MessageText, hello) != nil {
		return
	}
	go func() {
		defer cancel()
		for {
			select {
			case b := <-s.Send:
				if c.Write(ctx, websocket.MessageText, b) != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		_, b, err := c.Read(ctx)
		if err != nil {
			return
		}
		s.OnMsg(b)
	}
}

// newPin tire un code à 4 chiffres qui n'est pas déjà pris.
// shortcut: 4 chiffres sans limite d'essais, devinable en force brute ; passer à 6 chiffres + limite si le jeu devient public
func newPin(taken func(string) bool) string {
	for {
		if pin := fmt.Sprintf("%04d", rand.IntN(10000)); !taken(pin) {
			return pin
		}
	}
}
