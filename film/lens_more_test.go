package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// each gauge as published, its aspect worked from it, and its grain a
// clump of 46 to 49.3 microns, as the gauges' comment says. that holds
// the gauges to one another, as they were first chosen; it is not physics.
func TestTheGauges(t *testing.T) {
	for _, want := range []struct {
		g             Gauge
		width, height float64
		grain         float64
	}{
		{Super8, 5.79, 4.01, 0.008}, {Sixteen, 10.26, 7.49, 0.0045},
		{Academy, 22.0, 16.0, 0.0021}, {Super35, 24.89, 18.66, 0.0019},
		{FullFrame, 36.0, 24.0, 0.0013}, {Film65, 52.48, 23.01, 0.0009},
		{IMAX, 70.41, 52.63, 0.0007}, {WFC3, 61.4, 61.5, 0},
	} {
		g := want.g
		if g.Width != want.width || g.Height != want.height ||
			g.Grain != want.grain || g.Aspect() != want.width/want.height {
			t.Errorf("%s is %+v, aspect %v", g.Name, g, g.Aspect())
		}
		if clump := g.Grain * g.Width * 1000; g.Grain != 0 &&
			(clump < 46 || clump > 49.3) {
			t.Errorf("%s's grain is a clump of %v microns", g.Name, clump)
		}
	}
	if FullFrame.Aspect() != 1.5 {
		t.Errorf("full frame's aspect is %v", FullFrame.Aspect())
	}
	for _, g := range []Gauge{{Width: 3}, {Width: 3, Height: -1}} {
		if g.Aspect() != 1 {
			t.Errorf("%+v has an aspect of %v", g, g.Aspect())
		}
	}
	if (Gauge{Width: 3, Height: 0.5}).Aspect() != 6 {
		t.Error("a frame three wide and a half tall is not six to one")
	}
}

// hubble is a focal length of 57,600 mm on WFC3.
func TestHubblesFocalLength(t *testing.T) {
	if h := Hubble(); h.Focal != 57600 || h.Gauge != WFC3 {
		t.Errorf("hubble is %+v", h)
	}
}

// a lens of a focal length or a gauge under a millimetre is that lens, not the
// default: only zero and less are read as unset.
func TestASmallLensIsItself(t *testing.T) {
	small := Lens{Focal: 0.5}
	if got, want := small.Wide(), 2*math.Atan(24.89/(2*0.5)); got != want {
		t.Errorf("a half millimetre lens sees %v, want %v", got, want)
	}
	tiny := Lens{Gauge: Gauge{Width: 0.5, Height: 0.4}}
	if got, want := tiny.Wide(), 2*math.Atan(0.5/(2*50)); got != want {
		t.Errorf("a half millimetre gauge sees %v, want %v", got, want)
	}
	if Focal(1, Gauge{}) != Focal(1, Super35) {
		t.Error("the zero gauge is not super 35")
	}
	if got, want := Focal(1, Gauge{Width: 0.5}), 0.5/(2*math.Tan(0.5)); got !=
		want {
		t.Errorf("a half millimetre gauge's focal length is %v", got)
	}
	for _, strip := range [][2]float64{{0.5, 400}, {40, 0.5}} {
		if Swath(strip[0], strip[1]) == (Lens{Gauge: Super35}) {
			t.Errorf("a swath %v across from %v is the plain lens", strip[0],
				strip[1])
		}
	}
}

// a pinhole of no pixel shape is square; a picture one pixel wide is a
// picture, its rows looking up and down; a field of half a radian is that
// field, not the default.
func TestAPinholesSmallCases(t *testing.T) {
	if (Pinhole{}).Ray(3, 1, 8, 4, maths.Band{}) != (Pinhole{Pixel: 1}).Ray(3,
		1, 8, 4, maths.Band{}) {
		t.Error("no pixel shape is not square")
	}
	if r := (Pinhole{}).Ray(0, 0, 1, 5, maths.Band{}); r.Dir.Y <= 0 {
		t.Errorf("the top of a picture one pixel wide looks %v", r.Dir)
	}
	if (Pinhole{Wide: 0.5}).Ray(0, 0, 8, 4, maths.Band{}) ==
		(Pinhole{}).Ray(0, 0, 8, 4, maths.Band{}) {
		t.Error("a field of half a radian is the default's")
	}
}
