package skylight

import (
	"fmt"
	"math"
	"testing"
)

// The shell the sun is seen through.
const (
	handExtinction = 1e-5 // picked: an optical depth of one straight up
	handShare      = 1e-9 // picked: the chords' subtraction keeps nine
	// digits near grazing; a wrong path is out by percents
	metreExtinction = 1e-3 // picked: about three along the grazing chord
)

// a sighting is a height and a sun's zenith, in degrees.
type sighting struct{ height, zenith float64 }

// sightings are a few heights and angles, above and below the horizon.
var sightings = []sighting{
	{0, 0}, {0, 60}, {0, 90}, {airTop / two, 95}, // picked: a few
	{airTop / two, 120}, {airTop, 30}, {airTop, 100}, // picked: of each
}

// byHand is the sun's share at a height and zenith, through one uniform
// shell of extinction per metre from the ground to top, worked out
// along the straight line to the sun.
//
// From radius r at zenith z, the line meets the top sphere, of radius
// s, after t where s squared is r squared plus t squared plus 2 r t cos
// z: t is the root of s squared less r squared sin squared z, less r
// cos z.
//
// Below the horizon the planet hides the sun when the line passes
// nearer the centre than the ground: r sin z less than the radius.
func byHand(radius, top, extinction float64) func(sighting) float64 {
	return func(at sighting) float64 {
		r, zenith := radius+at.height, at.zenith*degree
		sine := r * math.Sin(zenith)
		if zenith > math.Pi/two && sine < radius {
			return none
		}
		outer := radius + top
		path := math.Sqrt((outer-sine)*(outer+sine)) - r*math.Cos(zenith)
		return math.Exp(-extinction * path)
	}
}

// flat is the same sun through a plane layer: the height left above
// over the zenith's cosine. It is wrong on a sphere, as the anti-test
// shows the check can tell.
func flat(top, extinction float64) func(sighting) float64 {
	return func(at sighting) float64 {
		slant := (top - at.height) / math.Cos(at.zenith*degree)
		return math.Exp(-extinction * slant)
	}
}

// agrees is nil when the field's transmittance matches want at every
// sighting, to a share of the larger.
func agrees(field *Field, places []sighting,
	want func(sighting) float64) error {
	for _, at := range places {
		got := field.Transmittance(at.height, at.zenith*degree)
		wanted := want(at)
		if !(math.Abs(got-wanted) <= handShare*max(got, wanted)) {
			return fmt.Errorf("%+v: %.12g, want %.12g", at, got, wanted)
		}
	}
	return nil
}

// handColumn is the one shell of the hand-worked sightings, split
// between its kinds of extinction so each is counted.
func handColumn() Column {
	shell := Shell{GasScattering: handExtinction / two,
		GasAbsorption:      handExtinction / two / two,
		ParticleExtinction: handExtinction / two / two,
		ParticleAlbedo:     half}
	return uniformColumn(earthRadius, airTop, shell, none)
}

// the sun's share is exp of minus the optical depth along the straight
// line to it, through a sphere's shell, at heights from the ground to
// the top and angles to below the horizon; and through a shell one
// metre thick at the planet's radius, along the grazing line.
func TestTransmittanceByHand(t *testing.T) {
	field := fieldOf(t, handColumn())
	want := byHand(earthRadius, airTop, handExtinction)
	if err := agrees(field, sightings, want); err != nil {
		t.Error(err)
	}
	metre := uniformColumn(earthRadius, one,
		Shell{GasAbsorption: metreExtinction}, none)
	grazing := []sighting{{0, 90}, {half, 90}, {half, 89.99}} // picked:
	// along the ground, and halfway up, level and just above it
	if err := agrees(fieldOf(t, metre), grazing,
		byHand(earthRadius, one, metreExtinction)); err != nil {
		t.Error(err)
	}
}

// the anti-test: the plane layer's slant, height over cosine, is not
// what the sphere gives at sixty degrees, so the check would catch a
// path worked out flat.
func TestAFlatPathIsCaught(t *testing.T) {
	field := fieldOf(t, handColumn())
	sixty := []sighting{{0, 60}} // picked: well above the horizon
	if agrees(field, sixty, flat(airTop, handExtinction)) == nil {
		t.Error("a flat path agrees with the sphere's")
	}
}

// fieldOf is the field of a column walked with a few photons, for what
// does not depend on them.
func fieldOf(t *testing.T, column Column) *Field {
	t.Helper()
	field, _, err := Solve(column, groundGrid(), workersOf, seedOf,
		workersOf)
	if err != nil {
		t.Fatal(err)
	}
	return field
}
