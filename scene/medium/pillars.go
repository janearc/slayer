package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Pillars is a nebula's pillars of gas, as the eagle's: Count columns standing
// on a wall of gas along the bottom, up y, none taller than Height, spread
// across Width, each tapering as it rises, rounded at the tip, and eaten into
// by fractal noise so the edges are ragged.
//
// The columns and the wall are Thick; a thin fog fills the rest of the box.
// Seed makes a nebula its own.
//
// Filaments is a dressing: fine streaks drawn up the columns by a finer noise
// stretched along them, as the pictures show and the gas at this scale does
// not, zero none and one all streaks.
type Pillars struct {
	Count         int
	Height, Width float64
	Thick         float64
	Seed          uint64
	Filaments     float64
}

// column is one of the pillars: where it stands, how tall, how wide
// at the foot.
type column struct {
	x, z, height, base float64
}

// columns is the pillars, each placed, sized and tapered by hash; a count of
// zero or less is none.
func (p Pillars) columns() []column {
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

// Tips is where the pillars end: the top of each column, where the
// young stars are found.
func (p Pillars) Tips() []maths.Vec {
	var out []maths.Vec
	for _, c := range p.columns() {
		out = append(out, maths.V(c.x, c.height, c.z))
	}
	return out
}

// Bounds is the box the pillars stand in, as deep as it is wide.
func (p Pillars) Bounds() maths.Box {
	w := p.Width / 2
	return maths.Box{Min: maths.V(-w, 0, -w), Max: maths.V(w, p.Height, w)}
}

// fog is the thin gas between the pillars, as a share of Thick: enough
// to catch a little starlight, not enough to grey the picture.
const fog = 0.003

// Density is the wall, the columns and the fog, whichever is most.
func (p Pillars) Density(at maths.Vec) float64 {
	if p.Height <= 0 || p.Width <= 0 || p.Count <= 0 {
		return 0
	}
	w := p.Width / 2
	if at.Y < 0 || at.Y > p.Height || math.Abs(at.X) > w ||
		math.Abs(at.Z) > w {
		return 0
	}
	most := fog
	// the wall along the bottom, a fifth of the height, ragged on top
	ragged := 0.06 * p.Height * noise.Fractal(p.Seed, at.X/p.Height*3, 0,
		at.Z/p.Height*3, 3)
	if top := 0.2*p.Height + ragged; at.Y < top {
		edge := maths.Smooth(top-0.08*p.Height, top, at.Y)
		most = math.Max(most, 1-edge)
	}
	// the columns, each its own
	for i, c := range p.columns() {
		h, base := c.height, c.base
		if at.Y > h {
			continue
		}
		r := base * (1 - 0.35*at.Y/h)
		// the tip, a dome a radius deep
		if from := h - r; at.Y > from {
			u := (at.Y - from) / r
			r *= math.Sqrt(math.Max(0, 1-u*u))
		}
		rho := math.Hypot(at.X-c.x, at.Z-c.z)
		// eaten into by noise: the edge wanders, the tip is knobbed
		f := 1.5 / base
		seed := p.Seed + 10 + uint64(i)
		x, y, z := at.X*f, at.Y*f, at.Z*f
		n := noise.Fractal(seed, x, y, z, 4)
		// built out by the noise, by up to about a whole base and in
		// by a seventh, half a base out on the whole, so the columns
		// are fat, solid and knobbed rather than wispy
		rho += (n - 0.5) * 0.9 * base
		got := 1 - maths.Smooth(r-0.3*base, r, rho)
		if p.Filaments > 0 && got > 0 {
			// streaks up the column: a noise stretched three
			// times along y, ragged at every scale, thinning
			// the gas where it is low
			g := 3 / base
			m := noise.Fractal(seed+50, at.X*g, at.Y*g/3, at.Z*g, 4)
			got *= 1 - p.Filaments*math.Min(1, math.Max(0, 0.5-m))
		}
		most = math.Max(most, got)
	}
	return p.Thick * most
}
