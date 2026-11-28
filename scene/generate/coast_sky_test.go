package generate

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// stripeCount is how many bands of land the striped ground has in half
// a turn round its meridian from minus x, and stripeShift how far round
// they are moved, so the walk starts in the sea and its first two shores
// are at an eighth and three eighths of half a turn.
const (
	stripeCount = 4
	stripeShift = math.Pi / 8
	firstShore  = stripeShift
	secondShore = stripeShift + math.Pi/stripeCount
)

// striped is a ground of bands of land and sea round the meridian from
// minus x, with a sky clouded from the start to a turn of cloudedTo: not
// a world, so the test says where its shores and clouds are.
type striped struct{ cloudedTo float64 }

// turnOf is how far round the meridian from minus x toward the north a
// place is.
func turnOf(p maths.Vec) float64 { return math.Atan2(p.Y, -p.X) }

// LandShare is land where the bands are and sea between them.
func (ground striped) LandShare(p maths.Vec, _ float64) float64 {
	if math.Sin(stripeCount*(turnOf(p)-stripeShift)) > 0 {
		return 1
	}
	return 0
}

// Spacing is a fifth of a radian.
func (ground striped) Spacing() float64 { return 0.2 }

// Clouded is all cloud from the start up to the turn given, counted
// once round, and clear past it.
func (ground striped) Clouded(p maths.Vec) float64 {
	if math.Mod(turnOf(p)+fullTurn, fullTurn) < ground.cloudedTo {
		return 1
	}
	return 0
}

// unclouded is the striped ground as a Ground that is not a Sky.
type unclouded struct{ ground striped }

// LandShare is the striped ground's.
func (ground unclouded) LandShare(p maths.Vec, pixel float64) float64 {
	return ground.ground.LandShare(p, pixel)
}

// Spacing is the striped ground's.
func (ground unclouded) Spacing() float64 { return ground.ground.Spacing() }

// shoreTurn is how far round the meridian the coast the cameras of a
// ground look at is.
func shoreTurn(t *testing.T, ground Ground) float64 {
	t.Helper()
	cameras, err := FindCoastOn(ground, 6.4e6, measure.Degrees(0),
		maths.V(-1, 0, 0), coastHeight)
	if err != nil {
		t.Fatal(err)
	}
	return turnOf(cameras.Low.LookAt)
}

// turnHair is how near a shore found must be to the one meant, as a
// turn: a small share of a band.
const turnHair = 0.01

// A coast camera that shows no coast fails its purpose: with the first
// shore of the walk under cloud, the coast found is the next, under
// clear sky.
func TestTheCoastFoundIsUnderClearSky(t *testing.T) {
	got := shoreTurn(t, striped{cloudedTo: (firstShore + secondShore) / 2})
	if math.Abs(got-secondShore) > turnHair {
		t.Errorf("the coast found is at %v, not the clear one at %v", got,
			secondShore)
	}
}

// Anti-test: a ground with no sky, or a clear one, gives the first shore
// of the walk, as it always did.
func TestWithNoCloudTheFirstShoreIsFound(t *testing.T) {
	for name, ground := range map[string]Ground{"no sky": unclouded{},
		"clear": striped{}} {
		if got := shoreTurn(t, ground); math.Abs(got-firstShore) >
			turnHair {
			t.Errorf("%s: the coast found is at %v, not %v", name, got,
				firstShore)
		}
	}
}

// skyPlaces is how many places round an Earth like planet the sky test
// asks.
const skyPlaces = 400

// The finder asks the planet's own clouds, as the renderer lays them:
// over an Earth like planet some places are under a patch's heart and
// some are clear. A planet with no clouds has no sky.
func TestTheSkyIsThePlanetsClouds(t *testing.T) {
	body, err := Make(coastPlanet())
	if err != nil {
		t.Fatal(err)
	}
	ground := worldGround{sky: skyOf(body)}
	clouded, clear := 0, 0
	for place := range skyPlaces {
		turn := fullTurn * float64(place) / skyPlaces
		over := ground.Clouded(maths.V(math.Cos(turn),
			math.Sin(stripeCount*turn)/2, math.Sin(turn)))
		if over > 0.5 {
			clouded++
		}
		if over == 0 {
			clear++
		}
	}
	if clouded == 0 || clear == 0 {
		t.Errorf("of %d places %d are clouded and %d clear", skyPlaces,
			clouded, clear)
	}
	dry := coastPlanet()
	dry.Makeup = Makeup{Metal: 0.32, Rock: 0.68}
	parched, err := Make(dry)
	if err != nil {
		t.Fatal(err)
	}
	if sky := skyOf(parched); sky != nil {
		t.Errorf("a dry planet has a sky of %+v", *sky)
	}
}

// Anti-test: a sky clouded everywhere is no reason to find no coast. The
// walk takes the least clouded shore it met, here the first.
func TestUnderCloudEverywhereACoastIsStillFound(t *testing.T) {
	if got := shoreTurn(t, striped{cloudedTo: fullTurn}); math.Abs(
		got-firstShore) > turnHair {
		t.Errorf("under cloud the coast found is at %v, not %v", got,
			firstShore)
	}
}
