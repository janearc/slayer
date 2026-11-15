//go:build draw

package gpuocean

import (
	"math"
	"strings"
	"testing"
)

func TestFresnelRejectsUnrepresentableIndexArithmetic(t *testing.T) {
	for _, index := range []float32{math.SmallestNonzeroFloat32, 1e-20, 1e20, math.MaxFloat32} {
		_, err := Fresnel([]FresnelSample{{Cosine: 0.5, Index: index}})
		if err == nil || !strings.Contains(err.Error(), "index") {
			t.Fatalf("index %g: %v", index, err)
		}
	}
}
