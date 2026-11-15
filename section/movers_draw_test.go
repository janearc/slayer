//go:build draw

package section_test

import (
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// What the mover trials pick, each by the test author.
const (
	// orbitRadius is how far a mover circles its centre: far enough that
	// a quarter turn takes it across several sections.
	orbitRadius = 3.0
	// orbitPeriod is a mover's turn: four seconds, so that the second
	// frame, a second on, finds it a quarter turn round.
	orbitPeriod = 4 * time.Second
	// moverRadius is the size of the ball or volume that moves.
	moverRadius = 0.8
	// fizzThick is the density at the middle of the foot of a fizz.
	fizzThick = 2.0
	// fizzHeight is how high the fizz column rises, in scene units.
	fizzHeight = 3.0
	// fizzRise is how fast the fizz's bubbles climb, units a second.
	fizzRise = 1.5
	// fizzChurn is how much of the density the churn may take away.
	fizzChurn = 0.8
	// fizzSeed picks the fizz's churn.
	fizzSeed = 7
	// discInner and discOuter are the radii of a thin disc painted on a
	// ball, in the ball's own units: a ring round its axis.
	discInner = 0.3
	discOuter = 2.0
	// discHot is the disc's hottest temperature, kelvin: well above the
	// 500 below which the disc is dark.
	discHot = 20000
	// discBright is the light of the disc where it is hottest.
	discBright = 2.0
	// discStreaks is how far the disc's streaks show: all of them.
	discStreaks = 1.0
	// discOmega is how fast the disc turns at radius one, radians a
	// second, enough to move its streaks in a second.
	discOmega = 3.0
	// discSeed picks the disc's streaks.
	discSeed = 11
	// keptAmongStars is the fewest sections kept when one mover shows in
	// a few sections of a frame of twenty-four: a third, picked after
	// counting the sections the movers can show in at the most.
	keptAmongStars = 8
)

// circling is a motion round the orbit centre, a quarter turn a second.
func circling(t *testing.T, phase float64) thing.Motion {
	t.Helper()
	return thing.Orbit{Centre: place(t, "orbit centre"), Radius: orbitRadius,
		Period: orbitPeriod, Phase: phase}
}

// stars are the two self-lit balls every mover trial has, which no
// mover touches.
func stars(t *testing.T) []*thing.Thing {
	t.Helper()
	return []*thing.Thing{glowing(t, starRadius, starGlow, "star left"),
		glowing(t, starRadius, starGlow, "star right")}
}

// a volume that moves keeps no section it can show in, and the picture
// is the same in every bit where one is kept.
func TestAMovingVolumeIsAMover(t *testing.T) {
	puff := puffAt(t, moverRadius, fizzThick, "puff centre")
	puff.Motion = circling(t, nothing)
	world := worldOf(stars(t), []*thing.Volume{puff}, sun(t))
	found := newTrial("a moving volume", world, world).strict(t)
	tried(t, found, keptAmongStars)
}

// a medium that is Timed changes with the moment though it stands still,
// and the sections it shows in are drawn again.
func TestATimedMediumIsAMover(t *testing.T) {
	cloud := medium.Fizz{Radius: moverRadius, Height: fizzHeight,
		Thick: fizzThick, Rise: fizzRise, Churn: fizzChurn, Seed: fizzSeed}
	foam := thing.Diffuse("foam", maths.Grey(picked(t, "smoke-grey")))
	made := thing.NewVolume(cloud, foam)
	made.Motion = thing.Still(placed(t, "puff right"))
	world := worldOf(stars(t), []*thing.Volume{&made}, sun(t))
	found := newTrial("a timed medium", world, world).strict(t)
	tried(t, found, keptAmongStars)
}

// a self-lit body whose texture is Timed changes with the moment though
// it stands still: it is a mover, not a steady self-lit body.
func TestATimedTextureOnASelfLitBodyIsAMover(t *testing.T) {
	disc := texture.Thin{Inner: discInner, Outer: discOuter, Hot: discHot,
		Bright: discBright, Streaks: discStreaks, Seed: discSeed,
		Omega: discOmega}
	made := thing.Diffuse("disc", maths.Grey(nothing))
	made.Shine = disc
	body := solid(starRadius*two, made, placed(t, "star right"))
	world := worldOf([]*thing.Thing{
		glowing(t, starRadius, starGlow, "star left"), body}, nil)
	found := newTrial("a timed texture", world, world).strict(t)
	tried(t, found, keptAmongStars)
}

// a thing in one frame only, a spark, is a mover where it appears and
// where it vanishes: either way no kept section differs.
func TestAThingInOneFrameOnlyIsAMover(t *testing.T) {
	base := stars(t)
	with := append(slices.Clone(base), glowing(t, moverRadius, starGlow, "spark"))
	has, lacks := worldOf(with, nil), worldOf(base, nil)
	appears := newTrial("a spark appears", lacks, has).strict(t)
	tried(t, appears, keptAmongStars)
	vanishes := newTrial("a spark vanishes", has, lacks).strict(t)
	tried(t, vanishes, keptAmongStars)
}

// a hidden thing is not drawn: one that moves, or comes to be shown, is
// still no section's picture changing in any bit where it is kept.
func TestAHiddenThingChangesNothingKept(t *testing.T) {
	hidden := glowing(t, moverRadius, starGlow, "spark")
	hidden.Attrs.Hidden = true
	hidden.Motion = circling(t, nothing)
	base := stars(t)
	before := worldOf(append(slices.Clone(base), hidden), nil)
	shown := *hidden
	shown.Attrs.Hidden = false
	after := worldOf(append(slices.Clone(base), &shown), nil)
	moved := newTrial("a hidden thing moves", before, before).strict(t)
	if moved.kept < keptAmongStars {
		t.Errorf("%d sections kept, not %d", moved.kept, keptAmongStars)
	}
	found := newTrial("a hidden thing is shown", before, after)
	if shown := found.strict(t); shown.changed == nothing {
		t.Errorf("a thing shown changed no section: nothing was tried")
	}
}

// a mover that does not touch the stars leaves their sections kept: the
// strict rule is not strict past what it must be. The sections the stars
// are in are all kept.
func TestAMoverLeavesTheSectionsItCannotShowInKept(t *testing.T) {
	ball := solid(moverRadius, thing.Diffuse("ball", maths.Grey(sunLight)),
		placed(t, "spark"))
	ball.Motion = circling(t, nothing)
	things := append(stars(t), ball)
	world := worldOf(things, nil, sun(t))
	found := newTrial("a ball circles two stars", world, world).strict(t)
	tried(t, found, keptAmongStars)
}
