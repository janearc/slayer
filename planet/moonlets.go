package planet

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths/noise"
)

// gravitation is newton's constant, codata 2018, cubic metres a
// kilogram a second squared
const gravitation = 6.67430e-11

// iceDensity is a moonlet's density, kilograms a cubic metre: about
// water ice's, picked; saturn's ring moonlets measure lower, porous.
const iceDensity = 900.0

// gapReach is how many hill radii either side of its orbit a moonlet
// clears: pan holds the encke gap 325 km wide at about 19 km of hill
// radius, daphnis the keeler gap 42 km wide at about 5, between 4 and
// 9 either side. picked between them.
const gapReach = 6.0

// wake is how many hill radii long a propeller's cleared lobes reach
// along the orbit, either side of the moonlet: picked.
const wake = 400.0

// a moonlet is a body in a ring: how far out it orbits, where round it
// is now, in radians, its radius, and its hill radius, all in metres,
// and whether it clears its whole orbit or only a propeller.
type moonlet struct {
	reach, angle, radius, hill float64
	opens                      bool
}

// Cleared is the ring with moonlets in it, so many, seeded, their radii
// from smallest to largest in metres by a power law of a slope, under a
// planet of a mass in kilograms.
//
// a moonlet of the radius opens or more clears a gap round its whole
// orbit; a smaller one, a propeller about itself. in saturn's rings opens
// is a few kilometres, daphnis's size; a larger one makes long propellers
// of moonlets that would gap.
func (ring Ring) Cleared(seed uint64, count int, smallest, largest,
	slope, opens, mass float64) (Ring, error) {
	if count < 0 || !(smallest > 0) || !(largest >= smallest) ||
		!(slope > 1) || !(opens > 0) || !(mass > 0) {
		return ring, fmt.Errorf("%d moonlets from %g to %g m, slope %g, "+
			"opening at %g, about %g kg", count, smallest, largest, slope,
			opens, mass)
	}
	ring.moonlets = make([]moonlet, count)
	for i := range ring.moonlets {
		radius := sized(noise.Hash(seed, int64(3*i)), smallest, largest,
			slope)
		reach := ring.inner + noise.Hash(seed, int64(3*i+1))*
			(ring.outer-ring.inner)
		weight := iceDensity * 4 / 3 * math.Pi * radius * radius * radius
		ring.moonlets[i] = moonlet{reach: reach,
			angle:  2 * math.Pi * noise.Hash(seed, int64(3*i+2)),
			radius: radius, hill: reach * math.Cbrt(weight/(3*mass)),
			opens: radius >= opens}
	}
	return ring, nil
}

// sized is a radius drawn from a power law of a slope between two sizes, for a
// draw from zero to one: many small, few large.
func sized(draw, smallest, largest, slope float64) float64 {
	low := math.Pow(smallest, 1-slope)
	high := math.Pow(largest, 1-slope)
	return math.Pow(low+draw*(high-low), 1/(1-slope))
}

// cleared is how much of a ring the moonlets leave at a point of its plane:
// zero in a gap or a propeller, rising smoothly to one at its edge.
func (ring Ring) cleared(x, z float64) float64 {
	if len(ring.moonlets) == 0 {
		return 1
	}
	reach, angle := math.Hypot(x, z), math.Atan2(z, x)
	left := 1.0
	for _, each := range ring.moonlets {
		gap := gapReach * each.hill
		out := math.Abs(reach-each.reach) / gap
		if each.opens {
			left *= edged(out)
			continue
		}
		along := math.Remainder(angle-each.angle, 2*math.Pi) * each.reach
		length := wake * each.hill
		left *= edged(math.Hypot(out, along/length))
	}
	return left
}

// edged is zero inside one, one past one and a half, smoothly between: a gap's
// soft edge.
func edged(apart float64) float64 {
	share := math.Max(0, math.Min(1, (apart-1)/0.5))
	return share * share * (3 - 2*share)
}

// Mass is the body's mass in kilograms, from its gravity at its radius.
func (body Body) Mass() float64 {
	return body.gravity * body.radius * body.radius / gravitation
}
