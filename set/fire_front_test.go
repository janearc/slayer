package set_test

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	"strings"
	"testing"
	"time"
)

func frontFixture(t *testing.T) string {
	raw := strings.Replace(fireRecipeFixture(t), "  film:", `  fire-fronts:
    through the grove:
      stand: the grove
      recipe: wood fire
      from: the fire
      bearing: {value: 90, unit: deg}
      speed: {value: 2, unit: m/s}
      starts: {value: 3, unit: s}
      burn-duration: {value: 8, unit: s}
      fade-time: {value: 0.2, unit: s}
      group-width: {value: 12, unit: m}
      ignition-height: {value: 1, unit: m}
      smoke-radius: {value: 8, unit: m}
  film:`, 1)
	return strings.Replace(raw, "  set:\n", "  set:\n    stands:\n      the grove: {kind: coast-redwood, count: 3, seed: 2}\n", 1)
}

func TestFireFrontReadsAnExplicitPrescribedMotionAndFindsItsFuel(t *testing.T) {
	made, err := set.Read("front.yaml", []byte(frontFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	fronts := made.FireFronts()
	if len(fronts) != 1 {
		t.Fatal("front lost")
	}
	front := fronts[0]
	spec := front.Motion(maths.V(1, 2, 3))
	if front.Stand() != "the grove" || front.Recipe() != "wood fire" || front.From() != "the fire" || spec.Speed != 2 ||
		spec.Start != 3*time.Second || spec.Duration != 8*time.Second || spec.Cell != 12 ||
		front.IgnitionHeight().InMetres() != 1 || front.SmokeRadius().InMetres() != 8 ||
		spec.Direction.Sub(maths.XAxis).Len() > 1e-12 {
		t.Fatal("front units or source references changed")
	}
}

func TestFireFrontRefusesUnknownSourcesWrongUnitsAndVolumeHints(t *testing.T) {
	for _, change := range [][2]string{
		{"stand: the grove", "stand: missing"}, {"recipe: wood fire", "recipe: missing"},
		{"from: the fire", "from: nowhere"}, {"unit: m/s", "unit: km/h"},
		{"speed: {value: 2", "speed: {value: 0"}, {"group-width: {value: 12", "group-width: {value: 0"},
		{"      smoke-radius: {value: 8, unit: m}\n", ""},
	} {
		raw := strings.Replace(frontFixture(t), change[0], change[1], 1)
		if _, err := set.Read("bad-front.yaml", []byte(raw)); err == nil {
			t.Fatalf("accepted %s", change[1])
		}
	}
}
