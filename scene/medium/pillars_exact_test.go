package medium

import (
	"math"
	"math/rand"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// oldColumns is Pillars.columns as it was written.
func oldColumns(p Pillars) []column {
	if p.Count <= 0 {
		return nil
	}
	w := p.Width / 2
	base := p.Width / (float64(p.Count) * 3.2)
	out := make([]column, 0, p.Count)
	for i := range p.Count {
		k := int64(i)
		out = append(out, column{
			x: -w + p.Width*(float64(i)+0.5)/float64(p.Count) +
				(noise.Hash(p.Seed+1, k)-0.5)*base,
			z: (noise.Hash(p.Seed+2, k) - 0.5) * 2 * base,
			height: p.Height *
				(0.55 + 0.45*noise.Hash(p.Seed+3, k)),
			base: base,
		})
	}
	return out
}

// oldDensity is Pillars.Density as it was written, the columns placed
// afresh by the call.
func oldDensity(p Pillars, at maths.Vec) float64 {
	if p.Height <= 0 || p.Width <= 0 || p.Count <= 0 {
		return 0
	}
	w := p.Width / 2
	if at.Y < 0 || at.Y > p.Height || math.Abs(at.X) > w ||
		math.Abs(at.Z) > w {
		return 0
	}
	most := fog
	ragged := 0.06 * p.Height * noise.Fractal(p.Seed, at.X/p.Height*3, 0,
		at.Z/p.Height*3, 3)
	if top := 0.2*p.Height + ragged; at.Y < top {
		edge := maths.Smooth(top-0.08*p.Height, top, at.Y)
		most = math.Max(most, 1-edge)
	}
	for i, c := range oldColumns(p) {
		h, base := c.height, c.base
		if at.Y > h {
			continue
		}
		r := base * (1 - 0.35*at.Y/h)
		if from := h - r; at.Y > from {
			u := (at.Y - from) / r
			r *= math.Sqrt(math.Max(0, 1-u*u))
		}
		rho := math.Hypot(at.X-c.x, at.Z-c.z)
		f := 1.5 / base
		seed := p.Seed + 10 + uint64(i)
		x, y, z := at.X*f, at.Y*f, at.Z*f
		n := noise.Fractal(seed, x, y, z, 4)
		rho += (n - 0.5) * 0.9 * base
		got := 1 - maths.Smooth(r-0.3*base, r, rho)
		if p.Filaments > 0 && got > 0 {
			g := 3 / base
			m := noise.Fractal(seed+50, at.X*g, at.Y*g/3, at.Z*g, 4)
			got *= 1 - p.Filaments*math.Min(1, math.Max(0, 0.5-m))
		}
		most = math.Max(most, got)
	}
	return p.Thick * most
}

// the density and the tips are the old arithmetic bit for bit, for a
// literal and for a prepared value, over several pillars including counts of
// zero, one, nine (past the stack buffer) and below, widths and heights of
// either zero and below, and a field that is no number, at seeded random
// points, from many goroutines at once.
func TestPillarsAreTheOldArithmeticBitForBit(t *testing.T) {
	r := rand.New(rand.NewSource(20260910))
	negZero := math.Copysign(0, -1)
	all := []Pillars{
		{Count: 3, Height: 7, Width: 10, Thick: 1.6, Seed: 9, Filaments: 0.5},
		{Count: 5, Height: 3, Width: 4, Thick: 0.7, Seed: 1234567},
		{Count: 9, Height: 3, Width: 4, Thick: 0.7, Seed: 5, Filaments: 1},
		{Count: 20, Height: 3, Width: 8, Thick: 1, Seed: 6},
		{Count: 1, Height: 1, Width: 1, Thick: 1, Seed: 0, Filaments: 1},
		{Count: 0, Height: 1, Width: 1}, {Count: -2, Height: 1, Width: 1},
		{Count: 2, Height: 2, Width: -3, Thick: 1},
		{Count: 2, Height: -2, Width: 3, Thick: 1},
		{Count: 2, Height: 2, Width: negZero, Thick: 1},
		{Count: 2, Height: negZero, Width: 3, Thick: 1},
		{Count: 2, Height: math.NaN(), Width: 3, Thick: 1},
		{Count: 2, Height: 2, Width: math.NaN(), Thick: 1},
		{Count: 2, Height: 2, Width: 3, Thick: math.NaN(), Filaments: math.NaN()},
	}
	same := func(a, b float64) bool { return math.Float64bits(a) == math.Float64bits(b) }
	var wg sync.WaitGroup
	for _, lit := range all {
		for name, p := range map[string]Pillars{"literal": lit, "prepared": lit.Prepared()} {
			for g := 0; g < 3; g++ {
				wg.Add(1)
				seed := r.Int63()
				go func() {
					defer wg.Done()
					rr := rand.New(rand.NewSource(seed))
					for n := 0; n < 2000; n++ {
						at := maths.V((rr.Float64()-0.5)*1.2*lit.Width, rr.Float64()*1.1*lit.Height,
							(rr.Float64()-0.5)*1.2*lit.Width)
						if got, want := p.Density(at), oldDensity(lit, at); !same(got, want) {
							t.Errorf("%s %+v at %v: %v, was %v", name, lit, at, got, want)
							return
						}
					}
				}()
			}
			tips, want := p.Tips(), oldColumns(lit)
			if len(tips) != len(want) {
				t.Fatalf("%s %+v: %d tips, %d columns", name, lit, len(tips), len(want))
			}
			for i, c := range want {
				if !same(tips[i].X, c.x) || !same(tips[i].Y, c.height) || !same(tips[i].Z, c.z) {
					t.Errorf("%s %+v tip %d: %v", name, lit, i, tips[i])
				}
			}
		}
	}
	wg.Wait()
}

// a prepared value's columns do not change when Thick or Filaments do, and
// its density is the old arithmetic's with the new values; it does not see a
// later change to Count.
func TestAPreparedPillarsDoesNotDependOnThickOrFilaments(t *testing.T) {
	p := Pillars{Count: 3, Height: 7, Width: 10, Thick: 1.6, Seed: 9, Filaments: 0.5}.Prepared()
	want := append([]column(nil), p.held...)
	for _, thick := range []float64{0, 0.1, 5} {
		for _, fil := range []float64{0, 0.3, 1} {
			q := p
			q.Thick, q.Filaments = thick, fil
			for i, c := range want {
				if q.held[i] != c {
					t.Fatalf("column %d moved with Thick %v Filaments %v", i, thick, fil)
				}
			}
			at := maths.V(0.3, 1.2, -0.4)
			lit := Pillars{Count: 3, Height: 7, Width: 10, Thick: thick, Seed: 9, Filaments: fil}
			if got, wantD := q.Density(at), oldDensity(lit, at); math.Float64bits(got) != math.Float64bits(wantD) {
				t.Errorf("Thick %v Filaments %v: %v, was %v", thick, fil, got, wantD)
			}
		}
	}
	q := p
	q.Count = 1
	if got := len(q.Tips()); got != 3 {
		t.Errorf("a prepared value saw a later Count: %d tips", got)
	}
}
