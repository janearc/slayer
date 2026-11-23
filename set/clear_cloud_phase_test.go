package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestFizzAndVaporPhaseRequireExplicitClearWater(t *testing.T) {
	fizz := fizzSet(t, "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        clear-water: true\n        phase-g: 0")
	read, err := set.Read("phase.yaml", []byte(fizz))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	g, explicit := world.Fizzes()[0].PhaseAsymmetry()
	if !explicit || g != 0 {
		t.Fatal("explicit zero HG was lost")
	}
	for _, bad := range []string{"-1", "1", ".nan", ".inf"} {
		edited := strings.Replace(fizz, "phase-g: 0", "phase-g: "+bad, 1)
		if _, err := set.Read("bad.yaml", []byte(edited)); err == nil {
			t.Fatalf("accepted HG %s", bad)
		}
	}
	opaque := strings.Replace(fizz, "clear-water: true", "clear-water: false", 1)
	if _, err := set.Read("not-clear.yaml", []byte(opaque)); err == nil {
		t.Fatal("phase silently ignored without clear water")
	}
}
