package shape

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// lumpy is a ball with bumps on it, made of rings of slices: two faces
// for every slice of every ring, and one at each pole.
func lumpy(slices, rings int) ([]maths.Vec, [][3]int) {
	vs := []maths.Vec{maths.V(0, 1, 0)}
	for i := 1; i < rings; i++ {
		th := math.Pi * float64(i) / float64(rings)
		for j := range slices {
			ph := 2 * math.Pi * float64(j) / float64(slices)
			r := 1 + 0.1*math.Sin(5*th)*math.Cos(7*ph)
			vs = append(vs, maths.V(r*math.Sin(th)*math.Cos(ph),
				r*math.Cos(th), r*math.Sin(th)*math.Sin(ph)))
		}
	}
	vs = append(vs, maths.V(0, -1, 0))
	ring := func(i, j int) int { return 1 + (i-1)*slices + j%slices }
	south := len(vs) - 1
	var fs [][3]int
	for j := range slices {
		fs = append(fs, [3]int{0, ring(1, j+1), ring(1, j)},
			[3]int{south, ring(rings-1, j), ring(rings-1, j+1)})
		for i := 1; i < rings-1; i++ {
			a, b := ring(i, j), ring(i, j+1)
			c, d := ring(i+1, j+1), ring(i+1, j)
			fs = append(fs, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	return vs, fs
}

// soup is n faces of every size from a hundredth to four tenths, strewn
// through the cube from -1 to 1, overlapping as they fall.
func soup(rng *rand.Rand, n int) ([]maths.Vec, [][3]int) {
	var vs []maths.Vec
	var fs [][3]int
	for range n {
		mid := within(rng, 1)
		size := 0.01 * math.Pow(40, rng.Float64())
		for range 3 {
			vs = append(vs, mid.Add(anyway(rng).Scale(size)))
		}
		fs = append(fs, [3]int{len(vs) - 3, len(vs) - 2, len(vs) - 1})
	}
	return vs, fs
}

// every is the nearest face the ray meets, found by asking every face.
func every(m *Mesh, r maths.Ray, near, far float64) (float64, int, bool) {
	q, best, which := ready(r), far, -1
	for i := range m.tris {
		if t, _, ok := q.meet(&m.tris[i], near, best); ok {
			best, which = t, i
		}
	}
	if which < 0 {
		return 0, -1, false
	}
	return best, which, true
}

// nearestOfAll is how far p is from the nearest face, squared, found by
// asking every face.
func nearestOfAll(m *Mesh, p maths.Vec) float64 {
	best := math.Inf(1)
	for _, tr := range m.tris {
		a, b, c := vec(tr.p[0]), vec(tr.p[1]), vec(tr.p[2])
		w := closest(p, a, b, c)
		d := a.Scale(w[0]).Add(b.Scale(w[1])).Add(c.Scale(w[2]))
		best = math.Min(best, d.Sub(p).Dot(d.Sub(p)))
	}
	return best
}

// benchRays are rays from all round a ball, aimed into it.
func benchRays() []maths.Ray {
	rng := rand.New(rand.NewPCG(14, 15))
	rays := make([]maths.Ray, 4096)
	for i := range rays {
		rays[i] = aimed(anyway(rng).Scale(3), inBall(rng, 1.1))
	}
	return rays
}

// BenchmarkMeshHit is rays a second against a hundred thousand faces.
func BenchmarkMeshHit(b *testing.B) {
	vs, fs := lumpy(250, 201)
	m := made(b, vs, fs, nil, false)
	rays, n := benchRays(), 0
	for b.Loop() {
		m.Hit(rays[n%len(rays)], 0, forever)
		n++
	}
	b.ReportMetric(float64(n)/b.Elapsed().Seconds(), "rays/s")
	b.ReportMetric(float64(len(fs)), "faces")
}

// BenchmarkMeshNormal is normals a second, where the rays met the faces.
func BenchmarkMeshNormal(b *testing.B) {
	vs, fs := lumpy(250, 201)
	m := made(b, vs, fs, nil, false)
	var at []maths.Vec
	for _, r := range benchRays() {
		if t, ok := m.Hit(r, 0, forever); ok {
			at = append(at, r.Point(t))
		}
	}
	n := 0
	for b.Loop() {
		m.Normal(at[n%len(at)])
		n++
	}
	b.ReportMetric(float64(n)/b.Elapsed().Seconds(), "normals/s")
}

// BenchmarkNewMesh is checking and building a hundred thousand faces.
func BenchmarkNewMesh(b *testing.B) {
	vs, fs := lumpy(250, 201)
	for b.Loop() {
		made(b, vs, fs, nil, false)
	}
}
