// Les cartes de Lombrics : toutes de la même taille (LW × LH), chacune son relief. La page les habille
// (couleurs, ciel, décor) d'après leur nom.
package main

import (
	"math"
	"math/rand/v2"
)

var mapNames = []string{"collines", "chateau", "plage", "banquise", "caverne", "ville"}

// genMap dessine le terrain de la carte g.Map.
func (g *WormGame) genMap() {
	g.land = make([]bool, LGW*LGH)
	switch g.Map {
	case "chateau":
		g.genCastle()
	case "plage":
		g.genBeach()
	case "banquise":
		g.genIce()
	case "caverne":
		g.genCave()
	case "ville":
		g.genCity()
	default:
		g.genHills()
	}
}

// edge : les deux bords de l'île plongent dans l'eau.
func edge(x, width float64) float64 { return math.Max(0, width-math.Min(x, LW-x)) * 2.5 }

// ground remplit de terre tout ce qui est sous la courbe top(x).
func (g *WormGame) ground(top func(x float64) float64) {
	for c := range LGW {
		t := top((float64(c) + 0.5) * LCell)
		for r := range LGH {
			if (float64(r)+0.5)*LCell > t {
				g.land[r*LGW+c] = true
			}
		}
	}
}

// rect remplit (ou vide) un rectangle.
func (g *WormGame) rect(x0, y0, x1, y1 float64, solid bool) {
	for r := max(0, int(y0/LCell)); r < min(LGH, int(math.Ceil(y1/LCell))); r++ {
		for c := max(0, int(x0/LCell)); c < min(LGW, int(math.Ceil(x1/LCell))); c++ {
			g.land[r*LGW+c] = solid
		}
	}
}

// merlons : les créneaux posés sur un haut de mur.
func (g *WormGame) merlons(x0, x1, top float64) {
	for x := x0; x+10 <= x1; x += 20 {
		g.rect(x, top-15, x+10, top, true)
	}
}

// topAt : la surface du terrain à l'abscisse x (la première terre en partant du haut).
func (g *WormGame) topAt(x float64) float64 {
	y := 0.0
	for y < LH && !g.solid(x, y) {
		y++
	}
	return y
}

func rnd(a, b float64) float64 { return a + rand.Float64()*(b-a) }

// Collines : du relief, des îlots flottants, des grottes.
func (g *WormGame) genHills() {
	p1, p2, f := rnd(0, 7), rnd(0, 7), rnd(2, 3.5)
	g.ground(func(x float64) float64 {
		return LH*0.5 + 110*math.Sin(f*x/LW*2*math.Pi+p1) + 45*math.Sin(4.3*f*x/LW*2*math.Pi+p2) + edge(x, 120)
	})
	for range 2 + rand.IntN(2) {
		g.fill(rnd(150, LW-150), rnd(LH*0.18, LH*0.3), rnd(50, 110), rnd(14, 24))
	}
	for range 6 + rand.IntN(4) {
		g.carve(rnd(80, LW-80), rnd(LH*0.6, LH*0.85), rnd(25, 53))
	}
}

// Château : un château fort au milieu (tours crénelées, donjon, grande salle à deux niveaux, portes), deux tours de
// guet sur les côtés.
func (g *WormGame) genCastle() {
	p := rnd(0, 7)
	ground := func(x float64) float64 { return LH*0.72 + 25*math.Sin(x/LW*2*math.Pi*2.5+p) + edge(x, 120) }
	g.ground(ground)
	x0, x1, wallTop, base := LW*0.3, LW*0.7, LH*0.48, LH*0.78
	g.rect(x0, wallTop, x1, base, true)
	g.merlons(x0, x1, wallTop)
	for i := range 4 { // tours
		tx, top := x0+float64(i)*(x1-x0)/3, rnd(LH*0.28, LH*0.34)
		g.rect(tx-35, top, tx+35, base, true)
		g.merlons(tx-35, tx+35, top)
		g.rect(tx-5, top+30, tx+5, top+55, false) // meurtrière
	}
	g.rect(LW/2-60, LH*0.14, LW/2+60, wallTop, true) // donjon
	g.merlons(LW/2-60, LW/2+60, LH*0.14)
	g.rect(LW/2-6, LH*0.2, LW/2+6, LH*0.26, false)
	hall := LH*0.72 - 10
	g.rect(x0+45, wallTop+35, x1-45, hall, false)   // grande salle
	g.rect(x0+110, LH*0.6, x1-110, LH*0.6+10, true) // son étage, ouvert aux deux bouts
	for _, d := range []float64{x0 - 50, x1 - 45} { // portes
		g.rect(d, hall-50, d+95, hall, false)
		g.carve(d+47, hall-50, 47)
	}
	for _, tx := range []float64{LW * 0.1, LW * 0.9} { // tours de guet
		top := ground(tx) - rnd(140, 200)
		g.rect(tx-25, top, tx+25, ground(tx)+20, true)
		g.merlons(tx-25, tx+25, top)
	}
}

// Plage : des dunes qui descendent dans la mer, des palmiers, un château de sable et un ponton.
func (g *WormGame) genBeach() {
	p1, p2 := rnd(0, 7), rnd(0, 7)
	top := func(x float64) float64 {
		return LH*0.7 + 30*math.Sin(x/LW*2*math.Pi*3+p1) + 12*math.Sin(x/LW*2*math.Pi*9+p2) +
			math.Max(0, x-LW*0.62)*0.55 + math.Max(0, 120-x)*2.5
	}
	g.ground(top)
	for k := range 3 { // palmiers : un tronc qui penche, et une touffe de palmes
		x, dir, h := rnd(100+float64(k)*280, 300+float64(k)*280), float64(1-2*rand.IntN(2)), rnd(100, 150)
		t := top(x)
		for d := 0.0; d < h; d += 5 {
			off := math.Sin(d/h*math.Pi/2) * 25 * dir
			g.rect(x+off-5, t-d-5, x+off+5, t-d+5, true)
		}
		cx := x + 25*dir
		g.fill(cx-28, t-h+4, 32, 8)
		g.fill(cx+28, t-h+4, 32, 8)
		g.fill(cx, t-h-6, 18, 10)
	}
	sx := rnd(LW*0.42, LW*0.55) // château de sable
	st := top(sx)
	g.rect(sx-35, st-40, sx+35, st+10, true)
	g.merlons(sx-35, sx+35, st-40)
	g.rect(sx-12, st-70, sx+12, st-40, true)
	g.merlons(sx-12, sx+12, st-70)
	g.rect(LW*0.8, LH*0.9, LW*0.98, LH*0.9+8, true) // ponton sur pilotis
	for x := LW * 0.8; x < LW*0.98; x += 65 {
		g.rect(x, LH*0.9, x+6, LH, true)
	}
	for range 2 {
		g.carve(rnd(150, LW*0.55), rnd(LH*0.8, LH*0.9), rnd(22, 35))
	}
}

// Banquise : des pics enneigés, des grottes de glace, des plaques de glace sur l'eau.
func (g *WormGame) genIce() {
	p1, p2 := rnd(0, 1), rnd(0, 7)
	tri := func(u float64) float64 { return math.Abs(2 * (u - math.Floor(u+0.5))) } // 0 au sommet, 1 au creux
	g.ground(func(x float64) float64 {
		return LH*0.32 + 300*tri(x/LW*3+p1) + 25*math.Sin(x/LW*2*math.Pi*7+p2) + edge(x, 200)
	})
	for range 4 + rand.IntN(3) {
		g.carve(rnd(250, LW-250), rnd(LH*0.62, LH*0.85), rnd(25, 45))
	}
	for _, x := range []float64{55, 150, LW - 150, LW - 55} { // plaques de glace
		g.fill(x+rnd(-10, 10), WaterY-3, rnd(30, 45), 9)
	}
}

// Caverne : de la roche partout, une grande galerie qui serpente, des galeries qui en partent, de grandes salles,
// et deux cheminées ouvertes sur le ciel (les caisses tombent par là). Le paradis de la corde ninja.
func (g *WormGame) genCave() {
	g.rect(0, 0, LW, LH, true)
	type pt struct{ x, y float64 }
	var path []pt
	y := rnd(LH*0.4, LH*0.6)
	for x := 0.0; x <= LW; x += 20 {
		y = math.Max(LH*0.3, math.Min(LH*0.72, y+rnd(-25, 25)))
		path = append(path, pt{x, y})
		g.carve(x, y, rnd(50, 75))
	}
	for range 3 { // galeries qui partent de la grande
		p := path[rand.IntN(len(path))]
		dy := rnd(10, 18) * float64(1-2*rand.IntN(2))
		dx := 20 * float64(1-2*rand.IntN(2))
		for range 18 {
			p.x, p.y = p.x+dx, math.Max(LH*0.15, math.Min(LH*0.85, p.y+dy))
			g.carve(p.x, p.y, rnd(32, 42))
		}
	}
	for range 3 { // grandes salles sur le chemin
		p := path[rand.IntN(len(path))]
		g.carve(p.x, p.y, rnd(85, 110))
	}
	for _, cx := range []float64{rnd(LW*0.15, LW*0.35), rnd(LW*0.65, LW*0.85)} { // cheminées
		for y := 0.0; y < LH*0.55; y += 12 {
			g.carve(cx+math.Sin(y/60)*20, y, 28)
		}
	}
}

// Ville : une rue, des immeubles à étages (on y entre par les portes et les fenêtres), des antennes sur les toits.
func (g *WormGame) genCity() {
	street := LH * 0.8
	g.ground(func(x float64) float64 { return street + edge(x, 70) })
	for x := rnd(70, 110); x < LW-170; {
		w, h := rnd(100, 190), rnd(LH*0.18, LH*0.55)
		x0, x1, top := x, x+w, street-h
		g.rect(x0, top, x1, street+5, true)
		side := rand.IntN(2)
		for fy := street - 38; fy >= top+12; fy -= 50 { // étages depuis la rue : 10 px de mur, 12 px de plancher
			g.rect(x0+10, fy, x1-10, fy+38, false)
			wx := x0 // une fenêtre par étage, d'un côté puis de l'autre
			if side == 1 {
				wx = x1 - 10
			}
			g.rect(wx, fy+10, wx+10, fy+30, false)
			side = 1 - side
		}
		g.rect(x0, street-38, x0+10, street, false) // portes au rez-de-chaussée
		g.rect(x1-10, street-38, x1, street, false)
		if rand.IntN(2) == 0 {
			g.rect(x0+w/2-2, top-35, x0+w/2+2, top, true) // antenne
		}
		x = x1 + rnd(45, 100)
	}
}
