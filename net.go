package main

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// Seat : la place d'un joueur dans un salon, quel que soit le jeu.
type Seat struct {
	ID     int
	Send   chan []byte      // états à envoyer au client
	OnMsg  func(msg []byte) // chaque message reçu du client
	Leave  func()           // quand le joueur quitte pour de bon
	Resume func()           // facultatif : à la reconnexion (renvoyer ce qui a pu être perdu)
	Room   string           // code du salon : le chat relie les places du même salon
	// facultatif : le chat est-il ouvert en ce moment ? (Dessine le coupe pendant le dessin)
	CanChat  func() bool
	name     string    // pseudo, pour signer les messages du chat
	group    string    // groupe du salon, gardé pour la reconnexion
	lastChat time.Time // anti-spam
}

// ---------- Chat : relayé en direct aux joueurs du même salon, jamais enregistré ----------

const (
	ChatMax   = 200 // caractères par message
	ChatEvery = 500 * time.Millisecond
)

var (
	chatMu  sync.Mutex
	chats   = map[string]map[*Seat]bool{} // jeu + code du salon → places connectées
	groupOf = map[string]string{}         // jeu + code du salon → groupe
	latest  = map[string]string{}         // groupe + jeu → dernier salon du groupe dans ce jeu
)

// Groupe : les joueurs d'un salon, qui changent de jeu ensemble sans se ré-inviter. Un salon reçoit son groupe à
// sa création : celui des joueurs qui arrivent d'un autre jeu, sinon un nouveau.
var groupJoin sync.Mutex // une arrivée groupée à la fois : deux joueurs du groupe ne créent pas chacun leur salon

// gamePages : les pages vers lesquelles un groupe peut partir (le salon et les jeux).
var gamePages = map[string]bool{"/salon.html": true, "/patate.html": true, "/ricochet.html": true, "/golf.html": true, "/dessin.html": true,
	"/foot.html": true, "/petanque.html": true, "/billard.html": true, "/artillerie.html": true, "/motos.html": true, "/lombrics.html": true, "/barrage.html": true}

// chatJoin inscrit la place au salon et rend le groupe du salon (group, ou un nouveau si vide, pour un salon neuf).
func chatJoin(key string, s *Seat, group string) string {
	chatMu.Lock()
	defer chatMu.Unlock()
	if chats[key] == nil {
		chats[key] = map[*Seat]bool{}
	}
	chats[key][s] = true
	if groupOf[key] == "" {
		if group == "" {
			group = newToken()
		}
		groupOf[key] = group
	}
	return groupOf[key]
}

func chatLeave(key string, s *Seat) {
	chatMu.Lock()
	defer chatMu.Unlock()
	delete(chats[key], s)
	if len(chats[key]) == 0 {
		delete(chats, key)
		path, _, _ := strings.Cut(key, "|")
		if gk := groupOf[key] + "|" + path; latest[gk] == key {
			delete(latest, gk)
		}
		delete(groupOf, key)
	}
}

// groupPin : le code du salon où retrouver son groupe dans ce jeu, "" s'il n'y en a plus.
func groupPin(group, path string) string {
	chatMu.Lock()
	defer chatMu.Unlock()
	key := latest[group+"|"+path]
	if key == "" || len(chats[key]) == 0 || groupOf[key] != group {
		return ""
	}
	return strings.TrimPrefix(key, path+"|")
}

// groupGo envoie tout le salon vers un autre jeu.
// shortcut: un client saturé rate l'ordre et reste sur place ; il peut toujours suivre avec le code
func groupGo(key, page string) {
	chatMu.Lock()
	defer chatMu.Unlock()
	b, _ := json.Marshal(map[string]string{"go": page, "group": groupOf[key]})
	for s := range chats[key] {
		trySend(s, b)
	}
}

// chatSay relaie un message à tout le salon (l'auteur compris). Rien n'est gardé.
func chatSay(key string, from *Seat, text string) {
	text = strings.Join(strings.Fields(text), " ") // pas de retours à la ligne ni d'espaces en rafale
	if r := []rune(text); len(r) > ChatMax {
		text = string(r[:ChatMax])
	}
	chatMu.Lock()
	defer chatMu.Unlock()
	if text == "" || time.Since(from.lastChat) < ChatEvery {
		return
	}
	from.lastChat = time.Now()
	if from.CanChat != nil && !from.CanChat() {
		b, _ := json.Marshal(map[string]any{"chat": map[string]string{"text": "Chat en pause pendant le dessin"}})
		trySend(from, b)
		return
	}
	b, _ := json.Marshal(map[string]any{"chat": map[string]any{"from": from.name, "id": from.ID, "text": text}})
	for s := range chats[key] {
		trySend(s, b)
	}
}

func trySend(s *Seat, b []byte) {
	select {
	case s.Send <- b:
	default: // client saturé : le message est perdu (le chat n'est pas garanti)
	}
}

// Reconnexion : quand la connexion lâche sans que le joueur ait quitté (réseau, écran verrouillé, 4G…),
// sa place reste réservée Grace secondes. Il la retrouve en revenant avec le jeton secret reçu à l'arrivée.
var Grace = 20 * time.Second // variable pour les tests

type held struct {
	seat  *Seat
	path  string
	timer *time.Timer
}

// live : une connexion en cours. Une reconnexion peut arriver avant que le serveur ait vu mourir l'ancienne
// connexion (téléphone qui se réveille) : la nouvelle prend alors la place, l'ancienne est coupée.
type live struct {
	seat *Seat
	path string
	kick context.CancelFunc
	over bool // place passée à une nouvelle connexion
}

var (
	heldMu  sync.Mutex
	helds   = map[string]*held{} // jeton → place réservée
	actives = map[string]*live{} // jeton → connexion en cours
)

func newToken() string {
	b := make([]byte, 16)
	crand.Read(b) // jeton imprévisible : il donne accès à la place d'un joueur
	return hex.EncodeToString(b)
}

// reclaim rend la place de ce jeton (et de ce jeu) : réservée, ou encore tenue par une vieille connexion qu'on coupe.
// nil si elle a expiré.
func reclaim(token, path string) *Seat {
	heldMu.Lock()
	defer heldMu.Unlock()
	if h := helds[token]; h != nil && h.path == path {
		delete(helds, token)
		h.timer.Stop()
		return h.seat
	}
	if lv := actives[token]; lv != nil && lv.path == path {
		lv.over = true
		delete(actives, token)
		lv.kick()
		return lv.seat
	}
	return nil
}

// release : la connexion se termine. Départ volontaire : le joueur quitte ; coupure : sa place est réservée.
func release(token string, lv *live, voluntary bool) {
	heldMu.Lock()
	if actives[token] == lv {
		delete(actives, token)
	}
	if lv.over { // place déjà reprise par une nouvelle connexion
		heldMu.Unlock()
		return
	}
	if !voluntary {
		holdLocked(token, lv.path, lv.seat)
		heldMu.Unlock()
		return
	}
	heldMu.Unlock()
	lv.seat.Leave()
}

// holdLocked réserve la place (heldMu tenu) ; sans retour avant Grace, le joueur quitte la partie.
func holdLocked(token, path string, s *Seat) {
	h := &held{seat: s, path: path}
	h.timer = time.AfterFunc(Grace, func() {
		heldMu.Lock()
		mine := helds[token] == h // pas repris entre-temps
		if mine {
			delete(helds, token)
		}
		heldMu.Unlock()
		if mine {
			s.Leave()
		}
	})
	helds[token] = h
}

// assetVersion : empreinte de style.css, common.js et qr.js, différente à chaque version du jeu. Les pages pointent
// vers /style.css?v=… : après une mise à jour l'adresse change, donc Cloudflare et les navigateurs ne peuvent plus
// servir une vieille copie (elle cassait l'affichage des nouvelles pages).
var assetVersion = func() string {
	var all []byte
	for _, f := range versioned {
		b, _ := files.ReadFile(f[1:])
		all = append(all, b...)
	}
	return sha256hex(all)[:10]
}()

var versioned = []string{"/style.css", "/common.js", "/qr.js"}

// serveFiles sert les pages (jamais gardées en cache, liens versionnés) et les fichiers statiques.
func serveFiles() http.Handler {
	static := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if strings.HasSuffix(name, ".html") {
			b, err := files.ReadFile(name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			for _, f := range versioned {
				b = bytes.ReplaceAll(b, []byte(`"`+f+`"`), []byte(`"`+f+"?v="+assetVersion+`"`))
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(b)
			return
		}
		if r.URL.Query().Get("v") == assetVersion {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // l'adresse change à chaque version
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	})
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
	// compression : les états JSON pèsent 2 à 2,5 fois moins (mesuré). Sans « context takeover » : peu de mémoire par connexion.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionNoContextTakeover, CompressionThreshold: 128})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(max(readLimit, 1024)) // au moins de quoi écrire un message de chat

	token := q.Get("resume")
	group := q.Get("group")
	var s *Seat
	var chatKey string
	if token != "" { // reconnexion
		if s = reclaim(token, r.URL.Path); s == nil {
			c.Close(websocket.StatusPolicyViolation, "Connexion perdue trop longtemps : rejoins la partie avec le code")
			return
		}
		chatKey = r.URL.Path + "|" + s.Room
		group = chatJoin(chatKey, s, s.group)
	} else {
		var why string
		grouped := group != "" && q.Get("pin") == "" // arrivée d'un autre jeu avec son groupe
		if grouped {
			groupJoin.Lock()
			if pin := groupPin(group, r.URL.Path); pin != "" {
				q.Set("pin", pin)
				s, why = join(name, q)
			}
			if s == nil { // pas encore de salon du groupe ici, ou plein : on en crée un
				q.Del("pin")
				s, why = join(name, q)
			}
		} else {
			s, why = join(name, q)
		}
		if s != nil {
			chatKey = r.URL.Path + "|" + s.Room
			group = chatJoin(chatKey, s, group)
			if grouped {
				chatMu.Lock()
				latest[group+"|"+r.URL.Path] = chatKey
				chatMu.Unlock()
			}
		}
		if grouped {
			groupJoin.Unlock()
		}
		if s == nil {
			c.Close(websocket.StatusPolicyViolation, why)
			return
		}
		token = newToken()
		s.name, s.group = name, group
	}
	defer chatLeave(chatKey, s)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	lv := &live{seat: s, path: r.URL.Path, kick: cancel}
	heldMu.Lock()
	actives[token] = lv
	heldMu.Unlock()
	closeCode := websocket.StatusCode(-1) // -1 : coupure sans au revoir
	defer func() {
		// fermeture volontaire (Quitter, onglet fermé) : on part tout de suite ; sinon la place est réservée
		release(token, lv, closeCode == websocket.StatusNormalClosure || closeCode == websocket.StatusGoingAway)
	}()
	start := time.Now()
	var why2 atomic.Value // raison de la coupure (texte), côté envoi ou côté réception : la première gagne
	defer func() {        // une ligne par déconnexion : utile pour comprendre les coupures (journal du service)
		log.Printf("déconnexion %s %q après %s : %v", r.URL.Path, name, time.Since(start).Round(time.Second), why2.Load())
	}()
	hello, _ := json.Marshal(map[string]any{"you": s.ID, "token": token, "group": group})
	if err := c.Write(ctx, websocket.MessageText, hello); err != nil {
		why2.Store("bonjour : " + err.Error())
		return
	}
	if q.Get("resume") != "" && s.Resume != nil {
		s.Resume()
	}
	go func() {
		defer cancel()
		for {
			select {
			case b := <-s.Send:
				if err := c.Write(ctx, websocket.MessageText, b); err != nil {
					why2.CompareAndSwap(nil, "envoi : "+err.Error())
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
			why2.CompareAndSwap(nil, "réception : "+err.Error())
			closeCode = websocket.CloseStatus(err)
			return
		}
		if bytes.HasPrefix(b, []byte(`{"chat":`)) { // message de chat : relayé, jamais transmis au jeu
			var m struct{ Chat string }
			if json.Unmarshal(b, &m) == nil {
				chatSay(chatKey, s, m.Chat)
			}
			continue
		}
		if bytes.HasPrefix(b, []byte(`{"go":`)) { // changement de jeu pour tout le salon
			var m struct{ Go string }
			if json.Unmarshal(b, &m) == nil && gamePages[m.Go] {
				groupGo(chatKey, m.Go)
			}
			continue
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
