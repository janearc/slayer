package searching

import (
	"math"
	"testing"

	"github.com/janearc/slayer/beats"
	"github.com/janearc/slayer/maths"
)

// flowFixture uses unevenly spaced vertices so index-based travel would
// disagree with the distance travelled by the light.
func flowFixture() *Edit {
	times := make([]float64, 64)
	for index := range times {
		times[index] = float64(index)
	}
	return &Edit{Grid: &beats.Grid{Score: beats.Score{Duration: 64,
		Beats: beats.Pulse{Times: times}}}, Plan: Plan{FlowStudy: &FlowPlan{
		Provenance: "authored-previs", Start: "1.1", Fold: "5.1", Answer: "7.1",
		Eye: [3]float64{0, 0, 50}, Wide: 72,
		Paths: []flowPath{{Width: 0.02,
			Points: [][3]float64{{10, 0, 0}, {9, 0, 0}, {6, 0, 0}}}},
		Landmarks: []flowPoint{{At: [3]float64{8, 2, -4}, Radius: 0.1,
			RGB: [3]float64{1, 1, 1}}},
	}}}
}

// TestFlowTravelUsesDistance pins a nonuniform path's physical fractions.
func TestFlowTravelUsesDistance(t *testing.T) {
	got := fractions(flowFixture().Plan.FlowStudy.Paths[0].Points)
	for index, want := range []float64{0, 0.25, 1} {
		if got[index] != want {
			t.Fatalf("vertex %d distance fraction %g, want %g", index, got[index], want)
		}
	}
	if packet(0, 1.2, 0.1) != 0 || packet(1, -0.2, 0.1) != 0 {
		t.Fatal("a pulse wrapped or leaked beyond the polyline")
	}
}

// TestFlowKeepsLandmarksAndJoinedEndpoints checks the actual world's
// transforms before and during the fold, including out-of-order sampling.
func TestFlowKeepsLandmarksAndJoinedEndpoints(t *testing.T) {
	edit := flowFixture()
	if err := edit.checkFlow(); err != nil {
		t.Fatal(err)
	}
	set, err := newFlow(edit.Plan.FlowStudy)
	if err != nil {
		t.Fatal(err)
	}
	landmark := set.world.Things[len(set.segments)]
	fixed := landmark.At(0)
	for _, at := range []float64{0, 18, 30, 18} {
		set.update(edit, at)
		if landmark.At(0) != fixed {
			t.Fatal("reference landmark moved during the fold")
		}
		left, right := set.segments[0].object.At(0), set.segments[1].object.At(0)
		if !left.Out(maths.V(0, 0, -1)).Near(right.Origin, 1e-12) {
			t.Fatal("adjacent filament segments separated")
		}
	}
	first := set.segments[1].object.At(0)
	set.update(edit, 4)
	set.update(edit, 18)
	if set.segments[1].object.At(0) != first {
		t.Fatal("fold depends on previous frame order")
	}
}

// TestFlowReturnTravelsOutward observes the materials after the inward
// pulses have expired: first the inner segment, then the outer one lights.
func TestFlowReturnTravelsOutward(t *testing.T) {
	edit := flowFixture()
	if err := edit.checkFlow(); err != nil {
		t.Fatal(err)
	}
	set, err := newFlow(edit.Plan.FlowStudy)
	if err != nil {
		t.Fatal(err)
	}
	set.update(edit, 26.25) // centre .625, inner segment's midpoint
	inner := set.segments[1].object.Material.Glow.Luminance()
	outer := set.segments[0].object.Material.Glow.Luminance()
	if inner <= outer+1 {
		t.Fatal("return did not start on the inner segment")
	}
	set.update(edit, 29.25) // centre .125, outer segment's midpoint
	if set.segments[0].object.Material.Glow.Luminance() <=
		set.segments[1].object.Material.Glow.Luminance()+1 {
		t.Fatal("return did not reach the outer segment")
	}
}

// TestFlowRefusesFalseProvenanceAndInvalidGeometry exercises preflight,
// including NaN, repeated vertices, invalid cues and excessive inputs.
func TestFlowRefusesFalseProvenanceAndInvalidGeometry(t *testing.T) {
	changes := []func(*FlowPlan){
		func(p *FlowPlan) { p.Provenance = "measured Laniakea" },
		func(p *FlowPlan) { p.Paths[0].Points[1] = p.Paths[0].Points[0] },
		func(p *FlowPlan) { p.Paths[0].Width = math.NaN() },
		func(p *FlowPlan) { p.Eye[0] = math.Inf(1) },
		func(p *FlowPlan) { p.Landmarks[0].Radius = 0 },
		func(p *FlowPlan) { p.Landmarks[0].RGB[0] = -1 },
		func(p *FlowPlan) { p.Fold = p.Start },
		func(p *FlowPlan) { p.Answer = "18.1" },
		func(p *FlowPlan) { p.ReturnPath = len(p.Paths) },
		func(p *FlowPlan) { p.Paths = make([]flowPath, 129) },
		func(p *FlowPlan) { p.Clouds = make([]flowCloud, 49) },
		func(p *FlowPlan) {
			p.Clouds = []flowCloud{{Radius: 1, Density: math.Inf(1)}}
		},
		func(p *FlowPlan) {
			p.Clouds = []flowCloud{{Radius: 1, RGB: [3]float64{math.NaN()}}}
		},
	}
	for index, change := range changes {
		edit := flowFixture()
		change(edit.Plan.FlowStudy)
		if edit.checkFlow() == nil {
			t.Fatalf("accepted invalid flow plan %d", index)
		}
	}
}
