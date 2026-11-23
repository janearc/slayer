// This file is the surf: a sea with no edge over a set's ground, its
// waves changed by the bottom as they come in. They shorten and slow in
// shallow water and their crests turn toward the shallows; they grow as
// they shoal and are cut down where they break; the breaking and the wind
// leave foam; and the swash wets the beach above the sea's level.

package arid

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// surf is what a coast does to a sea's waves, at each of the ground's
// samples. Between samples every value is read on the ground's triangles,
// as the depth is, so the waves change with the ground the eye sees.
type surf struct {
	shore *coast
	// toward is the way the waves travel, an east and a south of one
	// length: the wind's, as oceantex's bands are made.
	toward [2]float64
	// grown is how much of the open sea's height the waves have at a
	// sample of water once shoaled, before the depth cuts them; one on
	// land, where the depth leaves none. See gainAt.
	grown []float64
	// carries is the most of the open height the waves keep for each
	// metre of depth, by each bound the depth sets: the surf's and the
	// bed's. See shoal.
	carries [2]float64
	// most is the most gain anywhere, off the grid too.
	most float64
	// lags is, for each band, how far ahead of the open sea's its waves
	// are at a sample, in metres along toward.
	lags [][]float64
	// breaking is the share of the waves breaking at a sample.
	breaking []float64
	// wetness is the share of the time the swash covers a sample of the
	// beach: one under the sea, nought above the swash's reach.
	wetness []float64
	// whitecaps is the share of the open sea the wind whitens.
	whitecaps float64
	// deviation is the open surface's standard deviation in metres.
	deviation float64
	// counted is what the surf did, for the draw's line.
	counted Surfline
}

// arrival is the waves arriving over the coast from the open sea.
type arrival struct {
	// peak is the spectrum's peak, in radians a second.
	peak float64
	// open is the open sea's depth, nought for deep water.
	open float64
	// significant and rms are the waves' heights in metres.
	significant, rms float64
	// length is the deep-water length of the peak wave, in metres.
	length float64
	// reach is the furthest the open surface goes from its mean.
	reach float64
}

// Surf is the sea meeting a ground, as Ashore makes it, with what the
// bottom does to its waves as they come in. made is the sea the bands were
// made from, for its wind, its peak and its height. A sea with no wind is
// left as Ashore leaves it, and its line says it was calm.
//
// See surf for what is done, and the report on the surf for what is not:
// a wave's height is not changed by its bending, the swash does not run
// up the beach as water, and foam that a breaker leaves behind it is not
// kept once the wave has passed.
func (sea *Sea) Surf(made oceantex.Sea, ground Ground, east, south,
	level float64) (*Sea, Surfline, error) {
	if !finite(made.Wind) || !finite(made.Fetch) || !finite(made.Toward) ||
		!finite(made.Depth) || made.Depth < 0 {
		return nil, Surfline{}, errors.New("arid: the surf needs a sea " +
			"of finite wind, fetch, way and depth")
	}
	shored, shoreline, err := sea.Ashore(ground, east, south, level)
	if err != nil {
		return nil, Surfline{}, err
	}
	if !(made.Wind > 0 && made.Fetch > 0) || !(made.Height() > 0) ||
		!(sea.reach > 0) {
		return shored, Surfline{Shoreline: shoreline, Calm: true}, nil
	}
	waves := newSurf(sea, made, shored.coast)
	shored.surf = waves
	shored.steepest = waves.steepest(sea)
	shored.reach = sea.reach * waves.most
	counted := waves.counted
	counted.Shoreline = shored.shoreline(sea)
	return shored, counted, nil
}

// newSurf works out what the coast does to the sea's waves at every
// sample of its ground.
func newSurf(sea *Sea, made oceantex.Sea, shore *coast) *surf {
	peak := made.Peak()
	given := arrival{peak: peak, open: made.Depth,
		significant: made.Height(),
		rms:         made.Height() * rootMeanSquareShare,
		length:      twice * math.Pi * gravity / (peak * peak),
		reach:       sea.reach}
	waves := &surf{shore: shore,
		toward:    [2]float64{math.Cos(made.Toward), -math.Sin(made.Toward)},
		grown:     make([]float64, len(shore.height)),
		carries:   [2]float64{surfSaturated / given.rms, 1 / given.reach},
		breaking:  make([]float64, len(shore.height)),
		whitecaps: whitecapCover(made.Wind),
		deviation: given.significant / significantDeviations}
	waves.counted.Whitecaps = waves.whitecaps
	for at := range shore.height {
		waves.shoal(at, given)
	}
	waves.bend(sea, given)
	waves.swash(given)
	return waves
}

// shoal works out the waves' gain and how many break at one sample.
//
// They are shoaled by the peak wave's group speed (shoaling), as one wave
// stands for the whole sea. They are then cut so their root-mean-square
// height is at most 0.42 of the depth [TG82] and their highest crest and
// deepest trough are within the depth, so no trough goes through the bed.
// The cut is made where the surface is read, from the depth there.
func (waves *surf) shoal(at int, given arrival) {
	depth := -waves.shore.height[at]
	waves.grown[at] = 1
	if waves.shore.cut[at] {
		return
	}
	if depth <= 0 {
		// at the waterline the height over the depth is without end, so
		// every wave reaching it has broken (Q = 1); land carries that
		// share for the water between it and the next sample
		waves.breaking[at] = 1
		return
	}
	grown := shoaling(given.peak, depth, given.open)
	gain := math.Min(grown, math.Min(waves.carries[0]*depth,
		waves.carries[1]*depth))
	waves.grown[at] = grown
	waves.breaking[at] = breakingFraction(grown * given.rms /
		(breakerIndex * depth))
	waves.counted.Gain = math.Max(waves.counted.Gain, gain)
	if gain < grown {
		waves.counted.Capped++
		waves.counted.CappedFrom = math.Max(waves.counted.CappedFrom, depth)
	}
}

// bend works out every band's lag over the coast.
func (waves *surf) bend(sea *Sea, given arrival) {
	peak := waveNumber(given.peak, given.open)
	floor := bendFloor * given.rms
	for _, number := range bandNumbers(sea.bands, peak) {
		lags, rounds := waves.shore.lag(number, given.open, floor,
			waves.toward)
		waves.lags = append(waves.lags, lags)
		waves.counted.Rounds = append(waves.counted.Rounds, rounds)
	}
}

// height is the surface's height at an east and a south: the bands, each
// read where its lag over the coast moves it, summed and times the gain.
// A place that is not a number is the open sea's.
func (waves *surf) height(sea *Sea, east, south float64) float64 {
	if !finite(east) || !finite(south) {
		return sea.waves(east, south)
	}
	place := waves.shore.locate(east, south)
	gain, _, _ := waves.gainAt(place, east, south)
	if gain == 0 {
		return 0
	}
	return gain * waves.open(sea, place, east, south)
}

// gainAt is how much of the open sea's height the waves have at a spot,
// and how fast that changes a metre east and south: the shoaled gain read
// between the samples, cut to each bound the depth there sets. Where the
// ground is at or above the sea's level there is none.
func (waves *surf) gainAt(place spot, east, south float64) (gain, eastward,
	southward float64) {
	depth := -waves.shore.ground(place)
	if !(depth > 0) {
		return 0, 0, 0
	}
	gain, eastward, southward = waves.shore.slant(place, waves.grown, east,
		south)
	for _, carried := range waves.carries {
		if carried*depth < gain {
			gain, eastward, southward = carried*depth, -carried*place.rise,
				-carried*place.fall
		}
	}
	return gain, eastward, southward
}

// open is the bands' heights summed at a place, longest first, each read
// where the longer bands' water came from (Sea.waves) and then where its
// lag moves it: the open sea's waves, shortened and turned, before the gain.
func (waves *surf) open(sea *Sea, place spot, east, south float64) float64 {
	var total float64
	last := len(sea.bands) - 1
	for index, band := range sea.bands {
		lag := waves.shore.blend(place, waves.lags[index])
		readEast, readSouth := east+waves.toward[0]*lag,
			south+waves.toward[1]*lag
		half := band.Extent / 2
		if index == last {
			return total + periodicSample(band, band.Height, half+readEast,
				half-readSouth)
		}
		height, pushEast, pushNorth := carriedSample(band, half+readEast,
			half-readSouth)
		total += height
		east, south = east-pushEast, south+pushNorth
	}
	return total
}

// bandAt is a band's height at an east and a south, and its rise a metre
// east and a metre south, from the band's spectral slopes.
func bandAt(band *oceantex.Field, east, south float64) (height, eastward,
	southward float64) {
	half := band.Extent / 2
	height = periodicSample(band, band.Height, half+east, half-south)
	eastward = periodicSample(band, band.SlopeX, half+east, half-south)
	southward = -periodicSample(band, band.SlopeY, half+east, half-south)
	return height, eastward, southward
}

// normal is the way the surface faces at an east and a south. The surface
// is the gain times the sum of the bands (open), so its slope is the gain's
// times the sum, and the gain times the sum's own rise (opened).
func (waves *surf) normal(sea *Sea, east, south float64) maths.Vec {
	if !finite(east) || !finite(south) {
		return maths.V(0, 1, 0)
	}
	place := waves.shore.locate(east, south)
	gain, gainEast, gainSouth := waves.gainAt(place, east, south)
	open, riseEast, riseSouth := waves.opened(sea, place, east, south)
	riseEast = gain*riseEast + open*gainEast
	riseSouth = gain*riseSouth + open*gainSouth
	return maths.V(-riseEast, 1, -riseSouth).Unit()
}

// water is where the water a band is read at has been carried from, by
// the longer bands before it, and how that place moves with the point
// asked: a row for east and one for north, a column for a metre east and
// one for a metre north of the point.
type water struct {
	east, south float64
	moves       [2][2]float64
}

// opened is open's height, with its rise a metre east and a metre south.
// A band is read at the place its water came from (water) moved by its lag,
// so a metre east or north of the point moves that place by the water's
// own move, and by toward times the lag's rise; the band's rise is carried
// through that move. The water a longer band carries is its push, which
// moves by its stretch, as slopeFrom carries it for the open sea.
func (waves *surf) opened(sea *Sea, place spot, east, south float64) (
	total, riseEast, riseSouth float64) {
	carried := water{east: east, south: south,
		moves: [2][2]float64{{1, 0}, {0, 1}}}
	for index, band := range sea.bands {
		lag, lagEast, lagSouth := waves.shore.slant(place, waves.lags[index],
			east, south)
		moves := carried.moves
		toward, rise := [2]float64{waves.toward[0], -waves.toward[1]},
			[2]float64{lagEast, -lagSouth}
		for row := range moves {
			for column := range moves[row] {
				moves[row][column] += toward[row] * rise[column]
			}
		}
		readEast, readSouth := carried.east+waves.toward[0]*lag,
			carried.south+waves.toward[1]*lag
		height, bandEast, bandSouth := bandAt(band, readEast, readSouth)
		total += height
		riseEast += moves[0][0]*bandEast - moves[1][0]*bandSouth
		riseSouth -= moves[0][1]*bandEast - moves[1][1]*bandSouth
		carried = carried.pushed(band, readEast, readSouth, moves)
	}
	return total, riseEast, riseSouth
}

// pushed is the water carried on to the next band, from a band read at a
// place, moved by moves: the place less the band's push there, east and
// north, and the move less the band's stretch there taken through it.
func (carried water) pushed(band *oceantex.Field, east, south float64,
	moves [2][2]float64) water {
	half := band.Extent / 2
	across, up := half+east, half-south
	_, pushEast, pushNorth := carriedSample(band, across, up)
	stretch := [2][2]float64{
		{smoothSample(band, band.StretchXX, across, up),
			smoothSample(band, band.StretchXY, across, up)},
		{smoothSample(band, band.StretchXY, across, up),
			smoothSample(band, band.StretchYY, across, up)}}
	next := water{east: carried.east - pushEast,
		south: carried.south + pushNorth, moves: carried.moves}
	for row := range next.moves {
		for column := range next.moves[row] {
			next.moves[row][column] -= stretch[row][0]*moves[0][column] +
				stretch[row][1]*moves[1][column]
		}
	}
	return next
}

// Foam is how much of the surface at a point, in the sea's frame, is foam,
// from none to all, and what the foam reflects. A sea with no surf has
// none. See surf.foam.
func (sea *Sea) Foam(point maths.Vec) (float64, maths.Light) {
	if sea == nil || sea.surf == nil || !finite(point.X) ||
		!finite(point.Z) {
		return 0, maths.Light{}
	}
	return sea.surf.foam(sea, point.X, point.Z)
}

// Wet is the share of the time the swash covers the ground at a point, in
// the sea's frame: one under the sea, nought above the swash's reach and
// for a sea with no surf. It is for the ground's shading of the wet band.
func (sea *Sea) Wet(point maths.Vec) float64 {
	if sea == nil || sea.surf == nil || !finite(point.X) ||
		!finite(point.Z) {
		return 0
	}
	shore := sea.surf.shore
	return shore.blend(shore.locate(point.X, point.Z), sea.surf.wetness)
}

// foam is the foam at an east and a south. Of the sea there, the share
// that breaking or the wind whitens is the share of waves breaking and,
// of the rest, the whitecaps' cover. That share is laid on the highest
// of the open surface: where it stands over the height that so much of a
// normal surface of its deviation passes, it is foam, over foamEdge.
//
// White water reflects as fresh foam [WBG82], whitecaps as Koepke's
// whitecaps [Ko84], and the two mix by their shares. No water, no foam.
func (waves *surf) foam(sea *Sea, east, south float64) (float64,
	maths.Light) {
	place := waves.shore.locate(east, south)
	breaking := waves.shore.blend(place, waves.breaking)
	if gain, _, _ := waves.gainAt(place, east, south); !(gain > 0) {
		return 0, maths.Light{}
	}
	capped := waves.whitecaps * (1 - breaking)
	cover := breaking + capped
	if !(cover > 0) {
		return 0, maths.Light{}
	}
	over := math.Sqrt2 * math.Erfcinv(twice*cover)
	standing := waves.open(sea, place, east, south) / waves.deviation
	share := math.Min(math.Max((standing-over)/foamEdge, 0), 1)
	white := (surfFoamReflectance*breaking + whitecapReflectance*capped) /
		cover
	return share, maths.Light{R: white, G: white, B: white}
}

// steepest is the most the surf's surface can rise a metre, for the
// search to step by. Within a triangle of the ground the gain and every
// lag are planes, so each band's place moves by at most one and its lag's
// rise for every metre, and the surface's rise is within
//
//	most gain * sum(band rise * (1 + lag rise)) + gain rise * reach.
//
// The gain is the least of planes, so it rises no faster than the
// steepest of them that can be least in the triangle. Off the grid the
// values are held to its edge, which no triangle's rise exceeds; there the
// surface rises at most the most gain times the bands'. It also finds the
// most gain anywhere, which the sea's slab is made from.
func (waves *surf) steepest(sea *Sea) float64 {
	rises := make([]float64, len(sea.bands))
	var sum, most float64
	for index, band := range sea.bands {
		rises[index] = steepestRise(band)
		sum += rises[index]
	}
	shore := waves.shore
	for row := range shore.down - 1 {
		for column := range shore.across - 1 {
			for _, upper := range [...]bool{true, false} {
				var place spot
				shore.triangle(&place, column, row, upper)
				gain, rise := waves.triangleRise(place, rises, sea.reach)
				waves.most = math.Max(waves.most, gain)
				most = math.Max(most, rise)
			}
		}
	}
	return math.Max(most, waves.most*sum)
}

// triangleRise is the most gain in one triangle, and the most the surface
// rises a metre in it.
func (waves *surf) triangleRise(place spot, rises []float64,
	reach float64) (gain, most float64) {
	shore := waves.shore
	shallowest, deepest := math.Inf(1), math.Inf(-1)
	for _, corner := range place.corners {
		gain = math.Max(gain, waves.grown[corner])
		shallowest = math.Min(shallowest, -shore.height[corner])
		deepest = math.Max(deepest, -shore.height[corner])
	}
	east, south := shore.faceRise(place, waves.grown)
	gainRise := math.Hypot(east, south)
	grown := gain
	for _, carried := range waves.carries {
		gain = math.Min(gain, carried*math.Max(deepest, 0))
		if carried*shallowest < grown {
			gainRise = math.Max(gainRise,
				carried*math.Hypot(place.rise, place.fall))
		}
	}
	if !(gain > 0) {
		return 0, 0
	}
	most = gainRise * reach
	for index, rise := range rises {
		east, south = shore.faceRise(place, waves.lags[index])
		most += gain * rise * (1 + math.Hypot(east, south))
	}
	return gain, most
}

// String is the surf in a sentence, after the shoreline's.
func (counted Surfline) String() string {
	if counted.Calm {
		return counted.Shoreline.String() + "; the sea is calm, no surf"
	}
	return fmt.Sprintf("%s; surf: the waves stand at most %.3g of their "+
		"open height, are cut by the depth at %d heights from %.3g m of "+
		"water, and break spilling at %d, plunging at %d and surging at "+
		"%d heights of the waterline; the swash reaches %.3g to %.3g m "+
		"up and wets %d heights of land; whitecaps cover %.3g of the open "+
		"sea; the bands bent in %v rounds",
		counted.Shoreline.String(), counted.Gain, counted.Capped,
		counted.CappedFrom, counted.Spilling, counted.Plunging,
		counted.Surging, counted.RunupLeast, counted.RunupMost,
		counted.Swashed, counted.Whitecaps, counted.Rounds)
}
