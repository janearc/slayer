//go:build draw

package gpuocean

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestWavesAreDeterministicAndBounded(t *testing.T) {
	sea := oceantex.Sea{Wind: 8, Fetch: 1500, Toward: 0.3}
	first, err := Waves(sea, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Waves(sea, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || len(first) != 3 {
		t.Fatal("wave cascades are not deterministic")
	}
	march, err := MarchLayers(first)
	if err != nil || len(march) != 3 || march[0].Field != first[0].Field {
		t.Fatalf("march layers: %v %v", march, err)
	}
	for _, layer := range first {
		if layer.Field.N != 256 {
			t.Fatalf("grid: %d", layer.Field.N)
		}
		for _, height := range layer.Field.Height {
			if math.Abs(height) > layer.Reach {
				t.Fatal("reach does not bound height")
			}
		}
	}
}

func TestWavesRejectInvalidTime(t *testing.T) {
	if _, err := Waves(oceantex.Sea{}, math.NaN(), 0); err == nil {
		t.Fatal("nonfinite time accepted")
	}
	if _, err := Waves(oceantex.Sea{}, -1, 0); err == nil {
		t.Fatal("negative time accepted")
	}
}
