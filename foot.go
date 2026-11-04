// Foot : foot de pions au tour par tour pour 2 joueurs (ou contre un bot). Chaque équipe a 6 pions ;
// à son tour on lance un pion comme au golf (lance-pierre), il pousse le ballon et les autres pions.
// Un seul coup par tour, premier à 3 buts (réglable de 1 à 10).
package main

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	FW, FH    = 800.0, 500.0
	MouthH    = 150.0 // ouverture des buts
	DiscR     = 22.0
	FBallR    = 12.0
	DiscMass  = 3.0 // plus lourd que le ballon : un pion envoie le ballon loin
	FBallMass = 1.0
	KickMax   = 1100.0 // vitesse d'un pion lancé à pleine puissance (px/s)
	// frottement : une part fixe (px/s²) plus une part proportionnelle à la vitesse (/s). Les pièces filent vite puis
	// ralentissent en douceur, au lieu de freiner pareil jusqu'au bout et de s'arrêter net.
	DiscRoll    = 120.0
	FBallRoll   = 90.0
	RollDrag    = 0.3
	FBounce     = 0.8  // rebond sur les bords
	Restitute   = 0.9  // chocs entre pièces
	TurnTime    = 15.0 // pour jouer son coup
	SetupTime   = 10.0 // pour choisir sa disposition
	FootWin     = 3    // score à atteindre par défaut
	FootWinMax  = 10
	FootTick    = time.Second / 60
	FootSub     = 4 // sous-pas par image : aucune pièce ne traverse une autre
	FootRooms   = 1000
	BotThink    = 1.2 // secondes avant que le bot joue
	BotAimNoise = 0.07
)

// Dispositions de l'équipe de gauche (l'équipe de droite est en miroir).
var formations = map[string][6][2]float64{
	"defense":  {{45, 250}, {140, 90}, {130, 195}, {130, 305}, {140, 410}, {300, 250}},   // gardien, 4 derrière, 1 devant
	"balanced": {{120, 130}, {100, 250}, {120, 370}, {230, 170}, {230, 330}, {330, 250}}, // 3-2-1
	"attack":   {{110, 180}, {110, 320}, {290, 90}, {320, 200}, {320, 300}, {290, 410}},  // 2 derrière, 4 devant
}

type Piece struct {
	Team   int     `json:"team"` // 0 gauche, 1 droite, -1 ballon
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	vx, vy float64
}

func (p *Piece) r() float64 {
	if p.Team < 0 {
		return FBallR
	}
	return DiscR
}

func (p *Piece) mass() float64 {
	if p.Team < 0 {
		return FBallMass
	}
	return DiscMass
}

type Footer struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Bot       bool   `json:"bot"`
	Score     int    `json:"score"`
	Formation string `json:"formation"` // "" = pas encore choisie
	send      chan []byte
}

type Match struct {
	mu      sync.Mutex
	Pin     string     `json:"pin"`
	Players [2]*Footer `json:"players"` // index = équipe ; nil = place libre
	Discs   []*Piece   `json:"discs"`   // 6 par équipe : l'index sert à désigner le pion lancé
	Ball    *Piece     `json:"ball"`
	Phase   string     `json:"phase"` // wait, setup, aim, roll, goal, end
	Wait    float64    `json:"wait"`
	Turn    int        `json:"turn"` // équipe qui joue
	Kick    int        `json:"kick"` // équipe qui engage
	Winner  string     `json:"winner"`
	Goals   int        `json:"goals"` // premier à ce score
	Events  []Event    `json:"events"`
	nextID  int
}

func newMatch(pin string) *Match {
	m := &Match{Pin: pin, Goals: FootWin}
	m.reset()
	return m
}

func (m *Match) event(e Event) { m.Events = append(m.Events, e) }

// reset : nouvelle partie, dès qu'un joueur arrive ou part, ou après une victoire.
func (m *Match) reset() {
	m.Winner, m.Phase, m.Wait = "", "wait", 0
	m.Kick = rand.IntN(2)
	for _, p := range m.Players {
		if p != nil {
			p.Score = 0
		}
	}
	m.place()
	if m.Players[0] != nil && m.Players[1] != nil {
		m.setup()
	}
}

// setup : chacun choisit sa disposition avant le coup d'envoi.
func (m *Match) setup() {
	m.Phase, m.Wait = "setup", SetupTime
	for _, p := range m.Players {
		p.Formation = ""
		if p.Bot {
			p.Formation = []string{"defense", "balanced", "attack"}[rand.IntN(3)]
		}
	}
	m.place()
}

// place met les pions selon les dispositions choisies (Équilibrée par défaut) et le ballon au centre.
func (m *Match) place() {
	m.Discs = m.Discs[:0]
	for team, p := range m.Players {
		f := "balanced"
		if p != nil && p.Formation != "" {
			f = p.Formation
		}
		for _, xy := range formations[f] {
			x := xy[0]
			if team == 1 {
				x = FW - x
			}
			m.Discs = append(m.Discs, &Piece{Team: team, X: x, Y: xy[1]})
		}
	}
	m.Ball = &Piece{Team: -1, X: FW / 2, Y: FH / 2}
}

func (m *Match) choose(p *Footer, f string) {
	if _, ok := formations[f]; !ok || m.Phase != "setup" {
		return
	}
	p.Formation = f
	m.place()
	if m.Players[0].Formation != "" && m.Players[1].Formation != "" {
		m.kickoff()
	}
}

// setGoals change le score à atteindre, à tout moment tant qu'il reste au-dessus des scores actuels.
func (m *Match) setGoals(n int) {
	if n < 1 || n > FootWinMax || m.Phase == "end" {
		return
	}
	for _, p := range m.Players {
		if p != nil && p.Score >= n {
			return
		}
	}
	m.Goals = n
}

func (m *Match) kickoff() {
	for _, p := range m.Players {
		if p.Formation == "" {
			p.Formation = "balanced"
		}
	}
	m.place()
	m.Turn, m.Phase, m.Wait = m.Kick, "aim", TurnTime
}

func (m *Match) join(name string, bot bool) *Footer {
	slot := -1
	for i, p := range m.Players {
		if p == nil {
			slot = i
			break
		}
	}
	if slot < 0 && !bot { // un humain prend la place du bot
		for i, p := range m.Players {
			if p.Bot {
				slot = i
				break
			}
		}
	}
	if slot < 0 {
		return nil
	}
	m.nextID++
	p := &Footer{ID: m.nextID, Name: name, Bot: bot}
	if !bot {
		p.send = make(chan []byte, 8)
	}
	m.Players[slot] = p
	m.reset()
	return p
}

func (m *Match) leave(id int) {
	humans := 0
	for i, p := range m.Players {
		if p != nil && p.ID == id {
			m.Players[i] = nil
		} else if p != nil && !p.Bot {
			humans++
		}
	}
	if humans == 0 {
		m.Players = [2]*Footer{} // plus aucun humain : le bot part aussi, le salon se ferme
	}
	m.reset()
}

func (m *Match) removeBot() {
	for i, p := range m.Players {
		if p != nil && p.Bot {
			m.Players[i] = nil
			m.reset()
		}
	}
}

func (m *Match) team(p *Footer) int {
	if m.Players[1] == p {
		return 1
	}
	return 0
}

// shoot : l'équipe dont c'est le tour lance son pion d (angle en radians, puissance 0 à 1).
func (m *Match) shoot(team, d int, angle, power float64) {
	if m.Phase != "aim" || team != m.Turn || d < 0 || d >= len(m.Discs) || m.Discs[d].Team != team ||
		!(power > 0) || math.IsNaN(angle) || math.IsInf(angle, 0) {
		return
	}
	v := min(power, 1) * KickMax
	p := m.Discs[d]
	p.vx, p.vy = v*math.Cos(angle), v*math.Sin(angle)
	m.Phase = "roll"
	m.event(Event{Kind: "kick", A: team, X: p.X, Y: p.Y})
}

func (m *Match) pieces() []*Piece { return append(m.Discs[:len(m.Discs):len(m.Discs)], m.Ball) }

func (m *Match) step(dt float64) {
	switch m.Phase {
	case "setup":
		if m.Wait -= dt; m.Wait <= 0 {
			m.kickoff()
		}
	case "aim":
		if bot := m.Players[m.Turn]; bot != nil && bot.Bot && m.Wait <= TurnTime-BotThink {
			m.think()
			return
		}
		if m.Wait -= dt; m.Wait <= 0 { // temps écoulé : le tour passe
			m.Turn, m.Wait = 1-m.Turn, TurnTime
		}
	case "roll":
		moving := false
		for range FootSub {
			moving = m.physics(dt / FootSub)
			if m.Phase != "roll" { // but
				return
			}
		}
		if !moving {
			m.Turn, m.Phase, m.Wait = 1-m.Turn, "aim", TurnTime
		}
	case "goal":
		if m.Wait -= dt; m.Wait <= 0 {
			m.setup()
		}
	case "end":
		if m.Wait -= dt; m.Wait <= 0 { // revanche
			m.reset()
		}
	}
}

// physics avance d'un sous-pas : déplacement, frottement, bords, chocs. Renvoie true si quelque chose bouge encore.
func (m *Match) physics(dt float64) bool {
	ps := m.pieces()
	moving := false
	for _, p := range ps {
		sp := math.Hypot(p.vx, p.vy)
		if sp == 0 {
			continue
		}
		dec := DiscRoll + RollDrag*sp
		if p.Team < 0 {
			dec = FBallRoll + RollDrag*sp
		}
		if ns := sp - dec*dt; ns > 0 {
			p.vx, p.vy = p.vx*ns/sp, p.vy*ns/sp
			moving = true
		} else {
			p.vx, p.vy = 0, 0
		}
		p.X, p.Y = p.X+p.vx*dt, p.Y+p.vy*dt
		m.walls(p)
	}
	if m.Ball.X < -FBallR || m.Ball.X > FW+FBallR { // ballon entièrement passé la ligne, entre les poteaux
		scorer := 0 // ballon dans le but de droite : l'équipe de gauche marque
		if m.Ball.X < 0 {
			scorer = 1
		}
		m.goal(scorer)
		return false
	}
	for i, a := range ps {
		for _, b := range ps[i+1:] {
			m.collide(a, b)
		}
	}
	return moving
}

func inMouth(y float64) bool { return math.Abs(y-FH/2) < MouthH/2 }

// walls : rebond sur les bords. Seul le ballon passe par l'ouverture des buts ; les poteaux sont des points durs.
func (m *Match) walls(p *Piece) {
	r := p.r()
	if p.Y < r || p.Y > FH-r {
		lim := min(max(p.Y, r), FH-r)
		p.Y, p.vy = 2*lim-p.Y, -p.vy*FBounce
		m.event(Event{Kind: "wall", X: p.X, Y: lim})
	}
	open := p.Team < 0 && inMouth(p.Y)
	if !open && (p.X < r || p.X > FW-r) {
		lim := min(max(p.X, r), FW-r)
		p.X, p.vx = 2*lim-p.X, -p.vx*FBounce
		m.event(Event{Kind: "wall", X: lim, Y: p.Y})
	}
	if p.Team < 0 {
		for _, post := range [][2]float64{{0, FH/2 - MouthH/2}, {0, FH/2 + MouthH/2}, {FW, FH/2 - MouthH/2}, {FW, FH/2 + MouthH/2}} {
			dx, dy := p.X-post[0], p.Y-post[1]
			if d := math.Hypot(dx, dy); d < r && d > 0 {
				nx, ny := dx/d, dy/d
				p.X, p.Y = post[0]+nx*r, post[1]+ny*r
				if vn := p.vx*nx + p.vy*ny; vn < 0 {
					p.vx, p.vy = p.vx-(1+FBounce)*vn*nx, p.vy-(1+FBounce)*vn*ny
					m.event(Event{Kind: "post", X: post[0], Y: post[1]})
				}
			}
		}
	}
}

// collide : choc entre deux pièces de masses différentes.
func (m *Match) collide(a, b *Piece) {
	dx, dy := b.X-a.X, b.Y-a.Y
	d, rr := math.Hypot(dx, dy), a.r()+b.r()
	if d >= rr || d == 0 {
		return
	}
	nx, ny := dx/d, dy/d
	ia, ib := 1/a.mass(), 1/b.mass()
	push := (rr - d) / (ia + ib) // on les décolle, le plus léger bouge le plus
	a.X, a.Y, b.X, b.Y = a.X-nx*push*ia, a.Y-ny*push*ia, b.X+nx*push*ib, b.Y+ny*push*ib
	vn := (a.vx-b.vx)*nx + (a.vy-b.vy)*ny
	if vn <= 0 {
		return
	}
	j := (1 + Restitute) * vn / (ia + ib)
	a.vx, a.vy, b.vx, b.vy = a.vx-j*ia*nx, a.vy-j*ia*ny, b.vx+j*ib*nx, b.vy+j*ib*ny
	m.event(Event{Kind: "hit", X: a.X + nx*a.r(), Y: a.Y + ny*a.r(), A: int(min(vn, 2000))})
}

func (m *Match) goal(team int) {
	p := m.Players[team]
	p.Score++
	m.Kick = 1 - team // l'équipe qui a encaissé engage
	m.event(Event{Kind: "goal", A: team, X: m.Ball.X, Y: m.Ball.Y, Text: p.Name})
	for _, d := range m.pieces() {
		d.vx, d.vy = 0, 0
	}
	if p.Score >= m.Goals {
		m.Phase, m.Wait, m.Winner = "end", 6, p.Name
		m.event(Event{Kind: "win", A: team, Text: p.Name})
		return
	}
	m.Phase, m.Wait = "goal", 2.5
}

// think : le bot prend le pion le mieux placé derrière le ballon et le lance pour pousser le ballon vers le but adverse.
func (m *Match) think() {
	goalX := FW
	if m.Turn == 1 {
		goalX = 0
	}
	b := m.Ball
	gx, gy := goalX-b.X, FH/2-b.Y
	gl := math.Hypot(gx, gy)
	gx, gy = gx/gl, gy/gl
	cx, cy := b.X-gx*(DiscR+FBallR-2), b.Y-gy*(DiscR+FBallR-2) // où frapper le ballon
	best, bestScore := -1, math.Inf(1)
	for i, d := range m.Discs {
		if d.Team != m.Turn {
			continue
		}
		dx, dy := cx-d.X, cy-d.Y
		dist := math.Hypot(dx, dy)
		behind := (dx*gx + dy*gy) / dist // 1 : pion pile derrière le ballon
		if score := dist * (2 - behind); score < bestScore {
			best, bestScore = i, score
		}
	}
	d := m.Discs[best]
	angle := math.Atan2(cy-d.Y, cx-d.X) + (rand.Float64()*2-1)*BotAimNoise
	power := min(max(math.Hypot(cx-d.X, cy-d.Y)/450+0.35, 0.45), 1)
	m.shoot(m.Turn, best, angle, power)
}

// --- salons Foot (même principe que Ricochet, codes séparés) ---

var (
	matchesMu sync.Mutex
	matches   = map[string]*Match{}
)

func enterMatch(pin, name string) (*Match, *Footer, string) {
	matchesMu.Lock()
	defer matchesMu.Unlock()
	if pin == "" {
		if len(matches) >= FootRooms {
			return nil, nil, "Trop de parties en cours, réessaie plus tard"
		}
		m := newMatch(newPin(func(p string) bool { return matches[p] != nil }))
		matches[m.Pin] = m
		p := m.join(name, false) // avant run() : le salon n'est jamais vide quand la boucle démarre
		go m.run()
		return m, p, ""
	}
	m := matches[pin]
	if m == nil {
		return nil, nil, "Code inconnu"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if p := m.join(name, false); p != nil {
		return m, p, ""
	}
	return nil, nil, "Partie pleine (2 joueurs)"
}

func (m *Match) run() {
	t := time.NewTicker(FootTick)
	defer t.Stop()
	n := 0
	for range t.C {
		if m.closeIfEmpty() {
			return
		}
		m.mu.Lock()
		m.step(FootTick.Seconds())
		if n++; n%2 == 0 { // physique à 60 images/s, envoi à 30 : le navigateur lisse entre deux états
			b, _ := json.Marshal(m)
			m.Events = nil
			for _, p := range m.Players {
				if p != nil && p.send != nil {
					select {
					case p.send <- b:
					default: // client lent : on saute cette image
					}
				}
			}
		}
		m.mu.Unlock()
	}
}

func (m *Match) closeIfEmpty() bool {
	matchesMu.Lock()
	defer matchesMu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Players[0] != nil || m.Players[1] != nil {
		return false
	}
	delete(matches, m.Pin)
	return true
}

func serveFoot(w http.ResponseWriter, r *http.Request) {
	session(w, r, 256, func(name string, q url.Values) (*Seat, string) {
		m, p, why := enterMatch(q.Get("pin"), name)
		if p == nil {
			return nil, why
		}
		return &Seat{
			Room:  m.Pin,
			ID:    p.ID,
			Send:  p.send,
			Leave: func() { m.mu.Lock(); m.leave(p.ID); m.mu.Unlock() },
			OnMsg: func(b []byte) {
				var msg struct {
					Bot       int     `json:"bot"` // +1 : jouer contre un bot, -1 : le retirer
					Formation string  `json:"formation"`
					Goals     int     `json:"goals"` // premier à ce score
					D         *int    `json:"d"`     // pion lancé
					A         float64 `json:"a"`
					P         float64 `json:"p"`
				}
				if json.Unmarshal(b, &msg) != nil {
					return
				}
				m.mu.Lock()
				defer m.mu.Unlock()
				switch {
				case msg.Bot == 1:
					m.join("Bot", true)
				case msg.Bot == -1:
					m.removeBot()
				case msg.Goals != 0:
					m.setGoals(msg.Goals)
				case msg.Formation != "":
					m.choose(p, msg.Formation)
				case msg.D != nil:
					m.shoot(m.team(p), *msg.D, msg.A, msg.P)
				}
			},
		}, ""
	})
}
