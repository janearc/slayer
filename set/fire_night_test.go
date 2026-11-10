package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func nightRecipeFixture(t *testing.T) string {
	return strings.Replace(fireRecipeFixture(t), "  light: none",
		"  light: moon\n  moon: {azimuth: 115, altitude: 28}", 1)
}

func TestEcosystemSupportsItsMoonlightModel(t *testing.T) {
	made, err := set.Read("night.yaml", []byte(nightRecipeFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	moon, ok := made.Moonlit()
	if !ok {
		t.Fatal("explicit moon was lost")
	}
	around, up := moon.Where()
	if made.Light() != "moon" || around != 115 || up != 28 {
		t.Fatal("moon direction changed")
	}
}

func TestEcosystemMoonRejectsUnknownFieldsAndInvalidAngles(t *testing.T) {
	for _, change := range [][2]string{
		{"altitude: 28", "altitude: 91"},
		{"azimuth: 115", "azimuth: not-an-angle"},
		{"altitude: 28", "altitude: 28, typo: 1"},
	} {
		raw := strings.Replace(nightRecipeFixture(t), change[0], change[1], 1)
		if _, err := set.Read("invalid-night.yaml", []byte(raw)); err == nil {
			t.Fatal("invalid moon accepted")
		}
	}
}
