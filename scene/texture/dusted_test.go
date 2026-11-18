package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// the rock and dust the tests are drawn with
var (
	testRock = maths.RGB(0.06, 0.05, 0.046) // picked: a dark rock
	testDust = maths.RGB(0.03, 0.025, 0.02) // picked: a darker dust
)

// testPlaces is how many places along a line the tests look at. picked.
const testPlaces = 400

// coverOver is the mean dust over a line of places, and whether every
// colour lay between the rock's and the dust's.
func coverOver(dusted Dusted) (float64, bool) {
	var whole float64
	between := true
	for i := range testPlaces {
		p := maths.V(float64(i)*0.0137, 0.3, float64(i)*0.0071) // picked
		whole += dusted.cover(p)
		got := dusted.At(p)
		between = between && got.G <= testRock.G && got.G >= testDust.G
	}
	return whole / testPlaces, between
}

// Dust lies in patches: some places dusted, some bare, every colour
// between the rock's and the dust's, and more dust where more is asked.
func TestDustLiesInPatches(t *testing.T) {
	little, between := coverOver(Dusted{Rock: testRock, Dust: testDust,
		Share: 0.2, Patch: 0.05, Seed: 1}) // picked
	much, _ := coverOver(Dusted{Rock: testRock, Dust: testDust,
		Share: 0.7, Patch: 0.05, Seed: 1}) // picked
	if !between {
		t.Error("a colour lay outside the rock's and the dust's")
	}
	if !(little > 0 && little < much && much < 1) {
		t.Errorf("dust covers %v asked a fifth and %v asked most",
			little, much)
	}
}

// Anti: no dust asked is the bare rock everywhere.
func TestNoDustIsBareRock(t *testing.T) {
	dusted := Dusted{Rock: testRock, Dust: testDust, Patch: 0.05, Seed: 1}
	cover, _ := coverOver(dusted)
	if cover != 0 || dusted.At(maths.V(0.1, 0.2, 0.3)) != testRock {
		t.Errorf("no dust asked covers %v", cover)
	}
}
