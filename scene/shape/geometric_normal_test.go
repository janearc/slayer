package shape_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/mocks/shapemock"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"go.uber.org/mock/gomock"
)

// geometricNormalVectorFile is the table of points the tests of this file
// use.
const geometricNormalVectorFile = "testdata/geometric-normal-vectors.csv"

// geometricSolid is a shape that also says its geometric normal: a mock of
// each interface, put together.
type geometricSolid struct {
	*shapemock.MockShape
	*shapemock.MockGeometricNormaler
}

// geometricNormalHeight is the distance of a shape that is a floor: how
// high the point is above it.
func geometricNormalHeight(at maths.Vec) float64 { return at.Y }

// TestGeometricNormalAsksAShapeThatKnowsIt says that shape.GeometricNormal
// gives the answer of a shape that says its own geometric normal, asking it
// once at the point, and does not measure the distance.
func TestGeometricNormalAsksAShapeThatKnowsIt(t *testing.T) {
	controller := gomock.NewController(t)
	solid := geometricSolid{shapemock.NewMockShape(controller),
		shapemock.NewMockGeometricNormaler(controller)}
	point := groupVectorsIn(t, geometricNormalVectorFile).point("asked")
	known := solid.MockGeometricNormaler
	known.EXPECT().GeometricNormal(point).Return(maths.XAxis)
	if got := shape.GeometricNormal(solid, point); got != maths.XAxis {
		t.Errorf("geometric normal %v, want the one the shape gave", got)
	}
}

// TestGeometricNormalMeasuresAShapeThatDoesNotKnowIt is the anti-test of the
// one above: a shape that says no geometric normal and no normal is
// measured by its distance, six calls of it, a pair along each axis, and the
// answer is a unit vector along the way the distance grows.
func TestGeometricNormalMeasuresAShapeThatDoesNotKnowIt(t *testing.T) {
	measures := groupPickedCount(t, "geometric-normal-measures")
	controller := gomock.NewController(t)
	solid := shapemock.NewMockShape(controller)
	point := groupVectorsIn(t, geometricNormalVectorFile).point("asked")
	solid.EXPECT().Distance(gomock.Any()).DoAndReturn(
		geometricNormalHeight).Times(measures)
	got := shape.GeometricNormal(solid, point)
	tolerance := groupPicked(t, "geometric-normal-tolerance")
	unitTolerance := groupPicked(t, "geometric-normal-unit-tolerance")
	if !got.Near(maths.YAxis, tolerance) ||
		math.Abs(got.Len()-groupUnity) > unitTolerance {
		t.Errorf("measured normal %v, want up", got)
	}
}
