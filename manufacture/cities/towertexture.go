package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
)

// uniform is one colour at every point: the surface a material has before
// anything is laid over it, for a texture that needs a surface to lay on.
type uniform struct{ Colour maths.Light }

// At is the colour, wherever p is.
func (uniform uniform) At(maths.Vec) maths.Light { return uniform.Colour }

// product is one texture seen through another: a surface, with a grain laid
// over it by multiplying the light each gives back. A grain of whites and
// greys darkens the surface where it is grey and leaves it where it is
// white, so joints, board marks, crystals, patches and streaks, each its
// own texture, stand together on one face. It is band-limited if its two
// parts are: each is read by the pixel's footprint.
type product struct{ Surface, Grain texture.Texture }

// At is the surface at p, darkened by the grain.
func (product product) At(p maths.Vec) maths.Light {
	return multiplied(product.Surface.At(p), product.Grain.At(p))
}

// Within is the surface at p for a pixel footprint wide, darkened by the
// grain for that footprint, each as its own texture gives it.
func (product product) Within(p maths.Vec, footprint float64) maths.Light {
	return multiplied(texture.Sample(product.Surface, p, footprint),
		texture.Sample(product.Grain, p, footprint))
}

// multiplied is a light of each part of one times the same part of the
// other.
func multiplied(first, second maths.Light) maths.Light {
	return maths.RGB(first.R*second.R, first.G*second.G, first.B*second.B)
}

// ashlar is bricks or stone panels laid in courses, as Brick lays them,
// that fade to the average of the face and its joints once a joint is
// narrower than a pixel, so that a far wall of panels does not shimmer.
type ashlar struct{ brick }

// Within is the courses at p for a pixel footprint wide: the joints fade
// to the share of the surface they cover as they narrow below the pixel.
func (ashlar ashlar) Within(p maths.Vec, footprint float64) maths.Light {
	long, high := ashlar.Length+ashlar.Joint, ashlar.Height+ashlar.Joint
	solid := ashlar.Length * ashlar.Height / (long * high)
	average := ashlar.brick.Brick.Scale(solid).Add(
		ashlar.Mortar.Scale(1 - solid))
	shown := texture.Detail(footprint, ashlar.Joint)
	return ashlar.At(p).Scale(shown).Add(average.Scale(1 - shown))
}

// coursed is boards laid in courses, as Lap lays them, that fade to the
// average of a board and the shadow at its foot once the shadow is
// narrower than a pixel.
type coursed struct{ lap }

// Within is the boards at p for a pixel footprint wide.
func (coursed coursed) Within(p maths.Vec, footprint float64) maths.Light {
	average := coursed.Board.Scale(1 - coursed.Share*(1-coursed.Shadow))
	shown := texture.Detail(footprint, coursed.Height*coursed.Share)
	return coursed.At(p).Scale(shown).Add(average.Scale(1 - shown))
}
