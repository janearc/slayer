package generate

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// The detail the like-earth sets draw their ground at.
const coastDetail = 5

// coastHeight is the low camera's height the tests ask for, in metres.
const coastHeight = 20000

// coastPlanet is a planet like the Earth, its axis not tilted so that
// its frame is the set's.
func coastPlanet() Planet {
	planet := likeEarth()
	planet.Tilt = measure.Degrees(0)
	return planet
}

// metresAcross is a planet's radius in metres.
func metresAcross(t *testing.T, planet Planet) float64 {
	t.Helper()
	derived, err := Derive(planet)
	if err != nil {
		t.Fatal(err)
	}
	return derived.Radius.InKilometres() * metresPerKilometre
}

// shoreApart is how far either side of a shore found the test looks, in
// metres: a few, under the low camera's pixel of about twenty, finer than
// which the ground has no detail. At a hundred, four pixels, a coast that
// breaks finer to the pixel may turn back between the two.
const shoreApart = 5

// Where the walk meets a shore, the ground has land on one side and sea
// on the other, a few metres apart.
func TestTheShoreFoundHasLandOnOneSideAndSeaOnTheOther(t *testing.T) {
	body, err := Make(coastPlanet())
	if err != nil {
		t.Fatal(err)
	}
	ground, err := body.World(coastDetail)
	if err != nil {
		t.Fatal(err)
	}
	radius := metresAcross(t, coastPlanet())
	pixel := coastHeight / lowPixels / radius
	shore, found := newWalk(worldGround{made: ground}, maths.XAxis,
		pixel).shore()
	if !found {
		t.Fatal("no shore along the meridian of an Earth like planet")
	}
	across := shoreApart / radius
	sea := ground.LandShare(onCircle(shore.Point, shore.Seaward, across),
		pixel, coastMix)
	land := ground.LandShare(onCircle(shore.Point, shore.Seaward, -across),
		pixel, coastMix)
	if sea >= halfLand || land < halfLand {
		t.Errorf("seaward is %v land and landward is %v", sea, land)
	}
}

// The low camera stands the height asked over the shore and looks at
// the ground there; the high one is sixteen times as high, over the
// sea, and looks at the same shore, which is so in its frame.
func TestCoastCamerasStandAtTheirHeights(t *testing.T) {
	planet := coastPlanet()
	cameras, err := FindCoast(planet, coastDetail, maths.XAxis, coastHeight)
	if err != nil {
		t.Fatal(err)
	}
	radius := metresAcross(t, planet)
	want := map[string]float64{
		"low at": radius + coastHeight, "low look": radius,
		"high at": radius + coastHeight*highRise, "high look": radius}
	got := map[string]float64{"low at": cameras.Low.At.Len(),
		"low look": cameras.Low.LookAt.Len(), "high at": cameras.High.At.Len(),
		"high look": cameras.High.LookAt.Len()}
	for name, metres := range want {
		if math.Abs(got[name]-metres) > 1 {
			t.Errorf("%s is %v metres from the middle, not %v", name,
				got[name], metres)
		}
	}
	body, _ := Make(planet)
	ground, _ := body.World(coastDetail)
	over := ground.LandShare(cameras.High.At.Unit(), 0, coastMix)
	if over >= halfLand {
		t.Error("the high camera is over land, not the sea")
	}
	if cameras.High.LookAt != cameras.Low.LookAt {
		t.Errorf("the high camera looks at %v, not the shore at %v",
			cameras.High.LookAt, cameras.Low.LookAt)
	}
}

// Starting at a pole is no different but for the circle's choice.
func TestACoastIsFoundFromAPole(t *testing.T) {
	_, err := FindCoast(coastPlanet(), coastDetail, maths.Up, coastHeight)
	if err != nil && !errors.Is(err, ErrNoCoast) {
		t.Errorf("a pole start fails with %v, not a coast or none", err)
	}
	pole := meridianTangent(maths.Up)
	if math.Abs(pole.Dot(maths.Up)) > 1e-9 || math.Abs(pole.Len()-1) > 1e-9 {
		t.Errorf("a pole's tangent is %v", pole)
	}
}

// The same planet and start give the same cameras.
func TestCoastCamerasAreRepeatable(t *testing.T) {
	first, err := FindCoast(coastPlanet(), coastDetail, maths.XAxis,
		coastHeight)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := FindCoast(coastPlanet(), coastDetail, maths.XAxis,
		coastHeight)
	if first != again {
		t.Errorf("two finds differ: %v and %v", first, again)
	}
}

// halves is a ground that is land where x is over a line and sea under
// it: not a world, to show the finder asks only the Ground interface.
type halves struct{ line float64 }

// LandShare is wholly land over the line and wholly sea under it.
func (ground halves) LandShare(p maths.Vec, _ float64) float64 {
	if p.X > ground.line {
		return 1
	}
	return 0
}

// Spacing is a fifth of a radian.
func (ground halves) Spacing() float64 { return 0.2 }

// A ground that is not a world serves: with land for x over nought, a
// walk from minus x over the pole meets its coast at the pole.
func TestACoastIsFoundOnAnyGround(t *testing.T) {
	cameras, err := FindCoastOn(halves{}, 6.4e6, measure.Degrees(0),
		maths.V(-1, 0, 0), coastHeight)
	if err != nil {
		t.Fatal(err)
	}
	look := cameras.Low.LookAt.Unit()
	if look.Sub(maths.Up).Len() > 1e-6 {
		t.Errorf("the coast is at %v, not the pole", look)
	}
}

// Anti-test: a ground with no land, or none but all of it, has no coast
// whatever the walk, and gives no camera.
func TestAGroundOfOneKindHasNoCoast(t *testing.T) {
	for _, line := range []float64{-2, 2} {
		cameras, err := FindCoastOn(halves{line}, 6.4e6, measure.Degrees(0),
			maths.V(-1, 0, 0), coastHeight)
		if !errors.Is(err, ErrNoCoast) || cameras != (Cameras{}) {
			t.Errorf("a line at %v gives %v and %v", line, cameras, err)
		}
	}
}

// Anti-test: a world with no sea has no coast, and gives no camera over
// land.
func TestADryWorldHasNoCoast(t *testing.T) {
	planet := coastPlanet()
	planet.Makeup = Makeup{Metal: 0.32, Rock: 0.68}
	cameras, err := FindCoast(planet, coastDetail, maths.XAxis, coastHeight)
	if !errors.Is(err, ErrNoCoast) {
		t.Errorf("a dry world gives %v, not no coast", err)
	}
	if cameras != (Cameras{}) {
		t.Errorf("a dry world gives cameras %v", cameras)
	}
}

// Anti-test: a gas giant has no ground to walk, and says so, which is
// not the same as a ground with no coast.
func TestAGiantHasNoGroundToWalk(t *testing.T) {
	_, err := FindCoast(likeJupiter(), coastDetail, maths.XAxis, coastHeight)
	if err == nil || errors.Is(err, ErrNoCoast) {
		t.Errorf("a giant gives %v, not a refusal of its ground", err)
	}
}

// Anti-test: no height, a height past counting, or no direction to start
// from is refused by name, not walked.
func TestACoastIsNotFoundFromNothing(t *testing.T) {
	for _, height := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := FindCoast(coastPlanet(), coastDetail, maths.XAxis,
			height); err == nil || errors.Is(err, ErrNoCoast) {
			t.Errorf("a height of %v gives %v", height, err)
		}
	}
	if _, err := FindCoast(coastPlanet(), coastDetail, maths.Vec{},
		coastHeight); err == nil || errors.Is(err, ErrNoCoast) {
		t.Errorf("no direction gives %v", err)
	}
}
