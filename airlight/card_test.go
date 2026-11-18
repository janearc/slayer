package airlight

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// How finely the sky is summed over the half of it above a level card.
const (
	// cardSteps is how many steps of the cosine from up, and of the
	// azimuth, the sum takes. The middle of each step is exact for an
	// even sky, whose light on the card is the cosine's own integral.
	cardSteps = 64 // picked: see above
	// cardShare is how near the card must come to its grey: a few
	// thousand roundings of the sum.
	cardShare = 1e-9 // picked: see above
	// radianceShare is how far apart a card under the two conventions
	// must be in the anti-test: well short of pi, the factor between a
	// lamp taken as light landing and one taken as light seen.
	radianceShare = 1.5 // picked: see above
)

// skyOnCard is the light the sky lands on a level card: its light along
// each way above, times the way's cosine from up, over the half sphere,
// summed at the middle of equal steps of the cosine and the azimuth.
func skyOnCard(sky *Sky) maths.Light {
	var sum maths.Light
	step := one / float64(cardSteps)
	turn := fullTurn / float64(cardSteps)
	for up := range cardSteps {
		cosine := (float64(up) + half) * step
		across := math.Sqrt(one - cosine*cosine)
		for round := range cardSteps {
			about := (float64(round) + half) * turn
			way := maths.V(across*math.Sin(about), cosine,
				-across*math.Cos(about))
			sum = sum.Add(sky.Light(way).Scale(cosine * step * turn))
		}
	}
	return sum
}

// cardUnder is a level matte card of the grey card's reflectance as the
// tracer lights it, with the sun at its exposure along a way: the sun's
// lamp, light landing square to it, by its cosine; and the sky's light
// on it; both over pi. lampOver is what the lamp's part is divided by.
func cardUnder(t *testing.T, fields *Fields, way maths.Vec,
	lampOver float64) float64 {
	t.Helper()
	exposure, err := Meter(fields, way)
	if err != nil {
		t.Fatal(err)
	}
	lamp := Sun(fields, way, exposure).Scale(way.Unit().Y / lampOver)
	sky := skyOnCard(NewSky(fields, way, exposure)).Scale(one / math.Pi)
	return lamp.Add(sky).Scale(grey).Luminance()
}

// under the meter's exposure, a level grey card lit by the sun's lamp
// and the sky as the tracer lights a matte surface is its own grey: the
// lamp is light landing and the sky is light seen, in one unit.
func TestAGreyCardIsGreyUnderTheSunAndTheSky(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths, even(handSky))
	got := cardUnder(t, fields, wayAt(sixty*degree, none), math.Pi)
	if math.Abs(got-grey) > cardShare {
		t.Errorf("a grey card is %.12g in the picture, want %g", got, grey)
	}
}

// the anti-test: the lamp taken as light seen, not landing, so not
// divided by pi, makes the same card far brighter than its grey, so the
// check above sees the convention.
func TestALampTakenAsLightSeenIsTooBright(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths, even(handSky))
	got := cardUnder(t, fields, wayAt(sixty*degree, none), one)
	if !(got > radianceShare*grey) {
		t.Errorf("a card under a lamp taken as light seen is %g, not "+
			"above %g", got, radianceShare*grey)
	}
}
