package skylight

import "math"

// Suns is a span of the sun's zenith at the ground, in radians, that a
// walk is for: the bins an observer at the ground under such a sun
// reads.
type Suns struct {
	// From and To are the least and the most of the span, from nought
	// to pi.
	From, To float64
}

// everySun is the span of every sun: a walk of the whole planet.
var everySun = Suns{From: none, To: math.Pi}

// a band is the part of the disc the top sphere shows the sun that a
// walk starts its photons on: between two distances from the line
// through the planet's centre toward the sun, in metres.
type band struct {
	inner, outer float64
}

// bandOf is the part of the disc whose light can reach an observer at
// the ground under a sun in the span.
//
// The farthest air such an observer sees is on the top's horizon, an
// arc of acos(radius / top) away, so every point of air it sees has a
// sun within that arc of the span. A photon keeps its distance from
// the line to the sun until it first scatters, so a point of radius r
// whose sun is at zenith z is reached from a distance r sin z.
//
// So the band runs from the ground's radius times the sine of the
// span's least zenith less the arc, to the top's times the sine of its
// most plus the arc, or to the top where that passes the horizon.
//
// What it leaves out: light first scattered beyond the arc that a
// later scattering brings into it.
func bandOf(made *air, suns Suns) band {
	arc := math.Acos(made.radius / made.top)
	low, high := max(suns.From-arc, none), min(suns.To+arc, math.Pi)
	inner := made.radius * min(math.Sin(low), math.Sin(high))
	if high >= math.Pi/two {
		return band{inner: inner, outer: made.top}
	}
	return band{inner: inner, outer: made.top * math.Sin(high)}
}

// area is the band's area, square metres: what its photons share.
func (on band) area() float64 {
	return math.Pi * (on.outer*on.outer - on.inner*on.inner)
}

// reach is a distance from the line to the sun drawn evenly over the
// band's area, from a uniform draw from nought to one.
func (on band) reach(draw float64) float64 {
	inner := on.inner * on.inner
	return math.Sqrt(inner + draw*(on.outer*on.outer-inner))
}
