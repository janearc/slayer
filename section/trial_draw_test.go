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
	"github.com/janearc/slayer/trace"
)

const (
	// oneRay is one ray a pixel: the fastest draw, with nothing
	// averaged down.
	oneRay = 1
	// later is the moment of the second frame: a second after the first.
	later = time.Second
	// starRadius is how big a self-lit ball is, in scene units: a little
	// under the width of a section at its distance.
	starRadius = 1.2
	// starGlow is how bright a self-lit ball shows, in linear light:
	// bright enough to stand out from the empty sections.
	starGlow = 0.8
	// sunLight is the strength of the sun in the trials, as the existing
	// section tests have it.
	sunLight = 3.0
)

// trial is two worlds of a film, drawn through a lens one second apart.
type trial struct {
	name          string
	before, after *thing.World
	view          eye
	side          int
	was, now      time.Duration
}

// newTrial is two worlds seen through the pinhole of the section tests.
func newTrial(name string, before, after *thing.World) trial {
	view := eye{lens: film.Pinhole{}, wide: trialWide, high: trialHigh}
	return trial{name: name, before: before, after: after, view: view,
		side: trialSide, now: later}
}

// verdict is what a judged trial found: how many sections there are, how
// many were kept, how many differ in some bit between the two whole
// frames, and how many of the kept differ.
type verdict struct {
	total, kept, changed, keptChanged int
}

// glowing is a ball of a radius that shows only its own light, a grey of
// a strength, at a named place.
func glowing(t *testing.T, radius, strength float64, name string) *thing.Thing {
	t.Helper()
	made := thing.Diffuse(name, maths.Grey(nothing))
	made.Glow = maths.Grey(strength)
	return solid(radius, made, placed(t, name))
}

// worldOf is a world of things and volumes under lamps.
func worldOf(things []*thing.Thing, volumes []*thing.Volume,
	lamps ...thing.Lamp) *thing.World {
	world := thing.NewWorld()
	world.Things = append(world.Things, things...)
	world.Volumes = append(world.Volumes, volumes...)
	world.Lamps = append(world.Lamps, lamps...)
	return world
}

// sun is the trials' steady sun.
func sun(t *testing.T) thing.Lamp {
	t.Helper()
	return thing.Sun(place(t, "sun toward"), maths.Grey(sunLight))
}

// puffAt is a volume of smoke, a ball of a radius and thickness, at a
// named place, as lit by lamps as the section tests' puff is.
func puffAt(t *testing.T, radius, thick float64, name string) *thing.Volume {
	t.Helper()
	smoke := thing.Diffuse("smoke", maths.Grey(picked(t, "smoke-grey")))
	cloud := medium.Puff{Radius: radius, Thick: thick}
	made := thing.NewVolume(cloud, smoke)
	made.Motion = thing.Still(placed(t, name))
	return &made
}

// exactly is whether two frames are the same in every bit of every
// pixel of a box.
func exactly(first, second *frame.Frame, box image.Rectangle) bool {
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if first.At(x, y) != second.At(x, y) {
				return false
			}
		}
	}
	return true
}

// channels is a pixel's three colours.
func channels(pixel frame.Pixel) [3]float64 {
	return [3]float64{float64(pixel.R), float64(pixel.G), float64(pixel.B)}
}

// gapAndBright is the most any colour of any pixel of a box differs
// between two frames, and the brightest colour of the first frame there.
func gapAndBright(first, second *frame.Frame,
	box image.Rectangle) (gap, bright float64) {
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			was, now := channels(first.At(x, y)), channels(second.At(x, y))
			for at := range was {
				gap = math.Max(gap, math.Abs(was[at]-now[at]))
				bright = math.Max(bright, was[at])
			}
		}
	}
	return gap, bright
}

// tried fails the test unless the trial changed some section, so that
// something was tried, and kept at least a number of sections, so that
// the rule is not uselessly strict.
func tried(t *testing.T, found verdict, least int) {
	t.Helper()
	if found.changed == nothing {
		t.Errorf("no section changed: nothing was tried")
	}
	if found.kept < least {
		t.Errorf("%d sections kept, not %d of %d: the rule keeps too little",
			found.kept, least, found.total)
	}
}

// draw is a world as Sampled draws it at a moment, whole.
func (try trial) draw(t *testing.T, world *thing.World,
	at time.Duration) *frame.Frame {
	t.Helper()
	picture, err := trace.Sampled(world, maths.Local(), try.view.lens, at,
		try.view.wide, try.view.high, oneRay)
	if err != nil {
		t.Fatalf("%s: %v", try.name, err)
	}
	return picture
}

// sights are the sections of the trial's frame and the sight of each.
func (try trial) sights() ([]image.Rectangle, []section.Sight) {
	boxes := section.Split(try.view.wide, try.view.high, try.side)
	sights := make([]section.Sight, len(boxes))
	for at, box := range boxes {
		sights[at] = section.SightOf(try.view.look, box)
	}
	return boxes, sights
}

// judge is the verdict on two drawn frames under a drift: it fails the
// test for every kept section that differs in any bit with no drift, or
// by more than the drift's share of its brightest colour with one.
func (try trial) judge(t *testing.T, first, second *frame.Frame,
	drift float64) verdict {
	t.Helper()
	boxes, sights := try.sights()
	change := section.Between(try.before, try.after, try.was,
		try.now).Allowing(drift)
	found := verdict{total: len(boxes)}
	for at, box := range boxes {
		same := exactly(first, second, box)
		if !same {
			found.changed++
		}
		if !change.Keeps(sights[at]) {
			continue
		}
		found.kept++
		gap, bright := gapAndBright(first, second, box)
		if !same {
			found.keptChanged++
		}
		if !same && (!positive(drift) || !(gap <= drift*bright)) {
			t.Errorf("%s: section %v kept with drift %g and differs by "+
				"%g, its brightest %g", try.name, box, drift, gap, bright)
		}
	}
	return found
}

// strict is the verdict on the trial with no drift, the two frames drawn
// whole.
func (try trial) strict(t *testing.T) verdict {
	t.Helper()
	first := try.draw(t, try.before, try.was)
	second := try.draw(t, try.after, try.now)
	found := try.judge(t, first, second, nothing)
	t.Logf("%s: %d of %d sections kept, %d changed", try.name, found.kept,
		found.total, found.changed)
	return found
}
