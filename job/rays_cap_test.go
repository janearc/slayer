package job

import "testing"

// mostRays is the most rays along a side of a pixel the command line may
// ask for: sixty-four, 4,096 a pixel, so that a picture that is still
// grainy at sixteen can be drawn finer.
const mostRays = 64

// TestTheCommandLineMayAskForSixtyFourRaysASide holds the top of the
// range: an override of sixty-four rays a side is taken.
func TestTheCommandLineMayAskForSixtyFourRaysASide(t *testing.T) {
	job := read(t, film)
	over, err := job.With(Rays(mostRays))
	if err != nil {
		t.Fatal(err)
	}
	if rays, _ := over.Film().Rays(); rays != mostRays {
		t.Errorf("asked for %d rays a side, the job has %d", mostRays, rays)
	}
}

// TestMoreThanSixtyFourRaysASideIsRefused is the anti-test: one more than
// the top is refused by the film's rays.
// anti-test of TestTheCommandLineMayAskForSixtyFourRaysASide
func TestMoreThanSixtyFourRaysASideIsRefused(t *testing.T) {
	job := read(t, film)
	_, err := job.With(Rays(mostRays + 1))
	if err == nil {
		t.Fatal("sixty-five rays a side were taken")
	}
	saying(t, err, "/film/rays")
}
