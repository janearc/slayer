package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// bandPoints are places across a patch of a surface, in units.
func bandPoints() []maths.Vec {
	var points []maths.Vec
	for x := 0; x < 12; x++ {
		for z := 0; z < 12; z++ {
			across := 0.37 * float64(x)
			along := 0.41 * float64(z)
			points = append(points, maths.V(across, 0.1,
				along))
		}
	}
	return points
}

// spread is how far the brightest of a texture's paint is from its
// darkest, in the red, over the points.
func spread(paint func(maths.Vec) maths.Light) (low, high float64) {
	low, high = math.Inf(1), math.Inf(-1)
	for _, p := range bandPoints() {
		low, high = math.Min(low, paint(p).R), math.Max(high, paint(p).R)
	}
	return low, high
}

// bandTolerance is how near its average a far texture must paint, in the
// light's own units.
const bandTolerance = 1e-9

func TestDetailIsAllOfItNearNoneOfItFarAndFadesBetween(t *testing.T) {
	for _, c := range []struct{ footprint, want float64 }{{0, 1}, {-1, 1},
		{0.05, 1}, {0.5, 1}, {0.75, 0.5}, {1, 0}, {9, 0}} {
		if got := Detail(c.footprint, 1); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("a pixel of %v on a feature of 1 shows %v, wanted %v",
				c.footprint, got, c.want)
		}
	}
	if Detail(5, 0) != 1 {
		t.Errorf("a feature of no size is all shown")
	}
}

func TestDustedFarAwayIsItsAverageAndNearItStillVaries(t *testing.T) {
	dusted := Dusted{Rock: maths.Grey(0.8), Dust: maths.Grey(0.4), Share: 0.5,
		Patch: 1, Seed: 3}
	average := 0.6
	low, high := spread(func(p maths.Vec) maths.Light {
		return dusted.Within(p, 10)
	})
	if low < average-bandTolerance || high > average+bandTolerance {
		t.Errorf("far away it runs from %v to %v, wanted %v", low, high,
			average)
	}
	low, high = spread(func(p maths.Vec) maths.Light {
		return Sample(dusted, p, 0.01)
	})
	if high-low < 0.2 {
		t.Errorf("near it runs only from %v to %v", low, high)
	}
}

func TestWoodFarAwayIsItsAverageAndNearItStillVaries(t *testing.T) {
	wood := Wood{Early: maths.Grey(0.8), Late: maths.Grey(0.2), Ring: 0.1,
		LateShare: 0.25}
	average := 0.65
	low, high := spread(func(p maths.Vec) maths.Light {
		return Sample(wood, p, 5)
	})
	if low < average-bandTolerance || high > average+bandTolerance {
		t.Errorf("far away it runs from %v to %v, wanted %v", low, high,
			average)
	}
	low, high = spread(func(p maths.Vec) maths.Light {
		return Sample(wood, p, 0.001)
	})
	if high-low < 0.3 {
		t.Errorf("near it runs only from %v to %v", low, high)
	}
}

func TestTiledLinesFadeToAShareOfThePaintFarAndStayNear(t *testing.T) {
	tiled := Hexed{Tiling: hexes.Sphere{Splits: 4},
		Palette: []maths.Light{maths.Grey(0.5)}, Lines: 0.8}
	width := Tiled{Tiling: tiled.Tiling, Width: tiled.Width}.width()
	cover := math.Min(1, lineCover*width/tiled.Tiling.Spacing())
	want := 0.5 * (1 - 0.8*cover)
	var lowest = 1.0
	for _, p := range sphereRing() {
		far := tiled.Within(p, 10*width).R
		if math.Abs(far-want) > bandTolerance {
			t.Fatalf("far away a line is %v, wanted %v", far, want)
		}
		lowest = math.Min(lowest, tiled.Within(p, width/100).R)
	}
	if lowest > 0.5*(1-0.5*0.8) {
		t.Errorf("near, no point is on a line: darkest %v", lowest)
	}
}

// sphereRing is points round the unit sphere's equator, close enough
// together that some are on the lines between tiles.
func sphereRing() []maths.Vec {
	var points []maths.Vec
	for step := 0; step < 2000; step++ {
		angle := 2 * math.Pi * float64(step) / 2000
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		points = append(points, maths.V(cosine, 0, sine))
	}
	return points
}

func TestAWireFarAwayIsTheShareOfTheSurfaceItsLinesCover(t *testing.T) {
	wire := Wire{Tiling: hexes.Sphere{Splits: 4}, Light: maths.Grey(1)}
	want := float64(lineCover) / float64(wireShare)
	for _, p := range sphereRing()[:50] {
		if far := wire.Within(p, 100).R; math.Abs(far-want) > bandTolerance {
			t.Fatalf("far away the wire is %v, wanted %v", far, want)
		}
	}
	if (Wire{}).Within(maths.Vec{}, 1) != (maths.Light{}) {
		t.Errorf("a wire with no tiling is not black")
	}
}

func TestATextureWithNoFootprintIsPaintedAsItAlwaysWas(t *testing.T) {
	dusted := Dusted{Rock: maths.Grey(0.8), Dust: maths.Grey(0.4), Share: 0.5,
		Patch: 1, Seed: 3}
	for _, p := range bandPoints() {
		if dusted.Within(p, 0) != dusted.At(p) {
			t.Fatalf("a lone ray saw %v, not %v", dusted.Within(p, 0),
				dusted.At(p))
		}
	}
}
