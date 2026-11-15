//go:build draw

package section_test

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/section"
)

// a ringed is the giant with a puff of smoke before the eye, lit by a
// sun that does not change and by a small lamp far behind the eye that
// does: a fire a long way off.
type ringed struct {
	parts
	puff *thing.Volume
}

// newRinged is the giant, the clay ball and the puff.
func newRinged(t *testing.T) ringed {
	t.Helper()
	grey := maths.Grey(picked(t, "smoke-grey"))
	smoke := thing.Diffuse("smoke", grey)
	cloud := medium.Puff{Radius: picked(t, "puff-radius"),
		Thick: picked(t, "puff-thick")}
	puff := thing.NewVolume(cloud, smoke)
	puff.Motion = thing.Still(placed(t, "puff"))
	return ringed{parts: newParts(t), puff: &puff}
}

// staged is the world under the sun and the far lamp at a strength.
func (made ringed) staged(t *testing.T, far float64) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.Things = append(world.Things, made.giant, made.ball)
	world.Volumes = append(world.Volumes, made.puff)
	strength := maths.Grey(picked(t, "sun-strength"))
	toward := place(t, "sun toward")
	world.Lamp(thing.Sun(toward, strength))
	world.Lamp(film.Spot{At: place(t, "far lamp at"),
		Aim: place(t, "far lamp aim"), Strength: maths.Grey(far),
		Cone: picked(t, "far-cone")})
	return world
}

// apart is the largest difference between two frames in a box, and the
// brightest the first is there.
func apart(first, second *frame.Frame, box image.Rectangle) (most,
	bright float64) {
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			was, now := first.At(x, y), second.At(x, y)
			gap := math.Abs(float64(was.G - now.G))
			most = math.Max(most, gap)
			bright = math.Max(bright, float64(was.G))
		}
	}
	return most, bright
}

// kept is how many sections the loose change keeps and the strict one
// does not, and how many of those kept differ at all from the whole
// frame; it stops the test if a section is kept with no drift and
// changed, or kept with a drift and further from the whole frame than
// the drift's share of its brightest pixel.
func kept(t *testing.T, first, second *frame.Frame, boxes []image.Rectangle,
	sights []section.Sight, strict section.Change) (gained, moved int) {
	t.Helper()
	drift := picked(t, "drift")
	loose := strict.Allowing(drift)
	for at, box := range boxes {
		most, bright := apart(first, second, box)
		if strict.Keeps(sights[at]) && most != nothing {
			t.Fatalf("section %v was kept with no drift and changed", box)
		}
		if !loose.Keeps(sights[at]) {
			continue
		}
		if !(most <= drift*bright) {
			t.Fatalf("section %v was kept and is %g from the whole frame, "+
				"its brightest %g", box, most, bright)
		}
		if positive(most) {
			moved++
		}
		if !strict.Keeps(sights[at]) {
			gained++
		}
	}
	return gained, moved
}

// when a far lamp changes by a hundredth of the sun's light on a puff,
// no section the puff is in is kept with no drift allowed; with a drift
// of a twentieth they are kept, and no kept section is further from the
// frame drawn whole than a twentieth of its brightest pixel; and a
// section that did change by more, the clay ball's, is still drawn.
func TestADriftKeepsAVolumeAFarLampBarelyLights(t *testing.T) {
	made := newRinged(t)
	before := made.staged(t, picked(t, "far-dim"))
	after := made.staged(t, picked(t, "far-brighter"))
	first, boxes, sights := drawn(t, before)
	second, _, _ := drawn(t, after)
	strict := section.Between(before, after, nothing, time.Second)
	gained, moved := kept(t, first, second, boxes, sights, strict)
	if gained < counted(t, "gained-least") || moved < counted(t, "moved-least") {
		t.Fatalf("the drift kept %d more sections, %d of them changed at "+
			"all: nothing was tried", gained, moved)
	}
	clay := rect(t, row(t, windowsFile, "clay ball"))
	loose := strict.Allowing(picked(t, "drift"))
	index := counted(t, "clay-section-index")
	if most, _ := apart(first, second, clay); most == nothing ||
		loose.Keeps(sights[index]) {
		t.Fatal("the clay ball's section did not change or was kept")
	}
}

// puffsKept is how many of the sections that a change keeps have the
// puff in sight.
func puffsKept(t *testing.T, change section.Change, sights []section.Sight) (
	kept int) {
	t.Helper()
	puff := place(t, "puff")
	for _, sight := range sights {
		if change.Keeps(sight) && sight.Meets(puff, picked(t, "puff-radius")) {
			kept++
		}
	}
	return kept
}

// a far lamp that grows as bright on the puff as the sun is past any
// small drift, and something that moves between the puff and the sun
// could shadow it: neither keeps the puff's sections.
func TestADriftDoesNotKeepWhatChangedMuchOrMightBeShadowed(t *testing.T) {
	made := newRinged(t)
	drift := picked(t, "drift")
	before := made.staged(t, picked(t, "far-dim"))
	_, _, sights := drawn(t, before)
	bright := made.staged(t, picked(t, "far-bright"))
	change := section.Between(before, bright, nothing, time.Second)
	if kept := puffsKept(t, change.Allowing(drift), sights); kept != nothing {
		t.Errorf("%d of the puff's sections kept under a lamp as bright "+
			"as the sun", kept)
	}
	shaded := newRinged(t)
	shaded.giant, shaded.puff = made.giant, made.puff
	moved := *made.ball
	moved.Motion = thing.Still(placed(t, "ball shaded"))
	shaded.ball = &moved
	world := shaded.staged(t, picked(t, "far-dim"))
	change = section.Between(before, world, nothing, time.Second)
	if kept := puffsKept(t, change.Allowing(drift), sights); kept != nothing {
		t.Errorf("%d of the puff's sections kept with a ball moved "+
			"between the puff and the sun", kept)
	}
	same := made.staged(t, picked(t, "far-brighter"))
	change = section.Between(before, same, nothing, time.Second)
	if kept := puffsKept(t, change.Allowing(drift), sights); kept == nothing {
		t.Error("the puff's sections are never kept: nothing was tried")
	}
}
