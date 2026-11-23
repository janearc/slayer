package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The numbers the grazing, seam and leaving tests are held to, each a choice
// of the test.
const (
	// grazeShare is how far a reported distance may be from the true one, as
	// a share of it, for rounding.
	grazeShare = 1e-9
	// grazeHeight is how far above the water the grazing rays start.
	grazeHeight = 2.0
	// grazeFar is the far end of every search: as far as a tracer asks.
	grazeFar = math.MaxFloat64
	// seaExtent and seaSamples are the wide, wavy field the grazing rays
	// cross, 8 metres a cell, wide enough that a tenth of a degree from two
	// metres up arrives inside it.
	seaExtent, seaSamples = 4096.0, 513
	// seaAmplitude is how high the waves of that field are.
	seaAmplitude = 0.5
	// seaStart is how far south of the middle of the sea the rays start.
	seaStart = 1900.0
	// crestSamples and crestExtent are the grid of the ridge field: a metre
	// a cell, a crest of one metre along its middle column.
	crestSamples, crestExtent = 401, 400.0
	// crestGap is how far over or under the crest a ray passes.
	crestGap = 1e-3
	// pyramidSamples and pyramidExtent are the grid of the field with one
	// raised vertex: a metre a cell, so its vertices are exact numbers.
	pyramidSamples, pyramidExtent = 9, 8.0
	// pyramidRun is how far along each of its rays the peak is.
	pyramidRun = 4.0
	// pyramidMiss is how far above the peak the rays of the anti-test pass.
	pyramidMiss = 1e-3
)

// grazingDegrees are the angles below level the grazing tests draw at.
var grazingDegrees = []float64{5, 2, 1, 0.1}

// gridField makes a field of samples a side over extent, each height the
// function of where the sample is.
func gridField(t *testing.T, samples int, extent float64,
	height func(x, z float64) float64) *HeightField {
	t.Helper()
	step := extent / float64(samples-1)
	heights := make([]float64, samples*samples)
	for row := range samples {
		for column := range samples {
			heights[row*samples+column] = height(
				-extent/2+float64(column)*step, -extent/2+float64(row)*step)
		}
	}
	field, err := NewHeightField(heights, samples, extent)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// levelAcross is a field two samples a side, so one cell, at a level.
func levelAcross(t *testing.T, extent, level float64) *HeightField {
	t.Helper()
	return gridField(t, pair, extent, func(x, z float64) float64 {
		return level
	})
}

// seaField is the wide wavy field: waves a few tens of metres long.
func seaField(t *testing.T) *HeightField {
	t.Helper()
	return gridField(t, seaSamples, seaExtent, func(x, z float64) float64 {
		return seaAmplitude * math.Sin(2*math.Pi*(x/64+z/96))
	})
}

// downward is a unit ray from a start toward positive z, the given degrees
// below level; negative degrees are above it.
func downward(at maths.Vec, degrees float64) maths.Ray {
	radians := degrees * math.Pi / 180
	return maths.Ray{At: at, Dir: maths.V(0, -math.Sin(radians),
		math.Cos(radians))}
}

// meets checks that the field is met by the ray at the distance, and says so
// of the test when it is not.
func meets(t *testing.T, field *HeightField, ray maths.Ray, want float64) {
	t.Helper()
	got, hit := field.Hit(ray, 0, grazeFar)
	if !hit || math.Abs(got-want) > grazeShare*math.Max(1, want) {
		t.Errorf("ray %v met the field at %g, %t; want %g", ray, got, hit, want)
	}
}

// misses checks that the field is not met by the ray.
func misses(t *testing.T, field *HeightField, ray maths.Ray) {
	t.Helper()
	if got, hit := field.Hit(ray, 0, grazeFar); hit {
		t.Errorf("ray %v met the field at %g; want a miss", ray, got)
	}
}

// Level water two metres below the eye is met at the distance geometry says,
// at five, two, one and a tenth of a degree.
func TestHeightFieldHitMeetsLevelWaterAtGrazingAngles(t *testing.T) {
	field := levelAcross(t, seaExtent, 0)
	for _, degrees := range grazingDegrees {
		ray := downward(maths.V(0, grazeHeight, -seaStart), degrees)
		meets(t, field, ray, grazeHeight/math.Sin(degrees*math.Pi/180))
	}
}

// The anti-test of the one above: a ray cut short of the water, a ray that
// climbs, and a ray that starts above the water and goes level, do not meet
// it.
func TestHeightFieldHitDoesNotMeetLevelWaterItDoesNotReach(t *testing.T) {
	field := levelAcross(t, seaExtent, 0)
	for _, degrees := range grazingDegrees {
		ray := downward(maths.V(0, grazeHeight, -seaStart), degrees)
		want := grazeHeight / math.Sin(degrees*math.Pi/180)
		if got, hit := field.Hit(ray, 0, want*(1-grazeShare*1e3)); hit {
			t.Errorf("%g degrees: met at %g before the far end", degrees, got)
		}
		misses(t, field, downward(ray.At, -degrees))
	}
	misses(t, field, maths.Ray{At: maths.V(0, grazeHeight, -seaStart),
		Dir: maths.V(0, 0, 1)})
}

// Waves, not level water: the hit agrees with the plain sampling of the ray,
// at each grazing angle, and each of them arrives.
func TestHeightFieldHitMeetsWavesAtGrazingAngles(t *testing.T) {
	field := seaField(t)
	far := shapePropertyRangesIn(t).far
	for _, degrees := range grazingDegrees {
		ray := downward(maths.V(0, grazeHeight, -seaStart), degrees)
		if _, hit := field.Hit(ray, 0, grazeFar); !hit {
			t.Errorf("%g degrees over waves was not met", degrees)
		}
		if why := disagreement(field, ray, far); why != "" {
			t.Errorf("%g degrees over waves: %s", degrees, why)
		}
	}
}

// The anti-test of the one above: the same angles from over the crests of
// the waves, twelve metres up, do not arrive inside the field, and are
// misses.
func TestHeightFieldHitMissesWavesFromAboveTheirCrests(t *testing.T) {
	field := seaField(t)
	far := shapePropertyRangesIn(t).far
	for _, degrees := range []float64{0.1, 0.05} {
		ray := downward(maths.V(0, 12, -seaStart), degrees)
		misses(t, field, ray)
		if why := disagreement(field, ray, far); why != "" {
			t.Errorf("%g degrees from above the crests: %s", degrees, why)
		}
	}
}

// crestField is a ridge a metre high along the middle column, on level
// ground, a metre a cell.
func crestField(t *testing.T) *HeightField {
	t.Helper()
	return gridField(t, crestSamples, crestExtent, func(x, z float64) float64 {
		if x == 0 {
			return 1
		}
		return 0
	})
}

// A level ray a hair under a crest meets it, on the rising side, and a ray
// that skims a crest by a millimetre at half a degree passes it and meets the
// ground beyond, where the geometry says.
func TestHeightFieldHitMeetsWhatIsJustBelowACrestAndBeyondIt(t *testing.T) {
	field := crestField(t)
	under := 1 - 1e-6
	meets(t, field, maths.Ray{At: maths.V(-4, under, 0), Dir: maths.V(1, 0, 0)},
		4-1e-6)
	tilt := math.Tan(0.5 * math.Pi / 180)
	over := maths.Ray{At: maths.V(-4, 1+crestGap+4*tilt, 0),
		Dir: maths.V(1, -tilt, 0)}
	meets(t, field, over, 4+(1+crestGap)/tilt)
	below := maths.Ray{At: maths.V(-4, 1-crestGap+4*tilt, 0),
		Dir: maths.V(1, -tilt, 0)}
	meets(t, field, below, 4-crestGap/(1+tilt))
}

// The anti-test of the one above: a level ray a hair over the crest never
// meets anything, and a ray that skims the crest by a millimetre does not
// meet the crest.
func TestHeightFieldHitPassesWhatIsJustOverACrest(t *testing.T) {
	field := crestField(t)
	misses(t, field, maths.Ray{At: maths.V(-4, 1+1e-6, 0),
		Dir: maths.V(1, 0, 0)})
	tilt := math.Tan(0.5 * math.Pi / 180)
	skim := maths.Ray{At: maths.V(-4, 1+crestGap+4*tilt, 0),
		Dir: maths.V(1, -tilt, 0)}
	if got, hit := field.Hit(skim, 0, grazeFar); hit && got < 4+1 {
		t.Errorf("a ray over the crest met the crest at %g", got)
	}
}

// Level water meets nothing a ray leaves it with: from exactly on the surface
// it leaves by, at any angle above or below it, and from a tracer's lift off
// it away from it; and a ray lifted off the surface aimed into it is met
// where geometry says.
func TestHeightFieldHitLeavesTheSurfaceItStandsOn(t *testing.T) {
	field := levelAcross(t, seaExtent, 0)
	lift := testLift * seaExtent
	for _, degrees := range []float64{45, 5, 1} {
		for _, side := range []float64{1, -1} {
			on := downward(maths.V(0, 0, 0), -side*degrees)
			misses(t, field, on)
			lifted := downward(maths.V(0, side*lift, 0), -side*degrees)
			misses(t, field, lifted)
			aimed := downward(maths.V(0, side*lift, 0), side*degrees)
			meets(t, field, aimed, lift/math.Sin(degrees*math.Pi/180))
		}
	}
	along := maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(0, 0, 1)}
	misses(t, field, along)
}

// pyramidField is a field of a metre a cell with one raised vertex, a metre
// high, at a column and a row of its grid.
func pyramidField(t *testing.T, column, row int) *HeightField {
	t.Helper()
	peakX := float64(column) - pyramidExtent/2
	peakZ := float64(row) - pyramidExtent/2
	return gridField(t, pyramidSamples, pyramidExtent,
		func(x, z float64) float64 {
			if x == peakX && z == peakZ {
				return 1
			}
			return 0
		})
}

// pyramidAims are the ways to the peak: along a row of the grid, along a
// column, across cells diagonally, at slopes that are not a cell's, and
// straight down, each a direction of numbers that are exact in binary.
var pyramidAims = []maths.Vec{
	maths.V(1, -0.5, 0), maths.V(0, -0.5, 1), maths.V(1, -0.5, 1),
	maths.V(-1, -0.25, 0.5), maths.V(0.5, -0.25, -1), maths.V(0, -1, 0),
}

// A ray aimed through any vertex of the grid, from any of several ways, meets
// the peak there, as the surface, exactly where it is aimed, by either call.
func TestAHeightFieldIsMetAtAGridVertexFromEveryWay(t *testing.T) {
	for column := 1; column < pyramidSamples-1; column++ {
		for row := 1; row < pyramidSamples-1; row++ {
			field := pyramidField(t, column, row)
			peak := maths.V(float64(column)-pyramidExtent/2, 1,
				float64(row)-pyramidExtent/2)
			for _, aim := range pyramidAims {
				ray := maths.Ray{At: peak.Sub(aim.Scale(pyramidRun)), Dir: aim}
				meets(t, field, ray, pyramidRun)
				if got, hit := field.IntersectExact(ray, 0, grazeFar); !hit ||
					math.Abs(got-pyramidRun) > grazeShare {
					t.Errorf("vertex (%d,%d) aim %v: IntersectExact %g, %t",
						column, row, aim, got, hit)
				}
			}
		}
	}
}

// The anti-test of the one above: the same rays raised a millimetre pass over
// the peak, and are not met at it.
func TestAHeightFieldIsNotMetAtAVertexARayPassesOver(t *testing.T) {
	for column := 1; column < pyramidSamples-1; column++ {
		for row := 1; row < pyramidSamples-1; row++ {
			field := pyramidField(t, column, row)
			peak := maths.V(float64(column)-pyramidExtent/2, 1+pyramidMiss,
				float64(row)-pyramidExtent/2)
			for _, aim := range pyramidAims {
				ray := maths.Ray{At: peak.Sub(aim.Scale(pyramidRun)), Dir: aim}
				if got, hit := field.Hit(ray, 0, grazeFar); hit &&
					got < pyramidRun+pyramidMiss/2 {
					t.Errorf("vertex (%d,%d) aim %v: met at %g over the peak",
						column, row, aim, got)
				}
			}
		}
	}
}

// A ray from outside the square, parallel to an axis, along a line of the grid
// or across a cell; from far above the highest sample at a slope of a
// non-unit direction; and straight down from over a cell; each meets the
// surface where it is.
func TestHeightFieldHitEntersFromOutsideAndFromAbove(t *testing.T) {
	field := pyramidField(t, 4, 4)
	meets(t, field, maths.Ray{At: maths.V(-100, 0.5, 0),
		Dir: maths.V(1, 0, 0)}, 99.5)
	meets(t, field, maths.Ray{At: maths.V(0, 0.5, 100),
		Dir: maths.V(0, 0, -1)}, 99.5)
	meets(t, field, maths.Ray{At: maths.V(-100, 3, 0),
		Dir: maths.V(99.5, -2.5, 0)}, 1)
	meets(t, field, maths.Ray{At: maths.V(0.25, 50, 0.25),
		Dir: maths.V(0, -1, 0)}, 50-0.5625)
	meets(t, field, maths.Ray{At: maths.V(0.25, -50, 0.25),
		Dir: maths.V(0, 1, 0)}, 50+0.5625)
}

// The anti-test of the one above: a ray parallel to an axis above the highest
// sample, outside the square, below the lowest, or heading away from the
// square; and a ray from above going up; meet nothing.
func TestHeightFieldHitDoesNotMeetWhatItIsNotOver(t *testing.T) {
	field := pyramidField(t, 4, 4)
	for _, ray := range []maths.Ray{
		{At: maths.V(-100, 1.5, 0), Dir: maths.V(1, 0, 0)},
		{At: maths.V(-100, 0.5, 5), Dir: maths.V(1, 0, 0)},
		{At: maths.V(5, 0.5, -100), Dir: maths.V(0, 0, 1)},
		{At: maths.V(-100, -1, 0), Dir: maths.V(1, 0, 0)},
		{At: maths.V(10, 0.5, 0), Dir: maths.V(1, 0, 0)},
		{At: maths.V(0.25, 50, 0.25), Dir: maths.V(0, 1, 0)},
		{At: maths.V(0.25, -50, 0.25), Dir: maths.V(0, -1, 0)},
	} {
		misses(t, field, ray)
	}
}

// A ray whose hit is past the square's edge is a miss: nothing continues the
// surface there; and the same ray over a field wide enough to hold the hit
// meets it.
func TestHeightFieldHitLeavingOverTheEdgeIsAMiss(t *testing.T) {
	narrow := levelAcross(t, 100, 0)
	wide := levelAcross(t, seaExtent, 0)
	ray := downward(maths.V(0, grazeHeight, -40), 0.1)
	misses(t, narrow, ray)
	wideRay := downward(maths.V(0, grazeHeight, -seaStart), 0.1)
	meets(t, wide, wideRay, grazeHeight/math.Sin(0.1*math.Pi/180))
}
