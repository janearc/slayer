package emit

import (
	"github.com/janearc/celestial/bands"
	"github.com/janearc/slayer/maths"
)

// NewPoint describes a spectral source at a place, optionally directed by
// later cone controls. When evaluates its spectrum as linear display RGB.
// Construction alone does not check the public Point's mutable fields.
func NewPoint(at, toward maths.Vec, light bands.Light) Point {
	return Point{At: at, Toward: toward, Light: light}
}
