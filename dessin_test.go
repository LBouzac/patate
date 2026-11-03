package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const jpeg = "data:image/jpeg;base64,AAAA"

// sketchRound : 3 joueurs, partie lancée, photo envoyée par le 1er ; renvoie la partie, l'auteur, le dessinateur, celui qui devine.
func sketchRound(t *testing.T) (*Sketch, *Sketcher, *Sketcher, *Sketcher) {
	s := newSketch("0000")
	a, b, c := s.join("A"), s.join("B"), s.join("C")
	s.start()
	if s.Phase != "photo" || s.Owner != a.ID {
		t.Fatalf("A doit envoyer la 1re photo, obtenu %s / %d", s.Phase, s.Owner)
	}
	s.setPhoto(b, jpeg) // pas son tour : ignoré
	s.setPhoto(a, "data:text/html,<script>")
	if s.Phase != "photo" {
		t.Fatal("photo refusée attendue")
	}
	s.setPhoto(a, jpeg)
	if s.Phase != "draw" || s.Drawer == a.ID || s.Drawer == 0 {
		t.Fatalf("dessinateur tiré parmi les autres attendu, obtenu %s / %d", s.Phase, s.Drawer)
	}
	drawer, guesser := b, c
	if s.Drawer == c.ID {
		drawer, guesser = c, b
	}
	return s, a, drawer, guesser
}

func TestSketchScoring(t *testing.T) {
	s, owner, drawer, guesser := sketchRound(t)
	s.guess(drawer, "triche")
	s.guess(owner, "triche")
	s.step(DrawTime / 2)
	s.guess(guesser, "un chat")
	if len(s.Guesses) != 1 || s.Guesses[0].Pts != 55 {
		t.Fatalf("une seule proposition à 55 pts attendue (mi-temps), obtenu %+v", s.Guesses)
	}
	s.step(5) // le juge met du temps : les points ne baissent pas
	s.judge(guesser, s.Guesses[0].ID, true)
	if s.Guesses[0].OK != nil {
		t.Fatal("seul l'auteur de la photo juge")
	}
	s.judge(owner, s.Guesses[0].ID, true)
	if guesser.Score != 55 || drawer.Score != DrawerBonus {
		t.Fatalf("55 pour C et %d pour le dessinateur attendus, obtenu %d / %d", DrawerBonus, guesser.Score, drawer.Score)
	}
	if s.Phase != "reveal" {
		t.Fatalf("tout le monde a trouvé : révélation attendue, obtenu %s", s.Phase)
	}
}

func TestSketchJudgeAfterTimer(t *testing.T) {
	s, owner, _, guesser := sketchRound(t)
	s.guess(guesser, "chien")
	s.guess(guesser, "chat")
	s.step(DrawTime)
	if s.Phase != "judge" {
		t.Fatalf("temps de jugement attendu, obtenu %s", s.Phase)
	}
	s.guess(guesser, "trop tard")
	if len(s.Guesses) != 2 || s.Guesses[0].Pts != 100 {
		t.Fatalf("2 propositions à 100 pts attendues, obtenu %+v", s.Guesses)
	}
	s.judge(owner, s.Guesses[0].ID, false)
	if s.Phase != "judge" {
		t.Fatal("encore une proposition à juger")
	}
	s.judge(owner, s.Guesses[1].ID, true)
	if s.Phase != "reveal" || guesser.Score != 100 {
		t.Fatalf("révélation et 100 pts attendus, obtenu %s / %d", s.Phase, guesser.Score)
	}
}

func TestSketchHiddenAnswers(t *testing.T) {
	s, owner, _, guesser := sketchRound(t)
	other := s.join("D")
	s.guess(guesser, "chat")
	s.guess(other, "chien")
	s.judge(owner, s.Guesses[1].ID, false)
	texts := func(p *Sketcher) (out []string) {
		var m struct{ Guesses []Guess }
		json.Unmarshal(s.stateFor(p), &m)
		for _, g := range m.Guesses {
			out = append(out, g.Text)
		}
		return
	}
	if got := texts(other); got[0] != "" || got[1] != "chien" {
		t.Fatalf("D ne doit voir que sa proposition (fausse), obtenu %q", got)
	}
	if got := texts(owner); got[0] != "chat" {
		t.Fatalf("l'auteur de la photo voit tout, obtenu %q", got)
	}
}

func TestSketchTurnsAndLeave(t *testing.T) {
	s, _, drawer, _ := sketchRound(t)
	s.leave(drawer.ID) // dessinateur parti : manche annulée, mais il reste 2 joueurs → attente
	if s.Phase != "wait" {
		t.Fatalf("moins de 3 joueurs : attente attendue, obtenu %s", s.Phase)
	}
	s.join("E")
	s.join("F")
	s.start()
	turns := 0
	for s.Phase == "photo" {
		turns++
		s.setPhoto(s.get(s.Owner), jpeg)
		s.step(DrawTime)
		s.step(RevealTime)
	}
	if s.Phase != "end" || turns != 4 {
		t.Fatalf("4 tours puis fin attendus, obtenu %d tours / %s", turns, s.Phase)
	}
}

func TestSketchLines(t *testing.T) {
	s, _, drawer, guesser := sketchRound(t)
	s.addLine(guesser, []int{1, 2, 3, 4})
	s.addLine(drawer, []int{-5, 2, 3000, 4000})
	if len(s.lines) != 1 || s.lines[0][0] != 0 || s.lines[0][2] != CW || s.lines[0][3] != CH {
		t.Fatalf("un trait, borné au canevas, attendu : %v", s.lines)
	}
	late := s.join("Retard")
	if b := <-late.send; string(b) != `{"lines":[[0,2,1000,750]]}` {
		t.Fatalf("le dessin en cours doit être envoyé au nouveau venu, obtenu %s", b)
	}
}

// Une vraie photo dépasse largement les 256 octets des autres jeux : elle doit passer par le WebSocket.
func TestSketchPhotoOverWebSocket(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(serveSketch))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // échouer plutôt que bloquer
	defer cancel()
	dial := func(q string) *websocket.Conn {
		c, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/?"+q, nil)
		if err != nil {
			t.Fatal(err)
		}
		c.SetReadLimit(-1) // comme un navigateur
		return c
	}
	read := func(c *websocket.Conn, until func(map[string]any) bool) {
		for {
			var m map[string]any
			if err := wsjson.Read(ctx, c, &m); err != nil {
				t.Fatal(err)
			}
			if until(m) {
				return
			}
		}
	}
	a := dial("name=A")
	defer a.CloseNow()
	var pin string
	read(a, func(m map[string]any) bool { p, ok := m["pin"].(string); pin = p; return ok })
	b, c := dial("name=B&pin="+pin), dial("name=C&pin="+pin)
	defer b.CloseNow()
	defer c.CloseNow()
	wsjson.Write(ctx, a, map[string]bool{"start": true})
	read(a, func(m map[string]any) bool { return m["phase"] == "photo" })
	big := jpeg + strings.Repeat("A", 100_000)
	wsjson.Write(ctx, a, map[string]string{"photo": big})
	read(a, func(m map[string]any) bool { return m["photo"] == big })
}
