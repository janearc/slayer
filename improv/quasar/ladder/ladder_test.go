package ladder

import (
	"math"
	"testing"
)

// The numbers the tests choose: the quasar slice, 2026-11-12.
const (
	earthAcross = 1.27562e7 // twice the nominal equatorial radius, metres
	jetsAcross  = 1.98e23   // about Porphyrion's 6.43 Mpc, metres
	picture     = 1080
	slack       = 1e-9
)

func things(t *testing.T) []Thing {
	t.Helper()
	earth, err := NewThing("the earth", earthAcross, "a test")
	if err != nil {
		t.Fatal(err)
	}
	jets, err := NewThing("the jets", jetsAcross, "a test")
	if err != nil {
		t.Fatal(err)
	}
	return []Thing{earth, jets}
}

func TestEachStepIsTenTimesTheLastAndMarksIt(t *testing.T) {
	ladder, err := NewLadder(earthAcross*ten, things(t))
	if err != nil {
		t.Fatal(err)
	}
	steps := ladder.Steps()
	if steps[none].Marked() != none {
		t.Fatal("the first step marks a frame before it")
	}
	for i := one; i < len(steps); i++ {
		ratio := steps[i].Width() / steps[i-one].Width()
		share := steps[i].Marked() / steps[i].Width()
		if math.Abs(ratio-ten) > slack || math.Abs(share-tenth) > slack {
			t.Errorf("step %d is %v the last and marks %v of itself",
				i, ratio, share)
		}
	}
	if last := steps[len(steps)-one]; last.Width() < jetsAcross ||
		last.Width()/ten >= jetsAcross {
		t.Fatalf("the ladder ends at %v for jets of %v", last.Width(),
			jetsAcross)
	}
}

func TestAThingShowsBetweenAPixelAndTheFrame(t *testing.T) {
	earth := things(t)[none]
	ladder, err := NewLadder(earthAcross*ten, things(t))
	if err != nil {
		t.Fatal(err)
	}
	steps := ladder.Steps()
	if !steps[none].Shows(earth, picture) {
		t.Error("the earth does not show in a frame ten earths wide")
	}
	far := steps[len(steps)-one]
	if far.Shows(earth, picture) {
		t.Error("the earth shows in a frame the jets fill")
	}
	near := Step{width: earthAcross / ten}
	if near.Shows(earth, picture) {
		t.Error("the earth shows in a frame a tenth of it")
	}
}

func TestNewThingRefusesNoSizeNoNameNoSource(t *testing.T) {
	for _, size := range []float64{none, -one, math.NaN(), math.Inf(1)} {
		if _, err := NewThing("x", size, "a test"); err == nil {
			t.Errorf("a size of %v was taken", size)
		}
	}
	if _, err := NewThing("", one, "a test"); err == nil {
		t.Error("a thing with no name was taken")
	}
	if _, err := NewThing("x", one, ""); err == nil {
		t.Error("a thing with no source was taken")
	}
}

func TestNewLadderRefusesNoStartAndNothing(t *testing.T) {
	for _, first := range []float64{none, math.NaN(), math.Inf(1)} {
		if _, err := NewLadder(first, things(t)); err == nil {
			t.Errorf("a first width of %v was taken", first)
		}
	}
	if _, err := NewLadder(earthAcross, nil); err == nil {
		t.Error("a ladder past nothing was taken")
	}
}

func TestAThingSaysWhatItIs(t *testing.T) {
	earth := things(t)[none]
	if earth.Name() != "the earth" || earth.Size() != earthAcross ||
		earth.From() != "a test" {
		t.Fatalf("the thing is %q, %v, from %q", earth.Name(),
			earth.Size(), earth.From())
	}
}
