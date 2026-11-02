package film

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
)

// the gauges the defaults and the telescope stand on, by name in the
// table
const (
	// plain is the gauge a lens is on when none is given.
	plain = "super 35"
	// telescope is the gauge the space telescope's camera has.
	telescope = "hubble wfc3"
)

var (
	// lensFocal is the focal length of a lens when none is given.
	lensFocal = data.Picked("lens-focal-millimetres")
	// hubbleFocal is the focal length of the space telescope's lens.
	hubbleFocal = data.Picked("hubble-focal-millimetres")
	// standard and wfc3 are the two gauges Lens leans on. Tables without
	// them are mistakes in the code. They stop the program.
	standard, wfc3 = Shipped().Must(plain), Shipped().Must(telescope)
)

// Lens is a focal length on a gauge in front of a pinhole. It decides
// how much the picture sees. For now that is all it does. Glass bends
// the edges and splits the bands. It comes when a picture asks. The zero
// Lens is a 50 on Super 35.
//
// Its focal is the focal length in millimetres. Nought means 50. Its
// gauge is the frame. The zero gauge is Super 35. Its pixel is one
// pixel's width over its height. Nought means square.
//
//ninja:vow practice slayer's camera holds one; tests so far
type Lens struct {
	focal, pixel float64
	gauge        Gauge
}

// NewLens is a lens of a focal length in millimetres on a gauge. Its
// pixels have a shape, the width over the height.
//
//ninja:vow practice slayer's camera makes its lens here; tests so far
func NewLens(focal float64, gauge Gauge, pixel float64) Lens {
	return Lens{focal, pixel, gauge}
}

// Hubble is the space telescope as a lens. Its mirror is 2.4 metres
// across and its focal length is 57.6 metres on WFC3. It is a pinhole at
// that angle. The blur of diffraction is what the size of the mirror
// does. It is not modelled.
//
// The field here is the gauge seen through the bare telescope. That is
// about 220 arcseconds. The real one is 162 arcseconds on a side. The
// relay optics of WFC3 lengthen the focal length.
//
// The handbook gives 0.0395 arcseconds a pixel over 4096 pixels (WFC3
// Instrument Handbook 2.1 and 6.4).
//
// So smoke 18's jupiter is about a quarter small. The number stays until
// the picture gate exists, so that the change is seen when it is made.
//
//ninja:vow practice slayer's smoke 18 films the telescope here
func Hubble() Lens { return Lens{focal: hubbleFocal, gauge: wfc3} }

// length is the focal length, with zero and less read as the default.
func (lens Lens) length() float64 {
	if lens.focal <= none {
		return lensFocal
	}
	return lens.focal
}

// frame is the gauge, with the zero one read as Super 35.
func (lens Lens) frame() Gauge {
	if lens.gauge.Width() <= none {
		return standard
	}
	return lens.gauge
}

// Wide is how much the lens sees from the left edge of the frame to the
// right, in radians. It is the whole gauge width seen through the focal
// length.
//
//ninja:vow practice slayer's camera reads its field of view here
func (lens Lens) Wide() float64 {
	return pair * math.Atan(lens.frame().Width()/(pair*lens.length()))
}

// Ray is the pinhole's ray for the lens's angle. Pinholes bend every
// band alike. For now so does the lens.
func (lens Lens) Ray(x, y, width, height int, band maths.Band) maths.Ray {
	seen := NewPinhole(lens.Wide(), lens.pixel)
	return seen.Ray(x, y, width, height, band)
}

// Swath is the lens on super 35 that sees a strip of ground across so
// many kilometres wide from above so many kilometres up. It sees it were
// it aimed straight down. Where it is aimed is its camera's. Cameras
// named by what they cover are what reconnaissance flew.
//
// Strips of no width and strips from no height are super 35's ordinary
// lens.
//
// It is not any real one. Corona flew a 610 mm Petzval on 70 mm film
// from about 120 to 160 km. That was the KH-4B of 1967 to 1972. Its
// pictures were declassified in 1995. Nothing about it is secret.
// Nothing in slayer calls it yet.
//
//ninja:vow practice slayer's reconnaissance camera makes its lens here
func Swath(across, above float64) Lens {
	if !(across > none && above > none) {
		return Lens{gauge: standard}
	}
	wide := pair * math.Atan(across/(pair*above))
	return Lens{focal: Focal(wide, standard), gauge: standard}
}

// Focal is the focal length on a gauge that sees so many radians
// across. It is Wide undone. It serves a shot framed by what it must
// hold. The wide is greater than zero and below a half turn. Outside that
// there is no focal length.
//
//ninja:vow practice slayer's reconnaissance camera frames its shot here
func Focal(wide float64, gauge Gauge) float64 {
	if gauge.Width() <= none {
		gauge = standard
	}
	return gauge.Width() / (pair * math.Tan(wide/pair))
}
