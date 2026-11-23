package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// mutantsDiscBase is the disc of these tests with no profile, no Dense and no
// grain: from Inner to Outer, Thick across, read from the chosen numbers.
func mutantsDiscBase(t *testing.T) Disc {
	t.Helper()
	return Disc{Inner: picked(t, "mutants-disc-inner"),
		Outer: picked(t, "mutants-disc-outer"),
		Thick: picked(t, "mutants-disc-thick")}
}

// mutantsDiscOut is the point at a radius out along x, in the middle of the
// disc's thickness.
func mutantsDiscOut(radius float64) maths.Vec {
	return maths.V(radius, nought, nought)
}

// a disc runs from Inner to Outer and both are in it: with no profile,
// no Dense and no grain, it is one on a circle at Inner and one on a
// circle at Outer, in the middle of its thickness.
func TestADiscIncludesItsInnerAndOuterEdges(t *testing.T) {
	disc := mutantsDiscBase(t)
	for _, radius := range []float64{disc.Inner, disc.Outer} {
		if got := disc.Density(mutantsDiscOut(radius)); got != unity {
			t.Errorf("at radius %v the disc is %v, want 1", radius, got)
		}
	}
}

// Dense is the density where the profile is one, so a Dense under one
// thins the disc: half of it is a half.
func TestADiscOfDenseAHalfIsAHalf(t *testing.T) {
	disc := mutantsDiscBase(t)
	disc.Dense = halfway
	middle := mutantsDiscOut(picked(t, "mutants-disc-middle"))
	if got := disc.Density(middle); got != halfway {
		t.Errorf("a disc of Dense a half is %v, want 0.5", got)
	}
}

// a profile is its first entry at Inner and its last at Outer, and the entry
// between at the middle of three: of 1, 0.5, 0.25 that is 1, 0.5 and 0.25, at t
// of zero, a half and one.
func TestAProfileEndsAtItsLastEntry(t *testing.T) {
	disc := mutantsDiscBase(t)
	last := picked(t, "mutants-disc-profile-last")
	disc.Profile = []float64{unity, halfway, last}
	middle := picked(t, "mutants-disc-middle")
	wanted := map[float64]float64{disc.Inner: unity, middle: halfway,
		disc.Outer: last}
	for radius, want := range wanted {
		if got := disc.Density(mutantsDiscOut(radius)); got != want {
			t.Errorf("at radius %v the disc is %v, want %v", radius, got, want)
		}
	}
}

// Ringlets is how many ringlets lie across the disc, so the noise is read
// at t times Ringlets and a disc of one ringlet at t of 0.6 is the disc
// of the default sixty at t of 0.01: both read the noise at 0.6. the
// default is not one.
func TestRingletsScaleTheNoiseAcrossTheDisc(t *testing.T) {
	one := mutantsDiscBase(t)
	one.Grain, one.Ringlets = unity, unity
	one.Seed = uint64(pickedCount(t, "mutants-disc-ringlets-seed"))
	sixty := mutantsDiscBase(t)
	sixty.Grain = unity
	sixty.Seed = one.Seed
	// t of 0.6 is 2 + 0.6 * 4 out, t of 0.01 is 2 + 0.01 * 4
	oneRadius := picked(t, "mutants-disc-ringlet-one-radius")
	sixtyRadius := picked(t, "mutants-disc-ringlet-sixty-radius")
	got := one.Density(mutantsDiscOut(oneRadius))
	want := sixty.Density(mutantsDiscOut(sixtyRadius))
	if !closeTo(got, want, picked(t, "mutants-disc-tolerance")) {
		t.Errorf("one ringlet at 0.6 is %v, sixty at 0.01 is %v", got, want)
	}
}

// Grain is a share from zero to one of how far the ringlets thin the disc, and
// a Grain past one is one: a density is never less than zero, and a Grain of
// three is the density of a Grain of one.
func TestGrainPastOneIsOne(t *testing.T) {
	whole := mutantsDiscBase(t)
	whole.Grain = unity
	whole.Seed = uint64(pickedCount(t, "mutants-disc-grain-seed"))
	over := whole
	over.Grain = picked(t, "mutants-disc-grain-over")
	span := picked(t, "mutants-disc-sweep-span")
	samples := pickedCount(t, "mutants-disc-sweep-samples")
	for i := range samples {
		at := maths.V(whole.Inner+span*float64(i)/float64(samples), nought,
			nought)
		density := over.Density(at)
		if density < nought || !closeTo(density, whole.Density(at),
			picked(t, "mutants-disc-grain-tolerance")) {
			t.Fatalf("at %v a Grain of three is %v, of one %v", at, density,
				whole.Density(at))
		}
	}
}
