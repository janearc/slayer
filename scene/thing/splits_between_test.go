package thing

import "testing"

// A gloss above one mirrors no more than a gloss of one: SplitsBetween
// holds Gloss to between nought and one. Water against glass, face on.
func TestAGlossAboveOneSplitsAsOne(t *testing.T) {
	relative := picked(t, "splits-between-glass-index") /
		picked(t, "splits-between-water-index")
	full := &Material{Gloss: unity}
	over := &Material{Gloss: picked(t, "splits-between-over-gloss")}
	want := full.SplitsBetween(unity, relative)
	if !(want > nought) {
		t.Fatalf("a full gloss splits off %v face on: nothing to compare", want)
	}
	if got := over.SplitsBetween(unity, relative); got != want {
		t.Errorf("a gloss of 1.7 splits off %v, want a gloss of one's %v",
			got, want)
	}
	// the anti-test: a gloss below one does split off less
	dim := &Material{Gloss: picked(t, "splits-between-half-gloss")}
	if got := dim.SplitsBetween(unity, relative); !(got < want) {
		t.Errorf("a gloss of a half splits off %v, not less than %v", got, want)
	}
}
