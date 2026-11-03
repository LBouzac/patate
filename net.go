package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
)

// Seat : la place d'un joueur dans un salon, quel que soit le jeu.
type Seat struct {
	ID    int
	Send  chan []byte      // états à envoyer au client
	OnMsg func(msg []byte) // chaque message reçu du client
	Leave func()           // à la déconnexion
}

// session gère une connexion WebSocket commune à tous les jeux : pseudo nettoyé, join,
// envoi de {"you": id}, puis relais serveur → client et client → OnMsg.
// join renvoie la place, ou nil et la raison du refus (affichée au joueur).
func session(w http.ResponseWriter, r *http.Request, join func(name string, q url.Values) (*Seat, string)) {
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
	c.SetReadLimit(256)

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
