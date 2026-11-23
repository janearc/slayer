package set

import (
	"strings"
	"testing"
)

func TestVaporDispersalReadsIndependentAuthoredControls(t *testing.T) {
	raw := strings.Replace(vaporCavitationSet, "        fragment-seed: 705", "        wander-m: 1.2\n        wander-scale-m: 3\n        spread-start-m: 4\n        spread-per-m: 0.12\n        fragment-seed: 705", 1)
	w, err := Read("dispersal.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := w.World()
	a, b, c, d := world.VaporCavities()[0].Dispersal()
	if a != 1.2 || b != 3 || c != 4 || d != .12 {
		t.Fatal("dispersal controls lost")
	}
	for _, bad := range []string{strings.Replace(raw, "        wander-scale-m: 3\n", "", 1), strings.Replace(raw, "spread-per-m: 0.12", "spread-per-m: -1", 1)} {
		if _, err := Read("bad.yaml", []byte(bad)); err == nil {
			t.Fatal("invalid dispersal accepted")
		}
	}
}
