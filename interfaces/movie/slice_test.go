package movie

import "testing"

// TestSoundSliceOwnsOnlyItsInterval checks channel order, exact bounds and
// independent storage. Mutating the slice cannot alter its source.
func TestSoundSliceOwnsOnlyItsInterval(t *testing.T) {
	source, err := NewSound(44100, []float32{.1, .2, .3, .4, .5, .6})
	if err != nil {
		t.Fatal(err)
	}
	cut, err := source.Slice(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if cut.Rate() != 44100 || cut.Length() != 2 || cut.samples[0] != .3 || cut.samples[3] != .6 {
		t.Fatal("wrong interval", cut)
	}
	cut.samples[0] = 0
	if source.samples[2] != .3 {
		t.Fatal("slice aliases source")
	}
	for _, span := range [][2]int{{-1, 1}, {1, 1}, {2, 1}, {0, 4}} {
		if _, err := source.Slice(span[0], span[1]); err == nil {
			t.Fatal("accepted", span)
		}
	}
	if _, err := (Sound{}).Slice(0, 1); err == nil {
		t.Fatal("accepted absent sound")
	}
}
