package main

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/hud"
)

// zeroViewer restores the unconstructed, unlocked global between serial tests.
// No Cocoa window or event loop runs in these tests.
func zeroViewer[T any](value *T) { var empty T; *value = empty }

// viewerFixture gives the UI the same valid initial controls as main.
func viewerFixture(t *testing.T) {
	t.Helper()
	zeroViewer(&view)
	hudThemes = nil
	t.Cleanup(func() { zeroViewer(&view); hudThemes = nil })
	view.fov, view.placeFov, view.distance = 60, 60, 3.2
	view.width, view.lightness, view.hudSize = 800, .75, 2
	view.earth.marked = -1
	view.clock = time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	sheet := css.New()
	sheet.Set("ink", srgb.MustHex("#ffffff").Swatch())
	sheet.Set("ground", srgb.MustHex("#101010").Swatch())
	hudThemes = []namedTheme{{name: "test", theme: hud.NewTheme(sheet)}}
}

// closeFloat compares geometry against independently computed coordinates.
func closeFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-12*math.Max(1, math.Abs(want)) {
		t.Fatalf("got %.16g, want %.16g", got, want)
	}
}

func TestLongitudeLatitudeRoundTripsAcrossPolesAndSeam(t *testing.T) {
	for _, at := range [][2]float64{{0, 0}, {90, 0}, {-90, 0}, {179.99, 45}, {-179.99, -45}, {12, 89.99}, {-150, -89.99}} {
		x, y, z := unit(at[0], at[1])
		closeFloat(t, x*x+y*y+z*z, 1)
		lon, lat := lonLat([3]float64{x, y, z})
		closeFloat(t, lon, at[0])
		closeFloat(t, lat, at[1])
	}
	x, y, z := unit(90, 0)
	closeFloat(t, x, 0)
	closeFloat(t, y, 1)
	closeFloat(t, z, 0)
}

func TestSphereSubdivisionsPreserveUnitCornersAndFaceCount(t *testing.T) {
	for depth := 0; depth <= 2; depth++ {
		cells := cellsAt(depth)
		if len(cells) != 20*(1<<(2*depth)) {
			t.Fatalf("depth %d has %d faces", depth, len(cells))
		}
		seen := map[uint64]bool{}
		for _, cell := range cells {
			if seen[cell.Key()] || cell.Depth() != depth {
				t.Fatalf("bad cell %d", cell.Key())
			}
			seen[cell.Key()] = true
			corners := appendCorners([]float32{42}, cell)
			if len(corners) != 10 || corners[0] != 42 {
				t.Fatal("append lost prefix or corners")
			}
			for i := 1; i < len(corners); i += 3 {
				length := float64(corners[i]*corners[i] + corners[i+1]*corners[i+1] + corners[i+2]*corners[i+2])
				if math.Abs(length-1) > 2e-7 {
					t.Fatalf("corner left sphere: %g", length)
				}
			}
		}
	}
}

func TestCameraProjectsCentreAndKeepsRightHandedImage(t *testing.T) {
	at := [3]float64{2, 3, 4}
	matrix := camera([3]float64{0, 0, -2}, [3]float64{0, 1, 0}, at, 90, 800, 400)
	project := func(p [3]float64) [4]float64 {
		var q [4]float64
		for row := range 4 {
			q[row] = matrix[row]*p[0] + matrix[4+row]*p[1] + matrix[8+row]*p[2] + matrix[12+row]
		}
		return q
	}
	centre := project([3]float64{2, 3, 2})
	for i, want := range []float64{0, 0, 1, 2} {
		closeFloat(t, centre[i], want)
	}
	right, up := project([3]float64{3, 3, 2}), project([3]float64{2, 4, 2})
	closeFloat(t, right[0]/right[3], .5)
	closeFloat(t, up[1]/up[3], 1)
}

func TestPolarRotationComposesAndPreservesVectors(t *testing.T) {
	got := times(turnedBy(30), turnedBy(60))
	want := turnedBy(90)
	for i := range got {
		closeFloat(t, got[i], want[i])
	}
	for i, x := range []float64{0, 1, 0, 0} {
		closeFloat(t, got[i], x)
	}
	a, b := [3]float64{1, 2, 3}, [3]float64{-4, 5, 2}
	if plus(a, b) != [3]float64{-3, 7, 5} || scaled(a, 2) != [3]float64{2, 4, 6} {
		t.Fatal("vector arithmetic")
	}
	closeFloat(t, dot(a, b), 12)
	perpendicular := cross(a, b)
	closeFloat(t, dot(a, perpendicular), 0)
	closeFloat(t, dot(b, perpendicular), 0)
	closeFloat(t, dot(norm(a), norm(a)), 1)
}

func TestHorizonIsAnOrthonormalLocalFrameAndFacesCentre(t *testing.T) {
	viewerFixture(t)
	view.placeLon, view.placeLat = -122.4, 37.7
	zenith, north, east := horizon()
	for _, axis := range [][3]float64{zenith, north, east} {
		closeFloat(t, dot(axis, axis), 1)
	}
	closeFloat(t, dot(zenith, north), 0)
	closeFloat(t, dot(zenith, east), 0)
	closeFloat(t, dot(north, east), 0)
	for i, v := range cross(north, zenith) {
		closeFloat(t, east[i], v)
	}
	faceCentre()
	if view.altitude < horizonView || view.altitude < view.centreAlt || math.Abs(view.azimuth) > 180 {
		t.Fatalf("centre pointing %.3f %.3f %.3f", view.azimuth, view.altitude, view.centreAlt)
	}
	first := sidereal()
	view.clock = view.clock.Add(time.Hour)
	later := sidereal()
	turn := math.Mod(later-first+360, 360)
	if turn < 15 || turn > 15.1 {
		t.Fatalf("one UTC hour turned %g degrees", turn)
	}
}

func TestDragAndScrollUseEachViewsScaleAndClamp(t *testing.T) {
	viewerFixture(t)
	goDrag(10, 20)
	closeFloat(t, view.ra, 1.5)
	closeFloat(t, view.dec, 3)
	goScroll(100000)
	closeFloat(t, view.fov, .02)
	goScroll(-100000)
	closeFloat(t, view.fov, 120)
	view.mode = earthView
	goDrag(10, 20)
	closeFloat(t, view.lon, -3.3)
	closeFloat(t, view.lat, 6.6)
	goScroll(100000)
	closeFloat(t, view.distance, 1.02)
	goScroll(-100000)
	closeFloat(t, view.distance, 12)
	view.mode = placeView
	goDrag(10, 100000)
	closeFloat(t, view.azimuth, -1.5)
	closeFloat(t, view.altitude, 89.9)
	goDrag(0, -100000)
	closeFloat(t, view.altitude, -89.9)
	goScroll(100000)
	closeFloat(t, view.placeFov, .5)
	goScroll(-100000)
	closeFloat(t, view.placeFov, 120)
	view.width = 0
	view.mode = skyView
	goDrag(0, -100000)
	closeFloat(t, view.dec, -89.9)
}

func TestNonPaintingKeysChangeTheirOwnControlAndRespectBounds(t *testing.T) {
	viewerFixture(t)
	start := view.clock
	goKey(',')
	if !view.clock.Equal(start.Add(-time.Hour)) {
		t.Fatal("previous hour")
	}
	goKey('.')
	if !view.clock.Equal(start) {
		t.Fatal("next hour")
	}
	goKey('h')
	if view.hudSize != 0 || view.hudSaid != "" {
		t.Fatal("HUD size cycle")
	}
	goKey('o')
	if view.outline != 1 {
		t.Fatal("outline cycle")
	}
	goKey('t')
	if view.theme != 0 {
		t.Fatal("theme cycle")
	}
	for range 10 {
		goKey(']')
	}
	if view.line != len(widths)-1 {
		t.Fatal("outline upper bound")
	}
	for range 10 {
		goKey('[')
	}
	if view.line != 0 {
		t.Fatal("outline lower bound")
	}
	before := view.clock
	goKey(0)
	if !view.clock.Equal(before) {
		t.Fatal("unknown key changed clock")
	}
}

func TestHudCoordinatesAndModesTellTheViewerWhatIsSelected(t *testing.T) {
	viewerFixture(t)
	for _, row := range []struct {
		lat, lon float64
		want     string
	}{{37.7, -122.4, "37.70 N, 122.40 W"}, {-45, 190, "45.00 S, 170.00 W"}, {0, 360, "0.00 N, 0.00 E"}} {
		if got := latLon(row.lat, row.lon); got != row.want {
			t.Fatalf("%q want %q", got, row.want)
		}
	}
	if onOff(true) != "on " || onOff(false) != "off" || !strings.Contains(placed(), "not placed") {
		t.Fatal("layer/observer label")
	}
	view.placed = true
	view.placeLon = -122.4
	view.placeLat = 37.7
	if placed() != "37.70 N, 122.40 W" {
		t.Fatal(placed())
	}
	for _, mode := range []int{skyView, earthView, placeView} {
		view.mode = mode
		view.centreAlt = -30
		lines, help := hudWords()
		if len(lines) < 4 || len(help) != 2 || !strings.Contains(strings.Join(lines, "\n"), "test") {
			t.Fatal("incomplete HUD")
		}
		if mode == placeView && !strings.Contains(strings.Join(lines, "\n"), "below the horizon, by 30") {
			t.Fatal(lines)
		}
	}
	view.centreAlt = 12
	if lines, _ := hudWords(); !strings.Contains(strings.Join(lines, "\n"), "12 degrees above") {
		t.Fatal(lines)
	}
}

// uniforms calls the real cgo callback without importing C into a test file.
func uniforms[F ~float32, I ~int32](callback func(*F, I, I) I, width, height int) ([]float32, int) {
	values := make([]F, 72)
	mode := callback(&values[0], I(width), I(height))
	out := make([]float32, len(values))
	for i, v := range values {
		out[i] = float32(v)
	}
	return out, int(mode)
}

func TestRealUniformCallbackSetsModeCrosshairAndGroundBlocks(t *testing.T) {
	viewerFixture(t)
	for _, mode := range []int{skyView, earthView, placeView} {
		view.mode = mode
		got, selected := uniforms(goView, 800, 400)
		if selected != mode || view.width != 800 {
			t.Fatal("view selection")
		}
		if got[16] != float32(outlines[0]) || got[17] != float32(widths[0]) || got[23] != 1 || got[36+18] != 1 || got[36+34] != 0 {
			t.Fatalf("uniform controls %v", got)
		}
		if (got[34] == 1) != (mode == earthView) {
			t.Fatalf("crosshair mode %d", mode)
		}
		if mode == earthView && !slices.Equal(got[32:35], []float32{400, 200, 1}) {
			t.Fatal("crosshair centre")
		}
		for _, value := range got {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				t.Fatal("non-finite uniform")
			}
		}
		if view.hudSaid != "off" {
			t.Fatal("hidden HUD was not hidden")
		}
	}
}

func TestMatrixCompositionDoesNotCommuteScaleAndRotation(t *testing.T) {
	scaleAndTranslate := [16]float64{2, 0, 0, 0, 0, 3, 0, 0, 0, 0, 4, 0, 7, 8, 9, 1}
	product := times(scaleAndTranslate, turnedBy(90))
	want := [16]float64{0, 3, 0, 0, -2, 0, 0, 0, 0, 0, 4, 0, 7, 8, 9, 1}
	for i := range want {
		closeFloat(t, product[i], want[i])
	}
	reverse := times(turnedBy(90), scaleAndTranslate)
	if math.Abs(product[1]-reverse[1]) < .5 {
		t.Fatal("noncommuting composition became commutative")
	}
}

func TestLongitudeLatitudeAxesMatchTheDeclaredEquatorialFrame(t *testing.T) {
	for _, sample := range []struct {
		lon, lat float64
		want     [3]float64
	}{{0, 0, [3]float64{1, 0, 0}}, {90, 0, [3]float64{0, 1, 0}}, {0, 90, [3]float64{0, 0, 1}}, {180, 0, [3]float64{-1, 0, 0}}} {
		x, y, z := unit(sample.lon, sample.lat)
		for i, v := range []float64{x, y, z} {
			closeFloat(t, v, sample.want[i])
		}
	}
	for _, cell := range cellsAt(1) {
		vertices := appendCorners(nil, cell)
		a := [3]float64{float64(vertices[0]), float64(vertices[1]), float64(vertices[2])}
		b := [3]float64{float64(vertices[3]), float64(vertices[4]), float64(vertices[5])}
		c := [3]float64{float64(vertices[6]), float64(vertices[7]), float64(vertices[8])}
		minus := func(p, q [3]float64) [3]float64 { return [3]float64{p[0] - q[0], p[1] - q[1], p[2] - q[2]} }
		normal := cross(minus(b, a), minus(c, a))
		if dot(normal, plus(plus(a, b), c)) <= 0 {
			t.Fatal("sphere face wound inward")
		}
	}
}
