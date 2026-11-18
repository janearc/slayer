package searching

import "testing"

// TestRevealSurvivesTheCoda catches the old return to the observatory sets
// once the music passed the coda boundary.
func TestRevealSurvivesTheCoda(t *testing.T) {
	edit := &Edit{Plan: Plan{FullCut: true, FlowStudy: &FlowPlan{}}, FlowStart: 180, Coda: 220}
	for _, at := range []float64{180, 219, 220, 244} {
		if !edit.FlowAt(at) {
			t.Fatalf("lost reveal at %g", at)
		}
	}
	if edit.FlowAt(179.99) {
		t.Fatal("reveal started early")
	}
	edit.Plan.FlowStudy = nil
	if edit.FlowAt(244) {
		t.Fatal("selected absent reveal")
	}
}
