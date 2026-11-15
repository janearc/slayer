//go:build draw

package section_test

import (
	"image"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/section"
	"github.com/janearc/slayer/trace"
)

// stage is a small Melancholia: a giant up and to the left that shows
// only its own light, a clay ball low and to the right that a lamp
// lights, and the lamp, so bright by a number. The same giant and ball
// are in every stage made from the same parts.
type parts struct {
	giant, ball *thing.Thing
}

// solid is a ball of a radius, made of a material, at a place.
func solid(radius float64, material *thing.Material,
	where maths.Subspace) *thing.Thing {
	return &thing.Thing{Shape: shape.Sphere{Radius: radius},
		Material: material, Motion: thing.Still(where)}
}

// newParts is the giant and the ball.
func newParts(t *testing.T) parts {
	t.Helper()
	made := thing.Diffuse("giant", maths.Grey(nothing))
	made.Shine = texture.Checker(picked(t, "giant-check"),
		shade(t, "giant light"), shade(t, "giant dark"))
	grey := maths.Grey(picked(t, "clay-grey"))
	clay := thing.Diffuse("clay", grey)
	return parts{
		giant: solid(picked(t, "giant-radius"), made, placed(t, "giant")),
		ball:  solid(picked(t, "ball-radius"), clay, placed(t, "ball"))}
}

// staged is a world of the parts under a lamp of a strength.
func (made parts) staged(t *testing.T, strength float64) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.Things = append(world.Things, made.giant, made.ball)
	lamp, err := emit.NewDistant(place(t, "sun toward"), maths.Grey(strength))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(lamp)
	return world
}

// dim is the strength of the lamp before it brightens.
func dim(t *testing.T) float64 { return picked(t, "lamp-dim") }

// bright is the strength of the lamp after it brightens.
func bright(t *testing.T) float64 { return picked(t, "lamp-bright") }

// drawn is a world's picture and the sight of each of its sections.
func drawn(t *testing.T, world *thing.World) (*frame.Frame,
	[]image.Rectangle, []section.Sight) {
	t.Helper()
	view := newEye(t)
	picture, err := trace.Sampled(world, maths.Local(), view.lens, nothing,
		view.wide, view.high, counted(t, "sample-count"))
	if err != nil {
		t.Fatal(err)
	}
	boxes, sights := sightsOf(t, view)
	return picture, boxes, sights
}

// shows is whether a pixel is bright enough in red to be the giant's.
func shows(t *testing.T, red float32) bool {
	t.Helper()
	return float64(red) > picked(t, "giant-shows")
}

// sameBox is whether two pictures are the same in every pixel of a box,
// and whether the first shows the giant anywhere in it.
func sameBox(t *testing.T, first, second *frame.Frame,
	box image.Rectangle) (same, showing bool) {
	t.Helper()
	same = true
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			same = same && first.At(x, y) == second.At(x, y)
			showing = showing || shows(t, first.At(x, y).R)
		}
	}
	return same, showing
}

// tally is, over the sections of two pictures, how many were kept, how
// many of the kept show the giant, and how many changed; it fails if a
// kept section differs in any bit of any pixel.
func tally(t *testing.T, name string, before, after *thing.World) (kept,
	lit, changed int) {
	t.Helper()
	first, boxes, sights := drawn(t, before)
	second, _, _ := drawn(t, after)
	change := section.Between(before, after, nothing, time.Second)
	for at, box := range boxes {
		same, showing := sameBox(t, first, second, box)
		if !same {
			changed++
		}
		if !change.Keeps(sights[at]) {
			continue
		}
		if !same {
			t.Fatalf("%s: section %v was kept and its picture changed", name,
				box)
		}
		kept++
		if showing {
			lit++
		}
	}
	return kept, lit, changed
}

// when a lamp changes, the sections that show only the giant and the sky
// are kept and the ones the clay ball is in are not: no kept section's
// picture differs in any bit, and the giant's are among the kept.
func TestSectionsOfASelfLitBodyAreKeptWhenALampChanges(t *testing.T) {
	made := newParts(t)
	before, after := made.staged(t, dim(t)), made.staged(t, bright(t))
	kept, lit, changed := tally(t, "a brighter lamp", before, after)
	if changed == nothing {
		t.Fatal("the brighter lamp changed no section: nothing was tried")
	}
	total := counted(t, "frame-sections")
	if kept+changed > total || lit < counted(t, "giant-sections-least") ||
		kept < counted(t, "self-lit-kept-least") {
		t.Fatalf("%d sections kept, %d of them the giant's, %d changed, "+
			"of %d", kept, lit, changed, total)
	}
}

// nothing changed, and every section is kept.
func TestEverySectionIsKeptWhenNothingChanges(t *testing.T) {
	made := newParts(t)
	world := made.staged(t, dim(t))
	total := counted(t, "frame-sections")
	if kept, _, changed := tally(t, "the same world", world,
		world); kept != total || changed != nothing {
		t.Fatalf("%d sections kept and %d changed, of %d", kept, changed,
			total)
	}
}

// bend is a field that bends rays, in a world.
func bend(t *testing.T, world *thing.World) {
	t.Helper()
	still := thing.Still(maths.Local())
	field := &thing.Field{Mass: picked(t, "field-mass"),
		Reach: picked(t, "field-reach"), Motion: still}
	world.Fields = append(world.Fields, field)
}

// a thing that moves across the giant takes the sections it leaves and
// the ones it comes to out of the kept; a field that bends rays keeps
// none.
func TestSectionsAreDrawnAgainWhenTheyMightChange(t *testing.T) {
	made := newParts(t)
	before := made.staged(t, dim(t))
	moved := newParts(t)
	moved.giant = made.giant
	moved.ball.Motion = thing.Still(placed(t, "ball moved"))
	after := moved.staged(t, dim(t))
	kept, _, changed := tally(t, "the ball moved", before, after)
	if changed < counted(t, "changed-least") || kept == nothing ||
		kept+changed > counted(t, "frame-sections") {
		t.Fatalf("the ball moved: %d kept, %d changed", kept, changed)
	}
	bent := made.staged(t, dim(t))
	bend(t, bent)
	change := section.Between(before, bent, nothing, nothing)
	_, _, sights := drawn(t, before)
	for at, sight := range sights {
		if change.Keeps(sight) {
			t.Fatalf("section %d was kept in a world whose rays bend", at)
		}
	}
}

// a moon that shows only its own light and moves across the giant is
// still a change: the sections it leaves and the ones it comes to are
// drawn again, and no kept section's picture differs.
func TestAMovingSelfLitBodyIsAChange(t *testing.T) {
	made := newParts(t)
	made.ball.Material = thing.Diffuse("moon", maths.Grey(nothing))
	light, dark := picked(t, "moon-light"), picked(t, "moon-dark")
	made.ball.Material.Shine = texture.Checker(picked(t, "moon-check"),
		maths.Grey(light), maths.Grey(dark))
	before := made.staged(t, dim(t))
	moved := made
	moon := *made.ball
	moon.Motion = thing.Still(placed(t, "ball moved"))
	moved.ball = &moon
	after := moved.staged(t, dim(t))
	kept, _, changed := tally(t, "the moon moved", before, after)
	if changed < counted(t, "changed-least") || kept == nothing {
		t.Fatalf("the moon moved: %d kept, %d changed", kept, changed)
	}
}

// redraw draws again the sections it is asked for, and counts them.
type redraw struct {
	test  *testing.T
	world *thing.World
	asked int
}

// draw is the sections of the redraw's world that it is asked for.
func (again *redraw) draw(boxes []image.Rectangle) ([]*frame.Frame, error) {
	again.asked = len(boxes)
	view := newEye(again.test)
	return trace.SampledEach(again.world, maths.Local(), view.lens, nothing,
		view.wide, view.high, counted(again.test, "sample-count"), boxes)
}

// the frame after a lamp changes, made from the frame before by keeping
// what is kept and drawing the rest, is the later frame drawn whole in
// every bit; and the giant's sections were not drawn again.
func TestTheNextFrameFromTheLastIsTheNextFrame(t *testing.T) {
	made := newParts(t)
	before, after := made.staged(t, dim(t)), made.staged(t, bright(t))
	earlier, boxes, sights := drawn(t, before)
	whole, _, _ := drawn(t, after)
	again := &redraw{test: t, world: after}
	change := section.Between(before, after, nothing, time.Second)
	later, kept, err := change.Redrawn(earlier, boxes, sights, again.draw)
	if err != nil {
		t.Fatal(err)
	}
	if kept < counted(t, "self-lit-kept-least") ||
		kept+again.asked != len(boxes) {
		t.Fatalf("%d sections kept and %d drawn of %d", kept, again.asked,
			len(boxes))
	}
	view := newEye(t)
	for y := range view.high {
		for x := range view.wide {
			if later.At(x, y) != whole.At(x, y) {
				t.Fatalf("pixel %d, %d is %v, and drawn whole %v", x, y,
					later.At(x, y), whole.At(x, y))
			}
		}
	}
	probeX, probeY := counted(t, "probe-x"), counted(t, "probe-y")
	if earlier.At(probeX, probeY) == whole.At(probeX, probeY) {
		t.Fatal("the lamp changed nothing at the clay ball: nothing tried")
	}
}
