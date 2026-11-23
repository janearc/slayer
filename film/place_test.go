package film_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// The rays the places are held to.
const (
	// placedSide is defined as four rays along a pixel's side, and
	// placedCount as the sixteen rays a pixel that makes.
	placedSide  = 4
	placedCount = placedSide * placedSide
	// manyRays is defined as the most a film may say: 64 a side.
	manyRays = 4096
	// shapeRoom is how far the mean and the variance of 4096 places may
	// lie from their shape's. Picked: the places are the shape's own
	// quantiles, a 4096th apart, and this is forty times that.
	shapeRoom = 0.01
)

// The widths of the slope at each side of the three shapes, in pixels:
// none for the box, half for the trapezoid, all for the tent.
const (
	boxRamp       = 0
	trapezoidRamp = 0.5
	tentRamp      = 1
)

// centred is a sensor that has only its pixels' middles: a pinhole with
// no place to look through.
type centred struct{ film.Sensor }

// ramps is the three shapes' slopes.
func ramps() []float64 {
	return []float64{boxRamp, trapezoidRamp, tentRamp}
}

// A place is the same every time it is asked, and never further than
// its shape reaches: half of a pixel and the slope from the middle.
func TestAPlaceIsTheSameAndInsideItsShape(t *testing.T) {
	for _, ramp := range ramps() {
		reach := (1 + ramp) / 2
		for index := range manyRays {
			across, down := film.Place(index, manyRays, ramp)
			furthest := math.Max(math.Abs(across), math.Abs(down))
			if !(furthest < reach) {
				t.Fatalf("place %d is %g across and %g down, "+
					"outside a shape that reaches %g", index,
					across, down, reach)
			}
		}
	}
	across, down := film.Place(7, placedCount, tentRamp)
	again, under := film.Place(7, placedCount, tentRamp)
	if across != again || down != under {
		t.Errorf("one place asked twice is %g, %g and %g, %g", across,
			down, again, under)
	}
}

// A pixel's places lie as their shape does: their middle is the
// pixel's, and their variance along each side is that of a square one
// pixel wide smoothed by one as wide as the slope: a twelfth of a pixel
// squared, and a twelfth of the slope squared. The tent's is a sixth.
func TestPlacesLieAsTheirShapeDoes(t *testing.T) {
	for _, ramp := range ramps() {
		var sum, square float64
		for index := range manyRays {
			across, down := film.Place(index, manyRays, ramp)
			sum += across + down
			square += across*across + down*down
		}
		want := (1 + ramp*ramp) / 12
		mean, variance := sum/(2*manyRays), square/(2*manyRays)
		if !(math.Abs(mean) <= shapeRoom) ||
			!(math.Abs(variance-want) <= shapeRoom) {
			t.Errorf("a slope of %g: mean %g and variance %g, "+
				"want nought and %g within %g", ramp, mean,
				variance, want, shapeRoom)
		}
	}
}

// Anti: the slope is read. If it were dropped, the tent's rays would
// lie as the box's do, inside the pixel's own square.
func TestTheTentReachesPastThePixelsOwnSquare(t *testing.T) {
	outside := 0
	for index := range placedCount {
		across, down := film.Place(index, placedCount, tentRamp)
		if math.Abs(across) > 0.5 || math.Abs(down) > 0.5 {
			outside++
		}
	}
	if outside == 0 {
		t.Error("none of the tent's sixteen rays starts outside the " +
			"pixel's own square")
	}
}

// A pixel's sixteen rays start at sixteen heights and sixteen distances
// across, eight each side of the middle, where a grid of four by four
// has four of each: an edge along the grid is then drawn in seventeen
// shades and not five. One ray of one starts at the middle.
func TestAPixelsRaysShareNoRowAndNoColumn(t *testing.T) {
	rows, columns := map[float64]bool{}, map[float64]bool{}
	left, above := 0, 0
	for index := range placedCount {
		across, down := film.Place(index, placedCount, tentRamp)
		rows[down], columns[across] = true, true
		if across < 0 {
			left++
		}
		if down < 0 {
			above++
		}
	}
	if len(rows) != placedCount || len(columns) != placedCount {
		t.Errorf("sixteen rays start in %d rows and %d columns, want "+
			"sixteen of each", len(rows), len(columns))
	}
	if half := placedCount / 2; left != half || above != half {
		t.Errorf("%d rays start left of the middle and %d above it, "+
			"want %d of each", left, above, half)
	}
	across, down := film.Place(0, 1, tentRamp)
	if across != 0 || down != 0 {
		t.Errorf("one ray of one starts %g, %g from the middle", across,
			down)
	}
}

// Anti: the places are not the regular grid. If the sequence were
// dropped for the grid's own middles, four rays would share each row.
func TestThePlacesAreNotTheGrid(t *testing.T) {
	moved := 0
	for y := range placedSide {
		for x := range placedSide {
			across, down := film.Within(x, y, placedSide, boxRamp)
			if across != 0 || down != 0 {
				moved++
			}
		}
	}
	if moved != placedCount {
		t.Errorf("%d of sixteen rays left their own squares' middles, "+
			"want all sixteen", moved)
	}
}

// Within is Place in the fine picture's own pixels: a fine pixel's
// middle and its ray's start add up to the place in the pixel it
// belongs to. And every pixel has the same pattern.
func TestWithinIsPlaceInFinePixels(t *testing.T) {
	for y := range placedSide * 3 {
		for x := range placedSide * 3 {
			across, down := film.Within(x, y, placedSide, tentRamp)
			over, under := x%placedSide, y%placedSide
			index := under*placedSide + over
			wantAcross, wantDown := film.Place(index, placedCount,
				tentRamp)
			gotAcross := (float64(over)+0.5+across)/placedSide - 0.5
			gotDown := (float64(under)+0.5+down)/placedSide - 0.5
			if math.Abs(gotAcross-wantAcross) > 1e-12 ||
				math.Abs(gotDown-wantDown) > 1e-12 {
				t.Fatalf("fine pixel %d, %d starts %g, %g "+
					"from its pixel's middle, want %g, %g", x,
					y, gotAcross, gotDown, wantAcross, wantDown)
			}
		}
	}
}

// Half a stratum is half of one over the least power of two that holds
// the count, as a share in thirty-two bits.
func TestAStratumHalvesWithTheCount(t *testing.T) {
	for _, each := range []struct {
		count int
		want  uint32
	}{{1, 1 << 31}, {4, 1 << 29}, {9, 1 << 27}, {16, 1 << 27},
		{4096, 1 << 19}} {
		if got := film.Stratum(each.count); got != each.want {
			t.Errorf("half a stratum of %d rays is %d, want %d",
				each.count, got, each.want)
		}
	}
}

// A pinhole through a pixel's middle is its Ray to the bit, and through
// a place one whole pixel across is the next pixel's Ray.
func TestAPinholeLooksThroughAPlace(t *testing.T) {
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	middle := lens.RayThrough(5, 7, 0, 0, 32, 18, maths.Band{})
	if want := lens.Ray(5, 7, 32, 18, maths.Band{}); middle != want {
		t.Errorf("through the middle the ray is %v, not %v", middle,
			want)
	}
	moved := lens.RayThrough(5, 7, 1, 1, 32, 18, maths.Band{})
	want := lens.Ray(6, 8, 32, 18, maths.Band{})
	if moved.Dir.Sub(want.Dir).Len() > 1e-12 {
		t.Errorf("a pixel across and down the ray is %v, not %v", moved,
			want)
	}
}

// Anti: a place off the middle is another ray. If the place were
// dropped, every ray of a pixel would be its middle's.
func TestAPlaceOffTheMiddleIsAnotherRay(t *testing.T) {
	lens := film.Lens{Focal: 35}
	middle := lens.Ray(5, 7, 32, 18, maths.Band{})
	moved := lens.RayThrough(5, 7, 0.25, -0.25, 32, 18, maths.Band{})
	if moved == middle {
		t.Error("a quarter of a pixel off the middle the lens's ray " +
			"is the middle's")
	}
}

// LookThrough is the placed ray taken out of the eye's space; and a
// sensor with only its pixels' middles looks through the middle.
func TestLookThroughTakesThePlaceWhereTheSensorHasOne(t *testing.T) {
	eye := maths.Subspace{Origin: maths.V(1, 2, 3),
		Orientation: maths.LookAt(maths.V(1, 2, 3), maths.Vec{})}
	lens := film.Pinhole{}
	placed := film.LookThrough(eye, lens, 5, 7, 0.25, 0.5, 32, 18,
		maths.Band{})
	middle := film.Look(eye, lens, 5, 7, 32, 18, maths.Band{})
	if placed == middle || placed.At != middle.At {
		t.Errorf("the placed look is %v beside the middle's %v", placed,
			middle)
	}
	plain := film.LookThrough(eye, centred{lens}, 5, 7, 0.25, 0.5, 32,
		18, maths.Band{})
	if plain != middle {
		t.Errorf("a sensor with no places looks along %v, not its "+
			"middle's %v", plain, middle)
	}
}
