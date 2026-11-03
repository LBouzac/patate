package main

import (
	"math"
	"testing"
)

// La piste ne se chevauche jamais : deux morceaux éloignés le long du tracé restent séparés de plus d'une largeur.
func TestCircuitsDoNotOverlap(t *testing.T) {
	for _, c := range circuits {
		n := len(c.pts)
		if c.L < 2500 || n < 300 {
			t.Fatalf("%s : trop court (%.0f px)", c.Name, c.L)
		}
		for i := 0; i < n; i++ {
			p := c.pts[i]
			if p[0] < c.W/2 || p[1] < c.W/2 || p[0] > SumoW-c.W/2 || p[1] > SumoH-c.W/2 {
				t.Fatalf("%s : la piste sort de l'écran en %v", c.Name, p)
			}
			for j := i + 1; j < n; j++ {
				along := math.Abs(c.cum[j] - c.cum[i])
				along = math.Min(along, c.L-along)
				if along > 3*c.W && dist2(p, c.pts[j]) < c.W+20 {
					t.Fatalf("%s : la piste se touche entre %v et %v", c.Name, p, c.pts[j])
				}
			}
		}
		for _, b := range c.boxes {
			if _, d := c.nearest(b[0], b[1], 0); d > c.W/2 {
				i, _ := c.nearest(b[0], b[1], int(float64(n)*0.27))
				_ = i
			}
		}
	}
}

// Une course complète avec bots sur chaque circuit : tout le monde boucle ses 3 tours, points distribués.
func TestKartRaceOnEveryCircuit(t *testing.T) {
	g := newKart(sumoPads("Ana"))
	for range 3 {
		g.addBot()
	}
	ana := g.get(1)
	for race := 1; race <= len(circuits); race++ {
		if g.Phase != "count" || g.Race != race {
			t.Fatalf("course %d : départ attendu, obtenu %s (course %d)", race, g.Phase, g.Race)
		}
		for _, c := range g.Cars {
			if !c.In || c.dist >= 0 {
				t.Fatalf("%s doit être sur la grille, derrière la ligne", c.Name)
			}
		}
		for i := 0; i < 30*240 && g.Phase != "result"; i++ {
			if i%3 == 0 && g.Phase == "race" {
				g.brain(ana, 1) // Ana conduit comme un bot
			}
			g.Step(TVTick.Seconds())
			g.Screen()
		}
		if g.Phase != "result" {
			t.Fatalf("%s : la course ne se termine pas", g.Name)
		}
		done := 0
		for _, c := range g.Cars {
			if c.Done {
				done++
				if c.Lap != RaceLaps || c.Time < 20 {
					t.Fatalf("%s : %s arrivé au tour %d en %.1f s", g.Name, c.Name, c.Lap, c.Time)
				}
			}
		}
		if done < 3 {
			t.Fatalf("%s : %d voitures seulement ont fini", g.Name, done)
		}
		first := 0
		for _, c := range g.Cars {
			if c.Rank == 1 {
				first = c.Gain
			}
		}
		if first != racePoints[0] {
			t.Fatalf("le premier marque %d points", racePoints[0])
		}
		g.Step(RaceResult + 0.1)
	}
	if g.Race != len(circuits)+1 || g.Name != circuits[0].Name {
		t.Fatal("après le dernier circuit, on revient au premier")
	}
}

func TestKartItems(t *testing.T) {
	g := newKart(sumoPads("Ana", "Bob"))
	g.Step(3.1)
	a, b := g.get(1), g.get(2)
	box := g.c.boxes[0]
	a.body.X, a.body.Y = box[0], box[1]
	g.drive(0.001)
	if a.Item == "" || g.Boxes[0] {
		t.Fatal("en passant sur une boîte, on ramasse un objet et la boîte disparaît")
	}
	a.Item = "turbo"
	g.Input(&Pad{ID: 1}, []byte(`{"item":true}`))
	if a.boost <= 0 || a.Item != "" {
		t.Fatal("turbo utilisé")
	}
	a.Item = "huile"
	g.use(a)
	if len(g.Oils) != 1 {
		t.Fatal("flaque posée derrière la voiture")
	}
	o := g.Oils[0]
	b.body.X, b.body.Y = o.X, o.Y
	g.drive(0.001)
	if b.spin <= 0 || len(g.Oils) != 0 || b.pokes == 0 {
		t.Fatal("dans l'huile : tête-à-queue (et vibration), la flaque disparaît")
	}
	g.Step(BoxBack + 0.1)
	if !g.Boxes[0] {
		t.Fatal("la boîte revient")
	}
}

// Couper à travers l'herbe ne fait pas avancer : on est ramené là où on a quitté la piste.
func TestKartNoShortcut(t *testing.T) {
	g := newKart(sumoPads("Ana"))
	g.Step(3.1)
	a := g.get(1)
	before := a.dist
	far := g.c.at(a.idx + len(g.c.pts)/2)
	a.body.X, a.body.Y = far[0], far[1]
	for range 30 * 3 {
		g.Step(TVTick.Seconds())
	}
	if a.dist-before > 300 {
		t.Fatalf("raccourci : %.0f px gagnés", a.dist-before)
	}
	if _, d := g.c.nearest(a.body.X, a.body.Y, a.idx); d > g.c.W/2 {
		t.Fatal("ramené sur la piste après être resté dans l'herbe")
	}
}

func TestKartSteeringAndJoin(t *testing.T) {
	ps := sumoPads("Ana", "Bob", "Cy")
	g := newKart(ps[:1])
	g.Add(ps[1])
	if b := g.get(2); !b.In || len(g.Cars) != 2 {
		t.Fatal("arrivé pendant le compte à rebours : il prend place sur la grille")
	}
	g.Step(3.1)
	g.Add(ps[2])
	if c := g.get(3); c.In {
		t.Fatal("arrivé pendant la course : il attend la suivante")
	}
	g.Input(ps[0], []byte(`{"s":5}`))
	if a := g.get(1); a.steer != 1 {
		t.Fatal("direction bornée à ±1")
	}
	h := g.get(1).head
	for range 15 {
		g.Step(TVTick.Seconds())
	}
	if g.get(1).head <= h {
		t.Fatal("à droite, le cap tourne")
	}
	g.Input(ps[1], []byte(`{"bot":1}`))
	if len(g.Cars) != 3 {
		t.Fatal("seul le chef ajoute des bots")
	}
	g.Input(ps[0], []byte(`{"bot":1}`))
	if len(g.Cars) != 4 {
		t.Fatal("le chef ajoute un bot")
	}
	if v := g.View(ps[2]).(kartView); v.In {
		t.Fatal("sa manette indique qu'il attend")
	}
}
