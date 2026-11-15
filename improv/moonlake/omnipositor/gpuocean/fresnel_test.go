//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestFresnelMatchesOceanOptics(t *testing.T) {
	samples := []FresnelSample{{Cosine: 1, Index: 1.333}, {Cosine: 0, Index: 1.333}, {Cosine: 0.3, Index: 1.333}, {Cosine: 0.1, Index: 0.8}, {Cosine: 2, Index: 1.5}}
	got, err := Fresnel(samples)
	if err != nil {
		t.Fatal(err)
	}
	for index, sample := range samples {
		want := oceantex.Fresnel(float64(sample.Cosine), float64(sample.Index))
		if math.Abs(float64(got[index])-want) > 1e-5 {
			t.Fatalf("sample %d: GPU %v CPU %v", index, got[index], want)
		}
	}
}

func TestFresnelRefusesInvalidInputs(t *testing.T) {
	for _, sample := range []FresnelSample{{Cosine: float32(math.NaN()), Index: 1}, {Cosine: 1, Index: 0}, {Cosine: 1, Index: float32(math.Inf(1))}} {
		if _, err := Fresnel([]FresnelSample{sample}); err == nil {
			t.Fatalf("accepted %+v", sample)
		}
	}
}

func TestFresnelEmptyAndLimit(t *testing.T) {
	got, err := Fresnel(nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty: %v %v", got, err)
	}
	if _, err := Fresnel(make([]FresnelSample, MaxSamples+1)); err == nil {
		t.Fatal("sample limit not enforced")
	}
}
