package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// thinGas is a gas of Rayleigh's even lobe that, a metre, absorbs a
// millionth and scatters four millionths of every colour the eye sees:
// five millionths a metre of extinction, alike in the three channels.
func thinGas() gas.Gas {
	return gas.Gas{Name: "thin", Visible: func(float64) (float64, float64) {
		return 1e-6, 4e-6
	}}
}

// thinThick is the thickness of these tests' one step: a hundred metres
// at the gas's full density, an optical depth of 5e-4.
const thinThick = 100

// thinDepth is that optical depth.
const thinDepth = 5e-6 * thinThick

// thinStep is one step of alongZ through thinGas under a sun of two
// behind the eye, at a gain, begun from a passage. The sun is behind the
// eye and the step's point in front of the gas, so nothing dims the sun
// on its way.
func thinStep(gain float64, from passage) passage {
	return thinStepUnder(alongZ.Dir.Neg(), gain, from)
}

// thinStepUnder is thinStep with the sun in the way given, toward the
// sun from the step. The step's point is in front of the gas, so a sun
// overhead is not dimmed either.
func thinStepUnder(toward maths.Vec, gain float64, from passage) passage {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: gas.Of{
		Shape: plate(1, -1, 1), In: thinGas(), Unit: 1}})
	light := maths.Grey(2)
	world.Lamp(thing.Sun(toward, light))
	scene := stage(world, 0)
	inside := crossings(scene.volumes, alongZ, math.Inf(1))[0]
	return gasStep(scene, &inside, stride{thick: thinThick}, alongZ, gain,
		from)
}

// A gas step gathers the sun's light times the gas's phase at the angle
// the light turns through, times what the gas scatters over the step's
// thickness, times the field's gain. Light from behind the eye turns
// right round, a cosine of minus one, where Rayleigh's lobe
// 3 / (16 pi) (1 + c^2) is 3 / (8 pi). A sun of two through a hundred
// metres scattering four millionths a metre, at a gain of three, is
// 2 x 3/(8 pi) x 4e-4 x 3 in every channel.
//
// That is the step's light to first order in its optical depth. Whether
// a step is taken as thin or its own dimming is worked through it, the
// two differ by under half the depth, 2.5e-4 of the light, and the
// tolerance is that.
func TestAGasStepGathersTheSunByPhaseScatterThicknessAndGain(t *testing.T) {
	want := 2 * 3 / (8 * math.Pi) * 4e-6 * thinThick * 3
	got := thinStep(3, untouched()).added
	lightIs(t, "a gas step at a gain of three", got, maths.Grey(want),
		thinDepth/2*want)
}

// A sun overhead turns the light square across the eye, a cosine of
// nought, where Rayleigh's lobe 3 / (16 pi) (1 + c^2) is 3 / (16 pi),
// half what it is behind the eye. So the same step under a sun of two
// gathers 2 x 3/(16 pi) x 4e-4 x 3 in every channel, about 1.43e-4,
// at the same tolerance, 2.5e-4 of it.
func TestAGasStepGathersAnOverheadSunAtTheLobeSquareAcross(t *testing.T) {
	want := 2 * 3 / (16 * math.Pi) * 4e-6 * thinThick * 3
	got := thinStepUnder(maths.Up, 3, untouched()).added
	lightIs(t, "a gas step under an overhead sun", got, maths.Grey(want),
		thinDepth/2*want)
}

// Anti: a step that took every sun at the cosine of minus one would
// gather the behind-the-eye value, twice the overhead one, which the
// overhead check turns away.
func TestAGasStepIgnoringTheAngleIsTurnedAwayByTheOverheadCheck(
	t *testing.T) {
	overhead := 2 * 3 / (16 * math.Pi) * 4e-6 * thinThick * 3
	ignoring := thinStep(3, untouched()).added
	if within(ignoring.R, overhead, thinDepth/2*overhead) {
		t.Errorf("the behind-the-eye light %v passes for the overhead "+
			"value %v", ignoring, overhead)
	}
}

// The gain scales what a step gathers and nothing else: at a gain of
// three it gathers three times what it gathers at one, and keeps the
// same e^(-5e-4) of what is behind. And what it gathers shows through
// what the ray has already lost: begun from a passage that keeps a
// half, a quarter and an eighth, it adds by those shares.
func TestAGasStepsGainScalesItsLightAndNotItsKeep(t *testing.T) {
	one, three := thinStep(1, untouched()), thinStep(3, untouched())
	lightIs(t, "the light at a gain of three", three.added,
		one.added.Scale(3), 1e-12*one.added.R)
	kept := maths.Grey(math.Exp(-thinDepth))
	lightIs(t, "the keep at a gain of one", one.keep, kept, 1e-12)
	lightIs(t, "the keep at a gain of three", three.keep, kept, 1e-12)
	dimmed := passage{keep: maths.RGB(0.5, 0.25, 0.125),
		added: maths.Grey(0.75)}
	got := thinStep(1, dimmed)
	wantAdded := maths.Grey(0.75).Add(one.added.Times(dimmed.keep))
	lightIs(t, "the light begun dimmed", got.added, wantAdded,
		1e-12*one.added.R)
	lightIs(t, "the keep begun dimmed", got.keep, dimmed.keep.Times(kept),
		1e-12)
}

// Anti: a step of no gain gathers nothing and still dims what is
// behind it; the gain is not a switch on the gas.
func TestAGasStepOfNoGainGathersNothingAndStillDims(t *testing.T) {
	got := thinStep(0, untouched())
	if got.added != (maths.Light{}) {
		t.Errorf("at no gain the step gathers %v", got.added)
	}
	if !(got.keep.R < 1) || !within(got.keep.R, math.Exp(-thinDepth), 1e-12) {
		t.Errorf("at no gain the step keeps %v", got.keep)
	}
}
