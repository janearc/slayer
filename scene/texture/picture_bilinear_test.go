package texture

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// grid is a picture of four columns by four rows, made by hand so that
// bilinear sampling can be worked on paper. a pixel at column x and row
// y is 3x red, 3y green and 1+x+y blue, as values out of 255.
//
// every value is 9 or less, under the 10 where the srgb curve is still
// the straight stretch c/12.92 (10/255 is 0.0392, the bend is at
// 0.04045). so a pixel's light is v/255/12.92, and light between pixels
// is light between values.
//
// a place between pixels reads off the three formulas, x and y being
// pixels from the centre of the first.
func grid() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{uint8(3 * x), uint8(3 * y), uint8(1 + x + y), 255})
		}
	}
	return img
}

// linear is the light of a pixel value v, in the straight stretch of the
// srgb curve: v of 255, then divided by 12.92.
func linear(v float64) float64 { return v / 255 / 12.92 }

// toward is the direction at a longitude and a latitude, as the package
// reads them: the longitude is the angle from +X toward +Z.
func toward(lon, lat float64) maths.Vec {
	return maths.V(math.Cos(lat)*math.Cos(lon), math.Sin(lat), math.Cos(lat)*math.Sin(lon))
}

// A picture is read at the place a direction falls on it: u is the
// longitude over a turn, plus a half, v is a half less the latitude
// over pi, and the place in pixels is u*4 - 0.5 across and v*4 - 0.5
// down.
//
// the map is mirrored, as the package says: +Z reads to the right of the
// middle and -Z to the left.
func TestPictureIsReadAtTheWorkedPlace(t *testing.T) {
	pic := Picture{Image: grid()}
	pi := math.Pi
	for _, c := range []struct {
		why  string
		d    maths.Vec
		x, y float64
	}{
		{"u 0.4375, v 0.5625", toward(-pi/8, -pi/16), 1.25, 1.75},
		{"u 0.8125, v 0.4375", toward(5*pi/8, pi/16), 2.75, 1.25},
		{"u 0.6875, v 0.6875", toward(3*pi/8, -3*pi/16), 2.25, 2.25},
		{"+Z, u 0.75", toward(pi/2, 0), 2.5, 1.5},
		{"-Z, u 0.25", toward(-pi/2, 0), 0.5, 1.5},
		{"+X, u 0.5", maths.XAxis, 1.5, 1.5},
		{"a long way out", toward(-pi/8, -pi/16).Scale(40), 1.25, 1.75},
	} {
		want := maths.RGB(linear(3*c.x), linear(3*c.y), linear(1+c.x+c.y))
		if got := pic.At(c.d); !near(got, want) {
			t.Errorf("%s: %v, want %v", c.why, got, want)
		}
	}
}

// The poles are the first and last rows: v of 0 is 0.5 pixels above the
// first row's centre, v of 1 is 0.5 below the last's, and the rows past
// the edge are the edge's own. so the north pole is row 0 all along and
// the south pole is row 3, at u of a half, column 1.5.
func TestPictureEdgeRowsAreHeldAtThePoles(t *testing.T) {
	pic := Picture{Image: grid()}
	north := maths.RGB(linear(4.5), linear(0), linear(1+1.5+0))
	south := maths.RGB(linear(4.5), linear(9), linear(1+1.5+3))
	for _, p := range []maths.Vec{maths.V(0, 1, 0), maths.V(0, 7, 0)} {
		if got := pic.At(p); !near(got, north) {
			t.Errorf("the north pole at %v is %v, want %v", p, got, north)
		}
		if got := pic.At(p.Neg()); !near(got, south) {
			t.Errorf("the south pole at %v is %v, want %v", p.Neg(), got, south)
		}
	}
}

// A direction 1e-160 long has squares in the denormals, where Len is
// off, so its Unit has a Y of 1.0000056 or more, which the test checks
// first. the picture holds the latitude to the pole and not past it,
// where the arcsine would be no number.
func TestPictureHoldsALatitudePastTheirPole(t *testing.T) {
	pic := Picture{Image: grid()}
	if y := maths.V(0, 1e-160, 0).Unit().Y; !(y > 1) {
		t.Fatalf("the direction was meant to measure past 1, and is %v", y)
	}
	north := maths.RGB(linear(4.5), linear(0), linear(1+1.5+0))
	south := maths.RGB(linear(4.5), linear(9), linear(1+1.5+3))
	if got := pic.At(maths.V(0, 1e-160, 0)); !near(got, north) {
		t.Errorf("a hair of north is %v, want %v", got, north)
	}
	if got := pic.At(maths.V(0, -1e-160, 0)); !near(got, south) {
		t.Errorf("a hair of south is %v, want %v", got, south)
	}
}

// The picture wraps at the back, toward -X: u of 1 is u of 0, and x of
// -0.5 is half of the last column and half of the first, at y of 1.5.
//
// red is (9 + 0) / 2, green is the middle of rows 1 and 2, 4.5, and blue
// is column 3's 1 + 3 + 1.5 = 5.5 with column 0's 1 + 0 + 1.5 = 2.5,
// half and half: 4.
func TestPictureWrapsAtTheBack(t *testing.T) {
	pic := Picture{Image: grid()}
	want := maths.RGB(linear(4.5), linear(4.5), linear(4))
	for _, p := range []maths.Vec{maths.V(-1, 0, 0), maths.V(-1, 0, 1e-9), maths.V(-1, 0, -1e-9)} {
		if got := pic.At(p); !near(got, want) {
			t.Errorf("at the back, %v: %v, want %v", p, got, want)
		}
	}
}

// Turn moves the picture's left edge round by an angle that is added to
// the longitude: +X at a turn of a quarter lands on u of 0.75, which is
// x of 2.5, and at three quarters on u of 1.25 which wraps to 0.25, x of
// 0.5. a negative quarter is x of 0.5 as well.
func TestPictureTurnsByAddingToTheLongitude(t *testing.T) {
	for _, c := range []struct {
		turn, x float64
	}{{math.Pi / 2, 2.5}, {-math.Pi / 2, 0.5}, {3 * math.Pi / 2, 0.5}, {0, 1.5}} {
		pic := Picture{Image: grid(), Turn: c.turn}
		want := maths.RGB(linear(3*c.x), linear(4.5), linear(1+c.x+1.5))
		if got := pic.At(maths.XAxis); !near(got, want) {
			t.Errorf("a turn of %v: %v, want %v", c.turn, got, want)
		}
	}
}

// Gain scales the light, and a gain of zero is a gain of one.
func TestPictureGainScalesTheLight(t *testing.T) {
	d := toward(-math.Pi/8, -math.Pi/16)
	plain := maths.RGB(linear(3.75), linear(5.25), linear(4))
	for _, c := range []struct{ gain, by float64 }{{0, 1}, {1, 1}, {2, 2}, {0.5, 0.5}} {
		got := Picture{Image: grid(), Gain: c.gain}.At(d)
		if want := plain.Scale(c.by); !near(got, want) {
			t.Errorf("a gain of %v: %v, want %v", c.gain, got, want)
		}
	}
}

// Water is blue over red by more than 15 per cent and over green by more
// than 5 per cent, both together. the pixels are one by one, with srgb
// values under the 10 where the curve is straight, so the light is in the
// ratio of the values.
func TestPictureWaterIsBlueOverRedAndGreen(t *testing.T) {
	one := func(r, g, b uint8) Picture {
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		img.Set(0, 0, color.RGBA{r, g, b, 255})
		return Picture{Image: img}
	}
	for _, c := range []struct {
		why     string
		r, g, b uint8
		water   bool
	}{
		{"blue well over both", 2, 2, 10, true},
		{"blue over green but not over red (10 against 11.5)", 10, 2, 10, false},
		{"blue over red but not over green (10 against 10.5)", 2, 10, 10, false},
		{"blue over red by more than a seventh, not three times (6 against 3.45 and 9.9)", 3, 2, 6, true},
		{"blue over green by more than a twentieth, not three times (6 against 5.25 and 15.5)", 0, 5, 6, true},
		{"all one grey", 6, 6, 6, false},
	} {
		if got := one(c.r, c.g, c.b).Water(maths.XAxis); got != c.water {
			t.Errorf("%s: %v", c.why, got)
		}
	}
}

// Blue exactly over red by 1.15, or over green by 1.05, is not water: it
// takes more. the 16 bit pixels are picked so that the light comes out
// as exactly that in floating point, red 100 and blue 115, green 100 and
// blue 105, each over 65535; the test checks that first.
func TestPictureWaterTakesMoreThanTheEdge(t *testing.T) {
	one := func(r, g, b uint16) Picture {
		img := image.NewRGBA64(image.Rect(0, 0, 1, 1))
		img.SetRGBA64(0, 0, color.RGBA64{r, g, b, 65535})
		return Picture{Image: img}
	}
	red := one(100, 0, 115)
	if c := red.At(maths.XAxis); c.B != c.R*1.15 {
		t.Fatalf("the red pixel was meant to sit on the edge, and is %v", c)
	}
	if red.Water(maths.XAxis) {
		t.Error("blue exactly 1.15 over red is water")
	}
	green := one(0, 100, 105)
	if c := green.At(maths.XAxis); c.B != c.G*1.05 {
		t.Fatalf("the green pixel was meant to sit on the edge, and is %v", c)
	}
	if green.Water(maths.XAxis) {
		t.Error("blue exactly 1.05 over green is water")
	}
	if !one(100, 0, 116).Water(maths.XAxis) || !one(0, 100, 106).Water(maths.XAxis) {
		t.Error("a step over the edge is not water")
	}
}
