package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// The tables the tests of this file read: the ray at the steep ramp, and the
// heights of the samples of that ramp.
const (
	shapePropertyVectorFile = "testdata/shape-property-vectors.csv"
	shapePropertyRampFile   = "testdata/shape-property-ramp.csv"
)

// shapePropertyRanges are the ranges the property tests draw from, each a
// choice of the test, read from numbers.yaml.
type shapePropertyRanges struct {
	// samples is the most samples a side of a drawn field has.
	samples int
	// height is how high or low a drawn height may be.
	height float64
	// extent is the widest a drawn field is, across.
	extent float64
	// leastExtent is the narrowest a drawn field is, across.
	leastExtent float64
	// rise is how far above the highest sample a drawn ray starts, at most.
	rise float64
	// leastStep is the least a drawn ray starts above the field, and the
	// least it leans downward.
	leastStep float64
	// far is how far a drawn ray is followed.
	far float64
	// slack is the share of the distance a bound may exceed the truth by,
	// for rounding.
	slack float64
	// reach is how far a drawn place in a shape's own space is from its
	// zero.
	reach float64
	// huge and tiny are the sizes of a coordinate that is neither ordinary
	// nor infinite.
	huge, tiny float64
}

// shapePropertyRangesIn reads the ranges the property tests draw from.
func shapePropertyRangesIn(t *testing.T) shapePropertyRanges {
	t.Helper()
	return shapePropertyRanges{
		samples:     pickedCount(t, "shape-property-samples"),
		height:      picked(t, "shape-property-height"),
		extent:      picked(t, "shape-property-extent"),
		leastExtent: picked(t, "shape-property-least-extent"),
		rise:        picked(t, "shape-property-rise"),
		leastStep:   picked(t, "shape-property-least-step"),
		far:         picked(t, "shape-property-far"),
		slack:       picked(t, "shape-property-slack"),
		reach:       picked(t, "shape-property-range"),
		huge:        picked(t, "shape-property-huge"),
		tiny:        picked(t, "shape-property-tiny"),
	}
}

// extremes are the coordinates that are not ordinary: not a number,
// infinite, the largest and the smallest there are, and huge and tiny.
func (ranges shapePropertyRanges) extremes() []float64 {
	return []float64{math.NaN(), math.Inf(unity), math.Inf(-unity),
		math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64,
		ranges.huge, -ranges.huge, ranges.tiny}
}

// coordinate draws a coordinate that is ordinary, huge, tiny, infinite or
// not a number.
func (ranges shapePropertyRanges) coordinate() *rapid.Generator[float64] {
	ordinary := rapid.Float64Range(-ranges.reach, ranges.reach)
	extreme := rapid.SampledFrom(ranges.extremes())
	return rapid.OneOf(ordinary, extreme)
}

// drawPoint draws a point whose every coordinate is a coordinate.
func (ranges shapePropertyRanges) drawPoint(t *rapid.T) maths.Vec {
	x := ranges.coordinate().Draw(t, "x")
	y := ranges.coordinate().Draw(t, "y")
	z := ranges.coordinate().Draw(t, "z")
	return maths.V(x, y, z)
}

// point draws a point whose every coordinate is a coordinate.
func (ranges shapePropertyRanges) point() *rapid.Generator[maths.Vec] {
	return rapid.Custom(ranges.drawPoint)
}

// drawnField draws a field of a few samples a side over a few metres.
func (ranges shapePropertyRanges) drawnField(t *rapid.T) *HeightField {
	samples := rapid.IntRange(leastSamples, ranges.samples).Draw(t, "samples")
	aHeight := rapid.Float64Range(-ranges.height, ranges.height)
	heights := rapid.SliceOfN(aHeight, samples*samples,
		samples*samples).Draw(t, "heights")
	across := rapid.Float64Range(ranges.leastExtent, ranges.extent)
	extent := across.Draw(t, "extent")
	field, err := NewHeightField(heights, samples, extent)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// pastTheTruth is how far the field's Distance at the ray's start exceeds how
// far along the ray the surface is, which is at least the true distance, and
// whether the ray meets the surface at all. A bound is never past it.
func (ranges shapePropertyRanges) pastTheTruth(field *HeightField,
	ray maths.Ray) (float64, bool) {
	along, hit := field.IntersectExact(ray, nought, ranges.far)
	if !hit {
		return nought, false
	}
	return field.Distance(ray.At) - along*ray.Dir.Len(), true
}

// shapePropertyDraw is what the property tests draw against: the ranges,
// and the shapes asked at the points and rays drawn.
type shapePropertyDraw struct {
	ranges shapePropertyRanges
	solids map[string]Shape
	field  *HeightField
}

// distanceNeverExceeds draws a field and a start above it with a direction
// down, and holds the Distance at the start to no more than the distance
// along the ray to where IntersectExact meets the surface.
func (draw shapePropertyDraw) distanceNeverExceeds(t *rapid.T) {
	ranges := draw.ranges
	field := ranges.drawnField(t)
	reach := field.extent * half
	across := rapid.Float64Range(-reach, reach)
	rise := rapid.Float64Range(ranges.leastStep, ranges.rise)
	x := across.Draw(t, "x")
	height := field.top + rise.Draw(t, "rise")
	z := across.Draw(t, "z")
	lean := rapid.Float64Range(-unity, unity)
	dx := lean.Draw(t, "dx")
	dy := -rapid.Float64Range(ranges.leastStep, unity).Draw(t, "dy")
	dz := lean.Draw(t, "dz")
	ray := maths.Ray{At: maths.V(x, height, z), Dir: maths.V(dx, dy, dz)}
	past, hit := ranges.pastTheTruth(field, ray)
	if hit && past > ranges.slack*(unity+field.Distance(ray.At)) {
		t.Fatalf("Distance %v past the surface along %+v by %v",
			field.Distance(ray.At), ray, past)
	}
}

// panicAt asks every shape for its distance and normal at a point drawn.
func (draw shapePropertyDraw) panicAt(t *rapid.T) {
	point := draw.ranges.point().Draw(t, "point")
	for _, solid := range draw.solids {
		solid.Distance(point)
		Normal(solid, point)
		GeometricNormal(solid, point)
		solid.Bounds()
	}
}

// exactHit asks the sphere, the plane and the field for the hit of a ray
// drawn: a hit each finds is a distance, not NaN and not behind the ray.
func (draw shapePropertyDraw) exactHit(t *rapid.T) {
	at := draw.ranges.point().Draw(t, "at")
	direction := draw.ranges.point().Draw(t, "dir")
	ray := maths.Ray{At: at, Dir: direction}
	ball := Sphere{Radius: unity}
	far := math.Inf(unity)
	met := func(what string, distance float64, found bool) {
		if found && (math.IsNaN(distance) || distance < nought) {
			t.Fatalf("%s met %v at %v", what, ray, distance)
		}
	}
	distance, found := ball.Hit(ray, nought, far)
	met("the sphere", distance, found)
	enter, leave, found := ball.Span(ray, nought, far)
	met("the sphere's entry", enter, found)
	met("the sphere's exit", leave, found)
	distance, found = Plane{}.Hit(ray, nought, far)
	met("the plane", distance, found)
	distance, found = draw.field.IntersectExact(ray, nought, far)
	met("the field", distance, found)
}

// shapePropertyFlat is a field with no height, a few samples a side, over a
// few metres.
func shapePropertyFlat(t *testing.T) *HeightField {
	t.Helper()
	samples := pickedCount(t, "shape-property-flat-samples")
	field, err := NewHeightField(make([]float64, samples*samples), samples,
		picked(t, "shape-property-flat-extent"))
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// TestAHeightFieldDistanceNeverExceedsTheTruth says, for fields, starts above
// them and directions down, that the Distance at the start is no more than the
// distance along the ray to where IntersectExact meets the surface.
func TestAHeightFieldDistanceNeverExceedsTheTruth(t *testing.T) {
	draw := shapePropertyDraw{ranges: shapePropertyRangesIn(t)}
	rapid.Check(t, draw.distanceNeverExceeds)
}

// TestADistanceThatIgnoresTheSlopeIsCaught is the anti-test of the property
// above: the same check, run on a field whose slant is one, as if it were
// flat, finds a steep ramp's Distance past the truth. Worked by hand: a ramp
// rising ten metres over one, a ray from a metre over its foot, leaning
// toward its high side, meets it at half a metre along the vertical, and
// the true distance to the ramp is a tenth of a metre.
func TestADistanceThatIgnoresTheSlopeIsCaught(t *testing.T) {
	ranges := shapePropertyRangesIn(t)
	var heights []float64
	for _, each := range readSheet(t, shapePropertyRampFile) {
		heights = append(heights, cell(t, each, "height"))
	}
	ramp, err := NewHeightField(heights, pair,
		picked(t, "shape-property-ramp-extent"))
	if err != nil {
		t.Fatal(err)
	}
	ray := shapeVectorsIn(t, shapePropertyVectorFile).ray("ramp-ray")
	past, hit := ranges.pastTheTruth(ramp, ray)
	if !hit || past > ranges.slack {
		t.Fatalf("the ramp: %v past, %v; want the bound within the truth",
			past, hit)
	}
	flattened := *ramp
	flattened.slant = unity
	past, hit = ranges.pastTheTruth(&flattened, ray)
	if !hit || past <= ranges.slack {
		t.Errorf("a field with no slant: %v past, %v; want it caught", past,
			hit)
	}
}

// TestNoShapePanicsAtAnyPoint says that none of the shapes panics when
// asked for its distance or its normal at a point that is ordinary, huge,
// tiny, infinite or not a number.
func TestNoShapePanicsAtAnyPoint(t *testing.T) {
	tube := Tube{Radius: picked(t, "shape-property-tube-radius"),
		Thick:  picked(t, "shape-property-tube-wall"),
		Height: picked(t, "shape-property-tube-length")}
	draw := shapePropertyDraw{ranges: shapePropertyRangesIn(t),
		solids: map[string]Shape{
			"sphere": Sphere{Radius: unity},
			"plane":  Plane{},
			"tube":   tube,
			"field":  shapePropertyFlat(t),
			"group":  plateRow(t),
		}}
	rapid.Check(t, draw.panicAt)
}

// TestNoRayPanicsAnExactHit says that no sphere, plane or field panics when
// asked for the hit of a ray whose place and direction are ordinary, huge,
// tiny, infinite or not a number.
func TestNoRayPanicsAnExactHit(t *testing.T) {
	draw := shapePropertyDraw{ranges: shapePropertyRangesIn(t),
		field: shapePropertyFlat(t)}
	rapid.Check(t, draw.exactHit)
}
