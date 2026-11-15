//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestSampleFieldPeriodicInterpolation(t *testing.T) {
	field := &oceantex.Field{N: 2, Extent: 2}
	values := []float64{0, 2, 4, 6}
	points := []FieldPoint{{0, 0}, {0.5, 0.5}, {-0.5, -0.5}, {2.5, 0.5}}
	want := []float64{0, 3, 3, 3}
	got, err := SampleField(field, values, points)
	if err != nil {
		t.Fatal(err)
	}
	for index := range want {
		if math.Abs(float64(got[index])-want[index]) > 1e-6 {
			t.Fatalf("point %d: got %v want %v", index, got[index], want[index])
		}
	}
}

func TestSampleFieldRefusesInvalidInput(t *testing.T) {
	field := &oceantex.Field{N: 2, Extent: 2}
	for _, values := range [][]float64{{1}, {1, 2, math.NaN(), 4}} {
		if _, err := SampleField(field, values, []FieldPoint{{0, 0}}); err == nil {
			t.Fatal("invalid field accepted")
		}
	}
	if _, err := SampleField(field, []float64{1, 2, 3, 4}, []FieldPoint{{math.Inf(1), 0}}); err == nil {
		t.Fatal("invalid point accepted")
	}
}
