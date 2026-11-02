package film_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/film"
	"pgregory.net/rapid"
)

// anti-tests: radars of one range beam nothing in another. Radio radars
// are dark at the x-ray bands. X-ray radars are dark in radio.
func TestARadarBeamsNothingInAnotherRange(t *testing.T) {
	rapid.Check(t, radarIsDarkInOtherRanges)
}

func radarIsDarkInOtherRanges(t *rapid.T) {
	strength := rapid.Float64Range(whole, strengthMost).Draw(t, "strength")
	radio := film.NewRadar(maths.Vec{}, radiation.Radio(), strength)
	xray := film.NewRadar(maths.Vec{}, radiation.XRay(), strength)
	xrayShort, _ := radiation.XRay().Edges()
	radioShort, _ := radiation.Radio().Edges()
	inXray := maths.Band{Nanometres: xrayShort}
	inRadio := maths.Band{Nanometres: radioShort}
	if radio.Beams(inXray) != none || xray.Beams(inRadio) != none {
		t.Fatalf("a radar beamed in the range of another")
	}
}

// anti-tests: a radar is not behind the point it is seen from. The way
// toward it never points away from it.
func TestARadarIsNeverTheOtherWayFromThePoint(t *testing.T) {
	rapid.Check(t, radarIsNotAway)
}

func radarIsNotAway(t *rapid.T) {
	at := place(t, "at")
	from := apart(t, "from", at)
	glass := film.NewRadar(at, radiation.Radio(), whole)
	toward, _, _ := glass.Light(from, maths.Up)
	if toward.Dot(at.Sub(from)) <= none {
		t.Fatalf("from %v the radar at %v is toward %v", from, at, toward)
	}
}

// anti-tests: the sun has a direction and a distance that are never the
// zero vector or a place. Its direction is a unit and it is endless.
func TestTheSunIsNeverNowhereAndNeverNear(t *testing.T) {
	rapid.Check(t, sunIsNeverNowhere)
}

func sunIsNeverNowhere(t *rapid.T) {
	glass := film.NewSun(apart(t, "toward", maths.Vec{}), strong(t,
		"strength"))
	toward, dist, _ := glass.Light(place(t, "p"), place(t, "n"))
	if toward == (maths.Vec{}) || !math.IsInf(dist, whole) {
		t.Fatalf("the sun is %v at %v", toward, dist)
	}
}

// anti-tests: no surface gets more of a flood than its strength. Surfaces
// facing away get none. Floods have no direction.
func TestNoSurfaceGetsMoreOfAFloodThanItsStrength(t *testing.T) {
	rapid.Check(t, floodNeverExceedsItsStrength)
}

func floodNeverExceedsItsStrength(t *rapid.T) {
	up := apart(t, "up", maths.Vec{})
	strength := strong(t, "strength")
	glass := film.NewFlood(up, strength)
	toward, _, got := glass.Light(maths.Vec{}, apart(t, "normal",
		maths.Vec{}))
	_, _, down := glass.Light(maths.Vec{}, up.Neg())
	if got.R > strength.R+tight || got.G > strength.G+tight ||
		got.B > strength.B+tight || toward != (maths.Vec{}) ||
		down.R > tight || down.G > tight || down.B > tight {
		t.Fatalf("a flood of %v gave %v and %v facing down", strength, got,
			down)
	}
}

// anti-tests: a spot is not brighter further from the lamp. Twice the
// distance down the aim is dimmer.
func TestASpotIsDimmerFurtherDownItsAim(t *testing.T) {
	rapid.Check(t, spotIsDimmerFurther)
}

func spotIsDimmerFurther(t *rapid.T) {
	glass, aim, strength := aimed(t)
	if strength.R == none {
		t.Skip("a spot with no red")
	}
	near := rapid.Float64Range(wideLeast, eyeMost).Draw(t, "near")
	_, _, close := glass.Light(aim.Scale(near), maths.Up)
	_, _, far := glass.Light(aim.Scale(pair*near), maths.Up)
	if !(far.R < close.R) {
		t.Fatalf("at %v the spot gives %v and at twice that %v", near, close,
			far)
	}
}

// anti-tests: a spot gives no light that is negative or no number, at any point
// there is. The light is zero or more.
func TestASpotNeverGivesLessThanNoneOrNoNumber(t *testing.T) {
	rapid.Check(t, spotLightIsNeverNegative)
}

func spotLightIsNeverNegative(t *rapid.T) {
	glass, _, _ := aimed(t)
	_, _, got := glass.Light(place(t, "p"), maths.Up)
	for _, lamp := range []float64{got.R, got.G, got.B} {
		if lamp < none || math.IsNaN(lamp) {
			t.Fatalf("the spot gave %v", got)
		}
	}
}

// anti-tests: spots give nothing past their cone. Points further off the
// aim than the cone are dark.
func TestASpotGivesNothingPastItsCone(t *testing.T) {
	rapid.Check(t, spotIsDarkPastItsCone)
}

func spotIsDarkPastItsCone(t *rapid.T) {
	cone := rapid.Float64Range(wideLeast, coneMost).Draw(t, "cone")
	glass := film.NewSpot(maths.Vec{}, maths.V(none, -whole, none),
		maths.Grey(whole), cone, none)
	past := cone + rapid.Float64Range(wideLeast, math.Pi-coneMost).Draw(t,
		"past")
	if _, _, got := glass.Light(offAim(past), maths.Up); got != (maths.Light{}) {
		t.Fatalf("%v past a cone of %v the spot gave %v", past, cone, got)
	}
}

// anti-tests: a point on the aim is not dimmer than one off it at the
// same distance. The axis is the brightest.
func TestTheAxisOfASpotIsItsBrightest(t *testing.T) {
	rapid.Check(t, axisIsBrightest)
}

func axisIsBrightest(t *rapid.T) {
	cone := rapid.Float64Range(wideLeast, coneMost).Draw(t, "cone")
	glass := film.NewSpot(maths.Vec{}, maths.V(none, -whole, none),
		maths.Grey(whole), cone, cone/pair)
	off := rapid.Float64Range(none, math.Pi).Draw(t, "off")
	_, _, onAxis := glass.Light(offAim(none), maths.Up)
	_, _, away := glass.Light(offAim(off), maths.Up)
	if away.R > onAxis.R+tight {
		t.Fatalf("%v off the aim is brighter than on it", off)
	}
}

// anti-tests: spots aimed away from a point give it nothing. Points
// directly behind the aim are dark.
func TestASpotAimedAwayGivesNothing(t *testing.T) {
	rapid.Check(t, spotAimedAwayIsDark)
}

func spotAimedAwayIsDark(t *rapid.T) {
	glass, aim, _ := aimed(t)
	behind := aim.Scale(-rapid.Float64Range(whole, eyeMost).Draw(t,
		"behind"))
	if _, _, got := glass.Light(behind, maths.Up); got != (maths.Light{}) {
		t.Fatalf("a spot aimed away gave %v behind it", got)
	}
}

// anti-tests: a laser lights nothing more than three widths off its
// axis, whatever the point along it.
func TestALaserLightsNothingBeyondThreeWidthsOff(t *testing.T) {
	rapid.Check(t, laserIsDarkBeyondItsCutoff)
}

func laserIsDarkBeyondItsCutoff(t *rapid.T) {
	glass, at, aim, width, length := aligned(t)
	along := rapid.Float64Range(half, length*insideMost).Draw(t, "along")
	off := laserCutoff*width + rapid.Float64Range(wideLeast, whole).Draw(t,
		"off")
	point := at.Add(aim.Scale(along)).Add(crossing(aim).Scale(off))
	if _, _, got := glass.Light(point, maths.Up); got != (maths.Light{}) {
		t.Fatalf("%v off a beam of %v the laser gave %v", off, width, got)
	}
}

// anti-tests: a wider beam lights no less at the same offset. Doubling
// the width does not dim a point beside the axis.
func TestAWiderBeamLightsNoLessAtTheSameOffset(t *testing.T) {
	rapid.Check(t, widerBeamIsNotDimmer)
}

func widerBeamIsNotDimmer(t *rapid.T) {
	narrow, at, aim, width, length := aligned(t)
	wide := film.NewLaser(at, aim, pair*width, maths.Grey(whole), length)
	along := rapid.Float64Range(half, length*insideMost).Draw(t, "along")
	off := rapid.Float64Range(none, laserCutoff*width).Draw(t, "off")
	point := at.Add(aim.Scale(along)).Add(crossing(aim).Scale(off))
	_, _, first := narrow.Light(point, maths.Up)
	_, _, second := wide.Light(point, maths.Up)
	if second.R < first.R-tight {
		t.Fatalf("a wider beam gave %v where a narrow one gave %v", second,
			first)
	}
}

// anti-tests: a laser never lights more than its strength, and never
// lights anything dark where it is lit. The gaussian is at most one.
func TestALaserNeverLightsMoreThanItsStrength(t *testing.T) {
	rapid.Check(t, laserNeverExceedsItsStrength)
}

func laserNeverExceedsItsStrength(t *rapid.T) {
	glass, at, aim, width, length := aligned(t)
	along := rapid.Float64Range(none, length).Draw(t, "along")
	off := rapid.Float64Range(none, laserCutoff*width).Draw(t, "off")
	point := at.Add(aim.Scale(along)).Add(crossing(aim).Scale(off))
	if _, _, got := glass.Light(point, maths.Up); got.R > whole+tight ||
		got.R < none {
		t.Fatalf("the laser of strength one gave %v", got)
	}
}

// anti-tests: a point outside the bounds of a beam is dark. The box
// holds everything the beam lights and nothing needs to look past it.
func TestAPointOutsideTheBoundsOfABeamIsDark(t *testing.T) {
	rapid.Check(t, outsideBoundsIsDark)
}

func outsideBoundsIsDark(t *rapid.T) {
	glass, _, _, _, _ := aligned(t)
	box := glass.Bounds()
	past := rapid.Float64Range(whole, eyeMost).Draw(t, "past")
	for _, point := range []maths.Vec{
		maths.V(box.Max.X+past, box.Min.Y, box.Min.Z),
		maths.V(box.Min.X, box.Min.Y-past, box.Min.Z),
		maths.V(box.Min.X, box.Min.Y, box.Max.Z+past)} {
		if _, _, got := glass.Light(point, maths.Up); got != (maths.Light{}) {
			t.Fatalf("%v outside the box the laser gave %v", point, got)
		}
	}
}

// anti-tests: the bounds of a beam are not a point. The box is at least
// the length of the beam long and three widths round it.
func TestTheBoundsOfABeamAreNotSmallerThanTheBeam(t *testing.T) {
	rapid.Check(t, boundsHoldTheBeam)
}

func boundsHoldTheBeam(t *rapid.T) {
	glass, at, aim, _, length := aligned(t)
	box := glass.Bounds()
	end := at.Add(aim.Scale(length))
	for _, point := range []maths.Vec{at, end} {
		if point.X < box.Min.X-loose || point.X > box.Max.X+loose ||
			point.Y < box.Min.Y-loose || point.Y > box.Max.Y+loose ||
			point.Z < box.Min.Z-loose || point.Z > box.Max.Z+loose {
			t.Fatalf("%v is outside the box %v", point, box)
		}
	}
}
