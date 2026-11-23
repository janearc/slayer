package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// sooty is a steady medium thicker to some light than to other: each
// channel meets its tint's times the density.
type sooty struct {
	steady
	tint maths.Light
}

// Tint is how many times the density each channel meets.
func (soot sooty) Tint() maths.Light { return soot.tint }

// sootTint is the tint of these tests: half as thick to red, as thick to
// green, and twice as thick to blue.
var sootTint = maths.RGB(0.5, 1, 2)

// A tinted volume keeps of each channel e to the minus its own depth:
// the tint's times the density times the way through. Soot 0.7 thick and
// two deep, tinted a half, one and two, is 0.7, 1.4 and 2.8 deep to red,
// green and blue. Its glow of 0.6 comes through by what each channel
// took, one less what it kept. And one step 1.5 thick takes one less
// e^(-0.75), e^(-1.5) and e^(-3).
func TestATintedVolumeKeepsEachChannelByItsOwnDepth(t *testing.T) {
	soot := thing.Volume{Medium: sooty{sheet(0.7, -1, 1), sootTint},
		Material: &thing.Material{Glow: maths.Grey(0.6)}}
	scene := staging(soot)
	got := through(scene, alongZ, math.Inf(1))
	kept := maths.RGB(math.Exp(-0.7), math.Exp(-1.4), math.Exp(-2.8))
	lightIs(t, "soot keeps", got.keep, kept, 1e-12)
	lightIs(t, "soot adds", got.added, maths.RGB(0.6*(1-kept.R),
		0.6*(1-kept.G), 0.6*(1-kept.B)), 1e-12)
	took := maths.RGB(1-math.Exp(-0.75), 1-math.Exp(-1.5), 1-math.Exp(-3))
	lightIs(t, "a step of soot takes", absorbed(&scene.volumes[0], 1.5),
		took, 1e-15)
}

// Anti: a tint of one in every channel is no tint, and a medium that
// says none has none: each keeps e^(-1.4) of every channel alike.
func TestATintOfOneKeepsEveryChannelAlike(t *testing.T) {
	want := maths.Grey(math.Exp(-1.4))
	for name, made := range map[string]thing.Volume{
		"a tint of one": {Medium: sooty{sheet(0.7, -1, 1), maths.Grey(1)}},
		"no tint":       {Medium: sheet(0.7, -1, 1)},
	} {
		got := through(staging(made), alongZ, math.Inf(1)).keep
		lightIs(t, name, got, want, 1e-12)
	}
}

// overlapped is a plain sheet 0.3 thick from minus two to one, and soot
// 0.9 thick from minus one to three, staged together.
func overlapped() []placedVolume {
	return staging(thing.Volume{Medium: sheet(0.3, -2, 1)},
		thing.Volume{Medium: sooty{sheet(0.9, -1, 3), sootTint}}).volumes
}

// Toward a lamp, a tinted volume's depth is its tint's times in each
// channel, on top of what the plain volumes in the way take of all
// three. The plain sheet is 0.3 times 3 deep, 0.9; the soot 0.9 times 4,
// 3.6, which its tint makes 1.8, 3.6 and 7.2: so the lamp comes through
// by e^(-2.7), e^(-4.5) and e^(-8.1).
func TestTowardALampATintedVolumeDimsEachChannelByItsOwnDepth(t *testing.T) {
	got := paintedClear(overlapped(), alongZ.At, alongZ.Dir, 100)
	want := maths.RGB(math.Exp(-2.7), math.Exp(-4.5), math.Exp(-8.1))
	lightIs(t, "through a sheet and soot", got, want, 1e-12)
}

// Anti: soot beyond the lamp tints nothing. A lamp 48.5 along the way
// is half a unit into the plain sheet and short of the soot, so every
// channel keeps e^(-0.15) alike.
func TestSootBeyondTheLampTintsNothing(t *testing.T) {
	got := paintedClear(overlapped(), alongZ.At, alongZ.Dir, 48.5)
	dimmed := maths.Grey(math.Exp(-0.15))
	lightIs(t, "short of the soot", got, dimmed, 1e-12)
}
