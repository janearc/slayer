package planet

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
	"github.com/janearc/slayer/maths/noise"
)

// iceDensity is a moonlet's density, kilograms a cubic metre, from
// picks.yaml.
var iceDensity = pick("moonlet-density")

// gapReach is how many hill radii either side of its orbit a moonlet
// clears, from picks.yaml.
var gapReach = pick("moonlet-gap-reach")

// wake is how many hill radii long a propeller's cleared lobes reach along
// the orbit, either side of the moonlet, from picks.yaml.
var wake = pick("moonlet-wake")

// mostMoonlets is the most moonlets a ring may be given: a ray's share of
// the ring loops over every one, so the count is held to what a picture can
// afford. its value is in picks.yaml.
var mostMoonlets = pickCount("moonlet-most")

// drawsPerMoonlet is defined as how many draws of a seeded hash a moonlet
// takes: its radius, its reach and its angle.
const drawsPerMoonlet = 3

// The sphere's volume is four thirds pi times its radius cubed, and a
// moonlet's hill radius takes the cube root of its mass over three of its
// planet's: defined by the formulas.
const (
	// volumeNumerator is defined as the four of a sphere's four thirds.
	volumeNumerator = 4
	// volumeDenominator is defined as the three of a sphere's four thirds.
	volumeDenominator = 3
	// hillMassFactor is defined as the three a planet's mass is taken times
	// in the hill radius.
	hillMassFactor = 3
)

// a moonlet is a body in a ring: how far out it orbits, where round it
// is now, in radians, its radius, and its hill radius, all in metres,
// and whether it clears its whole orbit or only a propeller.
type moonlet struct {
	reach, angle, radius, hill float64
	opens                      bool
}

// Cleared is the ring with moonlets in it, so many, seeded, their radii
// from smallest to largest in metres by a power law of a slope, under a
// planet of a mass in kilograms. No more than mostMoonlets may be asked
// for, and no size, slope or mass may be endless.
//
// a moonlet of the radius opens or more clears a gap round its whole
// orbit; a smaller one, a propeller about itself. a larger opens makes
// long propellers of moonlets that would gap.
func (ring Ring) Cleared(seed uint64, count int, smallest, largest,
	slope, opens, mass float64) (Ring, error) {
	if count < nothing || count > mostMoonlets || !(smallest > nothing) ||
		!(largest >= smallest) || !(slope > whole) || !(opens > nothing) ||
		!(mass > nothing) || endless(largest) || endless(slope) ||
		endless(mass) {
		return ring, fmt.Errorf("%d moonlets from %g to %g m, slope %g, "+
			"opening at %g, about %g kg", count, smallest, largest, slope,
			opens, mass)
	}
	ring.moonlets = make([]moonlet, count)
	for i := range ring.moonlets {
		draw := noise.Hash(seed, int64(drawsPerMoonlet*i))
		radius := sized(draw, smallest, largest, slope)
		reach := ring.inner + noise.Hash(seed,
			int64(drawsPerMoonlet*i+adjacent))*(ring.outer-ring.inner)
		weight := iceDensity * volumeNumerator / volumeDenominator *
			math.Pi * radius * radius * radius
		ring.moonlets[i] = moonlet{reach: reach,
			angle: circle * noise.Hash(seed,
				int64(drawsPerMoonlet*i+double)),
			radius: radius,
			hill:   reach * math.Cbrt(weight/(hillMassFactor*mass)),
			opens:  radius >= opens}
	}
	return ring, nil
}

// sized is a radius drawn from a power law of a slope between two sizes, for a
// draw from zero to one: many small, few large.
func sized(draw, smallest, largest, slope float64) float64 {
	low := math.Pow(smallest, whole-slope)
	high := math.Pow(largest, whole-slope)
	return math.Pow(low+draw*(high-low), whole/(whole-slope))
}

// cleared is how much of a ring the moonlets leave at a point of its plane:
// zero in a gap or a propeller, rising smoothly to one at its edge.
func (ring Ring) cleared(x, z float64) float64 {
	if len(ring.moonlets) == nothing {
		return whole
	}
	reach, angle := math.Hypot(x, z), math.Atan2(z, x)
	var left float64 = whole
	for _, each := range ring.moonlets {
		gap := gapReach * each.hill
		out := math.Abs(reach-each.reach) / gap
		if each.opens {
			left *= edged(out)
			continue
		}
		along := math.Remainder(angle-each.angle, circle) * each.reach
		length := wake * each.hill
		left *= edged(math.Hypot(out, along/length))
	}
	return left
}

// edgeWidth is how wide a gap's soft edge is, in the widths of the gap; its
// value is in picks.yaml.
var edgeWidth = pick("moonlet-edge-width")

// edged is zero inside one, one past one and a half, smoothly between: a gap's
// soft edge.
func edged(apart float64) float64 {
	share := math.Max(nothing, math.Min(whole, (apart-whole)/edgeWidth))
	return eased(share)
}

// Mass is the body's mass in kilograms, from its gravity at its radius and
// the gravitational constant celestial holds.
func (body Body) Mass() float64 {
	return body.gravity * body.radius * body.radius / si.Gravitation()
}
