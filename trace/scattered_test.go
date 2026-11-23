package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// leaning is a steady medium that throws light one way more than
// another, by a Henyey-Greenstein asymmetry.
type leaning struct {
	steady
	asymmetry float64
}

// Asymmetry is the lobe's asymmetry.
func (mist leaning) Asymmetry() float64 { return mist.asymmetry }

// Known shortfall, pinned as it is. With no volume sky asked for, a bit
// of smoke sends toward the eye its colour times the sun's light times a
// half, whichever way the sun stands: 0.8, 0.4 and 0.2 of a sun of two,
// halved, is 0.8, 0.4 and 0.2.
//
// The right weight for a scatterer that sends light every way alike is
// its phase, one over the sphere's four pi steradians, 0.0796. The half
// is two pi times that. This test is to be rewritten to one in four pi
// when the legacy weight is put right.
func TestKnownShortfallLegacySmokeSendsHalfTheSunEveryWay(t *testing.T) {
	bit := smokeBit(sheet(0.7, -1, 1))
	want := maths.RGB(0.8, 0.4, 0.2)
	for name, toward := range sunWays {
		sunlight := sunlit(toward, maths.Grey(2))
		got := scattered(sunlight, &bit, midway, alongZ, false)
		lightIs(t, "smoke with the sun "+name, got, want, 1e-12)
	}
}

// Anti: smoke a sun cannot reach sends nothing of it. A ball stands
// between the bit and a sun overhead.
func TestSmokeBehindABallSendsNothingOfTheSun(t *testing.T) {
	bit := smokeBit(sheet(0.7, -1, 1))
	world := thing.NewWorld()
	sphereAt := resting(maths.V(0, 5, 0))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Motion: sphereAt})
	light := maths.Grey(2)
	world.Lamp(thing.Sun(maths.Up, light))
	got := scattered(stage(world, 0), &bit, midway, alongZ, false)
	if got != (maths.Light{}) {
		t.Errorf("smoke in a ball's shadow sends %v, want nothing", got)
	}
}

// The legacy weight is the constant evenly, and its value is a half:
// two pi times the phase of an even scatterer, one in four pi. Known
// shortfall, as the test above says; this holds the constant to the
// number the pictures were made with, and to how far that is from the
// phase the opted-in volume sky uses.
func TestKnownShortfallEvenlyIsAHalfAndTwoPiTimesTheEvenPhase(t *testing.T) {
	if evenly != 0.5 {
		t.Errorf("evenly is %v, want the half it has been", evenly)
	}
	if over := evenly / thing.IsotropicPhase(); !within(over, 2*math.Pi,
		1e-12) {
		t.Errorf("evenly is %v times the even phase, want two pi", over)
	}
}

// Over the sphere's four pi steradians the even phase adds up to one,
// as a phase must, which the half the pictures were made with does not.
func TestTheEvenPhaseAddsUpToOneOverTheSphere(t *testing.T) {
	if whole := thing.IsotropicPhase() * 4 * math.Pi; !within(whole, 1,
		1e-12) {
		t.Errorf("the even phase adds up to %v over the sphere", whole)
	}
}

// A lopsided mist sends more of a sun on ahead than back: its lobe at
// the angle the light turns through, over the even lobe's one in four
// pi. Henyey and Greenstein's lobe over the even one is
// (1 - g^2) / (1 + g^2 - 2 g c)^1.5 at a cosine c. At g of a half:
// straight on to the eye, c = 1, it is 0.75 / 0.5^3 = 6; turned right
// round, c = -1, 0.75 / 1.5^3 = 2/9; and square across, c = 0,
// 0.75 / 1.25^1.5.
//
// The mist is held to an even one under the same sun, so the weight an
// even scatterer takes does not come into it.
func TestALopsidedMistSendsASunByItsLobeOverTheEvenOne(t *testing.T) {
	lobes := map[string]float64{"behind the eye": 2.0 / 9,
		"ahead of the eye": 6, "overhead": 0.75 / math.Pow(1.25, 1.5)}
	even := smokeBit(sheet(0.7, -1, 1))
	mist := smokeBit(leaning{steady: sheet(0.7, -1, 1), asymmetry: 0.5})
	for name, toward := range sunWays {
		scene := sunlit(toward, maths.Grey(2))
		plain := scattered(scene, &even, midway, alongZ, false)
		got := scattered(scene, &mist, midway, alongZ, false)
		if !(plain.R > 0.1) {
			t.Fatalf("the even smoke is unlit with the sun %s", name)
		}
		lightIs(t, "mist with the sun "+name, got,
			plain.Scale(lobes[name]), 1e-12)
	}
}

// Anti: a lamp from everywhere has no way for a lobe to lean by, and a
// mist of no asymmetry has no lobe: both send what even smoke sends.
func TestAFloodOrAnEvenMistIsNotLobed(t *testing.T) {
	even := smokeBit(sheet(0.7, -1, 1))
	mist := smokeBit(leaning{steady: sheet(0.7, -1, 1), asymmetry: 0.5})
	level := smokeBit(leaning{steady: sheet(0.7, -1, 1)})
	flooded := thing.NewWorld()
	flooded.Lamp(film.Flood{Strength: maths.Grey(0.5)})
	scene := stage(flooded, 0)
	plain := scattered(scene, &even, midway, alongZ, false)
	if got := scattered(scene, &mist, midway, alongZ, false); got != plain ||
		!(plain.R > 0.01) {
		t.Errorf("a lopsided mist under a flood sends %v, even smoke %v",
			got, plain)
	}
	sun := sunlit(alongZ.Dir, maths.Grey(2))
	plain = scattered(sun, &even, midway, alongZ, false)
	if got := scattered(sun, &level, midway, alongZ, false); got != plain {
		t.Errorf("a mist of no asymmetry sends %v, even smoke %v", got,
			plain)
	}
}

// fixedGlowOf is a world whose glow is one radiance everywhere, of the
// one air given, with that air in it worn by smokeColour, and no lamp.
func fixedGlowOf(air medium.Medium, radiance maths.Light) *thing.World {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: air,
		Material: &thing.Material{Colour: smokeColour}})
	world.Glow = fixedGlow{light: radiance, air: air}
	return world
}

// A glow lights a bit of its own air by the air's colour times the
// glow's mean radiance there, channel by channel: 0.8, 0.4 and 0.2 of
// 0.5, 0.25 and 0.125 is 0.4, 0.1 and 0.025. And down a ray through a
// sheet of it 0.7 thick and two deep, where no lamp is and nothing
// shadows a glow, the ray gathers that times what the sheet takes,
// one less e to the minus 1.4.
func TestAGlowAddsTheAirsColourTimesItsMeanRadiance(t *testing.T) {
	air := sheet(0.7, -1, 1)
	glowing := fixedGlowOf(air, maths.RGB(0.5, 0.25, 0.125))
	scene := stage(glowing, 0)
	bit := crossings(scene.volumes, alongZ, math.Inf(1))[0]
	want := maths.RGB(0.4, 0.1, 0.025)
	lightIs(t, "a bit of glowing air",
		scattered(scene, &bit, midway, alongZ, false), want, 1e-12)
	took := 1 - math.Exp(-0.7*2)
	passed := through(scene, alongZ, math.Inf(1)).added
	lightIs(t, "a ray through glowing air", passed, want.Scale(took), 1e-12)
}

// Anti: a glow lights air that scatters nothing with nothing, and air it
// is not of with nothing.
func TestAGlowAddsNothingToBlackAirOrToAnotherAir(t *testing.T) {
	air := sheet(0.7, -1, 1)
	black := fixedGlowOf(air, maths.Grey(0.5))
	black.Volumes[0].Material = &thing.Material{}
	other := fixedGlowOf(air, maths.Grey(0.5))
	other.Glow = fixedGlow{light: maths.Grey(0.5), air: sheet(0.7, -2, 2)}
	for name, world := range map[string]*thing.World{
		"black air": black, "another air": other} {
		got := through(stage(world, 0), alongZ, math.Inf(1)).added
		if got != (maths.Light{}) {
			t.Errorf("a glow adds %v to %s, want nothing", got, name)
		}
	}
}

// selfShadowed is what a ray down alongZ gathers from a sheet of smoke
// 0.7 thick and two deep under a sun of two, over what one bit of that
// smoke sends with nothing in the sun's way.
func selfShadowed(toward maths.Vec) float64 {
	return selfShadowedIn(toward, strides)
}

// selfShadowedIn is selfShadowed with the ray's march cut into a number
// of steps across the sheet.
func selfShadowedIn(toward maths.Vec, cuts int) float64 {
	smoke := sheet(0.7, -1, 1)
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: smoke,
		Material: &thing.Material{Colour: smokeColour}})
	light := maths.Grey(2)
	world.Lamp(thing.Sun(toward, light))
	bit := smokeBit(smoke)
	sunlight := sunlit(toward, maths.Grey(2))
	one := scattered(sunlight, &bit, midway, alongZ, false)
	return throughGain(stage(world, 0), alongZ, math.Inf(1), nil,
		cuts).added.R / one.R
}

// stepError is how far a march of strides steps falls from the integral
// of an exponential through a sheet 1.4 deep in all, as a share, and
// the tolerance is twice it. Take d as a step's optical depth, 1.4 / 48.
//
// The shadow march toward the sun is exact here: the smoke's density is
// the same everywhere, and its samples stay inside the sheet. The ray's
// march takes each step's light at its middle, dimmed by the sun's way
// there, and weights it by what the eye keeps at the step's start times
// 1 - e^(-d).
//
// With the sun behind the eye that is e^(-d/2) at the middle and
// e^(-i d) at the start of step i. The sum over the 48 steps is
// e^(-d/2) (1 - e^(-2.8)) / (1 + e^(-d)), for the integral
// (1 - e^(-2.8)) / 2, so their ratio is 1 / cosh(d/2): short by d^2 / 8,
// about 1.06e-4, and the tolerance, 2.1e-4, is twice that. A pure
// middle rule on e^(-2t) would be out by d^2 / 6; this march is not
// that, since the keep and the share taken are the step's own.
//
// With the sun ahead the ratio is 2 sinh(d/2) / d, over by d^2 / 24,
// 3.5e-5, inside the same tolerance.
const stepError = 2 * (1.4 / strides) * (1.4 / strides) / 8

// Smoke between a bit and the sun dims the bit. With the sun behind the
// eye the light reaches depth t of the sheet through t of smoke, and
// comes back to the eye through t more, so the sheet sends one bit's
// light times the integral of e^(-2t) over its depth of 1.4:
// (1 - e^(-2.8)) / 2, 0.4696.
func TestSmokeLitFromBehindTheEyeIsDimmedByItsOwnDepthTwice(t *testing.T) {
	want := (1 - math.Exp(-2.8)) / 2
	if got := selfShadowed(alongZ.Dir.Neg()); !within(got, want,
		stepError*want) {
		t.Errorf("lit from behind the eye the sheet sends %v of a bit's "+
			"light, want %v", got, want)
	}
}

// Anti: the dimming is by the way to the sun, not by depth alone. With
// the sun ahead, light reaches depth t through the rest of the sheet,
// 1.4 - t, and the eye sees it through t: every bit is dimmed by
// e^(-1.4), and the sheet sends 1.4 e^(-1.4), 0.3452, where the depth
// alone would give the 0.4696 above.
func TestSmokeLitFromAheadIsDimmedByTheWholeSheetOnce(t *testing.T) {
	want := 1.4 * math.Exp(-1.4)
	got := selfShadowed(alongZ.Dir)
	if !within(got, want, stepError*want) {
		t.Errorf("lit from ahead the sheet sends %v of a bit's light, "+
			"want %v", got, want)
	}
	if behind := (1 - math.Exp(-2.8)) / 2; within(got, behind,
		stepError*behind) {
		t.Errorf("lit from ahead the sheet sends %v, which is what it "+
			"sends lit from behind", got)
	}
}

// coarseCuts is the steps of a march too coarse for the tolerance.
const coarseCuts = 4

// Anti: the tolerance is tight enough to tell a coarse march. Cut into
// coarseCuts steps, d is 1.4 / 4 = 0.35 and the march with the sun
// behind the eye falls short by d^2 / 8, about 0.015, seventy times
// stepError: the check that takes the real march turns this one away.
func TestACoarseMarchIsOutsideTheStepTolerance(t *testing.T) {
	want := (1 - math.Exp(-2.8)) / 2
	if got := selfShadowedIn(alongZ.Dir.Neg(), coarseCuts); within(got,
		want, stepError*want) {
		t.Errorf("a march of %d steps sends %v, within the tolerance "+
			"of %v", coarseCuts, got, want)
	}
}
