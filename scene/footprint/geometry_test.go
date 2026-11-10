package footprint_test

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
)

// These fields belong to the independent authored YAML fixture.
type pinholeBook struct {
	Width, Height int
	Field         float64               `yaml:"field-deg"`
	PixelAspect   float64               `yaml:"pixel-aspect"`
	Near          float64               `yaml:"near-m"`
	Cases         map[string]pinholeRow `yaml:"cases"`
}

type pinholeRow struct {
	Point        [3]float64
	Displacement float64 `yaml:"displacement-m"`
	PixelAspect  float64 `yaml:"pixel-aspect"`
	Visible      bool
	Limited      bool
	Numerator    int64 `yaml:"squared-bound-numerator"`
	Denominator  int64 `yaml:"squared-bound-denominator"`
}

func readPinhole(t *testing.T) pinholeBook {
	t.Helper()
	const path = "testdata/pinhole-contract.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var book pinholeBook
	if err := yamlfile.Read(path, raw, &book); err != nil {
		t.Fatal(err)
	}
	if len(book.Cases) != 7 {
		t.Fatal("the seven independent cases must all be present")
	}
	return book
}

func checkedView(t *testing.T, book pinholeBook,
	pixelAspect float64, eye maths.Subspace,
) footprint.Perspective {
	t.Helper()
	view, err := footprint.New(eye, measure.Radians(book.Field*math.Pi/180),
		book.Width, book.Height, pixelAspect, measure.Metres(book.Near))
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func pointBox(point maths.Vec) maths.Box {
	return maths.Box{Min: point, Max: point}
}

func checkAssessment(t *testing.T, got footprint.Assessment, row pinholeRow) {
	t.Helper()
	if got.Visible != row.Visible || got.Limited != row.Limited {
		t.Fatalf("visibility/near limit %v, want %v/%v",
			got, row.Visible, row.Limited)
	}
	if row.Limited {
		if got.Pixels != math.MaxFloat64 {
			t.Fatal("the near-limited error was not retained", got)
		}
		return
	}
	checkPixelBound(t, got.Pixels, row)
}

func checkPixelBound(t *testing.T, pixels float64, row pinholeRow) {
	t.Helper()
	if row.Numerator == 0 {
		if pixels != 0 {
			t.Fatal("zero displacement or invisible geometry has an error", pixels)
		}
		return
	}
	// The YAML supplies rational arithmetic, not a production result.
	// One ULP below sqrt permits its conversion rounding; the upper
	// allowance detects a gross bound while admitting outward arithmetic.
	want := math.Sqrt(float64(row.Numerator) / float64(row.Denominator))
	if !(pixels >= math.Nextafter(want, 0) && pixels <= want*(1+1e-10)) {
		t.Fatalf("pixel bound %.17g, independent rational bound %.17g", pixels, want)
	}
}

func TestPinholeBoundsMatchIndependentRationalFixtures(t *testing.T) {
	book := readPinhole(t)
	for name, row := range book.Cases {
		t.Run(name, func(t *testing.T) {
			aspect := row.PixelAspect
			if aspect == 0 {
				aspect = book.PixelAspect
			}
			view := checkedView(t, book, aspect, maths.Subspace{})
			point := maths.V(row.Point[0], row.Point[1], row.Point[2])
			got, err := view.Assess(pointBox(point),
				measure.Metres(row.Displacement))
			if err != nil {
				t.Fatal(err)
			}
			checkAssessment(t, got, row)
		})
	}
}

type planeCase struct {
	name         string
	point        maths.Vec
	displacement float64
	visible      bool
}

// The horizontal plane is x=+/-100 and vertical y=+/-50 at depth 100.
var planes = []planeCase{
	{"left", maths.V(-100, 0, -100), 0, true},
	{"right", maths.V(100, 0, -100), 0, true},
	{"top", maths.V(0, 50, -100), 0, true},
	{"bottom", maths.V(0, -50, -100), 0, true},
	{"near", maths.V(0, 0, -1), 0, true},
	{"past left", maths.V(-101, 0, -100), 0, false},
	{"past right", maths.V(101, 0, -100), 0, false},
	{"past top", maths.V(0, 51, -100), 0, false},
	{"past bottom", maths.V(0, -51, -100), 0, false},
	{"past near", maths.V(0, 0, -0.5), 0, false},
	{"behind", maths.V(0, 0, 100), 0, false},
	{"displacement reaches side", maths.V(101, 0, -100), 1, true},
}

func TestAllFiveFrustumPlanesRetainTheirBoundary(t *testing.T) {
	view := checkedView(t, readPinhole(t), 1, maths.Subspace{})
	for _, row := range planes {
		t.Run(row.name, func(t *testing.T) {
			got, err := view.Assess(pointBox(row.point),
				measure.Metres(row.displacement))
			if err != nil || got.Visible != row.visible {
				t.Fatalf("wrong plane decision: %v, %v", got, err)
			}
		})
	}
}

func TestAChangedPoseKeepsTheIndependentCentreBound(t *testing.T) {
	book := readPinhole(t)
	eye := maths.Subspace{Origin: maths.V(10, 20, 30), Scale: 1,
		Orientation: maths.Orientation{X: maths.V(0, 0, 1),
			Y: maths.V(0, 1, 0), Z: maths.V(-1, 0, 0)}}
	view := checkedView(t, book, 1, eye)
	// This camera looks east. The point is 100 metres east of its eye.
	got, err := view.Assess(pointBox(maths.V(110, 20, 30)), measure.Metres(1))
	if err != nil {
		t.Fatal(err)
	}
	checkAssessment(t, got, book.Cases["centre"])
	hidden, err := view.Assess(pointBox(maths.V(-90, 20, 30)), measure.Metres(0))
	if err != nil || hidden.Visible {
		t.Fatal("the same camera retained a point behind it", hidden, err)
	}
}

func TestInvalidFieldsAndPosesAreRefused(t *testing.T) {
	for _, field := range []float64{0, math.Pi, math.NaN()} {
		if _, err := footprint.New(maths.Subspace{}, measure.Radians(field),
			1000, 500, 1, measure.Metres(1)); err == nil {
			t.Fatal("accepted invalid field", field)
		}
	}
	for _, eye := range []maths.Subspace{{Scale: 2},
		{Origin: maths.V(math.NaN(), 0, 0)},
		{Orientation: maths.Orientation{X: maths.V(1, 0, 0),
			Y: maths.V(0, 1, 0), Z: maths.V(0, 0, -1)}}} {
		if _, err := footprint.New(eye, measure.Radians(math.Pi/2),
			1000, 500, 1, measure.Metres(1)); err == nil {
			t.Fatal("accepted invalid camera pose", eye)
		}
	}
}

func TestRasterAspectAndNearDistanceMustBeUsable(t *testing.T) {
	for _, row := range []struct {
		width, height int
		aspect, near  float64
	}{{0, 500, 1, 1}, {1000, 0, 1, 1}, {1000, 500, 0, 1},
		{1000, 500, 1, 0}, {1000, 500, 1, math.NaN()},
		{1000, 500, math.MaxFloat64, 1}} {
		if _, err := footprint.New(maths.Subspace{},
			measure.Radians(math.Pi/2), row.width, row.height,
			row.aspect, measure.Metres(row.near)); err == nil {
			t.Fatal("accepted invalid raster or near distance", row)
		}
	}
}

func TestUncheckedViewsAndInvalidBoundsCannotCull(t *testing.T) {
	box := pointBox(maths.V(0, 0, -100))
	unchecked := footprint.Perspective{}
	if _, err := unchecked.Assess(box, measure.Metres(0)); err == nil {
		t.Fatal("an unchecked view assessed geometry")
	}
	view := checkedView(t, readPinhole(t), 1, maths.Subspace{})
	for _, bad := range []maths.Box{{Min: maths.V(1, 0, 0)},
		{Max: maths.V(0, math.NaN(), 0)}} {
		if _, err := view.Assess(bad, measure.Metres(0)); err == nil {
			t.Fatal("accepted invalid bounds", bad)
		}
	}
	for _, bad := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, err := view.Assess(box, measure.Metres(bad)); err == nil {
			t.Fatal("accepted invalid displacement", bad)
		}
	}
}
