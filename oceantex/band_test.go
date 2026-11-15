package oceantex

import (
	"math"
	"reflect"
	"testing"
)

func TestWaveCascadesMatchViewSurface(t *testing.T) {
	sea := Sea{Wind: 8, Fetch: 1500, Toward: 0.3}
	actual, err := WaveCascades(sea, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	want, err := newSurface(sea, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != len(want.layers) {
		t.Fatal("cascade count differs")
	}
	for index, layer := range actual {
		if !reflect.DeepEqual(layer.Field, want.layers[index].field) || layer.Reach != want.layers[index].reach || layer.ResolvedAlong != want.layers[index].gotAlong || layer.ResolvedAcross != want.layers[index].gotAcross {
			t.Fatalf("cascade %d differs", index)
		}
	}
}

func TestWaveCascadesRejectInvalidTime(t *testing.T) {
	for _, seconds := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, err := WaveCascades(Sea{}, seconds, 3); err == nil {
			t.Fatalf("accepted time %v", seconds)
		}
	}
}
