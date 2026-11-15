//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestSlopeChanceMatchesCPU(t *testing.T) {
	samples := []SlopeSample{{0, 0, 0.03, 0.02}, {0.1, -0.2, 0.03, 0.02}, {-0.2, 0.1, 0.1, 0.2}}
	got, err := SlopeChance(samples)
	if err != nil {
		t.Fatal(err)
	}
	for index, sample := range samples {
		want := oceantex.SlopeChance(float64(sample.Along), float64(sample.Across), float64(sample.VarianceAlong), float64(sample.VarianceAcross))
		if math.Abs(float64(got[index])-want) > 1e-5 {
			t.Fatalf("sample %d: GPU %v CPU %v", index, got[index], want)
		}
	}
}

func TestSlopeChanceRejectsInvalidVariance(t *testing.T) {
	if _, err := SlopeChance([]SlopeSample{{VarianceAlong: 0, VarianceAcross: 1}}); err == nil {
		t.Fatal("zero variance accepted")
	}
}
