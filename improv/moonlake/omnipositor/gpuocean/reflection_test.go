//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestReflectionFlatWater(t *testing.T) {
	got, err := Reflections([]ReflectionSample{{Eye: oceantex.Vec{Z: 1}, Index: 1.333}})
	if err != nil {
		t.Fatal(err)
	}
	want := oceantex.Fresnel(1, 1.333)
	if math.Abs(got[0].Direction.Z-1) > 1e-6 || math.Abs(float64(got[0].Weight)-want) > 1e-5 {
		t.Fatalf("reflection: %+v want weight %v", got[0], want)
	}
}

func TestReflectionRejectsInvalidEye(t *testing.T) {
	if _, err := Reflections([]ReflectionSample{{Index: 1.333}}); err == nil {
		t.Fatal("zero eye accepted")
	}
}
