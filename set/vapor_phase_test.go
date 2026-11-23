package set

import (
	"strings"
	"testing"
)

func TestVaporPhaseRetainsExplicitHGAndRejectsNonClear(t *testing.T) {
	vapor := strings.Replace(vaporCavitationSet, "        clear-water: true", "        clear-water: true\n        phase-g: 0.8", 1)
	read, err := Read("vapor-phase.yaml", []byte(vapor))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	g, explicit := world.VaporCavities()[0].PhaseAsymmetry()
	if !explicit || g != .8 {
		t.Fatal("vapor HG not retained")
	}
	opaque := strings.Replace(vapor, "clear-water: true", "clear-water: false", 1)
	if _, err := Read("bad-phase.yaml", []byte(opaque)); err == nil {
		t.Fatal("vapor phase ignored without clear-water opt-in")
	}
}
