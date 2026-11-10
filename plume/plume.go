// Package plume models smoke carried above a fire by buoyant hot air. Measured
// fire-plume correlations set the mean column's rise, spreading, and slowing;
// eddies rise around that mean in eddies.go.
//
// The model assumes still air and a vertical column. Its finite bounding box
// truncates the plume; atmospheric stratification is not modelled.
package plume

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// the air's numbers, at the fire, and gravity: the standard's sea level
// for the air, which is near enough for a column's shape.
const (
	gravity  = 9.80665 // m s-2, standard gravity
	specific = 1.005   // kj kg-1 k-1, dry air at constant pressure
)

var errFire = errors.New("plume: a fire has a heat above nought, a " +
	"base wider than nought, a soot yield and a heat of combustion " +
	"above nought, an extinction of nought or more, air warmer than " +
	"nought and denser than nought, and a column taller than its start")

// Fire is what a plume is made from: the fire's heat release rate, its
// share of it carried by the hot air, the width of its base, how much
// soot it makes, what burning gives, how much light its soot stops,
// the air it burns in, and how high the column is drawn.
type Fire struct {
	Heat       float64 // kW, the whole heat release rate
	Convective float64 // the share of the heat the air carries up
	Base       float64 // m, the diameter of the fire's base
	Yield      float64 // g of soot for each g burned
	Combustion float64 // MJ given for each kg burned
	Extinction float64 // m2 of light stopped for each g of soot, 550 nm
	Air        float64 // k, the air's temperature
	Density    float64 // kg m-3, the air's density
	Top        float64 // m above the base, where the column is cut
	Seed       uint64  // which eddies, so two columns roll apart
}

// Column is a fire's plume, in the fire's own space: y up from the
// middle of its base, at a time.
type Column struct {
	fire   Fire
	start  float64 // m, where the plume region starts
	origin float64 // m, heskestad's virtual origin
	soot   float64 // g a second of soot the fire makes
	rises  float64 // m, where the soot begins, in the flame's tongues
	at     time.Duration
}

// New is a fire's plume. its soot rises out of the flame's tongues,
// mccaffrey's intermittent region, and is all there where his plume
// region starts.
func New(fire Fire) (*Column, error) {
	if !sane(fire) {
		return nil, errFire
	}
	start := plumeStarts * math.Pow(fire.Heat, 0.4)
	if !(fire.Top > start) {
		return nil, errFire
	}
	burned := fire.Heat / (fire.Combustion * 1000) // kg a second
	return &Column{fire: fire, start: start, origin: virtual(fire),
		rises: tonguesStart * math.Pow(fire.Heat, 0.4),
		soot:  fire.Yield * burned * 1000}, nil
}

// tonguesStart is where mccaffrey's intermittent region begins, the
// tongues above the continuous flame, as a height over the heat in kw
// to the two fifths: his table 1's weighted border, nbsir 79-1910, from
// the research data.
var tonguesStart = research.Value("tongues-from")

// risen is how much of the column's soot is there at a height: none
// below the tongues, all of it where the plume starts, and between, a
// smooth rise, so the smoke grows out of the flame with no edge. the
// rise's shape, smoothstep's, is picked.
func (column *Column) risen(height float64) float64 {
	if height >= column.start {
		return 1
	}
	share := (height - column.rises) / (column.start - column.rises)
	if !(share > 0) {
		return 0
	}
	return share * share * (3 - 2*share)
}

// above is a height, or where the plume starts if it is lower: below
// it the plume's relations do not hold, and the soot rising through
// the tongues takes the plume's shape where it starts.
func (column *Column) above(height float64) float64 {
	return math.Max(height, column.start)
}

// sane is a fire whose numbers can make a plume.
func sane(fire Fire) bool {
	positive := func(each ...float64) bool {
		for _, number := range each {
			if !(number > 0) || math.IsInf(number, 0) {
				return false
			}
		}
		return true
	}
	return positive(fire.Heat, fire.Convective, fire.Base, fire.Yield,
		fire.Combustion, fire.Air, fire.Density, fire.Top) &&
		fire.Convective <= 1 && fire.Extinction >= 0 &&
		!math.IsInf(fire.Extinction, 0)
}

// plumeStarts is where mccaffrey's plume region begins, as a height
// over the heat release rate in kw to the two fifths: his table 1's
// weighted border, from the research data.
//
// the table gives that border two ways, 0.201 by temperature and 0.195
// by speed, and weighs them to 0.198.
var plumeStarts = research.Value("plume-from")

// virtual is heskestad's virtual origin, the height the plume seems to
// rise from, as nist tn 1984 gives it (its equation 2): over the base,
// 1.4 times the fire's dimensionless heat to the two fifths, less 1.02.
func virtual(fire Fire) float64 {
	scale := fire.Density * specific * fire.Air *
		math.Sqrt(gravity*fire.Base) * fire.Base * fire.Base
	return fire.Base * (-1.02 + 1.4*math.Pow(fire.Heat/scale, 0.4))
}

// Speed is the plume's speed up its middle at a height, in m a second:
// mccaffrey's plume region, his plume speed times the heat release rate
// in kw to the third over the height to the third.
func (column *Column) Speed(height float64) float64 {
	return plumeSpeed * math.Cbrt(column.fire.Heat/height)
}

// plumeSpeed is mccaffrey's speed up the plume's middle, his table 1's
// weighted average, nbsir 79-1910, from the research data.
var plumeSpeed = research.Value("plume-speed")

// Rise is how much warmer than the air the plume's middle is at a
// height, in kelvin: heskestad's, as nist tn 1984 gives it (its
// equation 1), from the convective heat and the virtual origin.
func (column *Column) Rise(height float64) float64 {
	fire := column.fire
	air := math.Cbrt(fire.Air / (gravity * specific * specific *
		fire.Density * fire.Density))
	carried := fire.Convective * fire.Heat
	return 9.1 * air * math.Pow(carried, 2.0/3) *
		math.Pow(height-column.origin, -5.0/3)
}

// Radius is where the plume is half as much warmer than the air as its
// middle is, at a height, in m: heskestad's 0.12 times the square root
// of how much warmer the middle is, as a ratio, times the height above
// the virtual origin.
//
// recalled, not read: it is heskestad's in the sfpe handbook, which is
// copyrighted, and no public text that states it was found (a nrc
// licensee's deck names the method, a nist hangar report uses it).
// until it is read, the column's width rests on memory.
func (column *Column) Radius(height float64) float64 {
	warm := (column.fire.Air + column.Rise(height)) / column.fire.Air
	return 0.12 * math.Sqrt(warm) * (height - column.origin)
}

// Density is the soot's extinction at a point at the column's time, per
// metre at 550 nm: the mean's, rolled by the column's eddies.
func (column *Column) Density(p maths.Vec) float64 {
	if p.Y < column.rises || p.Y > column.fire.Top {
		return 0
	}
	moved, thicker := column.rolled(p)
	return column.Mean(moved) * thicker
}

// Mean is the soot's extinction at a point as the time's mean, per
// metre at 550 nm: its extinction for each gram times how many grams a
// cubic metre.
//
// the soot rises with the air, so through each height a second passes
// all the soot the fire makes in a second.
//
// across, the soot and the speed both fall to half at the plume's
// radius as a bell, so the soot at the middle is two log two times the
// soot a second, over the speed there times pi times the radius
// squared. the speed's bell as wide as the warmth's is picked.
//
// below where the plume starts, the soot has its shape there, as much
// of it as has risen.
func (column *Column) Mean(p maths.Vec) float64 {
	if p.Y < column.rises || p.Y > column.fire.Top {
		return 0
	}
	height := column.above(p.Y)
	radius, speed := column.Radius(height), column.Speed(height)
	middle := 2 * math.Ln2 * column.soot / (speed * math.Pi * radius *
		radius)
	across := (p.X*p.X + p.Z*p.Z) / (radius * radius)
	return column.fire.Extinction * middle * math.Exp2(-across) *
		column.risen(p.Y)
}

// Bounds is the box the column is drawn in: from where the soot rises
// to the top, and three radii out at the top, past which its mean soot
// is under a five hundredth of its middle's, and as far again as its
// eddies push it.
func (column *Column) Bounds() maths.Box {
	wide := (3 + sway) * column.Radius(column.fire.Top)
	return maths.Box{Min: maths.V(-wide, column.rises, -wide),
		Max: maths.V(wide, column.fire.Top, wide)}
}

// Tint is how many times its extinction at 550 nm each of the eye's
// three wavelengths meets: the same soot as the flame's, so the
// flame's tilt.
func (column *Column) Tint() maths.Light {
	return flame.Tilt()
}
