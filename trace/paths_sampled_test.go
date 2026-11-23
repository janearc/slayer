package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Sampled with one ray a pixel is Render, pixel for pixel.
func TestSampledOnceIsRender(t *testing.T) {
	lens := film.Pinhole{}
	once := sampled(smoking(), overTheScene, lens, 0, 16, 16, 1)
	plain := draw(smoking(), overTheScene, lens, 0, 16, 16)
	samePixels(t, once, plain, "sampled once,")
}

// meanOf is the mean of the four fine pixels under one coarse pixel,
// as light and cover, with the nearest of their depths.
func meanOf(fine *frame.Frame, x, y int) frame.Pixel {
	mean := frame.Pixel{Depth: float32(math.Inf(1))}
	four := []frame.Pixel{fine.At(2*x, 2*y), fine.At(2*x+1, 2*y),
		fine.At(2*x, 2*y+1), fine.At(2*x+1, 2*y+1)}
	for _, each := range four {
		mean.R += each.R / 4
		mean.G += each.G / 4
		mean.B += each.B / 4
		mean.Cover += each.Cover / 4
		mean.Depth = min(mean.Depth, each.Depth)
	}
	return mean
}

// alike is whether two pixels have the same light and cover to a
// millionth, and the same depth.
func alike(got, want frame.Pixel) bool {
	near := func(got, want float32) bool {
		return within(float64(got), float64(want), 1e-6)
	}
	return near(got.R, want.R) && near(got.G, want.G) &&
		near(got.B, want.B) && near(got.Cover, want.Cover) &&
		got.Depth == want.Depth
}

// Sampled with two rays per pixel each way is, at every pixel, the mean of the
// four rays' light and cover and the nearest of their depths. On the ball's
// edge that is a cover between none and all.
func TestSampledTwiceIsTheMeanOfFour(t *testing.T) {
	lens := film.Pinhole{}
	twice := sampled(smoking(), overTheScene, lens, 0, 16, 16, 2)
	fine := draw(smoking(), overTheScene, lens, 0, 32, 32)
	edges := 0
	for i, got := range twice.Pix {
		want := meanOf(fine, i%16, i/16)
		if !alike(got, want) {
			t.Fatalf("pixel %d is %v, its four's mean %v", i, got,
				want)
		}
		if got.Cover > 0 && got.Cover < 1 {
			edges++
		}
	}
	if edges == 0 {
		t.Error("no pixel lies across an edge, so the mean of cover " +
			"was never asked")
	}
}

// footprint is a texture that paints how wide it was asked to paint:
// what comes back says what pixel width the tracer handed down.
type footprint struct{}

// widthAsked is the pixel width a ball two units from the eye is asked to paint
// at, at the picture's middle, in a world whose Pixel is set or left at zero.
// the ball's matte face, square to a sun of one, gives back its colour over pi,
// so the width is pi times what comes back, over the distance.
func widthAsked(pixel float64) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Texture: footprint{}}})
	along := maths.V(0, 0, 1)
	light := maths.Grey(1)
	world.Lamp(thing.Sun(along, light))
	world.Pixel = pixel
	eye := lookingAt(maths.V(0, 0, 3), maths.Vec{})
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	const side, middle, away = 65, 32, 2.0
	return math.Pi * float64(draw(world, eye, lens, 0, side, side).
		At(middle, middle).R) / away
}

// A pixel's angle is the angle between the middle pixel's ray and its
// neighbour's. For a pinhole sixty degrees wide over sixty-five pixels
// that is the arctangent of one pixel's share of the picture's width.
// A world's own Pixel stands in for it when set.
func TestThePixelAngleIsThePicturesOrTheWorlds(t *testing.T) {
	share := 2 * math.Tan(30*maths.Degree) / 65
	want := math.Atan(share)
	if got := widthAsked(0); !within(got, want, 1e-4*want) {
		t.Errorf("the picture's own pixel is %v, want %v", got, want)
	}
	if got := widthAsked(0.01); !within(got, 0.01, 1e-7) {
		t.Errorf("with the world's Pixel at 0.01 the width is %v", got)
	}
}

// A thing with no shape is not there: a world holding one draws as the
// empty world does, under the same sky, and nothing panics.
func TestAThingWithNoShapeIsNotThere(t *testing.T) {
	lens := film.Pinhole{}
	empty := thing.NewWorld()
	empty.Sky = even(maths.Grey(0.5))
	shapeless := thing.NewWorld()
	shapeless.Sky = even(maths.Grey(0.5))
	shapeless.Add(thing.Thing{})
	want := draw(empty, overTheScene, lens, 0, 8, 8)
	got := draw(shapeless, overTheScene, lens, 0, 8, 8)
	samePixels(t, got, want, "with a shapeless thing,")
}

// At is black: asked with no width.
func (footprint) At(maths.Vec) maths.Light { return maths.Light{} }

// Within is the width asked for, as light.
func (footprint) Within(_ maths.Vec, wide float64) maths.Light {
	return maths.Grey(wide)
}
