// Physique partagée par les jeux de boules vues de haut (pétanque, billard) : roulement avec frottement
// et chocs entre boules de masses différentes.
package main

import "math"

type Body struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	vx, vy float64
	r, m   float64
	drag   float64 // décélération fixe (px/s²)
	fluid  float64 // plus une part proportionnelle à la vitesse (/s) : ralentit en douceur au lieu de s'arrêter net
}

func (b *Body) moving() bool { return b.vx != 0 || b.vy != 0 }

func (b *Body) speed() float64 { return math.Hypot(b.vx, b.vy) }

func (b *Body) launch(angle, speed float64) {
	b.vx, b.vy = speed*math.Cos(angle), speed*math.Sin(angle)
}

func (b *Body) stop() { b.vx, b.vy = 0, 0 }

// roll : frottement puis déplacement pendant dt.
func (b *Body) roll(dt float64) {
	sp := b.speed()
	if sp == 0 {
		return
	}
	if ns := sp - (b.drag+b.fluid*sp)*dt; ns > 0 {
		b.vx, b.vy = b.vx*ns/sp, b.vy*ns/sp
	} else {
		b.stop()
	}
	b.X, b.Y = b.X+b.vx*dt, b.Y+b.vy*dt
}

// bounce : rebond sur les bords du rectangle [x0,x1]×[y0,y1]. Renvoie true s'il y a eu rebond.
func (b *Body) bounce(x0, y0, x1, y1, e float64) bool {
	hit := false
	if b.X-b.r < x0 || b.X+b.r > x1 {
		lim := min(max(b.X, x0+b.r), x1-b.r)
		b.X, b.vx, hit = 2*lim-b.X, -b.vx*e, true
	}
	if b.Y-b.r < y0 || b.Y+b.r > y1 {
		lim := min(max(b.Y, y0+b.r), y1-b.r)
		b.Y, b.vy, hit = 2*lim-b.Y, -b.vy*e, true
	}
	return hit
}

// collide : choc entre deux boules (coefficient de restitution e). Renvoie la vitesse d'impact, 0 s'il n'y a pas eu choc.
func collide(a, b *Body, e float64) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	d, rr := math.Hypot(dx, dy), a.r+b.r
	if d >= rr || d == 0 {
		return 0
	}
	nx, ny := dx/d, dy/d
	ia, ib := 1/a.m, 1/b.m
	push := (rr - d) / (ia + ib) // on les décolle, la plus légère bouge le plus
	a.X, a.Y, b.X, b.Y = a.X-nx*push*ia, a.Y-ny*push*ia, b.X+nx*push*ib, b.Y+ny*push*ib
	vn := (a.vx-b.vx)*nx + (a.vy-b.vy)*ny
	if vn <= 0 {
		return 0
	}
	j := (1 + e) * vn / (ia + ib)
	a.vx, a.vy, b.vx, b.vy = a.vx-j*ia*nx, a.vy-j*ia*ny, b.vx+j*ib*nx, b.vy+j*ib*ny
	return vn
}
