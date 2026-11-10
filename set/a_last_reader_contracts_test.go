package set_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestLegacyBoundingBoxAcceptsEveryZeroDimension(t *testing.T) {
	raw := legacyNumericFixture(fireRecipeFixture(t))
	if strings.Count(raw, "  set:\n") != 1 {
		t.Fatal("ambiguous legacy set fixture")
	}
	raw = strings.Replace(raw, "  set:\n",
		"  set:\n    bounded_box: {size_x: 0, size_y: 0, size_z: 0}\n", 1)
	if _, err := set.Read("zero-box.yaml", []byte(raw)); err != nil {
		t.Fatalf("zero lies inside every legacy dimension's declared range: %v", err)
	}
}

func TestLegacyIntermittencyLimitsReportTheirOwnLocation(t *testing.T) {
	raw := legacyNumericFixture(fireRecipeFixture(t))
	raw = strings.Replace(raw, "      flame:\n",
		"      flame:\n        intermittency-start: 0.25\n", 1)
	requireLegacyNumericControl(t, raw)
	row := numericContract{"intermittency-start: ", "0.25",
		"intermittency-start", .001, .999}
	refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
	refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
}

func TestLegacyCoupledSparkRefusalsPointToTheSparkBlock(t *testing.T) {
	base := legacyNumericFixture(strings.Replace(fireRecipeFixture(t),
		"      flame:", sparkBlock+"      flame:", 1))
	requireLegacyNumericControl(t, base)
	for _, edit := range [][2]string{
		{"temperature: {value: 1650", "temperature: {value: 700"},
		{"rate: {value: 100,", "rate: {value: 8000,"},
	} {
		raw := strings.Replace(base, edit[0], edit[1], 1)
		if raw == base {
			t.Fatal("ineffective coupled spark fixture")
		}
		start := strings.Index(raw, "        rate:")
		if start < 0 {
			t.Fatal("missing spark mapping")
		}
		line := 1 + strings.Count(raw[:start], "\n")
		want := fmt.Sprintf("coupled-sparks.yaml:%d:9: "+
			"embers: invalid source or too many living particles", line)
		_, err := set.Read("coupled-sparks.yaml", []byte(raw))
		if err == nil || err.Error() != want {
			t.Fatalf("spark refusal: %v, want %s", err, want)
		}
	}
}

func TestLegacyFrontRefusesAStandWithNoDeclaredCount(t *testing.T) {
	base := legacyNumericFixture(frontFixture(t))
	requireLegacyNumericControl(t, base)
	raw := strings.Replace(base, "kind: coast-redwood, count: 3, seed: 2",
		"kind: coast-redwood, seed: 2", 1)
	if raw == base {
		t.Fatal("ineffective empty stand fixture")
	}
	_, err := set.Read("empty-fuel.yaml", []byte(raw))
	want := "empty-fuel.yaml:0:0: front through the grove " +
		"needs an existing unlit non-fire stand"
	if err == nil || err.Error() != want {
		t.Fatalf("empty fuel refusal: %v, want %s", err, want)
	}
}
