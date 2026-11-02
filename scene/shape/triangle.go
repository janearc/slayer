package shape

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// tri is a face as the tree keeps it: its corners, and which face it is.
type tri struct {
	p    [3][3]float64
	face int32
}

// vec is a corner as a point.
func vec(p [3]float64) maths.Vec { return maths.V(p[0], p[1], p[2]) }

// closest is the point of triangle a, b, c nearest p, as weights on its
// corners: inside the face, or on an edge or a corner when p is beyond
// it. It is Ericson's, from Real-Time Collision Detection.
func closest(p, a, b, c maths.Vec) [3]float64 {
	ab, ac, ap := b.Sub(a), c.Sub(a), p.Sub(a)
	d1, d2 := ab.Dot(ap), ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return [3]float64{1, 0, 0}
	}
	bp := p.Sub(b)
	d3, d4 := ab.Dot(bp), ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return [3]float64{0, 1, 0}
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		v := d1 / (d1 - d3)
		return [3]float64{1 - v, v, 0}
	}
	cp := p.Sub(c)
	d5, d6 := ab.Dot(cp), ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return [3]float64{0, 0, 1}
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		w := d2 / (d2 - d6)
		return [3]float64{1 - w, 0, w}
	}
	va := d3*d6 - d5*d4
	if va <= 0 && d4-d3 >= 0 && d5-d6 >= 0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return [3]float64{0, 1 - w, w}
	}
	v, w := vb/(va+vb+vc), vc/(va+vb+vc)
	return [3]float64{1 - v - w, v, w}
}

// query is a ray made ready to meet many triangles: turned so that it
// runs along an axis, kz, for the triangles, and with its direction
// inverted, for the boxes.
type query struct {
	o          [3]float64
	at, inv    maths.Vec
	kx, ky, kz int
	sx, sy, sz float64
}

// ready makes r a query. The axis it runs along is the one it runs most
// along; the other two are swapped when it runs backwards, so that a
// face's turn reads the same either way.
func ready(r maths.Ray) query {
	d := [3]float64{r.Dir.X, r.Dir.Y, r.Dir.Z}
	kz := 0
	if math.Abs(d[1]) > math.Abs(d[kz]) {
		kz = 1
	}
	if math.Abs(d[2]) > math.Abs(d[kz]) {
		kz = 2
	}
	kx, ky := (kz+1)%3, (kz+2)%3
	if d[kz] < 0 {
		kx, ky = ky, kx
	}
	return query{
		o:   [3]float64{r.At.X, r.At.Y, r.At.Z},
		at:  r.At,
		inv: maths.V(1/r.Dir.X, 1/r.Dir.Y, 1/r.Dir.Z),
		kx:  kx, ky: ky, kz: kz,
		sx: d[kx] / d[kz], sy: d[ky] / d[kz], sz: 1 / d[kz],
	}
}

// meet is where the ray meets triangle t, between near and far, and whether it
// meets the face's outside, where its corners turn counterclockwise.
//
// It is Woop, Benthin and Wald's watertight test (JCGT, 2013): Moller and
// Trumbore's solve for the weights on the corners, done after a shear that
// makes the ray the z axis. Each corner is sheared alone, so faces that share a
// corner share its sheared place to the last bit.
//
// Each weight is then one edge's two by two determinant, of that edge's two
// corners alone, and its sign is exact. A ray exactly on an edge is moved aside
// by an infinitesimal step, the same for every face, so the faces either side
// of the edge disagree about it.
//
// So a ray through an edge or a corner meets exactly one of the faces there,
// and never slips between them.
func (q *query) meet(t *tri, near, far float64) (float64, bool, bool) {
	ax, ay, az := q.shear(&t.p[0])
	bx, by, bz := q.shear(&t.p[1])
	cx, cy, cz := q.shear(&t.p[2])
	u, v, w := det(cx, cy, bx, by), det(ax, ay, cx, cy), det(bx, by, ax, ay)
	su := side(u, bx, by, cx, cy)
	sv := side(v, cx, cy, ax, ay)
	sw := side(w, ax, ay, bx, by)
	if su == 0 || su != sv || sv != sw {
		return 0, false, false
	}
	whole := u + v + w
	if whole == 0 {
		return 0, false, false // seen edge on
	}
	d := (u*az + v*bz + w*cz) / whole
	return d, whole > 0, d > near && d < far
}

// shear is where a corner lands once the ray is the z axis: x and y
// across the ray, and z along it, in the ray's own distances. The
// products are rounded before they are taken away, so every face that
// shares a corner puts it in the same place.
func (q *query) shear(p *[3]float64) (x, y, z float64) {
	a := [3]float64{p[0] - q.o[0], p[1] - q.o[1], p[2] - q.o[2]}
	x = a[q.kx] - float64(q.sx*a[q.kz])
	y = a[q.ky] - float64(q.sy*a[q.kz])
	return x, y, q.sz * a[q.kz]
}

// det is ax*by - ay*bx with its sign exact: Kahan's way, where a fused
// multiply recovers what rounding the first product lost. It is zero only
// when the two products are exactly equal.
func det(ax, ay, bx, by float64) float64 {
	w := float64(ay * bx)
	lost := math.FMA(-ay, bx, w)
	return math.FMA(ax, by, -w) + lost
}

// side is the sign of an edge's determinant e, for the edge from p to q:
// its own sign, and when it is zero, the sign it takes once the ray steps
// an infinitesimal way along x and an infinitely smaller one along y. The
// face across the edge runs it from q to p, and gets the other sign.
func side(e, px, py, qx, qy float64) int {
	switch {
	case e > 0:
		return 1
	case e < 0:
		return -1
	case qy > py:
		return 1
	case qy < py:
		return -1
	case px > qx:
		return 1
	case px < qx:
		return -1
	}
	return 0
}
