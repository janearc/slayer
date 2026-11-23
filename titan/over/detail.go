package over

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Each rule's noise has a seed of its own, the stretch's seed with one of
// these added, so the rules do not share a field. They are picked, and
// only have to differ.
const (
	saltFlute     = 1
	saltFluteFine = 2
	saltDip       = 3
	saltBed       = 4
	saltReach     = 5
	saltRoughness = 16
)

// Numbers the rules share. Each is picked, and each is a shape of the
// detail, not a measure of anything.
const (
	// exposureReach is how many times the slope that begins a rule the
	// rule is at full strength at: it eases in between.
	exposureReach = 1.25
	// zeroReach is how far from a surveyed sample, as a share of the map's
	// spacing, detail has eased up to its full size.
	zeroReach = 0.5
	// shortestOfSpacing is how many of the made spacing the shortest
	// roughness wavelength is: four samples to a wave, so that it is drawn
	// and not aliased.
	shortestOfSpacing = 4.0
	// surveyWavelength is the shortest wave a survey carries, in its
	// samples: two. Defined, by sampling.
	surveyWavelength = 2.0
)

// Flutes: where a channel is, in the noise's own units, and how the finer
// octave is added to the coarser.
const (
	// channelHalfWidth is how near zero the noise is within a channel: the
	// noise has a thin valley either side of its zero lines, and the channel
	// is that valley.
	channelHalfWidth = 0.12
	// fineScale and fineShare are how much finer than the coarse octave
	// the fine one is, and how much of it is added.
	fineScale = 2.3
	fineShare = 0.5
	// thirdAxis is how far through the noise's third axis the flutes'
	// pattern is taken. It is off the lattice, where the noise is nought.
	thirdAxis = 0.5
)

// Beds: the ledge's shape in a bed, and how the beds vary.
const (
	// bedEdge is how much of a bed's spacing a ledge's edges ease over.
	bedEdge = 0.08
	// bedThickness is how much of a bed's spacing stands out.
	bedThickness = 0.35
	// bedLeast is the weakest a bed stands out, as a share of the most.
	bedLeast = 0.35
	// bedDip is how far, in beds, a bed lifts and falls along the face.
	bedDip = 0.35
	// dipSpan is the metres, along the face, a bed takes to lift and fall;
	// dipRise is how many beds it takes up the face to change.
	dipSpan = 17.0
	dipRise = 3.0
	// reachLeast and reachSpan: a ledge does not run the length of a face;
	// it is weakest as this share of itself and reaches this many metres.
	reachLeast = 0.5
	reachSpan  = 23.0
)

// ground is the made detail: how far each place of the stretch moves along
// the shape's normal.
type ground struct {
	read    *shape
	stretch Stretch
	rules   Rules
	bound   float64  // metres: no place moves further
	octaves []octave // the sizes the roughness is made in
	scaling float64  // the power of width roughness grows by
	// eye is the fineness the ground is laid to, or nil where it is laid
	// to the rules' spacing alone, and widened is how many times coarser
	// than the eye wanted it was laid, to fit the faces.
	eye     Fineness
	widened float64
}

// How an octave of roughness is faded where the ground is laid as an eye
// sees it: none of a wave shorter than two corners, which its corners
// cannot carry, all of one four corners long or more. Two is defined, the
// least a sampled wave needs; four is picked.
const (
	drawnFrom = 2.0
	drawnFull = 4.0
)

// newGround is the made detail for a stretch cut to a spacing. Roughness is
// made at wavelengths from the map's spacing down to four samples of the
// made spacing: all shorter than the map's own, two of its samples.
func newGround(read *shape, stretch Stretch, rules Rules,
	spacing float64) ground {
	scaling := mostScaling
	if rules.Roughness.Continued && rules.Roughness.Gain > 0 {
		scaling = read.scaling(stretch)
	}
	return ground{read: read, stretch: stretch, rules: rules,
		bound: rules.Departure * read.step, scaling: scaling,
		octaves: octaves(rules.Seed+saltRoughness, read.step,
			shortestOfSpacing*spacing)}
}

// seenBy is the made detail laid to an eye's fineness, widened so many
// times. The eye decides only which octaves of roughness its corners can
// carry; the flutes, the beds and every coarser octave are the same from
// every eye.
func (made ground) seenBy(eye Fineness, widened float64) ground {
	made.eye, made.widened = eye, widened
	return made
}

// displacement is how far a place moves along the shape's normal, metres,
// outward positive: the three rules together, kept inside the bound, and
// eased to nothing at a sample and at the stretch's edge. The bound's
// tanh is no steeper than one, so it never widens a difference.
func (made ground) displacement(at place, point, normal maths.Vec) float64 {
	raw := made.flutes(at, point, normal) + made.beds(point, normal) +
		made.roughness(at, point, normal)
	return made.bound * math.Tanh(raw/made.bound) *
		made.sampleless(at) * made.inland(point)
}

// local is the spacing the ground is laid to at a point, metres: the
// eye's, widened, and no finer than the rules' spacing.
func (made ground) local(point maths.Vec) float64 {
	return made.widened * max(made.rules.Spacing, made.eye.Wanted(point))
}

// drawn is how much of an octave of roughness is made at a point, nought
// to one: all of it without an eye; with one, none of a wave under two
// corners long and all of one four or more, eased between.
//
// What an eye leaves out is under its pixel, by construction. An octave's
// height is its steepness times its wavelength over peakShare times waves
// that add to at most wavesPerOctave over two; with the steepness capped
// as roughness caps it, the octaves under four corners add to less than
// (wavesPerOctave/2/peakShare) (most/2 pi) 8 local, and Seen keeps most
// under mostSeenStand so that this is under local, one pixel widened.
func (made ground) drawn(point maths.Vec, wavelength float64) float64 {
	if made.eye == nil {
		return 1
	}
	local := made.local(point)
	return maths.Smooth(drawnFrom*local, drawnFull*local, wavelength)
}

// slope is how steep a normal says the ground is: rise over run.
func slope(normal maths.Vec) float64 {
	return math.Hypot(normal.X, normal.Z) / normal.Y
}

// exposure is how far a rule's ground is steep enough, nought to one: it
// is nothing below the slope the rule begins at.
func exposure(begins float64, normal maths.Vec) float64 {
	return maths.Smooth(begins, begins*exposureReach, slope(normal))
}

// sampleless is nought at a sample of the map and one half a spacing from
// it, eased between.
func (made ground) sampleless(at place) float64 {
	near := math.Hypot(min(at.across, 1-at.across), min(at.down, 1-at.down))
	return maths.Smooth(0, zeroReach, near)
}

// inland is nought at the stretch's edge and one a margin in, eased between.
func (made ground) inland(point maths.Vec) float64 {
	if made.rules.Margin == 0 {
		return 1
	}
	step := made.read.step
	east := point.X - float64(made.stretch.Column)*step
	south := point.Z - float64(made.stretch.Row)*step
	edge := min(east, float64(made.stretch.Across)*step-east, south,
		float64(made.stretch.Down)*step-south)
	return maths.Smooth(0, made.rules.Margin, edge)
}

// flutes are the channels: negative, cut in. A channel is cut where the
// noise across the fall line is near nought, as deep as the ground that
// drains through is much, on steep ground. Each corner of the place's
// triangle has a fall line of its own and says where the channels are by
// it; the three are blended by how much of each the place is.
func (made ground) flutes(at place, point, normal maths.Vec) float64 {
	rules := made.rules.Flutes
	steep := exposure(rules.Slope, normal)
	if rules.Depth == 0 || steep == 0 {
		return 0
	}
	var channel, drained float64
	for i, sample := range at.corners {
		channel += at.weights[i] * made.channel(point, made.read.fall[sample])
		drained += at.weights[i] *
			math.Min(1, math.Sqrt(made.read.drained[sample]/rules.Drainage))
	}
	return -rules.Depth * drained * channel * steep
}

// channel is how much of a channel a point is in, nought to one, for a
// given fall line. The noise is stretched along the way down, which is up
// and down the face, and not along the fall line's horizontal run, so that
// a channel on a wall runs the wall's height.
func (made ground) channel(point, fall maths.Vec) float64 {
	rules := made.rules.Flutes
	across := (point.Z*fall.X - point.X*fall.Z) / rules.Width
	along := point.Y / (rules.Width * rules.Length)
	seed := made.rules.Seed
	coarse := noise.Gradient(seed+saltFlute, across, along, thirdAxis)
	fine := noise.Gradient(seed+saltFluteFine, across*fineScale,
		along*fineScale, thirdAxis)
	valley := math.Abs(coarse+fineShare*fine) / (1 + fineShare)
	reach := 1 - min(1, valley/channelHalfWidth)
	return reach * reach
}

// beds are the ledges: level in height, a stated spacing apart, standing
// out of steep ground. Each bed stands out by its own share of the most, and
// the beds lift and fall along the face, so that they are not ruled lines.
func (made ground) beds(point, normal maths.Vec) float64 {
	rules := made.rules.Beds
	steep := exposure(rules.Slope, normal)
	if rules.Standout == 0 || steep == 0 {
		return 0
	}
	seed := made.rules.Seed
	rise := rules.Spacing * dipRise
	dip := bedDip * noise.Gradient(seed+saltDip, point.X/dipSpan,
		point.Y/rise, point.Z/dipSpan)
	level := point.Y/rules.Spacing + dip
	bed := math.Floor(level)
	strength := bedLeast + (1-bedLeast)*noise.Hash(seed+saltBed, int64(bed))
	reach := reachLeast + (1-reachLeast)*unitOf(noise.Gradient(
		seed+saltReach, point.X/reachSpan, point.Y/rise, point.Z/reachSpan))
	return rules.Standout * strength * reach * steep * ledge(level-bed)
}

// unitOf is a noise value, about minus one to one, as nought to one.
func unitOf(value float64) float64 {
	return min(1, max(0, (1+value)/2))
}

// ledge is how far out a bed stands, nought to one, at a place within it,
// nought at its lower edge to one at its upper: it is out for its thickness
// from its lower edge and the ground is not, the rest of the way up.
func ledge(within float64) float64 {
	return maths.Smooth(0, bedEdge, within) *
		(1 - maths.Smooth(bedThickness, bedThickness+bedEdge, within))
}

// roughness is relief made at wavelengths shorter than the map's, as steep
// at every one as the map's spread over its neighbourhood says, and no
// steeper than the ground is, nor than the rules let relief lean.
//
// A wave's amplitude is its wavelength times a steepness: the map's
// spread over a neighbourhood, over its width, times the gain, growing
// toward small waves as the survey's roughness falls with scale.
//
// Where relief is not continued it does not grow, the same slope at every
// scale, chosen and not measured. Where the ground is flat, or the map
// smooth round it, it is nothing.
//
// The cap is for the octaves together: the ground's own slope or the
// rules' stand, over two pi, over the square root of how many octaves
// there are. Octaves drawn from their own seeds are independent, so their
// root-mean-square slopes add in quadrature to no more than one's alone.
func (made ground) roughness(at place, point, normal maths.Vec) float64 {
	rules := made.rules.Roughness
	if rules.Gain == 0 {
		return 0
	}
	var spread float64
	for i, sample := range at.corners {
		spread += at.weights[i] * made.read.rough[sample]
	}
	most := min(rules.Stand, slope(normal)) / (2 * math.Pi) /
		math.Sqrt(float64(max(1, len(made.octaves))))
	survey := surveyWavelength * made.read.step
	var sum float64
	for _, each := range made.octaves {
		height := made.steepness(spread, each.wavelength, most) *
			each.wavelength / peakShare
		sum += height * each.along(point, normal, survey) *
			made.drawn(point, each.wavelength)
	}
	return sum
}

// steepness is an octave's: the spread over the neighbourhood's width,
// times the gain, grown toward small waves by the survey's scaling, and
// no more than most.
func (made ground) steepness(spread, wavelength, most float64) float64 {
	width := made.read.width()
	return min(most, made.rules.Roughness.Gain*spread/width*
		math.Pow(wavelength/width, made.scaling-mostScaling))
}
