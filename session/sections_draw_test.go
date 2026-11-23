//go:build draw

package session

import (
	_ "embed"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// draw is the picture, on the cpu or the gpu as asked, or exposed
// through a set's rig
func draw(world *thing.World, given asked,
	across float64) (*frame.Frame, error) {
	picture, _, err := drawMeasured(world, given, across)
	return picture, err
}

//go:embed testdata/sections.yaml
var sectionsRaw []byte

// sectionsScene is testdata/sections.yaml: the giant, the ball, the lamp,
// the frame, its sections, and the numbers the tests expect of them.
type sectionsScene struct {
	Giant struct {
		Radius, Across, Height, Depth, Scale float64
		Light, Dark                          struct{ Red, Green, Blue float64 }
	}
	Ball  struct{ Radius, Across, Height, Depth, Grey float64 }
	Lamp  struct{ Across, Height, Depth, First, Second float64 }
	Frame struct {
		Width, Height, Samples int
		Wide                   float64
	}
	Sections       struct{ Over, Size, Count int }
	AtLeast        struct{ Kept, Drawn int } `yaml:"at-least"`
	SmallFrameOver int                       `yaml:"small-frame-over"`
	MovedBy        float64                   `yaml:"moved-by"`
}

// sectionsFixture reads testdata/sections.yaml.
func sectionsFixture(t *testing.T) sectionsScene {
	t.Helper()
	var data sectionsScene
	if err := yamlfile.Read("sections.yaml", sectionsRaw, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

// lit is the world of the giant and the ball under a lamp of brightness.
func (data sectionsScene) lit(t *testing.T, giant, ball *thing.Thing,
	brightness float64) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.Things = append(world.Things, giant, ball)
	direction := maths.V(data.Lamp.Across, data.Lamp.Height, data.Lamp.Depth)
	lamp, err := emit.NewDistant(direction, maths.Grey(brightness))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(lamp)
	return world
}

// stillAt is the motion of a thing that stands at a place, unturned and
// not scaled.
func stillAt(at maths.Vec) thing.Motion {
	return thing.Still(stillSpace(at))
}

// stillSpace is the frame of something at a place, unturned and not
// scaled.
func stillSpace(at maths.Vec) maths.Subspace {
	return maths.NewSubspace(at, maths.Identity, nothing)
}

// keptScene is a giant that shows only its own light and a clay ball a
// lamp lights, seen from an eye that looks down the z axis, and the
// worlds of two frames of them: the lamp twice as bright in the second.
func keptScene(t *testing.T) (first, second *thing.World, given asked) {
	t.Helper()
	data := sectionsFixture(t)
	light, dark := data.Giant.Light, data.Giant.Dark
	shown := thing.Polished("giant", maths.Grey(nothing), nothing, nothing)
	shown.Shine = texture.Checker(data.Giant.Scale,
		maths.RGB(light.Red, light.Green, light.Blue),
		maths.RGB(dark.Red, dark.Green, dark.Blue))
	giantAt := maths.V(data.Giant.Across, data.Giant.Height, data.Giant.Depth)
	giant := &thing.Thing{Shape: shape.Sphere{Radius: data.Giant.Radius},
		Material: shown, Motion: stillAt(giantAt)}
	clay := thing.Polished("clay", maths.Grey(data.Ball.Grey), nothing, nothing)
	ballAt := maths.V(data.Ball.Across, data.Ball.Height, data.Ball.Depth)
	ball := &thing.Thing{Shape: shape.Sphere{Radius: data.Ball.Radius},
		Material: clay, Motion: stillAt(ballAt)}
	first = data.lit(t, giant, ball, data.Lamp.First)
	second = data.lit(t, giant, ball, data.Lamp.Second)
	eye := maths.Local()
	frame := data.Frame
	given = asked{width: frame.Width, height: frame.Height,
		samples: frame.Samples, wide: frame.Wide, eye: &eye,
		sections: sectionsAsked{over: data.Sections.Over,
			size: data.Sections.Size, keep: true}}
	return first, second, given
}

// sameFrame fails unless two frames are the same in every bit.
func sameFrame(t *testing.T, name string, got, want *frame.Frame) {
	t.Helper()
	for y := range want.H {
		for x := range want.W {
			if got.At(x, y) != want.At(x, y) {
				t.Fatalf("%s: pixel %d, %d is %v, and drawn whole %v", name, x,
					y, got.At(x, y), want.At(x, y))
			}
		}
	}
}

// keepNone is a film that says keep: false.
func keepNone(given *asked) { given.sections.keep = false }

// shrunk is a film whose frame is no wider than its sections start at.
func (data sectionsScene) shrunk(given *asked) {
	given.sections.over = data.SmallFrameOver
}

// a sequence's second frame, made from its first by keeping sections, is
// the second frame drawn whole in every bit, and the giant's sections
// were kept; a film that says keep: false, or a frame no wider than its
// sections start at, keeps none.
func TestASequenceKeepsSectionsThatCannotChange(t *testing.T) {
	first, second, given := keptScene(t)
	keep := &keeper{}
	if _, _, err := keep.picture(first, given, nothing); err != nil {
		t.Fatal(err)
	}
	given.at = time.Second
	got, _, err := keep.picture(second, given, nothing)
	if err != nil {
		t.Fatal(err)
	}
	want, err := draw(second, given, nothing)
	if err != nil {
		t.Fatal(err)
	}
	sameFrame(t, "kept", got, want)
	data := sectionsFixture(t)
	if keep.kept < data.AtLeast.Kept || keep.drawn < data.AtLeast.Drawn ||
		keep.kept+keep.drawn != data.Sections.Count {
		t.Fatalf("%d sections kept and %d drawn of %d", keep.kept,
			keep.drawn, data.Sections.Count)
	}
	for name, change := range map[string]func(*asked){
		"keep: false":   keepNone,
		"a small frame": data.shrunk,
	} {
		_, _, given := keptScene(t)
		change(&given)
		fresh := &keeper{}
		for _, world := range []*thing.World{first, second} {
			if _, _, err := fresh.picture(world, given, nothing); err != nil {
				t.Fatal(err)
			}
		}
		if fresh.kept != nothing || fresh.raw != nil {
			t.Errorf("%s: %d sections kept", name, fresh.kept)
		}
	}
}

// a camera that moved between frames keeps nothing of the last frame.
func TestAMovedCameraKeepsNoSection(t *testing.T) {
	first, second, given := keptScene(t)
	keep := &keeper{}
	if _, _, err := keep.picture(first, given, nothing); err != nil {
		t.Fatal(err)
	}
	movedBy := sectionsFixture(t).MovedBy
	moved := stillSpace(maths.V(movedBy, nothing, nothing))
	given.eye = &moved
	got, _, err := keep.picture(second, given, nothing)
	if err != nil {
		t.Fatal(err)
	}
	want, err := draw(second, given, nothing)
	if err != nil {
		t.Fatal(err)
	}
	sameFrame(t, "a moved camera", got, want)
	if keep.kept != nothing {
		t.Fatalf("%d sections kept though the camera moved", keep.kept)
	}
}
