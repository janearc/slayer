package rings

import (
	"fmt"
	"math"
)

// Disc is a disc of no thickness in the hole's equatorial plane, from an
// inner radius to an outer, in masses.
type Disc struct {
	inner, outer float64
}

// NewDisc is a disc from inner to outer masses. It refuses an inner
// radius that is not past the horizon, two masses, and an outer that is
// not past the inner or is not finite.
func NewDisc(inner, outer float64) (Disc, error) {
	if !(inner > horizon) {
		return Disc{}, fmt.Errorf("rings: a disc from %v masses "+
			"starts inside the horizon: give an inner radius past %v",
			inner, horizon)
	}
	if !(outer > inner) || math.IsInf(outer, 0) {
		return Disc{}, fmt.Errorf("rings: a disc out to %v masses "+
			"does not reach past its inner edge at %v: give a finite "+
			"outer radius past it", outer, inner)
	}
	return Disc{inner: inner, outer: outer}, nil
}

// horizon is the radius of a still hole's horizon, in masses: two,
// defined by the schwarzschild metric.
const horizon = 2

// Inner is the disc's inner radius, in masses.
func (disc Disc) Inner() float64 { return disc.inner }

// Outer is the disc's outer radius, in masses.
func (disc Disc) Outer() float64 { return disc.outer }

// holds is whether a radius is on the disc.
func (disc Disc) holds(radius float64) bool {
	return radius >= disc.inner && radius <= disc.outer
}

// Band is the angles off the hole's centre, above it, at which the eye
// sees one order of the disc's image: from near to far, in radians.
type Band struct {
	order     int
	near, far float64
}

// Order is the band's order: nought for the direct image, one for the
// image made by light that went half a turn more round the hole.
func (band Band) Order() int { return band.order }

// Near and Far are the band's edges, as angles off the hole's centre.
func (band Band) Near() float64 { return band.near }

// Far is the band's outer edge, as an angle off the hole's centre.
func (band Band) Far() float64 { return band.far }

// Width is the band's width, as an angle on the eye's sky.
func (band Band) Width() float64 { return band.far - band.near }

// Bands is, for each order from nought up to orders less one, the band
// of angles above the hole's centre at which the eye sees that image of
// the disc. It refuses fewer than one order, and says which order the
// search found no ray for: an image later than the third lies nearer
// the shadow's edge than the search looks.
func (lens Lens) Bands(disc Disc, orders int) ([]Band, error) {
	if orders < one {
		return nil, fmt.Errorf("rings: %d orders of image is none: "+
			"ask for one or more", orders)
	}
	edge := lens.Edge()
	angles, lands := lens.scan(edge, disc, orders)
	bands := make([]Band, orders)
	for order := range orders {
		first, last := span(lands[order])
		if first < none {
			return nil, fmt.Errorf("rings: no ray in the search "+
				"lands order %d on the disc: it lies nearer the "+
				"shadow's edge than %v radians", order, nearest)
		}
		bands[order] = Band{order: order,
			near: lens.boundary(angles, first, order, disc, true),
			far:  lens.boundary(angles, last, order, disc, false)}
	}
	return bands, nil
}

// scan sends the search's rays out past the shadow's edge, spaced in
// the logarithm of the angle past it, and says for each order which
// rays land it.
func (lens Lens) scan(edge float64, disc Disc, orders int) ([]float64,
	[][]bool) {
	angles := make([]float64, scanRays)
	lands := make([][]bool, orders)
	for order := range orders {
		lands[order] = make([]bool, scanRays)
	}
	ratio := math.Log(widest / nearest)
	for each := range scanRays {
		share := float64(each) / float64(scanRays-one)
		angles[each] = edge + nearest*math.Exp(share*ratio)
		path := lens.Follow(angles[each])
		for order := range orders {
			lands[order][each] = path.Lands(order, disc)
		}
	}
	return angles, lands
}

// span is the first and last ray that lands, or minus one for both
// when none does.
func span(lands []bool) (first, last int) {
	first, last = -one, -one
	for each, landed := range lands {
		if !landed {
			continue
		}
		if first < none {
			first = each
		}
		last = each
	}
	return first, last
}

// boundary is a band's edge, halved toward between the ray that lands
// at an index and its neighbour that does not: the one before it for
// the near edge, the one after for the far. A band that reaches the
// search's own end has its edge there.
func (lens Lens) boundary(angles []float64, at, order int, disc Disc,
	near bool) float64 {
	other := at + one
	if near {
		other = at - one
	}
	if other < none || other >= len(angles) {
		return angles[at]
	}
	in, out := angles[at], angles[other]
	for range halvings {
		mid := (in + out) / two
		if lens.Follow(mid).Lands(order, disc) {
			in = mid
			continue
		}
		out = mid
	}
	return in
}
