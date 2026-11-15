// Package diagram draws the photon ring short's top-down picture: the
// hole and its photon sphere seen from the side, the disc edge on, and
// rays the rings package followed back from the eye, each drawn to the
// crossing where it lands on the disc and the light it brings starts.
//
// It is a diagram, drawn to scale in masses, with no words: the short's
// labels are laid over it by the compositor, at the pixels Place says.
package diagram

import (
	"fmt"
	"image"
	"image/color"
	"math"

	xdraw "golang.org/x/image/draw"

	"github.com/janearc/slayer/improv/quasar/rings"
)

// Ray is a ray to draw: its path, the order it lands at on the disc, and
// its colour.
type Ray struct {
	Path   rings.Path
	Order  int
	Colour color.RGBA
}

// View is the diagram's frame: its side in pixels and how many masses
// from the hole's centre to its edge.
type View struct {
	side  int
	reach float64
}

// NewView is a square diagram so many pixels on a side, reaching so many
// masses from the hole's centre to each edge. It refuses a side under a
// pixel, and a reach that does not hold the horizon, two masses.
func NewView(side int, reach float64) (View, error) {
	if side < one {
		return View{}, fmt.Errorf("diagram: a side of %d pixels is "+
			"none: give one or more", side)
	}
	if !(reach > horizon) || math.IsInf(reach, 0) {
		return View{}, fmt.Errorf("diagram: a reach of %v masses does "+
			"not hold the horizon: give a finite reach past %v",
			reach, horizon)
	}
	return View{side: side, reach: reach}, nil
}

// Place is where a point in masses falls in the diagram, in pixels from
// its top left: along runs right, up runs up.
func (view View) Place(point rings.Point) (x, y float64) {
	scale := float64(view.side) / (two * view.reach)
	half := float64(view.side) / two
	return half + point.Along*scale, half - point.Up*scale
}

// Draw is the diagram: the hole, its photon sphere, the disc edge on,
// and each ray to where it lands.
func (view View) Draw(disc rings.Disc, rays []Ray) *image.RGBA {
	fine := newCanvas(view.side*finer, view.reach)
	fine.fill(background)
	fine.circle(photonSphere, sphereColour, thin, true)
	fine.disc(horizon, black)
	fine.circle(horizon, edgeColour, thin, false)
	for _, side := range []float64{one, -one} {
		fine.line(rings.Point{Along: side * disc.Inner()},
			rings.Point{Along: side * disc.Outer()}, discColour, thick)
	}
	for _, ray := range rays {
		fine.ray(ray, disc)
	}
	out := image.NewRGBA(image.Rect(none, none, view.side, view.side))
	xdraw.CatmullRom.Scale(out, out.Bounds(), fine.picture,
		fine.picture.Bounds(), xdraw.Src, nil)
	return out
}
