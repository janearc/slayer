package film

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Gauge is a film's frame: how wide and how tall the exposed picture is,
// in millimetres. With a focal length it says how much a lens sees, the
// way a cinematographer says it: on Super 35 a 32 or so is normal, a 50
// a little long, a 24 wide and a 200 long.
type Gauge struct {
	Name          string
	Width, Height float64
	// Grain is how coarse the film's grain is, as a share of the
	// frame's width: the same emulsion grows no finer on a bigger
	// frame, so a small gauge is grainy and a big one is clean. Nought
	// is no grain at all, as a detector has.
	Grain float64
}

// Aspect is how wide the frame is against how tall.
func (g Gauge) Aspect() float64 {
	if g.Height <= 0 {
		return 1
	}
	return g.Width / g.Height
}

// The gauges. The grain is the same emulsion's on each, a share of the
// frame's width that works out to a clump of 46 to 47 microns on most:
// Super 8's 0.008 of 5.79 mm is 46.3. IMAX's 0.0007 of 70.41 mm is the
// coarsest, at 49.3.
//
// the clump is a choice made early and kept, a look and not a
// measurement: no stock's published grain is behind it.
var (
	// Super8 is home movies: a frame under six millimetres across.
	Super8 = Gauge{"super 8", 5.79, 4.01, 0.008}
	// Sixteen is 16 mm, the documentary's and the student's gauge.
	Sixteen = Gauge{"16 mm", 10.26, 7.49, 0.0045}
	// Academy is 35 mm film as the talkies shot it, four perforations.
	Academy = Gauge{"academy", 22.0, 16.0, 0.0021}
	// Super35 is the whole width of 35 mm film, which most cinema and
	// most digital cinema cameras match.
	Super35 = Gauge{"super 35", 24.89, 18.66, 0.0019}
	// FullFrame is a 35 mm still frame, run sideways: nearly twice
	// Super 35's area.
	FullFrame = Gauge{"full frame", 36.0, 24.0, 0.0013}
	// Film65 is 65 mm, five perforations, the epic's gauge before imax.
	Film65 = Gauge{"65 mm", 52.48, 23.01, 0.0009}
	// IMAX is 70 mm film run sideways, fifteen perforations a frame.
	IMAX = Gauge{"imax", 70.41, 52.63, 0.0007}
	// WFC3 is hubble's wide field camera's visible detector: two chips
	// of 4096 by 2051 pixels of fifteen microns, side by side.
	WFC3 = Gauge{"hubble wfc3", 61.4, 61.5, 0}
)

// Hubble is the space telescope as a lens: a mirror of 2.4 metres at a
// focal length of 57.6, on WFC3. It is a pinhole at that angle; what the
// mirror's size does, the blur of diffraction, is not modelled.
//
// The field here is the gauge seen through the bare telescope, about 220
// arcseconds. The real one is 162 arcseconds on a side: WFC3's relay
// optics lengthen the focal length, and the handbook gives 0.0395
// arcseconds a pixel over 4096 pixels (WFC3 Instrument Handbook 2.1, 6.4).
//
// So smoke 18's jupiter is about a quarter small. The number stays until
// the picture gate exists, so that the change is seen when it is made.
func Hubble() Lens { return Lens{Focal: 57600, Gauge: WFC3} }

// Lens is a focal length on a gauge, in front of a pinhole: the lens
// decides how much the picture sees, and for now that is all it does.
// Glass, what bends the edges and splits the bands, comes when a picture
// asks. The zero Lens is a 50 on Super 35.
type Lens struct {
	// Focal is the focal length in millimetres. Zero means 50.
	Focal float64
	// Gauge is the frame. The zero gauge is Super 35.
	Gauge Gauge
	// Pixel is one pixel's width over its height; zero means square.
	Pixel float64
}

// focal is the focal length, with zero read as 50.
func (l Lens) focal() float64 {
	if l.Focal <= 0 {
		return 50
	}
	return l.Focal
}

// gauge is the gauge, with the zero one read as Super 35.
func (l Lens) gauge() Gauge {
	if l.Gauge.Width <= 0 {
		return Super35
	}
	return l.Gauge
}

// Wide is how much the lens sees from the left edge of the frame to the
// right, in radians: the whole gauge width seen through the focal length.
func (l Lens) Wide() float64 {
	return 2 * math.Atan(l.gauge().Width/(2*l.focal()))
}

// Ray is the pinhole's ray for the lens's angle. A pinhole bends every
// band alike, and so, for now, does the lens.
func (l Lens) Ray(x, y, w, h int, band maths.Band) maths.Ray {
	return Pinhole{Wide: l.Wide(), Pixel: l.Pixel}.Ray(x, y, w, h, band)
}

// Swath is the lens, on super 35, that sees a strip of ground across
// kilometres wide from above kilometres up, were it aimed straight down;
// where it is aimed is its camera's. A camera named by what it covers, as
// reconnaissance cameras are.
//
// a strip of no width, or one from no height, is super 35's ordinary lens.
//
// It is not any real one. Corona, the KH-4B of 1967 to 1972, flew a
// 610 mm Petzval on 70 mm film from about 120 to 160 km, and its pictures
// were declassified in 1995; nothing about it is secret. Nothing in slayer
// calls it yet; its test frames forty kilometres from four hundred.
func Swath(across, above float64) Lens {
	if !(across > 0 && above > 0) {
		return Lens{Gauge: Super35}
	}
	wide := 2 * math.Atan(across/(2*above))
	return Lens{Focal: Focal(wide, Super35), Gauge: Super35}
}

// Focal is the focal length on a gauge that sees wide radians across: Wide,
// undone, for a shot framed by what it must hold. wide is above zero and below
// a half turn; outside that it is no focal length.
func Focal(wide float64, g Gauge) float64 {
	if g.Width <= 0 {
		g = Super35
	}
	return g.Width / (2 * math.Tan(wide/2))
}
