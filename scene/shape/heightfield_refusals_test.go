package shape

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// What the tests of this file count in, each true by definition.
const (
	heightfieldRefusalsFirst = 0 // defined: the first of a list
)

// heightfieldRefusalsVectorFile is the table of points the tests of this
// file ask about.
const heightfieldRefusalsVectorFile = "testdata/heightfield-refusals-vectors.csv"

// heightfieldRefusalsRecover is deferred by refusal: it records in message
// what the function panicked with, and leaves it alone when there is no
// panic.
func heightfieldRefusalsRecover(message *string) {
	recovered := recover()
	if recovered == nil {
		return
	}
	words, _ := recovered.(string)
	if words == "" {
		words = "a panic that is not words"
	}
	*message = words
}

// refusal is what ask panics with, and empty when it does not panic.
func refusal(ask func()) (message string) {
	defer heightfieldRefusalsRecover(&message)
	ask()
	return ""
}

// heightfieldRefusalsQuery is a question put to a field at a point: each
// method asks one, and distance keeps its answer.
type heightfieldRefusalsQuery struct {
	field  *HeightField
	point  maths.Vec
	answer float64
}

// distance asks the field for its distance at the point.
func (query *heightfieldRefusalsQuery) distance() {
	query.answer = query.field.Distance(query.point)
}

// normal asks the field for its normal at the point.
func (query *heightfieldRefusalsQuery) normal() {
	query.field.Normal(query.point)
}

// geometry asks the field for its geometric normal at the point.
func (query *heightfieldRefusalsQuery) geometry() {
	query.field.GeometricNormal(query.point)
}

// asks are the three questions of a query, by what each asks for.
func (query *heightfieldRefusalsQuery) asks() map[string]func() {
	return map[string]func(){"distance": query.distance,
		"normal": query.normal, "geometry": query.geometry}
}

// sloped is a field of four samples a side, a metre across, that rises along
// x by a tenth of a metre a sample and does not change along z.
func sloped(t *testing.T) *HeightField {
	t.Helper()
	samples := pickedCount(t, "heightfield-refusals-samples")
	rise := picked(t, "heightfield-refusals-rise")
	heights := make([]float64, samples*samples)
	for row := range samples {
		for column := range samples {
			heights[row*samples+column] = rise * float64(column)
		}
	}
	field, err := NewHeightField(heights, samples,
		picked(t, "heightfield-refusals-extent"))
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// TestAHeightFieldMadeByNoOneRefusesInWords says that a field that was not
// made by NewHeightField refuses a query in words, as a Mesh does, and does
// not fail on a bare index.
func TestAHeightFieldMadeByNoOneRefusesInWords(t *testing.T) {
	var empty HeightField
	origin := maths.V(nought, nought, nought)
	query := &heightfieldRefusalsQuery{field: &empty, point: origin}
	for why, ask := range query.asks() {
		got := refusal(ask)
		if !strings.Contains(got, "made by NewHeightField") {
			t.Errorf("%s: refused with %q, want words about NewHeightField",
				why, got)
		}
	}
}

// TestAMadeHeightFieldDoesNotRefuse is the anti-test of the one above: a
// field made by NewHeightField answers the same queries without a panic.
func TestAMadeHeightFieldDoesNotRefuse(t *testing.T) {
	origin := maths.V(nought, nought, nought)
	query := &heightfieldRefusalsQuery{field: sloped(t), point: origin}
	for why, ask := range query.asks() {
		if got := refusal(ask); got != "" {
			t.Errorf("%s: panicked with %q", why, got)
		}
	}
}

// TestAHeightFieldAnswersAPlaceThatIsNoNumberWithNoNumber says that a point
// with an x or z that is not a number does not panic and does not get a
// plausible answer: its distance is not a number.
func TestAHeightFieldAnswersAPlaceThatIsNoNumberWithNoNumber(t *testing.T) {
	field := sloped(t)
	nan := math.NaN()
	for _, point := range []maths.Vec{maths.V(nan, nought, nought),
		maths.V(nought, nought, nan), maths.V(nan, nought, nan)} {
		query := &heightfieldRefusalsQuery{field: field, point: point}
		if got := refusal(query.distance); got != "" {
			t.Errorf("%v: panicked with %q", point, got)
		}
		if !math.IsNaN(query.answer) {
			t.Errorf("%v: distance %v, want not a number", point, query.answer)
		}
		if got := refusal(query.geometry); got != "" {
			t.Errorf("%v: normal panicked with %q", point, got)
		}
	}
}

// TestAHeightFieldAnswersAPlaceInTheSquareWithANumber is the anti-test of
// the one above: a place in the square has a distance that is a number.
func TestAHeightFieldAnswersAPlaceInTheSquareWithANumber(t *testing.T) {
	field := sloped(t)
	vectors := shapeVectorsIn(t, heightfieldRefusalsVectorFile)
	if got := field.Distance(vectors.point("in-the-square")); math.IsNaN(got) {
		t.Error("a place in the square has no distance")
	}
}

// TestAFieldWhoseCellsHaveNoWidthIsRefused says that an extent so small that
// the step between samples rounds to nothing is refused in words. Worked: the
// smallest positive number over two cells is half of itself, which rounds
// to nought.
func TestAFieldWhoseCellsHaveNoWidthIsRefused(t *testing.T) {
	samples := pickedCount(t, "heightfield-refusals-small-samples")
	heights := make([]float64, samples*samples)
	tiny := math.SmallestNonzeroFloat64
	field, err := NewHeightField(heights, samples, tiny)
	if err == nil || field != nil ||
		!strings.Contains(err.Error(), "no width") {
		t.Errorf("made %v, %v; want a refusal about the cell width", field, err)
	}
}

// TestAFieldWhoseCellsHaveAWidthIsMade is the anti-test of the one above:
// the same heights over an extent of a metre make a field with a slant that
// is a number.
func TestAFieldWhoseCellsHaveAWidthIsMade(t *testing.T) {
	samples := pickedCount(t, "heightfield-refusals-small-samples")
	field, err := NewHeightField(make([]float64, samples*samples), samples,
		picked(t, "heightfield-refusals-extent"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, slant := field.Pack(); math.IsNaN(slant) {
		t.Error("the slant of a flat field is not a number")
	}
}

// TestASmoothedFieldSharesItsHeights says that the copy WithApproximate
// makes for shading holds the same heights as the field it was made from, not
// a second set, whichever slopes it was given.
func TestASmoothedFieldSharesItsHeights(t *testing.T) {
	first := heightfieldRefusalsFirst
	plain := sloped(t)
	smooth, err := plain.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	slopes := make([]float64, len(plain.heights))
	spectral, err := plain.WithApproximateSpectralNormals(slopes, slopes)
	if err != nil {
		t.Fatal(err)
	}
	if &smooth.heights[first] != &plain.heights[first] ||
		&spectral.heights[first] != &plain.heights[first] {
		t.Error("a shading copy holds its own heights")
	}
}

// TestSpectralSlopesAreTheFieldsOwnCopy is the anti-test of the one above:
// the slopes a caller supplies are copied, so that the caller writing them
// afterward does not move the field's shading.
func TestSpectralSlopesAreTheFieldsOwnCopy(t *testing.T) {
	first := heightfieldRefusalsFirst
	plain := sloped(t)
	slopes := make([]float64, len(plain.heights))
	spectral, err := plain.WithApproximateSpectralNormals(slopes, slopes)
	if err != nil {
		t.Fatal(err)
	}
	slopes[first] = unity
	if spectral.smoothX[first] != nought {
		t.Error("a caller's write moved the field's slope")
	}
}

// TestShadingSlopesAreTheFieldsOwn says that ApproximateShadingSlopes gives
// the field's slopes and not a copy of each, and none for a plain field.
func TestShadingSlopesAreTheFieldsOwn(t *testing.T) {
	first := heightfieldRefusalsFirst
	plain := sloped(t)
	smooth, err := plain.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	x, z, ok := smooth.ApproximateShadingSlopes()
	if !ok || &x[first] != &smooth.smoothX[first] ||
		&z[first] != &smooth.smoothZ[first] {
		t.Error("the slopes are a copy, or are missing")
	}
	if _, _, ok := plain.ApproximateShadingSlopes(); ok {
		t.Error("a plain field has shading slopes")
	}
}
