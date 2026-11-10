package oceantex

import (
	"errors"
	"fmt"
	"math"
)

// The camera here is patterned on slayer's, which is in its ninjafilm/film
// and maths packages: a camera is a projection at a subspace.
//
// A Gauge is a film's frame; a Lens is a focal length on a gauge in front
// of a pinhole; a Projection gives each pixel a ray in its own space, down
// its minus z; a Subspace is where the eye is and which way it faces; Look
// takes the ray out into the world. The names, defaults and conventions
// are slayer's, so a shot framed for one frames the same in the other.
//
// Three things differ, and are said where they do: up in this world is z
// where slayer's is y; a ray here carries no band, since oceantex has
// folded the spectrum into colour before it reaches the camera; and a
// gauge carries no grain, which is the film's business and slayer's.

var (
	gaugeWidths  = nums("gauge.widths")
	gaugeHeights = nums("gauge.heights")
	lensFocal    = num("lens.focal")
	pinholeWide  = num("pinhole.wide") * degrees
	nearlyAlong  = num("lookat.nearly-along")
)

// gaugeNames are the gauges of slayer's table, in its order, which is the
// order of gauge.widths and gauge.heights in sources.yaml.
var gaugeNames = []string{"super 8", "16 mm", "academy", "super 35",
	"full frame", "65 mm", "imax", "hubble wfc3"}

// plain is the gauge a lens is on when none is given.
const plain = "super 35"

var errGauge = errors.New("oceantex: a gauge has a name, and a finite " +
	"width and height above nought in millimetres")

// Gauge is a film's frame. It says how wide and how tall the exposed
// picture is, in millimetres. With a focal length it says how much a lens
// sees, the way a cinematographer says it. On Super 35 a lens of 32 is
// normal, 50 is a little long and 24 is wide.
type Gauge struct {
	name          string
	width, height float64
}

// NewGauge is a gauge of a name and a size in millimetres. It says why
// not when it cannot be.
func NewGauge(name string, width, height float64) (Gauge, error) {
	finite := func(v float64) bool { return v > 0 && !math.IsInf(v, 0) }
	if name == "" || !finite(width) || !finite(height) {
		return Gauge{}, errGauge
	}
	return Gauge{name, width, height}, nil
}

// Name is what the gauge is called.
func (gauge Gauge) Name() string { return gauge.name }

// Width is how wide the frame is, in millimetres.
func (gauge Gauge) Width() float64 { return gauge.width }

// Height is how tall the frame is, in millimetres.
func (gauge Gauge) Height() float64 { return gauge.height }

// Aspect is how wide the frame is against how tall.
func (gauge Gauge) Aspect() float64 {
	if !(gauge.height > 0) {
		return 1
	}
	return gauge.width / gauge.height
}

// Gauges are the names of the gauges that ship: slayer's table.
func Gauges() []string { return append([]string(nil), gaugeNames...) }

// GaugeNamed is the gauge of a name in the table. Names not in it are
// refused, and the refusal lists the names that are.
func GaugeNamed(name string) (Gauge, error) {
	for i, each := range gaugeNames {
		if each == name {
			return NewGauge(name, gaugeWidths[i], gaugeHeights[i])
		}
	}
	return Gauge{}, fmt.Errorf("oceantex: no gauge is called %q; the "+
		"table holds %q", name, gaugeNames)
}

// standard is the gauge Lens leans on. A table without it is a mistake in
// sources.yaml, and stops the program.
var standard = func() Gauge {
	gauge, err := GaugeNamed(plain)
	if err != nil {
		panic(err)
	}
	return gauge
}()

// Ray is a place and a direction from it.
type Ray struct{ At, Dir Vec }

// Projection is the geometry of a picture. Each pixel of a picture has a
// ray it looks along. The projection gives it down its minus z, in its
// own space, with its x to the right and its y up the picture. The
// picture is so many pixels wide and high, and row zero is its top.
//
// Slayer's Projection is also told which band of the spectrum is asking,
// so that glass can bend red and blue differently. This one is not.
type Projection interface {
	Ray(x, y, width, height int) Ray
}

// Pinhole is the first projection. It is light through one point onto a
// flat picture. Straight lines stay straight and far things are smaller.
//
// Its wide is how much it sees from its left edge to its right, in
// radians. Nought means 60 degrees. Its pixel is one pixel's width over
// its height. Nought means square.
type Pinhole struct {
	wide, pixel float64
}

// NewPinhole is a pinhole that sees so many radians across, through
// pixels of a shape. Wides outside zero to a half turn are the default
// of 60 degrees. Pixels without a width greater than zero are square.
func NewPinhole(wide, pixel float64) Pinhole {
	if !(wide >= 0 && wide < math.Pi) {
		wide = 0
	}
	if !(pixel >= 0) {
		pixel = 0
	}
	return Pinhole{wide, pixel}
}

// Wide is how wide the pinhole sees, in radians.
func (pinhole Pinhole) Wide() float64 {
	if pinhole.wide == 0 {
		return pinholeWide
	}
	return pinhole.wide
}

// Ray is the ray through the middle of pixel x, y of a picture so many
// pixels wide and high. Row zero is the top of the picture. Pictures of
// no pixels look straight ahead.
func (pinhole Pinhole) Ray(x, y, width, height int) Ray {
	if width <= 0 || height <= 0 {
		return Ray{Dir: Vec{Z: -1}}
	}
	shape := pinhole.pixel
	if shape == 0 {
		shape = 1
	}
	reach := math.Tan(pinhole.Wide() / 2)
	across, down := float64(width), float64(height)
	// the picture measures the pixel's shape times the pixels across by
	// the pixels down, in pixel heights. Its aspect keeps circles round.
	aspect := across * shape / down
	right := (2*(float64(x)+0.5)/across - 1) * reach
	up := (1 - 2*(float64(y)+0.5)/down) * reach / aspect
	return Ray{Dir: Vec{right, up, -1}.unit()}
}

// Lens is a focal length on a gauge in front of a pinhole. It decides how
// much the picture sees, and that is all it does: there is no glass, no
// aperture and no focus. The zero Lens is a 50 on Super 35.
type Lens struct {
	focal, pixel float64
	gauge        Gauge
}

// NewLens is a lens of a focal length in millimetres on a gauge. Its
// pixels have a shape, the width over the height. A focal length of
// nought is 50, the zero gauge is Super 35, and a pixel of nought is
// square.
func NewLens(focal float64, gauge Gauge, pixel float64) Lens {
	return Lens{focal, pixel, gauge}
}

// Wide is how much the lens sees from the left edge of the frame to the
// right, in radians. It is the whole gauge width seen through the focal
// length.
func (lens Lens) Wide() float64 {
	focal, gauge := lens.focal, lens.gauge
	if focal <= 0 {
		focal = lensFocal
	}
	if gauge.Width() <= 0 {
		gauge = standard
	}
	return 2 * math.Atan(gauge.Width()/(2*focal))
}

// Ray is the pinhole's ray for the lens's angle.
func (lens Lens) Ray(x, y, width, height int) Ray {
	return NewPinhole(lens.Wide(), lens.pixel).Ray(x, y, width, height)
}

// Swath is the lens on Super 35 that sees a strip of sea so many
// kilometres across from so many kilometres up, were it aimed straight
// down. Where it is aimed is its camera's. Strips of no width and strips
// from no height are Super 35's ordinary lens.
func Swath(across, above float64) Lens {
	if !(across > 0 && above > 0) {
		return Lens{gauge: standard}
	}
	wide := 2 * math.Atan(across/(2*above))
	return Lens{focal: Focal(wide, standard), gauge: standard}
}

// Focal is the focal length on a gauge that sees so many radians across.
// It is Wide undone. The wide is greater than zero and below a half turn.
func Focal(wide float64, gauge Gauge) float64 {
	if gauge.Width() <= 0 {
		gauge = standard
	}
	return gauge.Width() / (2 * math.Tan(wide/2))
}

// Orientation is which way a subspace's axes point, as three arrows.
// They say where its own x, y and z go in the world. They are one unit
// long and at right angles to each other. The zero Orientation is no
// turn at all.
type Orientation struct{ X, Y, Z Vec }

// Identity is no turn. A camera with it looks straight down, since ahead
// is minus z and up in this world is z, with north up the picture.
var Identity = Orientation{Vec{X: 1}, Vec{Y: 1}, Vec{Z: 1}}

// Out takes a direction in the subspace out to the world.
func (turn Orientation) Out(direction Vec) Vec {
	if turn == (Orientation{}) {
		turn = Identity
	}
	return turn.X.times(direction.X).plus(turn.Y.times(direction.Y)).
		plus(turn.Z.times(direction.Z))
}

// Up is the world's up. Slayer's is y; this world's is z, with x east
// and y north.
var Up = Vec{Z: 1}

// LookAt is the orientation of something at from, facing to. Its up is
// kept as near the world's up as it can be. Forward is minus z.
func LookAt(from, to Vec) Orientation { return LookAtUp(from, to, Up) }

// LookAtUp is LookAt with an up of its own. Give it an up of nothing and
// it uses the world's. Looking at where you stand is no turn.
//
// Within about 2.56 degrees of looking straight along up, the top of the
// view is taken as north instead, and if the view is along north too, as
// east. Slayer does the same with its own axes, and so a camera sweeping
// past straight down rolls there, as its does.
func LookAtUp(from, to, up Vec) Orientation {
	back := Vec{from.X - to.X, from.Y - to.Y, from.Z - to.Z}
	if back == (Vec{}) {
		return Identity
	}
	if up == (Vec{}) {
		up = Up
	}
	back, up = back.unit(), up.unit()
	if math.Abs(back.dot(up)) > nearlyAlong {
		up = Vec{Y: 1}
		if math.Abs(back.dot(up)) > nearlyAlong {
			up = Vec{X: 1}
		}
	}
	x := up.cross(back).unit()
	return Orientation{x, back.cross(x), back}
}

// Subspace is where an eye is and which way it faces: where its zero
// sits in the world, in metres, and where its axes point. The empty
// Subspace is at the world's zero, looking straight down.
//
// Slayer's Subspace also has a scale. This one does not.
type Subspace struct {
	Origin      Vec
	Orientation Orientation
}

// Look is the ray a projection makes for a pixel, seen from an eye. It
// is taken out of the eye's space into the world. Nil projections are
// pinholes of 60 degrees.
func Look(eye Subspace, projection Projection, x, y, width, height int) Ray {
	if projection == nil {
		projection = Pinhole{}
	}
	made := projection.Ray(x, y, width, height)
	return Ray{At: eye.Origin.plus(eye.Orientation.Out(made.At)),
		Dir: eye.Orientation.Out(made.Dir).unit()}
}

// Camera is a projection at a subspace, and the picture it makes.
type Camera struct {
	// Eye is where the camera is and which way it faces. Its origin's z
	// is its height above the sea's mean level, in metres: a couple for
	// a camera on a beach, 400,000 for one in orbit.
	Eye Subspace
	// Lens is the projection: a Lens, a Pinhole, or one of your own. Nil
	// is a pinhole of 60 degrees.
	Lens Projection
	// Width and Height are the picture's size in pixels.
	Width, Height int
	// Seconds is the time, for the waves' motion.
	Seconds float64
	// Seed picks the sea. The same seed is the same sea.
	Seed uint64
}
