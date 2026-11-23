package airlight

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
)

// Sky is the sky an air's fields give an observer at their ground, with
// the sun along one way: the light arriving along each way, worked out
// once for a table of ways and read between them.
//
// The table has a point at the centre of each of the fields' view and
// azimuth bins, and at each point the light's slope toward the next
// point, by view and by azimuth. Between points it is read as a cubic
// that meets each point's light and slope, so the light and its slope
// run on unbroken across every point.
//
// Each slope is Steffen's (1990): it is nought where the light turns,
// and never so steep that the cubic passes above or below the points at
// its ends. So the reading makes no light the fields do not hold, and a
// point that is a bin's noise is a soft bump, not a crease.
//
// The sky above the horizon and the light coming up from below are read
// apart. Past the lowest point above, the sky is held to the horizon;
// past the highest point below, so is the light from below. Past the
// first and last azimuth points, it is held to the sun's own vertical
// and to the one opposite, where the sky turns back on itself.
//
// Below the horizon it is the fields' own light there: what comes up
// from the ground and the air below the eye. It is not the horizon's
// held down, as moonlight's sky is, since that light is in the field and
// lights whatever faces down.
//
// At the zenith and the nadir the azimuth bins meet at a point, where
// light cannot change with azimuth. Inside the centre of each polar view
// bin the light read round the azimuths is drawn toward that bin's mean
// round them, by a weight of (1 - s) squared times (1 + s).
//
// s is the sine of the look's angle from the pole over that of the bin's
// centre. The weight is one at the pole, and nought at the centre with
// no slope there, where the reading joins the cubic without a corner.
// It is the one cubic in s that does that and leaves the part of the
// light that turns with azimuth s + s^2 - s^3 of the bin's.
//
// So near the pole that part grows as s, the look's distance from the
// pole, as the first turn of a smooth sky does, and at the slope a
// straight line from the pole to the bin's centre has. A weight of (1 -
// s) squared alone grows it twice as fast, and a ring a degree from
// Earth's zenith turned by half again what the air's light does.
type Sky struct {
	round           float64
	views, azimuths int
	table           []maths.Light
	upward, across  []maths.Light
	poles           [two]maths.Light
	poleSine        float64
}

// a Sky is one of the scene's skies.
var _ sky.Sky = (*Sky)(nil)

// NewSky is the sky of the fields with the sun along a way in the
// world's space, x east, y up and north toward minus z, at an exposure:
// what a watt a square metre a steradian is in the picture. It refuses
// fields walked for suns that are not this one's, in Holds's words: the
// bins it would read hold only part of their light.
func NewSky(fields *Fields, sun maths.Vec, exposure float64) (*Sky,
	error) {
	if err := fields.Holds(sun); err != nil {
		return nil, err
	}
	return newSky(fields, sun, exposure), nil
}

// newSky is NewSky of fields that hold the sun.
func newSky(fields *Fields, sun maths.Vec, exposure float64) *Sky {
	zenith := zenithOf(sun)
	looks := make([][]float64, len(fields.wavelengths))
	for wave := range looks {
		looks[wave] = fields.slice(wave, zenith)
	}
	made := &Sky{round: roundOf(sun), views: fields.views,
		azimuths: fields.azimuths,
		table:    make([]maths.Light, fields.views*fields.azimuths)}
	for point := range made.table {
		made.table[point] = fields.seen(func(wave int) float64 {
			return looks[wave][point]
		}).Picture(exposure)
	}
	made.upward, made.across = made.slopes()
	made.poles = [two]maths.Light{made.rowMean(none),
		made.rowMean(made.views - one)}
	made.poleSine = sineOf(centre(made.views-one, made.views)*two - one)
	return made
}

// rowMean is the mean of a view bin's points round every azimuth: the
// azimuth bins are of one width, so it is the mean over the bin's circle.
func (made *Sky) rowMean(row int) maths.Light {
	var sum maths.Light
	for column := range made.azimuths {
		sum = sum.Add(made.table[row*made.azimuths+column])
	}
	return sum.Scale(one / float64(made.azimuths))
}

// sineOf is the sine of an angle from its cosine, nought for a cosine a
// rounding past one.
func sineOf(cosine float64) float64 {
	return math.Sqrt(max(none, one-cosine*cosine))
}

// polar is the side of the horizon a look's cosine from up is on, nought
// below and one above, and the weight that side's polar mean takes in
// the look's light: (1 - s) squared times (1 + s), s the look's sine
// over the polar bin centre's, held at one outside the centre, where the
// weight is none.
func (made *Sky) polar(cosine float64) (int, float64) {
	side := none
	if cosine >= none {
		side = one
	}
	share := min(one, sineOf(cosine)/made.poleSine)
	from := one - share
	return side, from * from * (one + share)
}

// zenithOf is a way's angle from straight up, in radians.
func zenithOf(way maths.Vec) float64 {
	return math.Acos(min(max(way.Unit().Y, -one), one))
}

// roundOf is a way's angle across the ground, from north toward east,
// in radians.
func roundOf(way maths.Vec) float64 {
	return math.Atan2(way.X, -way.Z)
}

// Light is the sky's light along a way: the table's, read by the way's
// cosine from up and its azimuth from the sun, as a cubic between the
// four nearest points that meets each one's light and slopes, and drawn
// toward its polar bin's mean near the zenith and the nadir.
func (made *Sky) Light(toward maths.Vec) maths.Light {
	toward = toward.Unit()
	side, weight := made.polar(toward.Y)
	if weight == none {
		return made.cubic(toward)
	}
	return made.cubic(toward).Scale(one - weight).
		Add(made.poles[side].Scale(weight))
}

// cubic is the table read along a unit way as a cubic between the four
// nearest points, with no draw toward the poles.
func (made *Sky) cubic(toward maths.Vec) maths.Light {
	view := made.viewAxis(toward.Y)
	round := axisOf(folded(roundOf(toward)-made.round)/math.Pi,
		made.azimuths)
	rows, columns := piece(view), piece(round)
	var light maths.Light
	for _, row := range rows {
		for _, column := range columns {
			point := row.bin*made.azimuths + column.bin
			light = light.Add(made.table[point].Scale(row.value *
				column.value)).
				Add(made.upward[point].Scale(row.slope * column.value)).
				Add(made.across[point].Scale(row.value * column.slope))
		}
	}
	return light
}

// viewAxis is where a cosine from up falls between the centres of the
// view bins on its own side of the horizon: the upper half of the bins
// for a look at or above it, the lower half for a look below. Past the
// half's first or last centre it is at that one.
func (made *Sky) viewAxis(cosine float64) axis {
	middle := made.views / two
	first, last := none, middle-one
	if cosine >= none {
		first, last = middle, made.views-one
	}
	place := (cosine+one)/two*float64(made.views) - half
	place = min(max(place, float64(first)), float64(last))
	low := int(place)
	return axis{low: low, high: min(low+one, last),
		share: place - float64(low)}
}

// a weight is what one end of a cubic piece gives at a place along it:
// the bin at that end, and how much of its light and of its slope.
type weight struct {
	bin          int
	value, slope float64
}

// piece is the weights of a piece of cubic between the two bins of an
// axis, at its share of the way along: Hermite's, of the light and the
// slope at each end, a slope being so much a bin.
func piece(at axis) [two]weight {
	share := at.share
	square := share * share
	cube := square * share
	toHigh := three*square - two*cube
	return [two]weight{
		{bin: at.low, value: one - toHigh, slope: cube - two*square + share},
		{bin: at.high, value: toHigh, slope: cube - square},
	}
}

// slopes is each point's slope by view and by azimuth, so much a bin:
// Steffen's slope from the points beside it on its row or column, and
// nought at the first and last point of each half of the view and of
// the azimuths, where the sky is held.
func (made *Sky) slopes() (upward, across []maths.Light) {
	upward = make([]maths.Light, len(made.table))
	across = make([]maths.Light, len(made.table))
	middle := made.views / two
	at := func(row, column int) maths.Light {
		return made.table[row*made.azimuths+column]
	}
	for row := range made.views {
		for column := range made.azimuths {
			point := row*made.azimuths + column
			if inside(row, middle) {
				upward[point] = steffen(at(row-one, column), at(row, column),
					at(row+one, column))
			}
			if column > none && column < made.azimuths-one {
				across[point] = steffen(at(row, column-one), at(row, column),
					at(row, column+one))
			}
		}
	}
	return upward, across
}

// inside is whether a view bin has a bin on each side of it on its own
// side of the horizon, which is between the bins before middle and the
// rest.
func inside(row, middle int) bool {
	return row != none && row != middle-one && row != middle &&
		row != two*middle-one
}

// steffen is the slope at a point, so much a bin, with the points before
// and after it, channel by channel.
func steffen(before, here, after maths.Light) maths.Light {
	return maths.RGB(steffenSlope(before.R, here.R, after.R),
		steffenSlope(before.G, here.G, after.G),
		steffenSlope(before.B, here.B, after.B))
}

// steffenSlope is Steffen's slope at a point of evenly spaced points:
// nought where the point is above or below both beside it, and else
// the least of the two steps to them and half their mean, so the cubic
// it makes keeps between the points.
func steffenSlope(before, here, after float64) float64 {
	down, up := here-before, after-here
	if down*up <= none {
		return none
	}
	least := min(math.Abs(down), math.Abs(up), math.Abs(down+up)/two/two)
	return math.Copysign(two*least, up)
}
