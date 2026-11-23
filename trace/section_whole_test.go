package trace

import (
	_ "embed"
	"image"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture/mocks"
	"github.com/janearc/slayer/scene/thing"
	"go.uber.org/mock/gomock"
)

// sectionScenes are worlds a section is tried on: smoke, a haze over a
// ground, and a mirror, so volumes, sky and bounces are all in it.
func sectionScenes(t *testing.T) map[string]*thing.World {
	return map[string]*thing.World{"smoke": smoking(),
		"a haze": weakHaze(), "a mirror": weakMirror(),
		"a floor painted by its pixel's width": measuredFloor(t)}
}

// widthOf is paint that is as bright as the pixel asking is wide, so a
// picture of it shows the width each ray was given.
func widthOf(_ maths.Vec, footprint float64) maths.Light {
	return maths.Grey(footprint)
}

// measuredFloor is a floor that glows by its pixel's width: its paint is a
// mock that is dark for a lone ray and as bright as the pixel is wide
// otherwise.
func measuredFloor(t *testing.T) *thing.World {
	t.Helper()
	paint := mocks.NewMockFootprinted(gomock.NewController(t))
	paint.EXPECT().At(gomock.Any()).Return(maths.Grey(none)).AnyTimes()
	paint.EXPECT().Within(gomock.Any(), gomock.Any()).DoAndReturn(widthOf).AnyTimes()
	world := thing.NewWorld()
	ruler := thing.Polished("ruler", maths.Grey(none), none, none)
	ruler.Shine = paint
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: ruler})
	return world
}

// positive is whether x is above none. NaN is not.
func positive(x float32) bool { return x > none }

// a frame drawn section by section is the frame drawn whole, in every
// bit of every pixel: sections of odd sizes, at one ray a pixel and at
// nine.
func TestSectionsAreTheWholeFrame(t *testing.T) {
	data := sectionFixture(t)
	width, height := data.Frame.Width, data.Frame.Height
	eye, lens := sectionEye(data)
	for name, world := range sectionScenes(t) {
		for _, rays := range data.Rays {
			whole, err := Sampled(world, eye, lens, atStart, width, height,
				rays)
			if err != nil {
				t.Fatal(err)
			}
			lit := none
			for _, tile := range data.Tiles {
				box := tile.box()
				part, err := sampledIn(world, eye, lens, atStart, width,
					height, rays, box)
				if err != nil {
					t.Fatal(err)
				}
				lit += sameSection(t, name, whole, part, box)
			}
			if lit < width*height/quarter {
				t.Fatalf("%s: only %d pixels lit: too dark to tell", name, lit)
			}
		}
	}
}

// sameSection fails unless a section is the whole frame's pixels in its
// box, and returns how many of them are lit.
func sameSection(t *testing.T, name string, whole, part *frame.Frame,
	box image.Rectangle) (lit int) {
	t.Helper()
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			want, got := whole.At(x, y), part.At(x-box.Min.X, y-box.Min.Y)
			if got != want {
				t.Fatalf("%s: pixel %d, %d of the section in %v is %v, of "+
					"the whole %v", name, x, y, box, got, want)
			}
			if positive(want.R) || positive(want.G) || positive(want.B) {
				lit++
			}
		}
	}
	return lit
}
