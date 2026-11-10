package set_test

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/set"
)

const sparkBlock = "      sparks:\n" +
	"        rate: {value: 100, unit: s^-1}\n" +
	"        lifetime: {value: 3, unit: s}\n" +
	"        radius: {value: 0.025, unit: m}\n" +
	"        release-height: {value: 3, unit: m}\n" +
	"        launch-speed: {value: 9, unit: m/s}\n" +
	"        spread: {value: 2, unit: m/s}\n" +
	"        updraft: {value: 8, unit: m/s}\n" +
	"        drag-time: {value: 0.8, unit: s}\n" +
	"        gravity: {value: 9.80665, unit: m/s^2}\n" +
	"        temperature: {value: 1650, unit: K}\n" +
	"        end-temperature: {value: 900, unit: K}\n"

func TestFireSparksKeepTheirPhysicalInputsAndOwnership(t *testing.T) {
	raw := strings.Replace(fireRecipeFixture(t), "      flame:", sparkBlock+"      flame:", 1)
	made, err := set.Read("spark.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	recipe, _ := made.Recipe("wood fire")
	spec, _ := recipe.Fire()
	if spec.Sparks == nil || spec.Sparks.Rate != 100 || spec.Sparks.Life != 3*time.Second ||
		spec.Sparks.Radius != measure.Metres(.025) || spec.Sparks.Hot != measure.Kelvin(1650) {
		t.Fatal("spark inputs changed")
	}
	spec.Sparks.Rate = 999
	again, _ := recipe.Fire()
	if again.Sparks.Rate != 100 {
		t.Fatal("caller changed stored sparks")
	}
}

func TestFireSparksRejectBadUnitsMissingFieldsAndColdStarts(t *testing.T) {
	for _, change := range [][2]string{
		{"unit: s^-1", "unit: Hz-ish"},
		{"rate: {value: 100", "rate: {value: -1"},
		{"lifetime: {value: 3", "lifetime: {value: 0"},
		{"temperature: {value: 1650", "temperature: {value: 700"},
		{"        gravity: {value: 9.80665, unit: m/s^2}\n", ""},
	} {
		raw := strings.Replace(fireRecipeFixture(t), "      flame:", strings.Replace(sparkBlock, change[0], change[1], 1)+"      flame:", 1)
		if _, err := set.Read("bad-spark.yaml", []byte(raw)); err == nil {
			t.Fatal("invalid sparks accepted")
		}
	}
}
