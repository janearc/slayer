package planet

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// sodiumD2 and sodiumD1 are sodium's d lines, nanometres in air, from
// picks.yaml.
var (
	sodiumD2 = pick("sodium-d2")
	sodiumD1 = pick("sodium-d1")
)

// grazing is the least cosine of the eye's height a thin glowing layer
// is taken at, so its path through the layer stays finite at the limb.
var grazing = pick("glow-grazing")

// frontStep is how many degrees a front is read across, and mostFront how
// many times one brightens the glow at most; both are in picks.yaml.
var (
	frontStep = pick("front-step")
	mostFront = pick("front-most")
)

// calmRows and calmRound are how many rows and columns of an even grid over
// the band the calm is read on.
var (
	calmRows  = pickCount("calm-rows")
	calmRound = pickCount("calm-round")
)

// poleCosineFloor is the least cosine of a latitude a front's east-west
// step is widened by, so the step stays finite at a pole.
var poleCosineFloor = pick("pole-cosine-floor")

// front is how strongly the airglow shows at a way of the face: none
// where its air's origin changes no more steeply than the band's calm,
// one where twice as steeply, up to mostFront. with no flow, one.
//
// so the band's inside is dark, and the whorls' edges and the band's
// boundary glow. the weather here is barotropic and has no compression;
// the strain that draws fronts is what this reads, standing in for it.
func (face *Face) front(up maths.Vec, finest float64) float64 {
	if face.flow == nil {
		return whole
	}
	steep := face.steepness(up, finest) / face.calm
	return math.Min(mostFront, math.Max(nothing, steep-whole))
}

// calmness is how steeply the band's air's origin changes everywhere but
// its fronts: the steepness its steepest share, its fronts, lies above,
// read on an even grid over the band. one with no flow, no glow, or a
// flat band.
func (face *Face) calmness() float64 {
	if face.flow == nil || face.airglow == nothing {
		return whole
	}
	steeps := make([]float64, nothing, calmRows*calmRound)
	for i := range calmRows {
		latitude := face.airband * (double*(float64(i)+half)/
			float64(calmRows) - whole)
		for k := range calmRound {
			longitude := fullTurnDegrees * float64(k) / float64(calmRound)
			up := face.ways.toward(latitude, longitude)
			steeps = append(steeps, face.steepness(up, nothing))
		}
	}
	slices.Sort(steeps)
	last := len(steeps) - adjacent
	count := float64(len(steeps))
	at := int((whole - face.fronts) * count)
	calm := steeps[min(last, at)]
	if !(calm > nothing) {
		return whole
	}
	return calm
}

// steepness is how sharply a way of the face's air's origin changes:
// degrees of origin across a degree of the sphere, read either side of it
// north-south and east-west, across the pixel or frontStep, whichever is
// wider. unstirred air changes one degree a degree.
func (face *Face) steepness(up maths.Vec, finest float64) float64 {
	latitude := latitudeOf(up, face.ways.pole)
	longitude := math.Atan2(up.Dot(face.ways.east),
		up.Dot(face.ways.front)) / maths.Degree
	step := math.Max(frontStep, finest/maths.Degree)
	wide := step / math.Max(math.Cos(latitude*maths.Degree), poleCosineFloor)
	northward := (face.originNear(latitude, longitude, step, nothing) -
		face.originNear(latitude, longitude, -step, nothing)) /
		(double * step)
	eastward := (face.originNear(latitude, longitude, nothing, wide) -
		face.originNear(latitude, longitude, nothing, -wide)) /
		(double * step)
	return math.Hypot(northward, eastward)
}

// originNear is the latitude in degrees whose air shows at a place of the
// face, a latitude and longitude in degrees, moved north and east by so many
// degrees.
func (face *Face) originNear(latitude, longitude, north,
	east float64) float64 {
	came, _, _ := face.flowed(face.ways.toward(latitude+north,
		longitude+east))
	return came
}

// Airglowing is the body with its air glowing in sodium's d lines under its
// x-rays, from a band so many degrees either side of its equator, where the
// x-rays leave.
//
// it glows so many watts a square metre a steradian looking straight down at a
// front twice as steep as the band's calm. fronts is the steepest share of the
// band, more than zero and under one, taken for its fronts.
//
// the glow is a thin layer, so it brightens toward the limb as the eye's path
// through it lengthens, as a sodium layer does seen from orbit, and shows on
// the night side as by day.
//
// it glows in the air the band's x-rays lit, so it goes where that air goes:
// the weather tears and folds it as it does the zones.
func (body Body) Airglowing(radiance, band, fronts float64) (Body,
	error) {
	if !(radiance >= nothing) ||
		!(band > nothing && band <= poleLatitude) ||
		!(fronts > nothing && fronts < whole) {
		return body, fmt.Errorf("an airglow of %g W/m²/sr over %g degrees, "+
			"its fronts the steepest %g", radiance, band, fronts)
	}
	body.airglow, body.airband, body.airfronts = radiance, band, fronts
	return body, nil
}

// heldIn is how much of sodium's d lines a band from lo to hi nanometres
// holds: half for each line.
func heldIn(lo, hi float64) float64 {
	var held float64
	for _, line := range []float64{sodiumD2, sodiumD1} {
		if line >= lo && line < hi {
			held += lineShare
		}
	}
	return held
}

// lineShare is how much of the glow each d line holds, as the two are
// alike.
var lineShare = pick("d-line-share")

// glowedAt is the airglow a point sends the eye whose air came from a
// latitude in degrees, in a band that holds so much of the d lines, watts a
// square metre a steradian, with the cosine of the eye's height there.
//
// it holds both d lines if the band does, as full at the band's edge as
// at the equator, so the edge, where the band's air meets the next, is a
// front like any other.
func (face *Face) glowedAt(latitude, eye, held float64) float64 {
	if face.airglow == nothing || math.Abs(latitude) >= face.airband {
		return nothing
	}
	return held * face.airglow / math.Max(eye, grazing)
}
