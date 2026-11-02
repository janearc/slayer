package lines_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/lines"
	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// elements is every gas lines has a spectrum for, by the name Of takes.
var elements = []string{"hydrogen", "neon", "argon"}

// The numbers these tests count in, each defined.
const (
	nothing = 0.0 // defined by the scale: no light
	whole   = 1.0 // defined by the scale: the luminance of one
)

// stopper is a test that can stop: a testing.T, or rapid's.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// glowOf is a gas's glow at a luminance, or the test stops.
func glowOf(test stopper, name string, luminance float64) maths.Light {
	test.Helper()
	gas, err := lines.Of(name)
	if err != nil {
		test.Fatal(err)
	}
	return gas.Glow(luminance)
}

// near is whether a value is within a share of what is wanted.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// Every gas glows at the luminance it is asked for.
func TestEveryGasGlowsAtTheLuminanceAskedFor(test *testing.T) {
	for _, name := range elements {
		got := glowOf(test, name, whole).Luminance()
		if !near(got, whole, picked("luminance-tolerance")) {
			test.Errorf("%s glows at %v, not %v", name, got, whole)
		}
	}
}

// The anti-test: a gas lines has no spectrum for is refused, its name
// in the refusal, and what comes back beside it glows with no light.
// Names are as Of lists them, so a capital is another name.
func TestAGasWithNoLinesIsRefused(test *testing.T) {
	for _, name := range []string{"helium", "", "Hydrogen"} {
		gas, err := lines.Of(name)
		if err == nil {
			test.Errorf("%q was not refused", name)
			continue
		}
		if glow := gas.Glow(whole); glow != (maths.Light{}) {
			test.Errorf("%q, refused, glows %v", name, glow)
		}
	}
}

// The property: at any luminance a gas glows at that luminance, in the
// same colour: its light over its luminance is its light at one.
func TestAGasKeepsItsColourAtAnyLuminance(test *testing.T) {
	rapid.Check(test, keepsItsColour)
}

// keepsItsColour is one draw of the property above.
func keepsItsColour(test *rapid.T) {
	name := rapid.SampledFrom(elements).Draw(test, "gas")
	luminance := rapid.Float64Range(picked("luminance-least"),
		picked("luminance-most")).Draw(test, "luminance")
	share := picked("luminance-tolerance")
	glow := glowOf(test, name, luminance)
	if !near(glow.Luminance(), luminance, share) {
		test.Fatalf("%s at %v glows at %v", name, luminance,
			glow.Luminance())
	}
	unit := glowOf(test, name, whole)
	scaled := glow.Scale(whole / luminance)
	for _, channel := range []struct{ got, want float64 }{
		{scaled.R, unit.R}, {scaled.G, unit.G}, {scaled.B, unit.B}} {
		if !near(channel.got, channel.want, share) {
			test.Fatalf("%s at %v is %v, at one %v", name, luminance,
				glow, unit)
		}
	}
}

// The anti-test: a glow of no luminance is no light, from any gas.
func TestAGlowOfNoLuminanceIsNoLight(test *testing.T) {
	for _, name := range elements {
		if glow := glowOf(test, name, nothing); glow != (maths.Light{}) {
			test.Errorf("%s at no luminance glows %v", name, glow)
		}
	}
}

// The colour comes from the gas. Hydrogen's balmer lines are led by
// h-alpha at 656 nm, so it glows red before green or blue; neon's lines
// are orange and red, so it glows red before blue.
func TestHydrogenAndNeonGlowRed(test *testing.T) {
	hydrogen := glowOf(test, "hydrogen", whole)
	if !(hydrogen.R > hydrogen.G && hydrogen.R > hydrogen.B) {
		test.Errorf("hydrogen glows %v, not red", hydrogen)
	}
	neon := glowOf(test, "neon", whole)
	if !(neon.R > neon.B) {
		test.Errorf("neon glows %v, not red before blue", neon)
	}
}

// The anti-test: a gas does not glow another's colour. Ionized argon's
// lines are blue and green, as an argon laser's, so it glows blue and
// green each before red, where hydrogen does not.
func TestArgonDoesNotGlowRed(test *testing.T) {
	argon := glowOf(test, "argon", whole)
	if !(argon.B > argon.R && argon.G > argon.R) {
		test.Errorf("argon glows %v, red before blue or green", argon)
	}
}

// A glow gives light and takes none away: no gas glows with a channel
// below nothing, though a line the three lamps cannot reach would ask
// for one.
func TestNoGlowHasAChannelBelowNothing(test *testing.T) {
	for _, name := range elements {
		glow := glowOf(test, name, whole)
		if glow.R < nothing || glow.G < nothing || glow.B < nothing {
			test.Errorf("%s glows %v", name, glow)
		}
	}
}
