package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestFizzAxisAllowsDownstreamPopulationAndRejectsZero(t *testing.T) {
	raw := fizzSet(t, "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        axis: [0.342, -0.94, 0.03]")
	if _, err := set.Read("oriented.yaml", []byte(raw)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"[0, 0, 0]", "[.nan, 0, 1]", "[1, 2]", "[2, 0, 0]"} {
		changed := strings.Replace(raw, "[0.342, -0.94, 0.03]", bad, 1)
		if _, err := set.Read("bad-axis.yaml", []byte(changed)); err == nil {
			t.Fatalf("accepted invalid axis %s", bad)
		}
	}
}

func TestFizzZeroPopulationIsExplicitlyClearWaterOnly(t *testing.T) {
	raw := fizzSet(t, "thick: 4", "thick: 0", "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        clear-water: true")
	if _, err := set.Read("zero-population.yaml", []byte(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := set.Read("legacy-zero.yaml", []byte(strings.Replace(raw, "clear-water: true", "clear-water: false", 1))); err == nil {
		t.Fatal("legacy standalone zero thickness silently became valid")
	}
}
