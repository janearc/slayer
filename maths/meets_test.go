package maths

import (
	"math"
	"math/rand"
	"testing"
)

// pick draws a number of a kind chosen to make boxes and rays that touch,
// graze, run along a face, or sit at the edge of what a float holds.
func pick(r *rand.Rand) float64 {
	switch r.Intn(9) {
	case 0:
		return float64(r.Intn(7) - 3) // a grid, so faces and origins meet exactly
	case 1:
		return float64(r.Intn(7)-3) / 4
	case 2:
		return 0
	case 3:
		return r.NormFloat64() * 1e-9
	case 4:
		return r.NormFloat64() * 1e9
	case 5:
		return [...]float64{math.Inf(1), math.Inf(-1), math.NaN(),
			math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
			math.MaxFloat64, -math.MaxFloat64, 1e-310, 1e300, -1e300,
			1e-300}[r.Intn(11)]
	default:
		return r.NormFloat64() * 3
	}
}

// For every box, ray, near and far a seeded draw makes, Meets answers as Hit
// does: the grid draws make rays that graze faces, edges and corners, and
// the others make zero, tiny, huge, infinite and NaN numbers.
func TestMeetsIsHitsAnswerForEveryRay(t *testing.T) {
	r := rand.New(rand.NewSource(20260910))
	const cases = 3_000_000
	hit := 0
	for i := range cases {
		var box Box
		var ray Ray
		// most draws keep the numbers sane, so the interesting answers
		// (a hit, a graze) are many
		tame := r.Intn(4) != 0
		draw := func() float64 {
			if tame {
				if r.Intn(2) == 0 {
					return float64(r.Intn(9)-4) / 2
				}
				return r.NormFloat64() * 2
			}
			return pick(r)
		}
		a, b := Vec{draw(), draw(), draw()}, Vec{draw(), draw(), draw()}
		if r.Intn(8) != 0 {
			// a box the right way round, as a tree holds
			box = Box{Min: V(min(a.X, b.X), min(a.Y, b.Y), min(a.Z, b.Z)),
				Max: V(max(a.X, b.X), max(a.Y, b.Y), max(a.Z, b.Z))}
			if math.IsNaN(a.X + b.X + a.Y + b.Y + a.Z + b.Z) {
				box = Box{Min: a, Max: b}
			}
		} else {
			box = Box{Min: a, Max: b}
		}
		dir := Vec{draw(), draw(), draw()}
		for _, axis := range []*float64{&dir.X, &dir.Y, &dir.Z} {
			if r.Intn(3) == 0 {
				*axis = 0
			}
		}
		ray = Ray{At: Vec{draw(), draw(), draw()}, Dir: dir}
		near, far := 0.0, math.Inf(1)
		switch r.Intn(4) {
		case 0:
			near, far = draw(), draw()
		case 1:
			near, far = 0, math.Abs(draw())
		case 2:
			far = 1e9
		}
		_, _, want := box.Hit(ray, near, far)
		if want {
			hit++
		}
		if got := box.Meets(ray, ray.Dir.Inverse(), near, far); got != want {
			t.Fatalf("case %d: box %v, ray %v, near %v, far %v: Meets "+
				"says %v and Hit says %v", i, box, ray, near, far, got,
				want)
		}
	}
	if hit < cases/20 || hit > cases-cases/20 {
		t.Errorf("%d of %d draws hit: the draws do not test both answers",
			hit, cases)
	}
}

// A ray that runs along a face of a box, or touches an edge or a corner,
// is met or missed as Hit says, whichever way the numbers round.
func TestMeetsAgreesAtTheFacesOfAUnitBox(t *testing.T) {
	box := Box{Min: V(0, 0, 0), Max: V(1, 1, 1)}
	steps := []float64{-1, -0.5, 0, 0.1, 0.5, 1, 1.5, 3, 1.0 / 3, 0.7}
	for _, ax := range steps {
		for _, ay := range steps {
			for _, dx := range steps {
				for _, dy := range steps {
					ray := Ray{At: V(ax, ay, -2), Dir: V(dx, dy, 1)}
					for _, far := range []float64{1, 2, 3, math.Inf(1)} {
						_, _, want := box.Hit(ray, 0, far)
						if got := box.Meets(ray, ray.Dir.Inverse(), 0,
							far); got != want {
							t.Fatalf("ray %v, far %v: Meets %v, Hit %v",
								ray, far, got, want)
						}
					}
				}
			}
		}
	}
}

// A ray that reaches a corner of a box at a distance, asked up to that
// distance and the ones a few places either side of it, is where the
// quotients Hit takes and the products Meets takes differ in the last
// place: Meets answers as Hit does.
func TestMeetsAgreesAtACornerReachedAtTheFar(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	const cases = 400_000
	for i := range cases {
		ray := Ray{At: V(r.NormFloat64(), r.NormFloat64(), r.NormFloat64()),
			Dir: V(r.NormFloat64(), r.NormFloat64(), r.NormFloat64())}
		along := r.Float64()*4 + 0.1
		corner := ray.At.Add(ray.Dir.Scale(along))
		other := V(corner.X+r.NormFloat64(), corner.Y+r.NormFloat64(),
			corner.Z+r.NormFloat64())
		box := Box{Min: V(min(corner.X, other.X), min(corner.Y, other.Y),
			min(corner.Z, other.Z)), Max: V(max(corner.X, other.X),
			max(corner.Y, other.Y), max(corner.Z, other.Z))}
		for _, far := range []float64{along, math.Nextafter(along, 0),
			math.Nextafter(math.Nextafter(along, 10), 10),
			math.Nextafter(math.Nextafter(math.Nextafter(along, 0), 0), 0)} {
			_, _, want := box.Hit(ray, 0, far)
			if got := box.Meets(ray, ray.Dir.Inverse(), 0, far); got != want {
				t.Fatalf("case %d: box %v, ray %v, far %v: Meets says %v "+
					"and Hit says %v", i, box, ray, far, got, want)
			}
		}
	}
}
