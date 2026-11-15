package section_test

import (
	"image"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/section"
)

// wanted is the sections of splits.csv, in order, each worked by hand.
func wanted(t *testing.T) []image.Rectangle {
	t.Helper()
	var want []image.Rectangle
	for _, each := range rows(t, splitsFile) {
		want = append(want, rect(t, each))
	}
	return want
}

// a frame splits into sections that cover every pixel once: the uneven
// frame of numbers.yaml is six sections, the right-hand ones narrower and
// the lower ones shorter; and a side of zero is a side of one.
func TestAFrameSplitsIntoSections(t *testing.T) {
	wide, high := counted(t, "split-wide"), counted(t, "split-high")
	boxes := section.Split(wide, high, counted(t, "split-side"))
	want := wanted(t)
	if len(boxes) != len(want) {
		t.Fatalf("%d sections, not %d: %v", len(boxes), len(want), boxes)
	}
	for at, box := range boxes {
		if box != want[at] {
			t.Errorf("section %d is %v, not %v", at, box, want[at])
		}
	}
	tiny := section.Split(counted(t, "tiny-wide"), counted(t, "tiny-high"),
		nothing)
	if got := len(tiny); got != counted(t, "tiny-sections") {
		t.Errorf("a side of zero gave %d sections of a tiny frame", got)
	}
}

// hits is whether a ray meets a ball.
func hits(ray maths.Ray, centre maths.Vec, radius float64) bool {
	toward := centre.Sub(ray.At)
	along := toward.Dot(ray.Dir)
	return positive(along) && toward.Sub(ray.Dir.Scale(along)).Len() <= radius
}

// sightsOf is the sight of each of the sections of the tests' frame.
func sightsOf(t *testing.T, view eye) ([]image.Rectangle, []section.Sight) {
	t.Helper()
	boxes := section.Split(view.wide, view.high, counted(t, "frame-side"))
	sights := make([]section.Sight, len(boxes))
	for at, box := range boxes {
		sights[at] = section.SightOf(view.look, box)
	}
	return boxes, sights
}

// ball is a ball the sight test puts where a pixel's ray goes, by its
// number.
type ball struct {
	centre maths.Vec
	radius float64
	far    float64
}

// aimed is the ball the sight test puts, by its number k, along the ray
// of a pixel picked by hashes.
func aimed(t *testing.T, view eye, k int64) ball {
	t.Helper()
	seedX, seedY := stream(t, "seed-pixel-x"), stream(t, "seed-pixel-y")
	acrossX := noise.Hash(seedX, k) * float64(view.wide)
	acrossY := noise.Hash(seedY, k) * float64(view.high)
	aim := view.look(int(acrossX), int(acrossY))
	far := picked(t, "ball-nearest") + picked(t, "ball-spread")*noise.Hash(
		stream(t, "seed-far"), k)
	centre := aim.At.Add(aim.Dir.Scale(far))
	first := noise.Hash(stream(t, "seed-size-first"), k)
	second := noise.Hash(stream(t, "seed-size-second"), k)
	radius := far * picked(t, "ball-size-share") * first * second
	return ball{centre: centre, radius: radius, far: far}
}

// unmet stops the test if a pixel of a box the sight says no to has a ray
// that meets the ball.
func unmet(t *testing.T, view eye, k int64, around ball, box image.Rectangle) {
	t.Helper()
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if hits(view.look(x, y), around.centre, around.radius) {
				t.Fatalf("ball %d is met by the ray of pixel %d, %d "+
					"and not by its section %v", k, x, y, box)
			}
		}
	}
}

// a sight never says no to a ball one of its section's rays meets: over
// thousands of balls of every size put where some pixel's ray goes, the
// section of every pixel whose ray meets the ball meets it. and it is
// worth having: a small ball is met by few of the sections.
func TestASightMeetsWhatItsRaysMeet(t *testing.T) {
	view := newEye(t)
	boxes, sights := sightsOf(t, view)
	balls := int64(counted(t, "balls"))
	met, small := nothing, nothing
	for k := range balls {
		around := aimed(t, view, k)
		for at, box := range boxes {
			if sights[at].Meets(around.centre, around.radius) {
				met++
				continue
			}
			unmet(t, view, k, around, box)
		}
		if around.radius < around.far*picked(t, "small-share") {
			small++
		}
	}
	share := float64(met) / float64(int(balls)*len(boxes))
	if !(share < picked(t, "met-share-most")) || small == nothing {
		t.Fatalf("balls are met by %.0f%% of sections: a sight that says "+
			"yes to everything is no use", percent*share)
	}
}

// a ball behind the eye, or well off to one side, is met by no section;
// one the eye is inside is met by all.
func TestASightKnowsBehindAndAbout(t *testing.T) {
	view := newEye(t)
	ahead := view.look(view.wide/two, view.high/two).Dir
	behind := ahead.Scale(picked(t, "behind-distance"))
	inside := ahead.Scale(picked(t, "inside-distance"))
	boxes, sights := sightsOf(t, view)
	for at, box := range boxes {
		if sights[at].Meets(behind, picked(t, "behind-radius")) {
			t.Errorf("section %v meets a ball behind the eye", box)
		}
		if !sights[at].Meets(inside, picked(t, "inside-radius")) {
			t.Errorf("section %v misses a ball the eye is inside", box)
		}
	}
}
