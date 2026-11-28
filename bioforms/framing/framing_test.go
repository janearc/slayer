package framing_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/bioforms/framing"
	"github.com/janearc/slayer/maths"
)

func TestSuggestionsFitLongPartsInAPortraitFrame(t *testing.T) {
	const aspect = .55
	wing := maths.NewBox(maths.V(-4, -.2, -.7), maths.V(4, .2, .7))
	placement := maths.NewSubspace(maths.V(1, 2, .7),
		maths.Rotation(maths.ZAxis, .42), 1)
	root := maths.NewSubspace(maths.V(8, -2, 3),
		maths.Rotation(maths.YAxis, .37), 1.7)
	source := func(time.Duration) (framing.Pose, error) {
		return framing.Pose{Root: root, Parts: []framing.Part{{
			Bounds: wing, Placement: placement,
		}}}, nil
	}
	track, err := framing.New(framing.Request{Source: source,
		Settings: framing.Settings{Aspect: aspect}})
	if err != nil {
		t.Fatal(err)
	}
	shots, err := track.Suggestions(0)
	if err != nil {
		t.Fatal(err)
	}
	want := []framing.View{framing.Front, framing.ThreeQuarter, framing.Profile}
	if len(shots) != len(want) {
		t.Fatalf("got %d suggestions, want %d", len(shots), len(want))
	}
	for index, shot := range shots {
		if shot.View() != want[index] {
			t.Fatalf("suggestion %d is %q, want %q", index, shot.View(), want[index])
		}
		half := math.Tan(shot.Wide()/2) / 1.13
		for _, x := range []float64{wing.Min.X, wing.Max.X} {
			for _, y := range []float64{wing.Min.Y, wing.Max.Y} {
				for _, z := range []float64{wing.Min.Z, wing.Max.Z} {
					corner := root.Out(placement.Out(maths.V(x, y, z)))
					inEye := shot.Eye().In(corner)
					if inEye.Z >= 0 || math.Abs(inEye.X/inEye.Z) > half+1e-12 ||
						aspect*math.Abs(inEye.Y/inEye.Z) > half+1e-12 {
						t.Errorf("%s leaves corner %v outside its padded frame", shot.View(), corner)
					}
				}
			}
		}
	}
}

func TestTrackingResamplesTheBoundsAndCanBakeTranslation(t *testing.T) {
	source := func(at time.Duration) (framing.Pose, error) {
		seconds := at.Seconds()
		root := maths.NewSubspace(maths.V(seconds, 2*seconds, 0), maths.Identity, 1)
		box := maths.NewBox(maths.V(-.5, -seconds, -.2), maths.V(.5, seconds, .2))
		return framing.Pose{Root: root, Parts: []framing.Part{{Bounds: box}}}, nil
	}
	track, err := framing.New(framing.Request{Source: source, View: framing.Front,
		Settings: framing.Settings{Aspect: 1}})
	if err != nil {
		t.Fatal(err)
	}
	samples := []time.Duration{time.Second, 3 * time.Second}
	last, err := track.At(samples[1])
	if err != nil {
		t.Fatal(err)
	}
	first, err := track.At(samples[0])
	if err != nil {
		t.Fatal(err)
	}
	if last.Target() != maths.V(3, 6, 0) || first.Target() != maths.V(1, 2, 0) ||
		last.Eye().Origin.Z <= first.Eye().Origin.Z || last.Wide() != first.Wide() {
		t.Fatal("tracking failed to refit a moving, taller subject with the same field")
	}
	path, err := track.Path(samples)
	if err != nil {
		t.Fatal(err)
	}
	for index, shot := range []framing.Shot{first, last} {
		eye, err := path.At(samples[index] - samples[0])
		if err != nil || *eye != shot.Eye() {
			t.Fatalf("path lost sample %d, got %v, %v", index, eye, err)
		}
	}
}
