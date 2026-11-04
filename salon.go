// Salon : on se retrouve avant de jouer, comme en mode télé. Le chef (le premier arrivé encore là) choisit le jeu,
// tout le monde y part ensemble ; en partie, le bouton Salon ramène tout le monde ici. Le groupe (net.go) relie le
// salon et les parties : personne n'a à se ré-inviter.
package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"sync"
)

const (
	LobbyMax   = 20 // personnes par salon
	LobbyRooms = 1000
)

type Member struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	send chan []byte
}

type Lobby struct {
	mu      sync.Mutex
	Pin     string    `json:"pin"`
	Members []*Member `json:"members"` // dans l'ordre d'arrivée : le premier est le chef
	nextID  int
}

// broadcast envoie l'état à tous (l.mu tenu).
func (l *Lobby) broadcast() {
	b, _ := json.Marshal(l)
	for _, m := range l.Members {
		select {
		case m.send <- b:
		default: // client saturé : il aura le prochain état
		}
	}
}

func (l *Lobby) join(name string) *Member {
	if len(l.Members) >= LobbyMax {
		return nil
	}
	l.nextID++
	m := &Member{ID: l.nextID, Name: name, send: make(chan []byte, 8)}
	l.Members = append(l.Members, m)
	l.broadcast()
	return m
}

var (
	lobbiesMu sync.Mutex
	lobbies   = map[string]*Lobby{}
)

func enterLobby(pin, name string) (*Lobby, *Member, string) {
	lobbiesMu.Lock()
	defer lobbiesMu.Unlock()
	if pin == "" {
		if len(lobbies) >= LobbyRooms {
			return nil, nil, "Trop de salons ouverts, réessaie plus tard"
		}
		l := &Lobby{Pin: newPin(func(p string) bool { return lobbies[p] != nil })}
		lobbies[l.Pin] = l
		return l, l.join(name), ""
	}
	l := lobbies[pin]
	if l == nil {
		return nil, nil, "Code inconnu"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if m := l.join(name); m != nil {
		return l, m, ""
	}
	return nil, nil, "Salon plein (20 personnes)"
}

func (l *Lobby) leave(id int) {
	lobbiesMu.Lock()
	defer lobbiesMu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Members = slices.DeleteFunc(l.Members, func(m *Member) bool { return m.ID == id })
	if len(l.Members) == 0 {
		delete(lobbies, l.Pin)
		return
	}
	l.broadcast() // le chef parti, le suivant prend la couronne
}

func serveSalon(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		l, m, why := enterLobby(q.Get("pin"), name)
		if m == nil {
			return nil, why
		}
		return &Seat{
			Room:   l.Pin,
			ID:     m.ID,
			Send:   m.send,
			Leave:  func() { l.leave(m.ID) },
			Resume: func() { l.mu.Lock(); l.broadcast(); l.mu.Unlock() },
			OnMsg: func(b []byte) {
				var msg struct {
					Pick string `json:"pick"` // page du jeu choisi par le chef
				}
				if json.Unmarshal(b, &msg) != nil || !gamePages[msg.Pick] || msg.Pick == "/salon.html" {
					return
				}
				l.mu.Lock()
				chef := len(l.Members) > 0 && l.Members[0].ID == m.ID
				l.mu.Unlock()
				if chef {
					groupGo(r.URL.Path+"|"+l.Pin, msg.Pick)
				}
			},
		}, ""
	})
}
