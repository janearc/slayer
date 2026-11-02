package smoke

import (
	"fmt"
	"testing"
)

// A smoke a caller is handed is its own: changing its Bands, Palette or
// Inputs, through Get or All, leaves the next caller's as they were.
func TestASmokeHandedOutIsACopy(t *testing.T) {
	var withBands, withPalette, withInputs int
	for _, s := range All() {
		if len(s.Bands) > 0 && withBands == 0 {
			withBands = s.Number
		}
		if len(s.Palette) > 0 && withPalette == 0 {
			withPalette = s.Number
		}
		if len(s.Inputs) > 0 && withInputs == 0 {
			withInputs = s.Number
		}
	}
	if withBands == 0 || withPalette == 0 || withInputs == 0 {
		t.Fatalf("no smoke with bands, a palette and inputs to try: %d %d %d",
			withBands, withPalette, withInputs)
	}
	for _, n := range []int{withBands, withPalette, withInputs} {
		// written out before anything is spoilt: a shared backing array
		// would spoil a Smoke held here as well
		first, _ := Get(n)
		before := fmt.Sprintf("%+v", withoutScene(first))
		spoilt, _ := Get(n)
		for i := range spoilt.Bands {
			spoilt.Bands[i].Nanometres = -1
		}
		for i := range spoilt.Palette {
			spoilt.Palette[i].R = -1
		}
		for i := range spoilt.Inputs {
			spoilt.Inputs[i] = "spoilt"
		}
		listed := All()[0]
		for _, s := range All() {
			if s.Number == n {
				listed = s
			}
		}
		for i := range listed.Bands {
			listed.Bands[i].Nanometres = -2
		}
		for i := range listed.Inputs {
			listed.Inputs[i] = "spoilt by All"
		}
		after, _ := Get(n)
		if got := fmt.Sprintf("%+v", withoutScene(after)); got != before {
			t.Errorf("smoke %d changed for later callers:\n%s\nwas\n%s",
				n, got, before)
		}
	}
	newest := Newest()
	for i := range newest.Bands {
		newest.Bands[i].Nanometres = -3
	}
	if got := Newest(); len(got.Bands) > 0 && got.Bands[0].Nanometres == -3 {
		t.Error("the newest smoke changed for later callers")
	}
}

// withoutScene is s with no Scene, whose printed address is not what is
// under test.
func withoutScene(s Smoke) Smoke {
	s.Scene = nil
	return s
}
