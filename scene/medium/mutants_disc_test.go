package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// a disc runs from Inner to Outer and both are in it: with no profile,
// no Dense and no grain, it is one on a circle at Inner and one on a
// circle at Outer, in the middle of its thickness.
func TestADiscIncludesItsInnerAndOuterEdges(t *testing.T) {
	d := Disc{Inner: 2, Outer: 6, Thick: 0.4}
	for _, r := range []float64{2, 6} {
		if got := d.Density(maths.V(r, 0, 0)); got != 1 {
			t.Errorf("at radius %v the disc is %v, want 1", r, got)
		}
	}
}

// Dense is the density where the profile is one, so a Dense under one
// thins the disc: half of it is a half.
func TestADiscOfDenseAHalfIsAHalf(t *testing.T) {
	d := Disc{Inner: 2, Outer: 6, Thick: 0.4, Dense: 0.5}
	if got := d.Density(maths.V(4, 0, 0)); got != 0.5 {
		t.Errorf("a disc of Dense a half is %v, want 0.5", got)
	}
}

// a profile is its first entry at Inner and its last at Outer, and the entry
// between at the middle of three: of 1, 0.5, 0.25 that is 1, 0.5 and 0.25, at t
// of zero, a half and one.
func TestAProfileEndsAtItsLastEntry(t *testing.T) {
	d := Disc{Inner: 2, Outer: 6, Thick: 0.4, Profile: []float64{1, 0.5, 0.25}}
	for r, want := range map[float64]float64{2: 1, 4: 0.5, 6: 0.25} {
		if got := d.Density(maths.V(r, 0, 0)); got != want {
			t.Errorf("at radius %v the disc is %v, want %v", r, got, want)
		}
	}
}

// Ringlets is how many ringlets lie across the disc, so the noise is read
// at t times Ringlets and a disc of one ringlet at t of 0.6 is the disc
// of the default sixty at t of 0.01: both read the noise at 0.6. the
// default is not one.
func TestRingletsScaleTheNoiseAcrossTheDisc(t *testing.T) {
	one := Disc{Inner: 2, Outer: 6, Thick: 0.4, Grain: 1, Ringlets: 1, Seed: 3}
	sixty := Disc{Inner: 2, Outer: 6, Thick: 0.4, Grain: 1, Seed: 3}
	// t of 0.6 is 2 + 0.6 * 4 out, t of 0.01 is 2 + 0.01 * 4
	got, want := one.Density(maths.V(4.4, 0, 0)), sixty.Density(maths.V(2.04, 0, 0))
	if !closeTo(got, want, 1e-12) {
		t.Errorf("one ringlet at 0.6 is %v, sixty at 0.01 is %v", got, want)
	}
}

// Grain is a share from zero to one of how far the ringlets thin the disc, and
// a Grain past one is one: a density is never less than zero, and a Grain of
// three is the density of a Grain of one.
func TestGrainPastOneIsOne(t *testing.T) {
	whole := Disc{Inner: 2, Outer: 6, Thick: 0.4, Grain: 1, Seed: 5}
	over := whole
	over.Grain = 3
	for i := range 300 {
		at := maths.V(2+4*float64(i)/300, 0, 0)
		d := over.Density(at)
		if d < 0 || !closeTo(d, whole.Density(at), 1e-15) {
			t.Fatalf("at %v a Grain of three is %v, of one %v", at, d,
				whole.Density(at))
		}
	}
}
