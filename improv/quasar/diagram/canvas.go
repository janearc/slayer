package diagram

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/improv/quasar/rings"
)

// The numbers the drawing is made of, defined.
const (
	none         = 0
	one          = 1
	two          = 2
	horizon      = 2 // a still hole's horizon, in masses
	photonSphere = 3 // where light goes round a still hole, in masses
	opaque       = 255
)

// The numbers the drawing picked: the quasar slice, 2026-11-12.
const (
	// finer is how many times larger the drawing is made before it is
	// scaled down, so its edges are smooth.
	finer = 3
	// thin and thick are line widths, in fine pixels.
	thin  = 3
	thick = 9
	// rayWidth is a ray's width, in fine pixels.
	rayWidth = 7
	// landing is the radius of the dot where a ray lands, in fine pixels.
	landing = 14
	// dashes is how many dashes the photon sphere is drawn in.
	dashes = 48
	// stride is how far apart, in fine pixels, a line's stamps are.
	stride = 0.5
)

// The colours the drawing picked: the quasar slice, 2026-11-12.
var (
	background   = color.RGBA{R: 10, G: 12, B: 22, A: opaque}
	black        = color.RGBA{A: opaque}
	edgeColour   = color.RGBA{R: 90, G: 90, B: 100, A: opaque}
	sphereColour = color.RGBA{R: 120, G: 120, B: 140, A: opaque}
	discColour   = color.RGBA{R: 255, G: 160, B: 60, A: opaque}
)

// canvas is a picture being drawn on, and how many masses its half side
// is.
type canvas struct {
	picture *image.RGBA
	view    View
}

// newCanvas is a canvas so many pixels on a side reaching so many masses.
func newCanvas(side int, reach float64) canvas {
	return canvas{picture: image.NewRGBA(image.Rect(none, none, side,
		side)), view: View{side: side, reach: reach}}
}

// fill paints the whole canvas one colour.
func (paper canvas) fill(colour color.RGBA) {
	bounds := paper.picture.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			paper.picture.SetRGBA(x, y, colour)
		}
	}
}

// dot paints a round dot of a radius in pixels at a pixel place; what
// falls off the canvas is left.
func (paper canvas) dot(x, y, radius float64, colour color.RGBA) {
	for row := int(y - radius); row <= int(y+radius); row++ {
		for column := int(x - radius); column <= int(x+radius); column++ {
			across, down := float64(column)-x, float64(row)-y
			if across*across+down*down <= radius*radius {
				paper.picture.SetRGBA(column, row, colour)
			}
		}
	}
}

// line paints a straight line between two points in masses.
func (paper canvas) line(from, to rings.Point, colour color.RGBA,
	width float64) {
	x0, y0 := paper.view.place(from)
	x1, y1 := paper.view.place(to)
	length := math.Hypot(x1-x0, y1-y0)
	marks := int(length/stride) + one
	for mark := range marks + one {
		share := float64(mark) / float64(marks)
		paper.dot(x0+share*(x1-x0), y0+share*(y1-y0), width/two, colour)
	}
}

// disc paints a filled circle about the hole's centre of a radius in
// masses.
func (paper canvas) disc(radius float64, colour color.RGBA) {
	x, y := paper.view.place(rings.Point{})
	scale := float64(paper.view.side) / (two * paper.view.reach)
	paper.dot(x, y, radius*scale, colour)
}

// circle paints a circle about the hole's centre of a radius in masses,
// dashed if asked.
func (paper canvas) circle(radius float64, colour color.RGBA,
	width float64, dashed bool) {
	steps := dashes * two * dashes
	for step := range steps {
		if dashed && (step*dashes*two/steps)%two == one {
			continue
		}
		angle := two * math.Pi * float64(step) / float64(steps)
		point := rings.Point{Along: radius * math.Cos(angle),
			Up: radius * math.Sin(angle)}
		x, y := paper.view.place(point)
		paper.dot(x, y, width/two, colour)
	}
}

// ray paints a ray's path from the eye to its crossing of the disc's
// plane of its order, and a dot there, where the light it brings starts.
// A ray that does not land at its order is drawn whole, with no dot.
func (paper canvas) ray(ray Ray, disc rings.Disc) {
	points := ray.Path.Points()
	lands := ray.Path.Lands(ray.Order, disc)
	crossings := none
	for each := one; each < len(points); each++ {
		from, to := points[each-one], points[each]
		if (from.Up > none) != (to.Up > none) {
			crossings++
		}
		if lands && crossings > ray.Order {
			end := onPlane(from, to)
			paper.line(from, end, ray.Colour, rayWidth)
			x, y := paper.view.place(end)
			paper.dot(x, y, landing, ray.Colour)
			return
		}
		paper.line(from, to, ray.Colour, rayWidth)
	}
}

// onPlane is where the straight step between two points meets the
// disc's plane, as the rings package places a crossing.
func onPlane(from, to rings.Point) rings.Point {
	share := from.Up / (from.Up - to.Up)
	return rings.Point{Along: from.Along + share*(to.Along-from.Along)}
}
