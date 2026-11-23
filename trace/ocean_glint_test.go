package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// satinBed is the material of a seabed that leaves highlights: a Satin
// of a half, and all of fresnel's share mirrored.
var satinBed = &thing.Material{Gloss: 1, Satin: 0.5}

// bedEye is the way from a point of the seabed to the eye of these
// tests: up at a cosine of 0.8, leaning away from the sun.
var bedEye = maths.V(-0.6, 0.8, 0)

// refractedSun is the way a sun at an angle from straight up in the air
// arrives two units under a level sea of water, toward the sun, and its
// light there. By Snell's law the sine in the water is the sine in the
// air over the index. A sun of ten on a surface square to it lands on
// the level sea by its cosine in the air, passes by what fresnel does
// not mirror, and is spread over a beam wider by the cosine in the
// water; and the way to the sea is two over that cosine, through water
// that takes 0.2, 0.1 and 0.05 a unit of red, green and blue.
func refractedSun(angle float64) (maths.Vec, maths.Light) {
	sine := math.Sin(angle) / waterIndex
	cosine := math.Sqrt(1 - sine*sine)
	passes := 10 * (1 - mirrorShare(math.Cos(angle), waterIndex)) *
		math.Cos(angle) / cosine
	way := 2 / cosine
	return maths.V(sine, cosine, 0), maths.RGB(passes*math.Exp(-0.2*way),
		passes*math.Exp(-0.1*way), passes*math.Exp(-0.05*way))
}

// bedGlint is the highlight oceanGlint leaves at a point two units under
// the tagged sea, on a satin surface facing a way, seen from bedEye, by
// a look that is in the sea's water or in none.
func bedGlint(t *testing.T, angle float64, facing maths.Vec,
	inWater bool) (*staged, maths.Light) {
	t.Helper()
	world, _, _, seen, _ := oceanSunFixture(t, angle)
	if !inWater {
		seen = look{}
	}
	scene := stage(world, 0)
	return scene, scene.oceanGlint(maths.V(0, -2, 0), facing, bedEye,
		satinBed, seen, &thing.Thing{})
}

// Under the tagged sea the sun's highlight is the sun's light as it
// arrives through the water, by the surface's highlight lobe and its
// mirrored share at the way that light arrives: the refracted way, not
// the sun's way in the air. Neither lamp adds a highlight by the
// straight way, since the sea's own surface stands in it.
//
// The light is asked from a hair above the point, the scene's lift:
// a millionth of the scene's reach, and the sea is twenty units across,
// so of the order of 2e-5 of a unit. That shortens the water by as
// much: the tolerance is a ten thousandth of the highlight.
func TestOceanGlintIsTheRefractedSunsHighlight(t *testing.T) {
	for _, angle := range []float64{0, 0.5} {
		_, got := bedGlint(t, angle, maths.Up, true)
		toward, light := refractedSun(angle)
		half := toward.Add(bedEye).Unit()
		lobe := thing.Microfacet(half.Y, toward.Y, bedEye.Y, 0.5*0.5)
		facing := half.Dot(bedEye)
		want := light.Times(satinBed.Reflects(facing)).Scale(lobe)
		if !(want.B > want.R && want.R > 0) {
			t.Fatalf("the test's own highlight is %v", want)
		}
		lightIs(t, "the highlight under the sea", got, want, 1e-4*want.R)
	}
}

// Anti: a surface turned from the refracted sun takes no highlight of
// it, and a look that is not in the sea's water takes the lamps by the
// straight way as any surface does, where the sea's surface hides them.
func TestOceanGlintIsDarkTurnedFromTheSunOrSeenFromOutsideTheWater(
	t *testing.T) {
	facingAway := maths.V(-1, 0, 0)
	if toward, _ := refractedSun(0.5); !(facingAway.Dot(toward) < 0 &&
		facingAway.Dot(bedEye) > 0) {
		t.Fatal("the test's surface is not turned from the sun alone")
	}
	if _, got := bedGlint(t, 0.5, facingAway, true); got != (maths.Light{}) {
		t.Errorf("turned from the sun the highlight is %v", got)
	}
	scene, got := bedGlint(t, 0.5, maths.Up, false)
	plain := scene.glint(maths.V(0, -2, 0), maths.Up, bedEye, satinBed)
	if got != plain || got != (maths.Light{}) {
		t.Errorf("out of the water the highlight is %v, and the plain "+
			"one %v, want neither", got, plain)
	}
}
