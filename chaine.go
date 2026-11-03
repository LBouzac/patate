// Dessin en chaîne (mode télé) : chacun écrit une phrase sur son téléphone. Le voisin la dessine, le suivant
// décrit ce dessin, le suivant dessine la description… Chaque chaîne passe par tout le monde (8 étapes au plus).
// À la fin, la télé révèle chaque chaîne maillon par maillon ; le chef avance avec son téléphone.
package main

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"strings"
)

const (
	ChainMin     = 3
	ChainSteps   = 8
	ChainWrite   = 45.0
	ChainDraw    = 80.0
	ChainGuess   = 40.0
	ChainLate    = 3.0  // après le chrono, les téléphones envoient ce qu'ils ont
	ChainShow    = 20.0 // révélation : on avance tout seul si le chef ne le fait pas
	ChainText    = 80   // caractères
	ChainStrokes = 600
	ChainNums    = 16000 // nombres au total dans un dessin (couleur, épaisseur, x, y…)
	DrawW, DrawH = 1000, 750
	DrawColors   = 8
)

type Artist struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Done  bool   `json:"done"`
	Gone  bool   `json:"gone"`
	idea  string
}

// Link : un maillon de chaîne, une phrase ou un dessin.
type Link struct {
	Kind  string          `json:"kind"` // text, draw
	By    string          `json:"by"`
	Color string          `json:"color"`
	Text  string          `json:"text,omitempty"`
	Draw  json.RawMessage `json:"draw,omitempty"` // [[couleur, épaisseur, x, y, x, y…], …]
	Skip  bool            `json:"skip,omitempty"` // personne ne l'a rempli
	done  bool
}

type Chain struct {
	Phase string    `json:"phase"` // write, draw, guess, reveal, done
	K     int       `json:"step"`  // étape en cours (0 : écrire)
	Steps int       `json:"steps"`
	Left  float64   `json:"left"`
	Late  bool      `json:"late"`
	P     []*Artist `json:"p"`
	RC    int       `json:"rc"` // révélation : chaîne montrée
	RS    int       `json:"rs"` // et nombre de maillons visibles
	Owner string    `json:"owner"`
	Show  []*Link   `json:"show"`
	links [][]*Link // links[c][k] : maillon k de la chaîne commencée par le joueur c
	auto  float64
}

func newChain(pads []*Pad) *Chain {
	g := &Chain{}
	ideas := rand.Perm(len(chainIdeas))
	for i, p := range pads {
		g.P = append(g.P, &Artist{ID: p.ID, Name: p.Name, Color: p.Color, idea: chainIdeas[ideas[i%len(ideas)]]})
	}
	g.Steps = min(len(g.P), ChainSteps)
	g.links = make([][]*Link, len(g.P))
	g.start(0)
	return g
}

func kindOf(step int) string {
	if step%2 == 1 {
		return "draw"
	}
	return "text"
}

// chainOf : à l'étape k, le joueur j s'occupe de la chaîne (j - k) : chaque chaîne passe une fois par chacun.
func (g *Chain) chainOf(j, k int) int { return ((j-k)%len(g.P) + len(g.P)) % len(g.P) }

func (g *Chain) start(k int) {
	g.K, g.Late = k, false
	switch {
	case k == 0:
		g.Phase, g.Left = "write", ChainWrite
	case kindOf(k) == "draw":
		g.Phase, g.Left = "draw", ChainDraw
	default:
		g.Phase, g.Left = "guess", ChainGuess
	}
	for j, a := range g.P {
		c := g.chainOf(j, k)
		l := &Link{Kind: kindOf(k), By: a.Name, Color: a.Color}
		g.links[c] = append(g.links[c], l)
		a.Done = false
		if a.Gone {
			g.close(j)
		}
	}
	g.checkAll()
}

func (g *Chain) mine(j int) *Link {
	c := g.chainOf(j, g.K)
	return g.links[c][g.K]
}

func (g *Chain) index(id int) int {
	for j, a := range g.P {
		if a.ID == id {
			return j
		}
	}
	return -1
}

func (g *Chain) checkAll() {
	if g.Phase == "reveal" || g.Phase == "done" {
		return
	}
	for _, a := range g.P {
		if !a.Done {
			return
		}
	}
	g.next()
}

func (g *Chain) next() {
	if g.K+1 < g.Steps {
		g.start(g.K + 1)
		return
	}
	g.Phase, g.RC, g.RS, g.auto = "reveal", 0, 1, ChainShow
}

// close remplit le maillon du joueur j s'il est resté vide (chrono écoulé, ou joueur parti).
// Une phrase de départ manquante est remplacée par son idée, pour que la chaîne ait quand même un sujet.
func (g *Chain) close(j int) {
	if l := g.mine(j); !l.done {
		l.done = true
		switch {
		case g.K == 0:
			l.Text = g.P[j].idea
		case l.Kind == "text":
			l.Text, l.Skip = "…", true
		default:
			l.Skip = true
		}
	}
	g.P[j].Done = true
}

func (g *Chain) Add(*Pad) {} // les nouveaux jouent à la partie suivante

func (g *Chain) Remove(id int) {
	j := g.index(id)
	if j < 0 {
		return
	}
	g.P[j].Gone = true
	if g.Phase != "reveal" && g.Phase != "done" {
		g.close(j)
		g.checkAll()
	}
}

func (g *Chain) Input(p *Pad, b []byte) {
	var m struct {
		Text *string          `json:"text"`
		Draw *json.RawMessage `json:"draw"`
		Next bool             `json:"next"`
	}
	if json.Unmarshal(b, &m) != nil {
		return
	}
	if m.Next && p.Host && g.Phase == "reveal" {
		g.advance()
		return
	}
	j := g.index(p.ID)
	if j < 0 || g.Phase == "reveal" || g.Phase == "done" || g.P[j].Done {
		return
	}
	l := g.mine(j)
	switch {
	case l.Kind == "text" && m.Text != nil:
		t := strings.Join(strings.Fields(*m.Text), " ")
		if r := []rune(t); len(r) > ChainText {
			t = string(r[:ChainText])
		}
		if t == "" {
			if g.Phase == "write" {
				t = g.P[j].idea // rien écrit : on prend son idée
			} else {
				t, l.Skip = "…", true
			}
		}
		l.Text = t
	case l.Kind == "draw" && m.Draw != nil:
		d, ok := cleanDrawing(*m.Draw)
		if !ok {
			return
		}
		l.Draw = d
	default:
		return
	}
	l.done, g.P[j].Done = true, true
	g.checkAll()
}

// cleanDrawing vérifie un dessin et le remet au propre (bornes, taille).
func cleanDrawing(raw []byte) (json.RawMessage, bool) {
	var strokes [][]int
	if json.Unmarshal(raw, &strokes) != nil || len(strokes) > ChainStrokes {
		return nil, false
	}
	nums := 0
	out := make([][]int, 0, len(strokes))
	for _, s := range strokes {
		if len(s) < 4 || len(s)%2 != 0 {
			continue
		}
		if nums += len(s); nums > ChainNums {
			return nil, false
		}
		s[0] = min(max(s[0], 0), DrawColors-1)
		s[1] = min(max(s[1], 1), 3)
		for i := 2; i < len(s); i += 2 {
			s[i] = min(max(s[i], 0), DrawW)
			s[i+1] = min(max(s[i+1], 0), DrawH)
		}
		out = append(out, s)
	}
	b, _ := json.Marshal(out)
	return b, true
}

func (g *Chain) advance() {
	g.auto = ChainShow
	if g.RS < len(g.links[g.RC]) {
		g.RS++
		return
	}
	if g.RC+1 < len(g.links) {
		g.RC, g.RS = g.RC+1, 1
		return
	}
	g.Phase = "done"
}

func (g *Chain) Step(dt float64) {
	switch g.Phase {
	case "done":
		return
	case "reveal":
		if g.auto -= dt; g.auto <= 0 {
			g.advance()
		}
		return
	}
	if g.Left -= dt; g.Left > 0 {
		return
	}
	if !g.Late { // chrono écoulé : les téléphones envoient ce qu'ils ont
		g.Late, g.Left = true, ChainLate
		return
	}
	for j := range g.P {
		g.close(j)
	}
	g.checkAll()
}

func (g *Chain) Screen() any {
	s := *g
	s.Left = math.Round(max(0, g.Left)*10) / 10
	if g.Phase == "reveal" {
		s.Owner = g.P[g.RC].Name
		s.Show = g.links[g.RC][:g.RS]
	}
	return s
}

type chainView struct {
	Phase  string `json:"phase"`
	Step   int    `json:"step"`
	Steps  int    `json:"steps"`
	Left   int    `json:"left"`
	Due    bool   `json:"due"`  // envoie ce que tu as
	Done   bool   `json:"done"` // déjà envoyé
	Prompt string `json:"prompt"`
	Idea   string `json:"idea"`
	Out    bool   `json:"out"` // arrivé en cours de partie
	RC     int    `json:"rc"`
	N      int    `json:"n"`
	RS     int    `json:"rs"`
}

func (g *Chain) View(p *Pad) any {
	v := chainView{Phase: g.Phase, Step: g.K, Steps: g.Steps, Left: int(math.Ceil(max(0, g.Left))), Due: g.Late,
		RC: g.RC, N: len(g.links), RS: g.RS}
	if g.Late {
		v.Left = 0
	}
	j := g.index(p.ID)
	if j < 0 {
		v.Out = true
		return v
	}
	var big json.RawMessage
	if g.Phase == "write" || g.Phase == "draw" || g.Phase == "guess" {
		v.Done = g.P[j].Done
		if g.K == 0 {
			v.Idea = g.P[j].idea
		} else if !v.Done {
			prev := g.links[g.chainOf(j, g.K)][g.K-1]
			if prev.Kind == "text" {
				v.Prompt = prev.Text
			} else {
				big = prev.Draw
				if big == nil {
					big = json.RawMessage("[]")
				}
			}
		}
	}
	return tvBig{V: v, Big: big}
}

var chainIdeas = []string{
	"Un chat qui fait du ski", "Une girafe au restaurant", "Un pirate qui a peur de l'eau",
	"Un robot amoureux d'un grille-pain", "Une vache sur la Lune", "Un dinosaure qui fait du vélo",
	"Le Père Noël en vacances à la plage", "Un escargot qui gagne une course", "Un fantôme qui fait la vaisselle",
	"Un pingouin perdu dans le désert", "Une pizza géante qui attaque la ville", "Un chien qui conduit un bus",
	"Une sorcière qui rate son atterrissage", "Un cowboy sur un hamster", "Un éléphant caché derrière un arbre",
	"Un astronaute qui va à la pêche", "Une licorne coincée dans un embouteillage", "Un vampire chez le dentiste",
	"Un bonhomme de neige qui bronze", "Une tortue qui fait du skate", "Un roi qui a perdu sa couronne",
	"Un dragon qui fait griller des chamallows", "Des canards qui traversent la route", "Un ninja qui glisse sur une banane",
	"Un cactus qui fait un câlin", "Un mouton qui n'arrive pas à dormir", "Un extraterrestre qui fait ses courses",
	"Une baleine dans une baignoire", "Un singe qui joue du piano", "Une souris qui fait peur à un éléphant",
	"Un chevalier qui combat un moustique", "Un hérisson chez le coiffeur", "Un ours qui fait du yoga",
	"Un gâteau d'anniversaire qui prend feu", "Un volcan qui éternue", "Une pieuvre qui jongle",
	"Un facteur poursuivi par un chien", "Une momie qui se déroule", "Un crocodile qui se brosse les dents",
	"Un lapin magicien sorti d'un chapeau",
}
