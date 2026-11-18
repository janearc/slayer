package searching

import (
	"math"
	"testing"

	"github.com/janearc/slayer/beats"
)

// halfSecondEdit is an edit over eighty beats half a second apart, so beat
// n falls at n/2 seconds and a bar lasts two seconds. Lock is at bar five,
// radio at bars seven and nine, the reveal at bar eleven and the coda at
// bar seventeen.
func halfSecondEdit() *Edit {
	times := make([]float64, 80)
	marks := make([]int, 80)
	for index := range times {
		times[index] = float64(index) / 2
		marks[index] = 1
	}
	return &Edit{
		Plan: Plan{FPS: 4, SearchCutBars: 2, BeatMotion: []float64{1, 0.5, 0.25, 0.125},
			Azimuth: 25, SearchDays: 2},
		Grid: &beats.Grid{Score: beats.Score{Duration: 40, Meter: 4, Status: "estimated",
			Beats: beats.Pulse{Times: times, Measured: marks}}},
		Lock: 8, RadioOne: 12, RadioTwo: 16, Reveal: 20, Coda: 32, Accents: []float64{3},
	}
}

// within reports whether two numbers agree to a tolerance.
func within(got, want, tolerance float64) bool { return math.Abs(got-want) <= tolerance }

// TestFramesRoundsUp checks the frame count for a length that is not a
// whole number of frames.
func TestFramesRoundsUp(t *testing.T) {
	edit := halfSecondEdit()
	if edit.Frames() != 160 {
		t.Fatalf("%d frames for forty seconds at four a second", edit.Frames())
	}
	edit.Grid.Score.Duration = 40.1
	if edit.Frames() != 161 {
		t.Fatalf("%d frames for 40.1 seconds at four a second", edit.Frames())
	}
}

// TestSearchCutsBetweenSitesEveryTwoBars checks the site and the beat-four
// colour drain in the search, with positions worked by hand.
func TestSearchCutsBetweenSitesEveryTwoBars(t *testing.T) {
	edit := halfSecondEdit()
	for _, each := range []struct {
		at         float64
		site       string
		saturation float64
	}{
		{0.25, "hawaii", 1}, {1.75, "hawaii", 0.825}, {3.9, "hawaii", 1 - 0.35*0.8},
		{4.5, "atacama", 1}, {7.9, "atacama", 1 - 0.35*0.8},
	} {
		state := edit.At(each.at)
		if state.Site != each.site || state.Chapter != "search" || !within(state.Saturation, each.saturation, 1e-9) {
			t.Errorf("At(%g) = %s, %s, saturation %g; wanted %s, search, %g",
				each.at, state.Site, state.Chapter, state.Saturation, each.site, each.saturation)
		}
	}
}

// TestFlashFollowsDownbeatsAndAccents checks the flash at a downbeat, off
// a beat, at an accent, and where an accent sits on a downbeat.
func TestFlashFollowsDownbeatsAndAccents(t *testing.T) {
	edit := halfSecondEdit()
	for _, each := range []struct{ at, want float64 }{
		{0, 1}, {2, 1}, {1, 0}, {3, 0.5}, {2.5, 0},
	} {
		if got := edit.At(each.at).Flash; got != each.want {
			t.Errorf("At(%g).Flash = %g, wanted %g", each.at, got, each.want)
		}
	}
	edit.Accents = []float64{2}
	if got := edit.At(2).Flash; got != 0.5 {
		t.Errorf("an accent on a downbeat gave %g, wanted 0.5", got)
	}
}

// TestSearchAimSwingsThenSettles checks the aim before the settle, inside
// it and at the lock against values worked from the sine by hand.
func TestSearchAimSwingsThenSettles(t *testing.T) {
	edit := halfSecondEdit()
	for _, each := range []struct{ at, aim float64 }{
		{1, 43.275234},  // camera 1.5, site hawaii
		{4.5, 58.36824}, // camera 4.75, site atacama
		{7, 36.838588},  // half way through the settle, from atacama
		{8, 25},
	} {
		if got := edit.At(each.at).Aim; !within(got, each.aim, 1e-5) {
			t.Errorf("At(%g).Aim = %g, wanted %g", each.at, got, each.aim)
		}
	}
	state := edit.At(4)
	if want := 0.5 * 2 * 86400; state.SceneSeconds != want {
		t.Errorf("scenery clock %g at the half way point, wanted %g", state.SceneSeconds, want)
	}
	if got := edit.At(30).SceneSeconds; got != 2*86400 {
		t.Errorf("scenery clock after the lock %g, wanted two days", got)
	}
}

// TestLockHoldsAndAlternatesSites checks the lock chapter's camera drift
// and its site on even and odd bars.
func TestLockHoldsAndAlternatesSites(t *testing.T) {
	edit := halfSecondEdit()
	first, second := edit.At(8), edit.At(10)
	if first.Chapter != "lock" || first.Site != "hawaii" || second.Site != "atacama" {
		t.Fatalf("lock: %s %s then %s", first.Chapter, first.Site, second.Site)
	}
	if !within(first.Camera, 7.5, 1e-9) || !within(second.Camera, 7.5+2*0.03, 1e-9) {
		t.Fatalf("camera %g then %g", first.Camera, second.Camera)
	}
}

// TestRadioChaptersSlewToTheSharedAzimuth checks the radio arrays' aim at
// the start of each slew and half way through it.
func TestRadioChaptersSlewToTheSharedAzimuth(t *testing.T) {
	edit := halfSecondEdit()
	for _, each := range []struct {
		at   float64
		site string
		aim  float64
	}{
		{12, "radio-hawaii", 25 - 90}, {15, "radio-hawaii", 25 - 45},
		{16, "radio-atacama", 25 + 100}, {19, "radio-atacama", 25 + 50},
	} {
		state := edit.At(each.at)
		if state.Site != each.site || state.Chapter != "radio response" || !within(state.Aim, each.aim, 1e-9) {
			t.Errorf("At(%g) = %s, %q, %g; wanted %s, aim %g", each.at, state.Site, state.Chapter, state.Aim, each.site, each.aim)
		}
	}
}

// TestRevealCyclesThroughTheSpectrum checks the blazar views bar by bar,
// and the wrap back to the first.
func TestRevealCyclesThroughTheSpectrum(t *testing.T) {
	edit := halfSecondEdit()
	for index, want := range []string{"radio", "infrared", "visible", "x-ray", "gamma", "radio"} {
		at := 20 + 2*float64(index)
		state := edit.At(at)
		if state.Site != "blazar" || state.Chapter != "reveal" || state.Spectrum != want {
			t.Errorf("At(%g) = %s, %s, %q; wanted blazar, reveal, %q", at, state.Site, state.Chapter, state.Spectrum, want)
		}
	}
}

// TestCodaReturnsToTheSummits checks the coda's site on even and odd bars
// and that it clears the flash.
func TestCodaReturnsToTheSummits(t *testing.T) {
	edit := halfSecondEdit()
	edit.Accents = []float64{34}
	for _, each := range []struct {
		at   float64
		site string
	}{{32, "hawaii"}, {34, "atacama"}} {
		state := edit.At(each.at)
		if state.Chapter != "coda" || state.Site != each.site || state.Aim != 25 || state.Flash != 0 {
			t.Errorf("At(%g) = %+v", each.at, state)
		}
	}
}

// TestSearchBeforeTheFirstBeatStaysInTheFirstBar checks a time before the
// score starts.
func TestSearchBeforeTheFirstBeatStaysInTheFirstBar(t *testing.T) {
	state := halfSecondEdit().At(-0.25)
	if state.Site != "hawaii" || state.Chapter != "search" || state.Saturation != 1 {
		t.Fatalf("state %+v", state)
	}
}

// TestSmoothIsACubicEaseClampedToItsEnds checks the ease at its ends, its
// middle and outside them.
func TestSmoothIsACubicEaseClampedToItsEnds(t *testing.T) {
	for _, each := range []struct{ in, want float64 }{
		{-3, 0}, {0, 0}, {0.25, 0.15625}, {0.5, 0.5}, {0.75, 0.84375}, {1, 1}, {7, 1},
	} {
		if got := smooth(each.in); !within(got, each.want, 1e-12) {
			t.Errorf("smooth(%g) = %g, wanted %g", each.in, got, each.want)
		}
	}
}
