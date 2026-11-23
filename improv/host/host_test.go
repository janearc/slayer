package host

import (
	"image"
	"testing"
)

// The numbers the tests choose: the quasar slice, 2026-11-12.
const (
	seed     = 7
	loud     = 0.9
	soft     = 0.5
	silent   = 0.05
	strong   = 0.95
	weak     = 0.2
	hostSide = 100
	boxWide  = 600
	boxTall  = 300
	boxLeft  = 50
	boxTop   = 900
	seconds  = 3.25
	frames   = 500
	frameGap = 1.0 / 24
)

func TestPickFollowsTheMusicAndTheWords(t *testing.T) {
	cases := []struct {
		at   moment
		want string
	}{
		{moment{Loudness: silent, Kick: strong, Speaking: true}, quiet},
		{moment{Loudness: loud, Kick: strong, Speaking: true}, flash},
		{moment{Loudness: loud, Kick: weak, Speaking: true}, speaking},
		{moment{Loudness: soft, Kick: strong}, resting},
		{moment{Loudness: loud, Kick: weak}, resting},
	}
	for _, each := range cases {
		if got := pick(each.at); got != each.want {
			t.Errorf("%+v picks %s, want %s", each.at, got, each.want)
		}
	}
}

func testBox(t *testing.T) Box {
	t.Helper()
	box, err := NewBox(image.Rect(boxLeft, boxTop, boxLeft+boxWide,
		boxTop+boxTall), image.Pt(hostSide, hostSide))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestPlaceNeverLeavesTheBox(t *testing.T) {
	box := testBox(t)
	for i := range frames {
		at := moment{Loudness: loud, Kick: float64(i%two) * strong}
		place := box.place(at, float64(i)*frameGap, seed)
		host := image.Rectangle{Min: place, Max: place.Add(box.size)}
		if !host.In(box.area) {
			t.Fatalf("frame %d puts the host at %v, outside %v", i,
				host, box.area)
		}
	}
}

func TestPlaceIsTheSameTwiceAndJumpsOnAKick(t *testing.T) {
	box := testBox(t)
	still := moment{Loudness: loud}
	first, second := box.place(still, seconds, seed), box.place(still, seconds, seed)
	if first != second {
		t.Fatal("the same moment placed the host twice differently")
	}
	kicked := moment{Loudness: loud, Kick: one}
	if !(box.place(kicked, seconds, seed).Y <=
		box.place(still, seconds, seed).Y) {
		t.Fatal("a kick did not lift the host")
	}
	if box.place(still, seconds, seed) == box.place(still, seconds,
		seed+one) && box.place(still, seconds+one, seed) ==
		box.place(still, seconds+one, seed+one) {
		t.Fatal("another seed drifts the host the same way")
	}
}

func TestPlaceHoldsStillInABreak(t *testing.T) {
	box := testBox(t)
	hushed := moment{Loudness: silent}
	if box.place(hushed, seconds, seed) != box.place(hushed,
		seconds*two, seed+one) {
		t.Fatal("the host moved in a break")
	}
}

func TestNewBoxRefusesAHostThatDoesNotFit(t *testing.T) {
	area := image.Rect(none, none, hostSide, hostSide)
	for _, size := range []image.Point{{hostSide + one, hostSide},
		{none, hostSide}, {hostSide, hostSide + one}} {
		if _, err := NewBox(area, size); err == nil {
			t.Errorf("a host %v fitted a box %v", size, area)
		}
	}
}

func TestFlickerStaysInItsDepthAndRepeats(t *testing.T) {
	for i := range frames {
		at := float64(i) * frameGap
		bright := Flicker(at, seed)
		if bright < one-flickerDepth || bright > one {
			t.Fatalf("frame %d flickers to %v", i, bright)
		}
		if Flicker(at, seed) != bright {
			t.Fatal("a flicker drawn twice differed")
		}
	}
}

func TestLiftOnlyOnAKickInAPartThatLifts(t *testing.T) {
	lifts := map[string]bool{"hairline": true}
	kicked := moment{Part: "hairline", Kick: one, Loudness: loud}
	if kickLift(kicked, lifts) != lifted {
		t.Fatalf("a full kick lifts %v", kickLift(kicked, lifts))
	}
	for _, at := range []moment{
		{Part: "thin", Kick: one, Loudness: loud},
		{Part: "hairline", Kick: none, Loudness: loud},
		{Part: "hairline", Kick: one, Loudness: silent},
	} {
		if kickLift(at, lifts) != none {
			t.Errorf("%+v lifts the picture", at)
		}
	}
}
