package gas

import (
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// Optical is a medium that knows its gas: what a tracer asks a volume's
// medium for, to scatter and absorb by band rather than by one density.
type Optical interface {
	medium.Medium
	// Gas is the medium's optics, per unit of the medium's own space,
	// at unit density; the density at a point scales them.
	Gas() Gas
}

// Of is a medium of a gas: the medium gives the shape and the density from
// place to place, the gas what light meets there. Unit is how many metres one
// of the medium's units is, since a gas's numbers are per metre; zero is one.
type Of struct {
	Shape medium.Medium
	In    Gas
	Unit  float64
}

// Density is the shape's.
func (o Of) Density(p maths.Vec) float64 { return o.Shape.Density(p) }

// Bounds is the shape's.
func (o Of) Bounds() maths.Box { return o.Shape.Bounds() }

// Span is the shape's own when it has one, else its box's.
func (o Of) Span(r maths.Ray, near, far float64) (enter, leave float64,
	ok bool) {
	if sp, has := o.Shape.(medium.Spanner); has {
		return sp.Span(r, near, far)
	}
	return o.Shape.Bounds().Hit(r, near, far)
}

// Tick is the shape ticked, the gas kept.
func (o Of) Tick(t time.Duration) medium.Medium {
	if moving, ok := o.Shape.(medium.Timed); ok {
		return Of{Shape: moving.Tick(t), In: o.In, Unit: o.Unit}
	}
	return o
}

// Gas is the gas per unit of the medium's space.
func (o Of) Gas() Gas {
	unit := o.Unit
	if unit <= 0 {
		unit = 1
	}
	return o.In.Per(unit)
}

// Per is the gas with its numbers per a unit of so many metres.
func (g Gas) Per(metres float64) Gas {
	out := g
	out.Ranges = map[radiation.Range]Optics{}
	for r, o := range g.Ranges {
		out.Ranges[r] = Optics{Absorb: o.Absorb * metres,
			Scatter: o.Scatter * metres}
	}
	if g.Visible != nil {
		was := g.Visible
		out.Visible = func(nm float64) (absorb, scatter float64) {
			a, s := was(nm)
			return a * metres, s * metres
		}
	}
	if g.Lines != nil {
		out.Lines = map[float64]float64{}
		for nm, k := range g.Lines {
			out.Lines[nm] = k * metres
		}
	}
	return out
}
