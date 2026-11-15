package thing

import "testing"

// A gloss above one mirrors no more than a gloss of one: SplitsBetween
// holds Gloss to between nought and one. Water against glass, face on.
func TestAGlossAboveOneSplitsAsOne(t *testing.T) {
	const relative = 1.5 / 1.33 // defined: glass over water, the tracer's own test indices
	full := &Material{Gloss: 1}
	over := &Material{Gloss: 1.7}
	want := full.SplitsBetween(1, relative)
	if !(want > 0) {
		t.Fatalf("a full gloss splits off %v face on: nothing to compare", want)
	}
	if got := over.SplitsBetween(1, relative); got != want {
		t.Errorf("a gloss of 1.7 splits off %v, want a gloss of one's %v", got, want)
	}
	// the anti-test: a gloss below one does split off less
	half := &Material{Gloss: 0.5}
	if got := half.SplitsBetween(1, relative); !(got < want) {
		t.Errorf("a gloss of a half splits off %v, not less than %v", got, want)
	}
}
