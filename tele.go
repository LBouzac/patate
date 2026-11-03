// Mode télé : la télé (ou tout grand écran) affiche le jeu, les téléphones servent de manettes.
// On ouvre /tele.html sur la télé : un salon se crée avec un code et un QR code. Chacun le scanne avec son
// téléphone (/manette.html). Le premier arrivé est le chef : il choisit le jeu, puis tout le monde suit,
// d'un jeu à l'autre, sans rescanner.
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"
)

const (
	TVMax    = 10                      // une couleur par joueur
	TVTick   = time.Second / 30        // la physique de Sumo tourne à 30 images/s
	TVSlow   = 3                       // les jeux au tour par tour n'envoient qu'une image sur 3 (10/s)
	TVBeat   = 1500 * time.Millisecond // sans changement, un signe de vie (le navigateur se reconnecte après 4 s de silence)
	TVRooms  = 200
	ScreenID = 1000 // les écrans ont des numéros à part, après ceux des joueurs
)

// TVGame : un jeu du mode télé. Le salon appelle toutes ces méthodes en tenant son verrou.
type TVGame interface {
	Add(p *Pad)             // un joueur arrive (en cours de partie, il attend souvent la manche suivante)
	Remove(id int)          // un joueur part
	Input(p *Pad, b []byte) // un message de sa manette
	Step(dt float64)        // une image (TVTick)
	Screen() any            // ce que montre la télé : public
	View(p *Pad) any        // ce que montre le téléphone de ce joueur : peut contenir un secret
}

type tvKind struct {
	ID   string
	Min  int  // joueurs au minimum pour lancer
	Fast bool // temps réel : une image envoyée à chaque tick
	New  func(pads []*Pad) TVGame
}

var tvGames = []tvKind{
	{"sumo", 1, true, func(p []*Pad) TVGame { return newSumo(p) }},
	{"quiz", 1, false, func(p []*Pad) TVGame { return newQuiz(p) }},
	{"imposteur", 3, false, func(p []*Pad) TVGame { return newImpostor(p) }},
	{"chaine", 3, false, func(p []*Pad) TVGame { return newChain(p) }},
}

func tvKindOf(id string) *tvKind {
	for i := range tvGames {
		if tvGames[i].ID == id {
			return &tvGames[i]
		}
	}
	return nil
}

// out : un destinataire (télé ou téléphone). On n'envoie que ce qui a changé, plus un signe de vie de temps en temps.
type out struct {
	send chan []byte
	last []byte
	sent time.Time
}

var beat = []byte(`{"hb":1}`)

func (o *out) deliver(b []byte, now time.Time) {
	msg, same := b, bytes.Equal(b, o.last)
	if same {
		if now.Sub(o.sent) < TVBeat {
			return
		}
		msg = beat
	}
	select {
	case o.send <- msg:
		if !same {
			o.last = b
		}
		o.sent = now
	default: // client lent : on réessaie à la prochaine image, rien n'est perdu
	}
}

type Pad struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Host  bool   `json:"host"` // le chef choisit le jeu
	out
	lastBig []byte
}

// tvBig : une vue de téléphone avec une grosse partie (un dessin), envoyée à part et seulement quand elle change :
// le reste de la vue (le chrono) change chaque seconde, on ne renvoie pas le dessin à chaque fois.
type tvBig struct {
	V   any
	Big json.RawMessage
}

type Party struct {
	mu      sync.Mutex
	Pin     string
	pads    []*Pad // dans l'ordre d'arrivée : le premier est le chef
	screens map[int]*out
	nextID  int
	nextScr int
	kind    *tvKind // nil : on est au menu
	game    TVGame
	runs    int // numéro de partie : change à chaque lancement (même jeu relancé)
	ticks   int
}

func newParty(pin string) *Party { return &Party{Pin: pin, screens: map[int]*out{}, nextScr: ScreenID} }

func (pt *Party) join(name string) *Pad {
	if len(pt.pads) >= TVMax {
		return nil
	}
	used := map[string]bool{}
	for _, p := range pt.pads {
		used[p.Color] = true
	}
	pt.nextID++
	p := &Pad{ID: pt.nextID, Name: name, Color: colors[0], out: out{send: make(chan []byte, 16)}}
	for _, c := range colors {
		if !used[c] {
			p.Color = c
			break
		}
	}
	pt.pads = append(pt.pads, p)
	pt.crown()
	if pt.game != nil {
		pt.game.Add(p)
	}
	return p
}

func (pt *Party) leave(id int) {
	pt.pads = slices.DeleteFunc(pt.pads, func(p *Pad) bool { return p.ID == id })
	pt.crown()
	if pt.game != nil {
		pt.game.Remove(id)
	}
}

// crown : le plus ancien joueur est le chef.
func (pt *Party) crown() {
	for i, p := range pt.pads {
		p.Host = i == 0
	}
}

func (pt *Party) pad(id int) *Pad {
	for _, p := range pt.pads {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// start lance un jeu avec les joueurs présents (le chef seulement).
func (pt *Party) start(id string) {
	k := tvKindOf(id)
	if k == nil || len(pt.pads) < k.Min {
		return
	}
	pt.kind, pt.game = k, k.New(slices.Clone(pt.pads))
	pt.runs++
}

// input : messages du salon (choix du jeu, menu, rejouer), sinon transmis au jeu en cours.
// Les noms play, menu et again sont réservés au salon : aucun jeu ne doit les utiliser.
func (pt *Party) input(p *Pad, b []byte) {
	var m struct {
		Play  string `json:"play"`
		Menu  bool   `json:"menu"`
		Again bool   `json:"again"`
	}
	if json.Unmarshal(b, &m) == nil && (m.Play != "" || m.Menu || m.Again) {
		if !p.Host {
			return
		}
		switch {
		case m.Play != "":
			pt.start(m.Play)
		case m.Menu:
			pt.kind, pt.game = nil, nil
		case pt.kind != nil:
			pt.start(pt.kind.ID)
		}
		return
	}
	if pt.game != nil {
		pt.game.Input(p, b)
	}
}

type screenMsg struct {
	Pin     string `json:"pin"`
	Players []*Pad `json:"players"`
	Game    string `json:"game"`
	Run     int    `json:"run"`
	S       any    `json:"s"`
}

type padMsg struct {
	Pin     string `json:"pin"`
	Me      *Pad   `json:"me"`
	Players []*Pad `json:"players"`
	Game    string `json:"game"`
	Run     int    `json:"run"`
	V       any    `json:"v"`
}

// flush envoie à chaque écran et à chaque téléphone ce qu'il doit voir, s'il a changé.
func (pt *Party) flush(now time.Time) {
	game := ""
	if pt.kind != nil {
		game = pt.kind.ID
	}
	var s any
	if pt.game != nil {
		s = pt.game.Screen() // une fois par image : les jeux y vident leurs événements (sons)
	}
	players := pt.pads
	if players == nil {
		players = []*Pad{} // [] et pas null pour le navigateur
	}
	if len(pt.screens) > 0 {
		b, _ := json.Marshal(screenMsg{pt.Pin, players, game, pt.runs, s})
		for _, o := range pt.screens {
			o.deliver(b, now)
		}
	}
	for _, p := range pt.pads {
		var v any
		if pt.game != nil {
			v = pt.game.View(p)
		}
		if sp, ok := v.(tvBig); ok {
			v = sp.V
			bb, _ := json.Marshal(struct {
				Big json.RawMessage `json:"big"`
			}{sp.Big})
			if !bytes.Equal(bb, p.lastBig) {
				select {
				case p.send <- bb:
					p.lastBig = bb
				default:
					continue // la vue partira avec le dessin à la prochaine image, jamais avant lui
				}
			}
		}
		b, _ := json.Marshal(padMsg{pt.Pin, p, players, game, pt.runs, v})
		p.deliver(b, now)
	}
}

func (pt *Party) run() {
	tk := time.NewTicker(TVTick)
	defer tk.Stop()
	for now := range tk.C {
		if pt.closeIfEmpty() {
			return
		}
		pt.mu.Lock()
		if pt.game != nil {
			pt.game.Step(TVTick.Seconds())
		}
		pt.ticks++
		if pt.ticks%TVSlow == 0 || (pt.kind != nil && pt.kind.Fast) {
			pt.flush(now)
		}
		pt.mu.Unlock()
	}
}

var (
	partiesMu sync.Mutex
	parties   = map[string]*Party{}
)

// Le salon vit tant que quelqu'un y est : la télé peut se recharger, les téléphones restent.
func (pt *Party) closeIfEmpty() bool {
	partiesMu.Lock()
	defer partiesMu.Unlock()
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.pads) > 0 || len(pt.screens) > 0 {
		return false
	}
	delete(parties, pt.Pin)
	return true
}

// openScreen : la télé crée un salon, ou revient dans le sien (page rechargée, lien ?pin=).
func openScreen(pin string) (*Party, int, *out, string) {
	partiesMu.Lock()
	defer partiesMu.Unlock()
	pt := parties[pin]
	if pt == nil {
		if len(parties) >= TVRooms {
			return nil, 0, nil, "Trop de salons ouverts, réessaie plus tard"
		}
		pt = newParty(newPin(func(p string) bool { return parties[p] != nil }))
		parties[pt.Pin] = pt
		defer func() { go pt.run() }() // l'écran est déjà là quand la boucle démarre
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.screens) >= 4 {
		return nil, 0, nil, "Trop d'écrans sur ce salon"
	}
	pt.nextScr++
	o := &out{send: make(chan []byte, 16)}
	pt.screens[pt.nextScr] = o
	return pt, pt.nextScr, o, ""
}

func openPad(pin, name string) (*Party, *Pad, string) {
	partiesMu.Lock() // tenu jusqu'au bout : le salon ne peut pas fermer pendant qu'on y entre
	defer partiesMu.Unlock()
	pt := parties[pin]
	if pt == nil {
		return nil, nil, "Code inconnu : regarde le code affiché sur la télé"
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if p := pt.join(name); p != nil {
		return pt, p, ""
	}
	return nil, nil, "Salon plein (10 joueurs)"
}

func serveScreen(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(_ string, q url.Values) (*Seat, string) {
		pt, id, o, why := openScreen(q.Get("pin"))
		if pt == nil {
			return nil, why
		}
		return &Seat{
			Room:   pt.Pin,
			ID:     id,
			Send:   o.send,
			OnMsg:  func([]byte) {}, // la télé ne joue pas
			Leave:  func() { pt.mu.Lock(); delete(pt.screens, id); pt.mu.Unlock() },
			Resume: func() { pt.mu.Lock(); o.last = nil; pt.mu.Unlock() },
		}, ""
	})
}

func servePad(w http.ResponseWriter, r *http.Request) {
	session(w, r, 160<<10, func(name string, q url.Values) (*Seat, string) { // un dessin pèse jusqu'à ~100 Ko
		pt, p, why := openPad(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			Room:   pt.Pin,
			ID:     p.ID,
			Send:   p.send,
			OnMsg:  func(b []byte) { pt.mu.Lock(); pt.input(p, b); pt.mu.Unlock() },
			Leave:  func() { pt.mu.Lock(); pt.leave(p.ID); pt.mu.Unlock() },
			Resume: func() { pt.mu.Lock(); p.last, p.lastBig = nil, nil; pt.mu.Unlock() },
		}, ""
	})
}
