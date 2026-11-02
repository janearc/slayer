// Package texture describes spatially varying surface appearance. Textures are
// evaluated at three-dimensional local coordinates rather than only through a
// wrapped image. A checker pattern forms squares on a floor and cuts through a
// sphere as cubes, like grain exposed by carving marble.
package texture

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// Texture is how much of each colour of light a surface gives back at a
// point in the thing's own space, each part from 0 to 1.
type Texture interface {
	At(p maths.Vec) maths.Light
}

// Timed is a texture that moves: Tick is the texture as it is at a
// time, which the tracer asks for once a picture. A texture that is
// not Timed is the same at every time.
type Timed interface {
	Texture
	Tick(t time.Duration) Texture
}

// Footprinted is a texture that paints differently by how big the pixel asking
// is: a world that elaborates where the eye is close and leaves it where the
// eye is far. The footprint is the pixel's width at p, in the thing's own
// units; zero is no pixel at all, as a lone ray asks.
type Footprinted interface {
	Texture
	Within(p maths.Vec, footprint float64) maths.Light
}

// Sample is a texture's paint at p for a pixel footprint wide: by the
// footprint where the texture can, and otherwise as it always paints.
func Sample(t Texture, p maths.Vec, footprint float64) maths.Light {
	if f, ok := t.(Footprinted); ok && footprint > 0 {
		return f.Within(p, footprint)
	}
	return t.At(p)
}

// checker is cubes of side size, alternating between a and b.
type checker struct {
	size float64
	a, b maths.Light
}

// Checker is cubes of side size alternating between two colours, like a
// chessboard in every direction.
//
// The cubes are centred on the origin, so a floor at height zero lies in the
// middle of one layer and never on the boundary between two. Otherwise the
// last bit of arithmetic noise in a point on the floor would decide which
// layer it was in, and the floor would speckle.
func Checker(size float64, a, b maths.Light) Texture {
	return checker{size, a, b}
}

// Checks is a texture that can say it is a checker, and of what: for
// anything that writes a scene back out, as the css interface does.
type Checks interface {
	Checks() (size float64, a, b maths.Light)
}

// Checks is the checker's size and colours.
func (c checker) Checks() (float64, maths.Light, maths.Light) {
	return c.size, c.a, c.b
}

// At is a or b by which cell of the checker p falls in, counting cells
// along all three axes so the pattern is a checker in every direction.
func (c checker) At(p maths.Vec) maths.Light {
	cell := func(v float64) int {
		return int(math.Floor(v/c.size + 0.5))
	}
	if (cell(p.X)+cell(p.Y)+cell(p.Z))%2 == 0 {
		return c.a
	}
	return c.b
}
