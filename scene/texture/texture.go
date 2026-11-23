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
	At(point maths.Vec) maths.Light
}

// Timed is a texture that moves: Tick is the texture as it is at a
// time, which the tracer asks for once a picture. A texture that is
// not Timed is the same at every time.
type Timed interface {
	Texture
	Tick(elapsed time.Duration) Texture
}

// Oriented is a texture that paints by where a point stands in the world
// and which way the surface faces there, which a point in the thing's own
// space does not say: weather settles on what faces up and runs down
// what faces across.
//
// Facing takes the point in the thing's own space, the
// same point in the world, and the surface's unit normal in the world,
// turned toward the eye and taken before any wrinkle tilts it. A texture
// that is Oriented still paints by At where nothing says which way the
// surface faces.
type Oriented interface {
	Texture
	Facing(local, world, normal maths.Vec) maths.Light
}

// Footprinted is a texture that paints differently by how big the pixel asking
// is: a world that elaborates where the eye is close and leaves it where the
// eye is far. The footprint is the pixel's width at the point, in the thing's
// own units; zero is no pixel at all, as a lone ray asks.
type Footprinted interface {
	Texture
	Within(point maths.Vec, footprint float64) maths.Light
}

// Sample is a texture's paint at a point for a pixel footprint wide: by the
// footprint where the texture can, and otherwise as it always paints.
func Sample(texture Texture, point maths.Vec, footprint float64) maths.Light {
	if footprinted, ok := texture.(Footprinted); ok && footprint > nought {
		return footprinted.Within(point, footprint)
	}
	return texture.At(point)
}

// checker is cubes of side size, alternating between first and second.
type checker struct {
	size          float64
	first, second maths.Light
}

// Checker is cubes of side size alternating between two colours, like a
// chessboard in every direction. A size that is not over nought has no
// cubes to tell apart, and is the first colour everywhere.
//
// The cubes are centred on the origin, so a floor at height zero lies in the
// middle of one layer and never on the boundary between two. Otherwise the
// last bit of arithmetic noise in a point on the floor would decide which
// layer it was in, and the floor would speckle.
func Checker(size float64, first, second maths.Light) Texture {
	return checker{size, first, second}
}

// cellOf is which cell of a checker of the size a coordinate falls in
// along one axis.
func cellOf(coordinate, size float64) int {
	return int(math.Floor(coordinate/size + half))
}

// At is the first colour or the second by which cell of the checker the
// point falls in, counting cells along all three axes so the pattern is a
// checker in every direction.
func (check checker) At(point maths.Vec) maths.Light {
	if !(check.size > nought) {
		return check.first
	}
	cells := cellOf(point.X, check.size) + cellOf(point.Y, check.size) +
		cellOf(point.Z, check.size)
	if cells%pair == nought {
		return check.first
	}
	return check.second
}
