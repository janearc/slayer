package movie

import (
	"slices"
	"testing"
)

// every slice in the table of taken ones holds exactly the instants asked
// for, in channel order, at the source's rate, and owns its samples:
// clearing the slice leaves the source as it was.
func TestSoundSliceOwnsOnlyItsInterval(t *testing.T) {
	for _, row := range casesOf(t, slicesTakenFile) {
		source := soundIn(t, row)
		given := samplesIn(t, row, "samples")
		want := samplesIn(t, row, "want")
		cut, err := source.Slice(wholeIn(t, row, "first"),
			wholeIn(t, row, "last"))
		if err != nil {
			t.Errorf("%s: %v", row["case"], err)
			continue
		}
		if cut.Rate() != source.Rate() || !slices.Equal(cut.samples, want) {
			t.Errorf("%s: %v at %d, want %v at %d", row["case"],
				cut.samples, cut.Rate(), want, source.Rate())
		}
		clear(cut.samples)
		if !slices.Equal(source.samples, given) {
			t.Errorf("%s: the slice aliases its source", row["case"])
		}
	}
}

// the anti-test: every slice in the table of refused ones is refused, and
// so is any slice of no sound.
func TestSoundSliceRefusesWhatIsNotAnInterval(t *testing.T) {
	for _, row := range casesOf(t, slicesRefusedFile) {
		first, last := wholeIn(t, row, "first"), wholeIn(t, row, "last")
		if _, err := soundIn(t, row).Slice(first, last); err == nil {
			t.Errorf("%s: [%d,%d) accepted", row["case"], first, last)
		}
	}
	if _, err := (Sound{}).Slice(zero, one); err == nil {
		t.Errorf("a slice of no sound accepted")
	}
}
