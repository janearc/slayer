package planet

import (
	"math"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// even is starlight of one watt a square metre a nanometre from 400 to
// 700 nm.
func even() bands.Light {
	spectrum := map[int]float64{}
	for nm := 400; nm <= 700; nm++ {
		spectrum[nm] = 1
	}
	return bands.Split(spectrum)
}

// bareBody is a body with no air over a deck of an albedo, one zone, or
// two split at the equator with a layer over the north.
func bareBody(t *testing.T, deck float64, north []Layer) Body {
	t.Helper()
	whole, err := NewZone(-90, 90, nil)
	zones := []Zone{whole}
	if north != nil {
		south, _ := NewZone(-90, 0, nil)
		top, _ := NewZone(0, 90, north)
		zones = []Zone{south, top}
	}
	if err != nil {
		t.Fatal(err)
	}
	body, err := NewBody(6e7, 25, deck, zones)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// seen is light in bands as the tracer's three channels.
func seen(light bands.Light) maths.Light {
	red, green, blue := srgb.Light(observer.Swatch(light))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}

// close is whether two lights agree in every channel to a share.
func close(got, want maths.Light, share float64) bool {
	return near(got.R, want.R, share) && near(got.G, want.G, share) &&
		near(got.B, want.B, share)
}

// a bare white deck under a star overhead, seen from overhead, sends
// back the star's light over pi: a cosine-law white.
func TestABareWhiteFaceIsACosineReflector(t *testing.T) {
	body := bareBody(t, 1, nil)
	up := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), up, up.Scale(1e9), even(), 2)
	got := face.At(up.Scale(6e7))
	want := seen(even().Scale(2 / math.Pi))
	if !close(got, want, 1e-6) {
		t.Errorf("%+v, want %+v", got, want)
	}
}

// the night side sends nothing toward the eye, and neither does a point
// the eye cannot see.
func TestTheNightSideIsDark(t *testing.T) {
	body := bareBody(t, 1, nil)
	up := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), up, up.Scale(1e9), even(), 1)
	for _, p := range []maths.Vec{up.Scale(-6e7), maths.V(6e7, 0, -1)} {
		if got := face.At(p); got != (maths.Light{}) {
			t.Errorf("at %v: %+v", p, got)
		}
	}
}

// a zone under an absorbing layer is darker than one with none, by its
// depth both ways.
func TestAZonesAirDimsIt(t *testing.T) {
	dark := mustGas(t, "dark", flat(0), 0.02).Absorbing(flat(1e-30))
	layer, err := NewLayer(0, 1e3, []Gas{dark}, []float64{1}, Haze{})
	if err != nil {
		t.Fatal(err)
	}
	body := bareBody(t, 1, []Layer{layer})
	depth := layer.optics(550, 25).absorb
	star := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), star, star.Scale(1e12), even(),
		1)
	tilt := 10 * maths.Degree
	north := maths.V(0, math.Sin(tilt), math.Cos(tilt)).Scale(6e7)
	south := maths.V(0, -math.Sin(tilt), math.Cos(tilt)).Scale(6e7)
	slant := 2 / math.Cos(tilt)
	got := face.At(north).G / face.At(south).G
	if want := math.Exp(-depth * slant); !near(got, want, 1e-3) {
		t.Errorf("north over south %g, want %g", got, want)
	}
}

// zones that leave a gap, overlap, or stop short of a pole are refused.
func TestZonesMustCoverThePlanet(t *testing.T) {
	if _, err := NewZone(10, 10, nil); err == nil {
		t.Error("a zone of no width")
	}
	if _, err := NewZone(-91, 0, nil); err == nil {
		t.Error("a zone past the pole")
	}
	south, _ := NewZone(-90, 0, nil)
	north, _ := NewZone(10, 90, nil)
	short, _ := NewZone(0, 80, nil)
	for _, zones := range [][]Zone{{south, north}, {south, short}, {north}} {
		if _, err := NewBody(6e7, 25, 1, zones); err == nil {
			t.Errorf("zones %+v", zones)
		}
	}
}

// layerOf is a layer of one gas between two pressures, or the test
// stops.
func layerOf(t *testing.T, gas Gas, top, bottom float64, haze Haze) Layer {
	t.Helper()
	layer, err := NewLayer(top, bottom, []Gas{gas}, []float64{1}, haze)
	if err != nil {
		t.Fatal(err)
	}
	return layer
}

// a column's layers must run down: each starts at or below where the
// one above it ends, and one starting higher is refused, wherever it
// stands in the column.
func TestAColumnRunsDown(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	one := layerOf(t, gas, 0, 100, Haze{})
	two := layerOf(t, gas, 100, 200, Haze{})
	three := layerOf(t, gas, 200, 300, Haze{})
	early := layerOf(t, gas, 150, 300, Haze{})
	if _, err := NewZone(-90, 90, []Layer{one, two, three}); err != nil {
		t.Errorf("touching layers refused: %v", err)
	}
	for _, column := range [][]Layer{{one, two, early}, {two, one},
		{one, three, two}} {
		if _, err := NewZone(-90, 90, column); err == nil {
			t.Errorf("a column running up was taken")
		}
	}
	if _, err := NewZone(0, 90.5, nil); err == nil {
		t.Error("a zone past the north pole")
	}
}

// a body's numbers at their edges are taken, and past them refused.
func TestABodysEdges(t *testing.T) {
	whole, _ := NewZone(-90, 90, nil)
	zones := []Zone{whole}
	for _, fine := range [][3]float64{{0.5, 0.5, 0}, {1, 1, 1}} {
		if _, err := NewBody(fine[0], fine[1], fine[2], zones); err != nil {
			t.Errorf("%v refused: %v", fine, err)
		}
	}
	for _, wrong := range [][3]float64{{0, 1, 0.5}, {1, 0, 0.5},
		{1, 1, -0.1}, {1, 1, 1.1}} {
		if _, err := NewBody(wrong[0], wrong[1], wrong[2],
			zones); err == nil {
			t.Errorf("%v taken", wrong)
		}
	}
	if _, err := NewHaze(0, 0, 0); err != nil {
		t.Errorf("a clear haze refused: %v", err)
	}
	if _, err := NewHaze(0.5, 1, 0.9); err != nil {
		t.Errorf("a white haze refused: %v", err)
	}
}

// a latitude on a zone's edge belongs to the zone north of it, the pole
// to the last zone, and latitudes are degrees.
func TestLatitudesFindTheirZones(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	dark := layerOf(t, gas.Absorbing(flat(1e-28)), 0, 1e4, Haze{})
	south, _ := NewZone(-90, 30, nil)
	north, _ := NewZone(30, 90, []Layer{dark})
	body, err := NewBody(1, 1, 1, []Zone{south, north})
	if err != nil {
		t.Fatal(err)
	}
	face := NewFace(body, maths.V(0, 1, 0), maths.V(0, 1, 0),
		maths.V(0, 1e6, 0), even(), 1)
	if face.zoneOf(30) != 1 || face.zoneOf(90) != 1 || face.zoneOf(20) != 0 {
		t.Errorf("zones %d %d %d", face.zoneOf(30), face.zoneOf(90),
			face.zoneOf(20))
	}
	at := 20 * maths.Degree
	lit := face.At(maths.V(math.Cos(at), math.Sin(at), 0))
	if !(lit.G > 0) {
		t.Errorf("20 degrees north fell in the dark zone: %+v", lit)
	}
}

// at the terminator and on the night side nothing is sent, even where
// air would scatter light that came from below the horizon.
func TestNoLightFromBelowTheHorizon(t *testing.T) {
	gas := mustGas(t, "gas", flat(1e-31), 0.02)
	whole, _ := NewZone(-90, 90, []Layer{layerOf(t, gas, 0, 1e5, Haze{})})
	body, _ := NewBody(1, 1, 0.5, []Zone{whole})
	up := maths.V(0, 0, 1)
	for _, star := range []maths.Vec{maths.V(1, 0, 0),
		maths.V(1, 0, -0.5), maths.V(0, 0, -1)} {
		face := NewFace(body, maths.V(0, 1, 0), star, up.Scale(10), even(), 1)
		if got := face.At(up); got != (maths.Light{}) {
			t.Errorf("star %v: %+v", star, got)
		}
	}
	face := NewFace(body, maths.V(0, 1, 0), up, maths.V(10, 0, 0), even(), 1)
	if got := face.At(up); got != (maths.Light{}) {
		t.Errorf("seen edge on: %+v", got)
	}
}

// a haze leaning forward is brighter lit from behind than from in front,
// by the turn between the star and the eye.
func TestHazeIsBrightAgainstTheStar(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	haze, _ := NewHaze(0.05, 1, 0.8)
	whole, _ := NewZone(-90, 90, []Layer{layerOf(t, gas, 0, 1, haze)})
	body, _ := NewBody(1, 1, 0, []Zone{whole})
	tilt := 60 * maths.Degree
	behind := maths.V(math.Sin(tilt), 0, math.Cos(tilt))
	eye := maths.V(-math.Sin(tilt), 0, math.Cos(tilt)).Scale(1e9)
	up := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), behind, eye, even(), 1)
	got := face.At(up).G
	stack := []optics{{haze: 0.05, hazeAlbedo: 1, hazeLean: 0.8}}
	turn := -behind.Dot(eye.Sub(up).Unit())
	want := seen(even().Scale(reflected(stack, 0, up.Dot(behind),
		up.Dot(eye.Sub(up).Unit()), turn))).G
	if !near(got, want, 1e-6) {
		t.Errorf("%g, want %g", got, want)
	}
}

// a dot an ulp past one, either way, is still a pole, not no number;
// and a latitude is in degrees.
func TestLatitudesAreHeldAtThePoles(t *testing.T) {
	pole := maths.V(1, 0, 0)
	past := 1 + 1e-15
	if got := latitudeOf(maths.V(past, 0, 0), pole); got != 90 {
		t.Errorf("past the north pole: %g", got)
	}
	if got := latitudeOf(maths.V(-past, 0, 0), pole); got != -90 {
		t.Errorf("past the south pole: %g", got)
	}
	at := 40 * maths.Degree
	if got := latitudeOf(maths.V(math.Cos(at), math.Sin(at), 0),
		maths.V(0, 1, 0)); !near(got, 40, 1e-12) {
		t.Errorf("40 degrees read as %g", got)
	}
}

// an eye exactly on a point's horizon sees nothing of it.
func TestAnEyeOnTheHorizonSeesNothing(t *testing.T) {
	gas := mustGas(t, "gas", flat(1e-31), 0.02)
	whole, _ := NewZone(-90, 90, []Layer{layerOf(t, gas, 0, 1e5, Haze{})})
	body, _ := NewBody(1, 1, 0.5, []Zone{whole})
	up := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), up, maths.V(10, 0, 1), even(), 1)
	if got := face.At(up); got != (maths.Light{}) {
		t.Errorf("%+v", got)
	}
}
