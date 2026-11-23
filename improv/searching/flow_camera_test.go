package searching

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

// TestFlowTravelKeepsScaleAndHolds pins endpoints, logarithmic distance and
// absolute-time sampling across the end of the camera move.
func TestFlowTravelKeepsScaleAndHolds(t *testing.T) {
	e := flowFixture()
	e.Plan.FlowStudy.Eye = [3]float64{0, 0, 2}
	e.Plan.FlowStudy.Travel = &flowTravel{To: [3]float64{0, 0, 50}, Until: "5.1"}
	if err := e.checkFlow(); err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct{ at, distance float64 }{{0, 2}, {8, 10}, {16, 50}, {25, 50}, {8, 10}} {
		eye, aim := e.flowCamera(sample.at)
		if !aim.Near(maths.Vec{}, 1e-12) || math.Abs(eye.Sub(aim).Len()-sample.distance) > 1e-10 {
			t.Fatalf("at %g eye %v aim %v", sample.at, eye, aim)
		}
	}
}

// TestFlowTravelRefusesSingularViews ensures malformed recipes fail before
// native camera construction and cannot move through their own target.
func TestFlowTravelRefusesSingularViews(t *testing.T) {
	for _, travel := range []flowTravel{
		{Until: "1.1", To: [3]float64{0, 0, 50}},
		{Until: "5.1"},
		{Until: "5.1", To: [3]float64{0, 0, -50}},
		{Until: "5.1", To: [3]float64{math.NaN(), 0, 50}},
		{Until: "7.1", To: [3]float64{0, 0, 50}},
	} {
		e := flowFixture()
		e.Plan.FlowStudy.Travel = &travel
		if e.checkFlow() == nil {
			t.Fatal("accepted singular or invalid camera", travel)
		}
	}
}
