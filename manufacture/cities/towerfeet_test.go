package cities

import "testing"

// footLength is the length of the walls the open-wall tests use, in metres.
// picked: any length will do.
const footLength = 10.0

// footStorey is the height of the open storey the tests ask for, in
// metres. picked: any height above nought will do.
const footStorey = 3.5

// TestAWallIsOpenOnlyIfItsOneClearStretchIsAllOfIt checks the three things
// a wall's clear stretches can be: all of it, a part of it, and none.
func TestAWallIsOpenOnlyIfItsOneClearStretchIsAllOfIt(t *testing.T) {
	half := footLength / 2
	for name, test := range map[string]struct {
		clear [][2]float64
		open  bool
	}{
		"all of it":     {[][2]float64{{-half, half}}, true},
		"a part of it":  {[][2]float64{{-half, 0}}, false},
		"two parts":     {[][2]float64{{-half, -1}, {1, half}}, false},
		"none of it":    {nil, false},
		"short of one":  {[][2]float64{{-half + 1, half}}, false},
		"short of both": {[][2]float64{{-half, half - 1}}, false},
	} {
		face := FootFace{Length: footLength, Clear: test.clear}
		if got := face.open(); got != test.open {
			t.Errorf("%s: open %v, want %v", name, got, test.open)
		}
	}
}

// footedTower is a tower of the pins and a seed, or the test's end.
func footedTower(t *testing.T, pins Pins) *Tower {
	t.Helper()
	tower, err := NewTower(21, Irregularity{}, pins)
	if err != nil {
		t.Fatal(err)
	}
	return tower
}

// TestAPlainRectangleIsFootedAndATowerOnABaseIsNot checks that a single
// shaft of a plain rectangle in plan is lifted on an open storey of the
// height asked, and that one of two shafts, which stands on a base, is not.
func TestAPlainRectangleIsFootedAndATowerOnABaseIsNot(t *testing.T) {
	off := false
	pins := Pins{Plan: "rectangle", Skin: "concrete", Storeys: 8, Width: 36,
		Taper: &off, Twist: &off, Setbacks: &off, Spire: &off}
	plain := footedTower(t, pins)
	lifted, err := plain.Footed(footStorey)
	if err != nil || !lifted || plain.Foot() != footStorey {
		t.Errorf("a plain rectangle: lifted %v, foot %v, error %v", lifted,
			plain.Foot(), err)
	}
	pins.Shafts = 2
	based := footedTower(t, pins)
	lifted, err = based.Footed(footStorey)
	if err != nil || lifted || based.Foot() != 0 {
		t.Errorf("a tower on a base: lifted %v, foot %v, error %v", lifted,
			based.Foot(), err)
	}
}

// TestNoStoreyOfNoHeightIsOpened checks that a height of nought opens
// nothing, whatever the plan.
func TestNoStoreyOfNoHeightIsOpened(t *testing.T) {
	off := false
	tower := footedTower(t, Pins{Plan: "rectangle", Skin: "concrete",
		Storeys: 8, Width: 36, Taper: &off, Twist: &off, Setbacks: &off,
		Spire: &off})
	if lifted, err := tower.Footed(0); err != nil || lifted ||
		tower.Foot() != 0 {
		t.Errorf("a storey of no height: lifted %v, foot %v, error %v", lifted,
			tower.Foot(), err)
	}
}
