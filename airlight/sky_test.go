package airlight

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

// counted is a sky whose radiance in each bin is one more than its
// place, so no two bins are alike.
func counted(fields *Fields) func(int, int, int) float64 {
	return func(zenith, view, round int) float64 {
		return float64(fields.index(zenith, view, round) + one)
	}
}

// countedFields is hand fields of one zenith bin, so the sun's zenith
// reads that bin and nothing between, each bin its own radiance.
func countedFields() *Fields {
	plain := handFields(one, handViews, handAzimuths, even(none))
	return handFields(one, handViews, handAzimuths, counted(plain))
}

// pointWay is the way the table's point at a view and an azimuth bin
// looks, with the sun round from north by so much.
func pointWay(fields *Fields, view, round int, sunRound float64) maths.Vec {
	cosine := centre(view, fields.views)*two - one
	about := centre(round, fields.azimuths)*math.Pi + sunRound
	across := math.Sqrt(one - cosine*cosine)
	return maths.V(across*math.Sin(about), cosine,
		-across*math.Cos(about))
}

// the sky's table at its own points is the fields there, through the
// spectrum's route, and the sky read along each point's way is it.
func TestTheSkysTableAtItsPointsIsTheFields(t *testing.T) {
	fields := countedFields()
	made := NewSky(fields, wayAt(thirty*degree, none), handExposure)
	for view := range fields.views {
		for round := range fields.azimuths {
			want := pictured(fields.radiance[none][fields.index(none, view,
				round)], handExposure)
			if got := made.table[view*fields.azimuths+round]; got != want {
				t.Errorf("point %d, %d: %v, want %v", view, round, got, want)
			}
			look := pointWay(fields, view, round, none)
			if got := made.Light(look); !near(got, want) {
				t.Errorf("along point %d, %d: %v, want %v", view, round,
					got, want)
			}
		}
	}
}

// the anti-test: the sky turns with the sun, so one way in the world,
// under a sun turned half round, reads another bin.
func TestTheSkyTurnsWithTheSun(t *testing.T) {
	fields := countedFields()
	look := pointWay(fields, handViews-one, none, none)
	facing := NewSky(fields, wayAt(thirty*degree, none), handExposure)
	turned := NewSky(fields, wayAt(thirty*degree, math.Pi), handExposure)
	if near(turned.Light(look), facing.Light(look)) {
		t.Error("the sky does not turn with the sun")
	}
}

// below and above is a sky whose light coming up is one radiance and
// whose light coming down is another.
func belowAndAbove(upward, downward float64) func(int, int, int) float64 {
	return func(_, view, _ int) float64 {
		if view < handViews/two {
			return upward
		}
		return downward
	}
}

// straightDown and straightUp are the ways down and up.
var straightDown, straightUp = maths.V(none, -one, none),
	maths.V(none, one, none)

// a sky looked at below the horizon is the fields' light coming up.
func TestBelowTheHorizonIsTheUpwardLight(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths,
		belowAndAbove(handUpward, handSky))
	made := NewSky(fields, wayAt(thirty*degree, none), handExposure)
	if got, want := made.Light(straightDown),
		pictured(handUpward, handExposure); !near(got, want) {
		t.Errorf("below the horizon %v, want %v", got, want)
	}
}

// the anti-test: below the horizon is not the horizon's light held down.
// The light coming up changed alone changes the sky below and leaves it
// above as it was.
func TestTheUpwardLightIsNotTheHorizonsHeldDown(t *testing.T) {
	way := wayAt(thirty*degree, none)
	plain := NewSky(handFields(handZeniths, handViews, handAzimuths,
		belowAndAbove(handUpward, handSky)), way, handExposure)
	brighter := NewSky(handFields(handZeniths, handViews, handAzimuths,
		belowAndAbove(two*handUpward, handSky)), way, handExposure)
	if near(brighter.Light(straightDown), plain.Light(straightDown)) {
		t.Error("the sky below does not follow the light coming up")
	}
	if !near(brighter.Light(straightUp), plain.Light(straightUp)) {
		t.Error("the light coming up changed the sky above")
	}
}

// The fields the reading's slope is checked on: a sky whose light rises
// faster and faster up from the horizon and round from the sun, so no
// bin's slope is nought, and a little light coming up from below.
const (
	smoothViews    = 8    // picked: four bins above the horizon, four below
	smoothAzimuths = 4    // picked: four bins round from the sun
	smoothBelow    = 0.5  // picked: the light coming up
	nudge          = 1e-6 // picked: a step either side of a point
	creaseShare    = 1e-3 // picked: far past a smooth step's change
	horizonCosine  = 0.01 // picked: a look a little off the horizon
	betweenRows    = 0.5  // defined: halfway between two bins' centres
	smoothRow      = 5    // picked: a bin above the horizon, inside
	smoothColumn   = 1    // picked: an azimuth bin, inside
)

// smoothRise is the light in each bin above the horizon by its place
// from the horizon up, and smoothRound by its place round from the sun:
// each doubling, or near it, so each step is larger than the last.
var (
	smoothRise  = []float64{1, 2, 4, 8}       // picked: see above
	smoothRound = []float64{1, 1.5, 2.5, 4.5} // picked: see above
)

// smoothFields is hand fields of one zenith bin whose sky rises as
// smoothRise and smoothRound say, the two multiplied.
func smoothFields() *Fields {
	return handFields(one, smoothViews, smoothAzimuths,
		func(_, view, round int) float64 {
			if view < smoothViews/two {
				return smoothBelow
			}
			return smoothRise[view-smoothViews/two] * smoothRound[round]
		})
}

// lookAt is the way at a cosine from up and an azimuth, in radians, round
// from north, where the hand sky's sun stands.
func lookAt(cosine, azimuth float64) maths.Vec {
	return wayAt(math.Acos(cosine), azimuth)
}

// crease is how much a reading's slope changes across a place, as a
// share of the slope: read a nudge before and after it along a way
// that moves by the nudge.
func crease(read func(maths.Vec) maths.Light,
	along func(float64) maths.Vec) float64 {
	luminance := func(at float64) float64 {
		return read(along(at)).Luminance()
	}
	before := (luminance(none) - luminance(-nudge)) / nudge
	after := (luminance(nudge) - luminance(none)) / nudge
	return math.Abs(after-before) / math.Max(math.Abs(before),
		math.Abs(after))
}

// creases is the crease of a reading across a view bin's centre, along
// the view's cosine, at an azimuth bin's centre and between two; and
// across an azimuth bin's centre, round from the sun, at a view bin's
// centre and between two.
func creases(fields *Fields, read func(maths.Vec) maths.Light) []float64 {
	cosine := centre(smoothRow, fields.views)*two - one
	between := (centre(smoothRow, fields.views)+centre(smoothRow+one,
		fields.views))*betweenRows*two - one
	azimuth := centre(smoothColumn, fields.azimuths) * math.Pi
	middle := (centre(smoothColumn, fields.azimuths) + centre(smoothColumn+
		one, fields.azimuths)) * betweenRows * math.Pi
	var found []float64
	for _, round := range []float64{azimuth, middle} {
		found = append(found, crease(read, func(by float64) maths.Vec {
			return lookAt(cosine+by, round)
		}))
	}
	for _, up := range []float64{cosine, between} {
		found = append(found, crease(read, func(by float64) maths.Vec {
			return lookAt(up, azimuth+by)
		}))
	}
	return found
}

// the sky's slope runs on unbroken across a bin's centre, by the view's
// cosine and by the azimuth: no crease for the eye to see as a seam.
func TestTheSkysSlopeRunsOnAcrossABinCentre(t *testing.T) {
	fields := smoothFields()
	made := NewSky(fields, wayAt(thirty*degree, none), one)
	for index, found := range creases(fields, made.Light) {
		if found > creaseShare {
			t.Errorf("crease %d: the slope changes by %.3g of itself",
				index, found)
		}
	}
}

// linearly is the same table read linearly between its four nearest
// points, over the whole view axis, as the sky read before.
func linearly(made *Sky) func(maths.Vec) maths.Light {
	return func(toward maths.Vec) maths.Light {
		toward = toward.Unit()
		view := axisOf((toward.Y+one)/two, made.views)
		round := axisOf(folded(roundOf(toward)-made.round)/math.Pi,
			made.azimuths)
		at := func(row, column int) maths.Light {
			return made.table[row*made.azimuths+column]
		}
		near := blend(at(view.low, round.low), at(view.low, round.high),
			round.share)
		far := blend(at(view.high, round.low), at(view.high, round.high),
			round.share)
		return blend(near, far, view.share)
	}
}

// blend is a light a share of the way toward another.
func blend(from, to maths.Light, share float64) maths.Light {
	return from.Scale(one - share).Add(to.Scale(share))
}

// the anti-test: the same table read linearly creases at a bin's centre
// both ways, the seam the sunward picture showed, and the check sees it.
func TestALinearReadingCreasesAtABinCentre(t *testing.T) {
	fields := smoothFields()
	made := NewSky(fields, wayAt(thirty*degree, none), one)
	found := creases(fields, linearly(made))
	for _, index := range []int{none, two} {
		if !(found[index] > creaseShare) {
			t.Errorf("crease %d: a linear reading changes its slope by "+
				"only %.3g of itself", index, found[index])
		}
	}
}

// the sky just above the horizon is the lowest bin's own light, held to
// the horizon: the light coming up from below is not blended into it.
func TestTheSkyIsHeldToTheHorizon(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths,
		belowAndAbove(handUpward, handSky))
	made := NewSky(fields, wayAt(thirty*degree, none), handExposure)
	look := lookAt(horizonCosine, none)
	if got, want := made.Light(look),
		pictured(handSky, handExposure); !near(got, want) {
		t.Errorf("just above the horizon %v, want %v", got, want)
	}
}

// the anti-test: just below the horizon is the light from below, held
// up to it; the sky above is not blended into it either.
func TestTheLightFromBelowIsHeldToTheHorizon(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths,
		belowAndAbove(handUpward, handSky))
	made := NewSky(fields, wayAt(thirty*degree, none), handExposure)
	look := lookAt(-horizonCosine, none)
	if got, want := made.Light(look),
		pictured(handUpward, handExposure); !near(got, want) {
		t.Errorf("just below the horizon %v, want %v", got, want)
	}
}

// Steffen's slope is nought where a point turns, the least step's twice
// at most, and half the mean of two steps at most; its sign is theirs.
func TestSteffensSlope(t *testing.T) {
	for _, each := range []struct{ before, here, after, want float64 }{
		{1, 2, 4, 1.5},   // defined: half of three, the mean of 1 and 2
		{4, 2, 1, -1.5},  // defined: the same, falling
		{1, 3, 2, none},  // defined: a top
		{1, 1.1, 5, 0.2}, // defined: twice the lesser step, 0.1
	} {
		got := steffenSlope(each.before, each.here, each.after)
		if math.Abs(got-each.want) > handShare {
			t.Errorf("between %g, %g and %g: %g, want %g", each.before,
				each.here, each.after, got, each.want)
		}
	}
}

// the anti-test: a flat run has no slope, and a slope is not the plain
// mean of the steps where that would carry the cubic past a point.
func TestSteffensSlopeKeepsBetweenThePoints(t *testing.T) {
	if got := steffenSlope(two, two, two); got != none {
		t.Errorf("a flat run's slope %g", got)
	}
	before, here, after := 1.0, 1.1, 5.0 // defined: steps of 0.1 and 3.9
	mean := (after - before) / two
	if got := steffenSlope(before, here, after); !(got < mean) {
		t.Errorf("the slope %g is the plain mean %g", got, mean)
	}
}

// fields read back from their bytes are the same fields, to the bit.
func TestFieldsReadBackToTheBit(t *testing.T) {
	fields := countedFields()
	read, err := FieldsFromBytes(fields.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(read, fields) {
		t.Error("the fields read back are not the fields")
	}
}

// the anti-test: bytes too short, of a length no number has, or whose
// header counts nothing, are refused.
func TestBytesThatAreNotFieldsAreRefused(t *testing.T) {
	raw := countedFields().Bytes()
	empty := make([]byte, len(raw))
	for name, made := range map[string][]byte{
		"short":            raw[:len(raw)-float64Bytes],
		"a broken number":  raw[:len(raw)-one],
		"a header of none": empty,
	} {
		if _, err := FieldsFromBytes(made); err == nil {
			t.Errorf("%s bytes read as fields", name)
		}
	}
}
