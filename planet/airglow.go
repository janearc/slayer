package planet

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// sodium's d lines, nanometres in air: d2 and d1, nist's atomic spectra
// database (nist-asd-na-i-visible on the shelf), both rated aa.
const (
	sodiumD2 = 588.995095
	sodiumD1 = 589.592424
)

// grazing is the least cosine of the eye's height a thin glowing layer
// is taken at, so its path through the layer stays finite at the limb:
// picked.
const grazing = 0.05

// a front is read across a degree, about three weather cells at 1024
// round, so the solver's ripples a cell wide are not taken for fronts;
// and one brightens the glow at most so many times, so the steepest edge
// the weather draws stays finite. both picked.
const (
	frontStep = 1.0
	mostFront = 6.0
)

// front is how strongly the airglow shows at a way of the face: none
// where its air's origin changes no more steeply than the band's calm,
// one where twice as steeply, up to mostFront. with no flow, one.
//
// so the band's inside is dark, and the whorls' edges and the band's
// boundary glow.
//
// as asked: brightest at the fronts, the whorls' edges and the boundary
// between the bands, and the band's inside darker. the weather here is
// barotropic and has no compression; the strain that draws fronts is
// what this reads, standing in for it.
func (face *Face) front(up maths.Vec, finest float64) float64 {
	if face.flow == nil {
		return 1
	}
	steep := face.steepness(up, finest) / face.calm
	return math.Min(mostFront, math.Max(0, steep-1))
}

// calmness is how steeply the band's air's origin changes everywhere but
// its fronts: the steepness its steepest share, its fronts, lies above,
// read on an even grid over the band. one with no flow, no glow, or a
// flat band.
func (face *Face) calmness() float64 {
	if face.flow == nil || face.airglow == 0 {
		return 1
	}
	const rows, round = 32, 128
	steeps := make([]float64, 0, rows*round)
	for i := range rows {
		latitude := face.airband * (2*(float64(i)+0.5)/rows - 1)
		for k := range round {
			longitude := 360 * float64(k) / round
			up := face.ways.toward(latitude, longitude)
			steeps = append(steeps, face.steepness(up, 0))
		}
	}
	slices.Sort(steeps)
	calm := steeps[min(len(steeps)-1,
		int((1-face.fronts)*float64(len(steeps))))]
	if !(calm > 0) {
		return 1
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
	wide := step / math.Max(math.Cos(latitude*maths.Degree), 0.05)
	origin := func(north, east float64) float64 {
		came, _, _ := face.flowed(face.ways.toward(latitude+north,
			longitude+east))
		return came
	}
	northward := (origin(step, 0) - origin(-step, 0)) / (2 * step)
	eastward := (origin(0, wide) - origin(0, -wide)) / (2 * step)
	return math.Hypot(northward, eastward)
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
// through it lengthens, as earth's own sodium layer does seen from orbit, and
// shows on the night side as by day.
//
// it glows in the air the band's x-rays lit, so it goes where that air goes:
// the weather tears and folds it as it does the zones.
func (body Body) Airglowing(radiance, band, fronts float64) (Body,
	error) {
	if !(radiance >= 0) || !(band > 0 && band <= 90) ||
		!(fronts > 0 && fronts < 1) {
		return body, fmt.Errorf("an airglow of %g W/m²/sr over %g degrees, "+
			"its fronts the steepest %g", radiance, band, fronts)
	}
	body.airglow, body.airband, body.airfronts = radiance, band, fronts
	return body, nil
}

// glowedAt is the airglow a point sends the eye whose air came from a
// latitude in degrees, in a band from lo to hi nanometres, watts a
// square metre a steradian, with the cosine of the eye's height there.
//
// it holds both d lines if the band does, as full at the band's edge as
// at the equator, so the edge, where the band's air meets the next, is a
// front like any other.
func (face *Face) glowedAt(latitude, eye, lo, hi float64) float64 {
	if face.airglow == 0 || math.Abs(latitude) >= face.airband {
		return 0
	}
	held := 0.0
	for _, line := range []float64{sodiumD2, sodiumD1} {
		if line >= lo && line < hi {
			held += 0.5
		}
	}
	return held * face.airglow / math.Max(eye, grazing)
}
