package job

import (
	"math"
	"testing"

	sensor "github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// TestTallIsWhatAPinholeSeesDownThePicture holds WideFor to film's own
// arithmetic: a pinhole as wide as WideFor says sees tall down the
// picture, for a portrait frame and a wide one.
func TestTallIsWhatAPinholeSeesDownThePicture(t *testing.T) {
	made := read(t, still+"film: {tall: 60}\n").Film()
	tall, _ := made.Tall()
	for _, preset := range []string{"portrait", "4k"} {
		chosen, _ := presetNamed(preset)
		width, height := chosen.Size()
		wide, said := made.WideFor(width, height)
		pinhole := sensor.Pinhole{Wide: wide * maths.Degree}
		middle := width / halves
		top := pinhole.Ray(middle, nothing, width, height, maths.Band{})
		// the top row's ray passes through its pixel's middle, half a
		// pixel in from the edge of a picture height pixels tall
		inset := float64(height-one) / float64(height)
		want := math.Atan(math.Tan(tall*maths.Degree/halves) * inset)
		got := math.Atan2(top.Dir.Y, -top.Dir.Z)
		if !said || math.Abs(got-want)/maths.Degree > number(t, "within") {
			t.Errorf("%s: the pinhole sees %v degrees up, want %v",
				preset, got/maths.Degree, want/maths.Degree)
		}
	}
}

// TestWideStandsAsSaid is the anti-test: a wide the job says is the
// angle across whatever the size, and with neither there is none.
// anti-test of TestTallIsWhatAPinholeSeesDownThePicture
func TestWideStandsAsSaid(t *testing.T) {
	made := read(t, still+"film: {wide: 55}\n").Film()
	for _, preset := range []string{"portrait", "4k"} {
		chosen, _ := presetNamed(preset)
		wide, said := made.WideFor(chosen.Size())
		if want, _ := made.Wide(); !said || wide != want {
			t.Errorf("%s: wide 55 is %v across", preset, wide)
		}
	}
	if _, said := read(t, still).Film().WideFor(one, one); said {
		t.Error("a film that says no angle says one across")
	}
}
