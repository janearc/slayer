//go:build draw

package section_test

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// What the drift trials pick, each by the test author.
const (
	// driftSmall, driftMedium and driftLarge are the drifts tried: a
	// thousandth, a hundredth and a tenth.
	driftSmall  = 0.001
	driftMedium = 0.01
	driftLarge  = 0.1
	// The shares of the sun's light that a changed lamp brings the volume,
	// each just under or well over a drift: the engine's decision lies
	// between, so the bound is tried where it is most likely to break.
	shareA = 0.0005
	shareB = 0.0009
	shareC = 0.005
	shareD = 0.009
	shareE = 0.05
	shareF = 0.09
	shareG = 0.3
	// wholeCone is a cone of nearly a half turn, so a spot lights every
	// direction from where it is.
	wholeCone = 3.1
	// narrowCone is a cone of a hundredth of a radian: seventy centimetres
	// across at seventy units, a spot that lights a patch of the volume
	// and none of its corners.
	narrowCone = 0.01
	// bigPuffRadius and bigPuffThick are a volume much larger than the
	// frame, whose box holds the eye: forty units, thin enough to see
	// through.
	bigPuffRadius = 40.0
	bigPuffThick  = 0.02
	// lampsChanged is how many lamps change at once in the scene of
	// several changed lamps.
	lampsChanged = 3
)

// sweepShares are the shares of the sun's light tried.
func sweepShares() []float64 {
	return []float64{shareA, shareB, shareC, shareD, shareE, shareF, shareG}
}

// sweepDrifts are the drifts tried.
func sweepDrifts() []float64 {
	return []float64{driftSmall, driftMedium, driftLarge}
}

// builder is a scene of a volume lit by a steady sun and some lamps that
// change, by the share of the sun's light they bring the volume.
type builder func(t *testing.T, share float64) (before, after *thing.World)

// brings is a spot at a place that lights the volume at a place by a
// share of the sun's light: its strength is the share of the sun, times
// the square of the distance, for light falls off as the square.
func brings(t *testing.T, at, toward maths.Vec, share, cone float64,
	centre maths.Vec) thing.Lamp {
	t.Helper()
	far := centre.Sub(at).Len()
	return film.Spot{At: at, Aim: toward, Strength: maths.Grey(
		share * sunLight * far * far), Cone: cone}
}

// volumeBase is the self-lit ball and the volume of a name every drift
// scene has, and the world of them under lamps.
func volumeBase(t *testing.T, star *thing.Thing, puff *thing.Volume,
	lamps ...thing.Lamp) *thing.World {
	t.Helper()
	return worldOf([]*thing.Thing{star}, []*thing.Volume{puff}, lamps...)
}

// sweep draws a scene for each share, and judges the pair of frames under
// each drift: it fails for every kept section further from the whole
// frame than the drift's share of its brightest colour, in any colour.
func sweep(t *testing.T, name string, build builder) {
	t.Helper()
	kept, moved := nothing, nothing
	for _, share := range sweepShares() {
		before, after := build(t, share)
		try := newTrial(name, before, after)
		first := try.draw(t, before, try.was)
		second := try.draw(t, after, try.now)
		for _, drift := range sweepDrifts() {
			found := try.judge(t, first, second, drift)
			kept, moved = kept+found.kept, moved+found.keptChanged
			t.Logf("%s: share %g drift %g: %d kept, %d of them differ, %d "+
				"differ at all", name, share, drift, found.kept,
				found.keptChanged, found.changed)
		}
	}
	if kept == nothing || moved == nothing {
		t.Logf("%s: kept %d, of which %d differ: nothing of the bound "+
			"was tried", name, kept, moved)
	}
}

// severalChanged is a puff under the sun and three far lamps, all of
// which change.
func severalChanged(t *testing.T, share float64) (*thing.World,
	*thing.World) {
	t.Helper()
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	centre := place(t, "puff centre")
	var steady []thing.Lamp
	steady = append(steady, sun(t))
	var dim, bright []thing.Lamp
	for _, name := range []string{"lamp a", "lamp b", "lamp c"} {
		at := place(t, name)
		dim = append(dim, brings(t, at, centre.Sub(at), share/lampsChanged,
			wholeCone, centre))
		bright = append(bright, brings(t, at, centre.Sub(at), share,
			wholeCone, centre))
	}
	return volumeBase(t, star, puff, append(steady, dim...)...),
		volumeBase(t, star, puff, append(steady, bright...)...)
}

// oneLampAt is a puff under the sun and one lamp at a named place that
// changes: dim before, by the share after.
func oneLampAt(t *testing.T, name string, star *thing.Thing,
	puff *thing.Volume, share float64) (*thing.World, *thing.World) {
	t.Helper()
	centre, at := place(t, "puff centre"), place(t, name)
	aim := centre.Sub(at)
	dim := brings(t, at, aim, nothing, wholeCone, centre)
	bright := brings(t, at, aim, share, wholeCone, centre)
	return volumeBase(t, star, puff, sun(t), dim), volumeBase(t, star, puff, sun(t),
		bright)
}

// insideTheBox is a lamp inside the volume's bounding box that changes.
func insideTheBox(t *testing.T, share float64) (*thing.World, *thing.World) {
	t.Helper()
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	return oneLampAt(t, "lamp inside", star, puff, share)
}

// nearACorner is a lamp very near one corner of the box that changes.
func nearACorner(t *testing.T, share float64) (*thing.World, *thing.World) {
	t.Helper()
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	return oneLampAt(t, "lamp corner", star, puff, share)
}

// aHugeVolume is a volume much larger than the frame, with a lamp far off
// that changes.
func aHugeVolume(t *testing.T, share float64) (*thing.World, *thing.World) {
	t.Helper()
	puff := puffAt(t, bigPuffRadius, bigPuffThick, "big puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	centre, at := place(t, "big puff centre"), place(t, "lamp huge")
	aim := centre.Sub(at)
	dim := brings(t, at, aim, nothing, wholeCone, centre)
	bright := brings(t, at, aim, share, wholeCone, centre)
	return volumeBase(t, star, puff, sun(t), dim), volumeBase(t, star, puff, sun(t),
		bright)
}

// aDirectionalLamp is a puff under the sun and a second sun, from
// another way, that changes.
func aDirectionalLamp(t *testing.T, share float64) (*thing.World,
	*thing.World) {
	t.Helper()
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	other := place(t, "sun other")
	dim := thing.Sun(other, maths.Grey(nothing))
	bright := thing.Sun(other, maths.Grey(share*sunLight))
	return volumeBase(t, star, puff, sun(t), dim), volumeBase(t, star, puff, sun(t),
		bright)
}

// aNarrowBeam is a lamp whose cone lights a patch of the volume and none
// of the points the drift looks at: its corners, its middle.
func aNarrowBeam(t *testing.T, share float64) (*thing.World, *thing.World) {
	t.Helper()
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	centre, at := place(t, "puff centre"), place(t, "lamp huge")
	aim := place(t, "narrow aim point").Sub(at)
	dim := brings(t, at, aim, nothing, narrowCone, centre)
	bright := brings(t, at, aim, share, narrowCone, centre)
	return volumeBase(t, star, puff, sun(t), dim), volumeBase(t, star, puff, sun(t),
		bright)
}

// aShadowingMover is a puff under the sun and a far lamp that changes,
// and in the second frame a ball in the way of the sun, standing above
// the frame, where no section's rays reach it.
func aShadowingMover(t *testing.T, share float64) (*thing.World,
	*thing.World) {
	t.Helper()
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	star := glowing(t, starRadius, starGlow, "star left")
	before, after := oneLampAt(t, "lamp a", star, puff, share)
	blocker := clay(t, casterRadius*two, "blocker")
	after.Things = append(after.Things[:len(after.Things):len(after.Things)],
		blocker)
	return before, after
}

// the same for a ball that moves, out of every section's sight, into the
// way of the sun: the volume is shadowed, so no section it shows in is
// kept, and none kept differs.
func TestDriftHoldsWhenAMoverShadowsTheVolume(t *testing.T) {
	sweep(t, "a shadowing mover", aShadowingMover)
}

// with a drift, no section kept is further from the whole frame than the
// drift's share of its brightest colour: several lamps changing at once.
func TestDriftHoldsWithSeveralChangedLamps(t *testing.T) {
	sweep(t, "several lamps", severalChanged)
}

// the same for a changed lamp inside the volume's bounding box.
func TestDriftHoldsWithALampInsideTheBox(t *testing.T) {
	sweep(t, "a lamp inside the box", insideTheBox)
}

// the same for a changed lamp very near one corner of the box.
func TestDriftHoldsWithALampNearACorner(t *testing.T) {
	sweep(t, "a lamp near a corner", nearACorner)
}

// the same for a volume much larger than the frame.
func TestDriftHoldsWithAVolumeLargerThanTheFrame(t *testing.T) {
	sweep(t, "a huge volume", aHugeVolume)
}

// the same for a directional lamp that changes.
func TestDriftHoldsWithAChangedDirectionalLamp(t *testing.T) {
	sweep(t, "a directional lamp", aDirectionalLamp)
}

// the same for a lamp whose cone is narrow and lights part of the volume.
func TestDriftHoldsWithANarrowBeam(t *testing.T) {
	sweep(t, "a narrow beam", aNarrowBeam)
}
