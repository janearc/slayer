package film_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"pgregory.net/rapid"
)

// the tables the lens tests are run over
const (
	knownLenses = "testdata/lenses-known.csv"
	knownSwaths = "testdata/swaths-known.csv"
	plainSwaths = "testdata/swaths-plain.csv"
)

// the gauges the lens tests name
const (
	standardName  = "super 35"
	telescopeName = "hubble wfc3"
)

// degrees is a number of radians as degrees.
func degrees(radians float64) float64 { return radians / maths.Degree }

// gauged is a gauge of the table that ships, or the test stops.
func gauged(t stopper, name string) film.Gauge {
	t.Helper()
	made, err := film.Shipped().Named(name)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// stopper is a test that can be stopped: a testing.T or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// gaugeNames are the gauges of the table that ships.
func gaugeNames() []string {
	return []string{"super 8", "16 mm", "academy", "super 35",
		"full frame", "65 mm", "imax", "hubble wfc3"}
}

// lens draws a lens of a focal length. It has a gauge and a pixel's shape.
func lens(t *rapid.T) film.Lens {
	length := rapid.Float64Range(focalLeast, focalMost).Draw(t, "focal")
	name := rapid.SampledFrom(gaugeNames()).Draw(t, "gauge")
	shape := rapid.Float64Range(pixelLeast, pixelMost).Draw(t, "pixel")
	return film.NewLens(length, gauged(t, name), shape)
}

// Lenses see the angles that are known of them. The zero lens is a 50 on
// Super 35. That is about 28 degrees. Twenty four is wide and two
// hundred is long. The angle follows the gauge.
func TestLensesSeeTheAnglesKnownOfThem(t *testing.T) {
	for _, row := range table(t, knownLenses) {
		name, _ := row.Text("gauge")
		made := film.NewLens(number(t, row, "focal"), gauged(t, name), none)
		got, want := degrees(made.Wide()), number(t, row, "degrees")
		if within := number(t, row, "within"); math.Abs(got-want) > within {
			label, _ := row.Text("label")
			t.Errorf("%s sees %.3f degrees, want %v", label, got, want)
		}
	}
	var zero film.Lens
	length := data.Picked("lens-focal-millimetres")
	fifty := film.NewLens(length, gauged(t, standardName), none)
	if zero.Wide() != fifty.Wide() {
		t.Errorf("the zero lens sees %v, a 50 on super 35 sees %v",
			zero.Wide(), fifty.Wide())
	}
}

// The telescope as a lens sees about 220 arcseconds. So does the bare
// telescope. That is wider than the real 162 of WFC3.
func TestTheTelescopeSeesWhatItDid(t *testing.T) {
	seen := degrees(film.Hubble().Wide())
	if math.Abs(seen-hubbleSeen) > hubbleWithin {
		t.Errorf("hubble sees %v degrees, want %v", seen, hubbleSeen)
	}
	want := film.NewLens(data.Picked("hubble-focal-millimetres"),
		gauged(t, telescopeName), none)
	if film.Hubble() != want {
		t.Errorf("hubble is %+v, want %+v", film.Hubble(), want)
	}
}

// Focal undoes Wide: the focal length that sees what a lens sees is the
// lens's own.
func TestFocalUndoesWide(t *testing.T) {
	rapid.Check(t, focalUndoesWide)
}

func focalUndoesWide(t *rapid.T) {
	length := rapid.Float64Range(focalLeast, focalMost).Draw(t, "focal")
	name := rapid.SampledFrom(gaugeNames()).Draw(t, "gauge")
	frame := gauged(t, name)
	seen := film.NewLens(length, frame, none).Wide()
	if got := film.Focal(seen, frame); math.Abs(got-length) > loose*length {
		t.Fatalf("a %v on %s came back as %v", length, name, got)
	}
}

// Lens rays start where the pinhole's do. They are one unit long.
func TestALensRaysArePinholeRays(t *testing.T) {
	rapid.Check(t, lensRaysArePinholeRays)
}

func lensRaysArePinholeRays(t *rapid.T) {
	glass := lens(t)
	across, down := picture(t)
	x := rapid.IntRange(none, across-whole).Draw(t, "x")
	y := rapid.IntRange(none, down-whole).Draw(t, "y")
	band := maths.Band{}
	got := glass.Ray(x, y, across, down, band)
	if length := got.Dir.Len(); math.Abs(length-whole) > tight {
		t.Fatalf("the lens's ray is %v long", length)
	}
	flat := film.NewPinhole(glass.Wide(), none)
	if wide := flat.Ray(x, y, across, down, band); wide.At != got.At {
		t.Fatalf("the lens's ray starts at %v, not %v", got.At, wide.At)
	}
}

// Lenses see more through longer gauges and less through longer focal
// lengths.
func TestLongerGaugesSeeMoreAndLongerLensesSeeLess(t *testing.T) {
	rapid.Check(t, anglesFollowGaugeAndFocal)
}

func anglesFollowGaugeAndFocal(t *rapid.T) {
	short := rapid.Float64Range(focalLeast, focalMost).Draw(t, "short")
	long := short * (whole + rapid.Float64Range(wideLeast,
		wideMost).Draw(t, "stretch"))
	frame := gauged(t, rapid.SampledFrom(gaugeNames()).Draw(t, "gauge"))
	near := film.NewLens(short, frame, none).Wide()
	far := film.NewLens(long, frame, none).Wide()
	if !(far < near) {
		t.Fatalf("a %v sees %v and a %v sees %v on %v", short, near, long,
			far, frame)
	}
}

// Focal lengths below a millimetre are that lens and not the default. So are
// gauges so small. Only zero and less are read as unset.
func TestASmallLensIsItself(t *testing.T) {
	small := film.NewLens(half, gauged(t, standardName), none)
	standard := gauged(t, standardName)
	if got, want := small.Wide(), pair*math.Atan(standard.Width()/(pair*half)); got !=
		want {
		t.Errorf("a half millimetre lens sees %v, want %v", got, want)
	}
	if film.Focal(whole, film.Gauge{}) != film.Focal(whole, standard) {
		t.Error("the zero gauge is not super 35")
	}
	tiny, err := film.NewGauge("a tiny one", half, half, none)
	if err != nil {
		t.Fatal(err)
	}
	want := tiny.Width() / (pair * math.Tan(half))
	if got := film.Focal(whole, tiny); got != want {
		t.Errorf("a half millimetre gauge's focal length is %v", got)
	}
	glass := film.NewLens(none, tiny, none)
	length := data.Picked("lens-focal-millimetres")
	if got, want := glass.Wide(), pair*math.Atan(tiny.Width()/(pair*length)); got != want {
		t.Errorf("a half millimetre gauge sees %v, want %v", got, want)
	}
}

// Swaths see what they cover. Forty kilometres from four hundred up is
// five and three quarter degrees.
func TestASwathSeesWhatItCovers(t *testing.T) {
	rapid.Check(t, swathSeesWhatItCovers)
}

func swathSeesWhatItCovers(t *rapid.T) {
	above := rapid.Float64Range(wideLeast, eyeMost).Draw(t, "above")
	across := rapid.Float64Range(wideLeast, eyeMost).Draw(t, "across")
	got := film.Swath(across, above).Wide()
	want := pair * math.Atan(across/(pair*above))
	if math.Abs(got-want) > loose*want {
		t.Fatalf("a swath %v across from %v sees %v, want %v", across,
			above, got, want)
	}
}

// Swaths see the angles known of them.
func TestASwathSeesTheAngleKnownOfIt(t *testing.T) {
	for _, row := range table(t, knownSwaths) {
		got := degrees(film.Swath(number(t, row, "across"),
			number(t, row, "above")).Wide())
		if math.Abs(got-number(t, row, "degrees")) > number(t, row, "within") {
			label, _ := row.Text("label")
			t.Errorf("%s sees %v degrees", label, got)
		}
	}
}

// anti-tests: strips of no width and strips from no height are Super 35's
// ordinary lens. They are not lenses of no angle or of a bent one.
func TestASwathOfNothingIsAPlainLens(t *testing.T) {
	plain := film.NewLens(none, gauged(t, standardName), none)
	for _, row := range table(t, plainSwaths) {
		got := film.Swath(number(t, row, "across"), number(t, row, "above"))
		if got != plain {
			label, _ := row.Text("label")
			t.Errorf("%s is %+v", label, got)
		}
	}
	for _, strip := range [][]float64{{half, eyeMost}, {eyeMost, half}} {
		if film.Swath(strip[none], strip[whole]) == plain {
			t.Errorf("a swath of %v is the plain lens", strip)
		}
	}
}
