//go:build draw && darwin && arm64

package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Recorded from the original root builder at befd86b1, before moving coral.
// The plain colony is covered by TestSharedLoftKeepsPreExtractionGeometry.
func TestCoralDetailedGeometryAndPaintWitness(t *testing.T) {
	for _, tc := range []struct {
		name, digest string
		corallites   bool
	}{
		{"fine", "49cf0943137cc543c30a5ed0ad82e95c83ecfbb83d3971a244ab64cad8f36d39", false},
		{"corallites", "1940b93c94e920f3d04565cf010ba3bd39c4d80305430fbca57e1982e8341f26", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := DefaultCoral()
			spec.Levels, spec.FineSurface, spec.Corallites = 1, !tc.corallites, tc.corallites
			kind, err := Coral(spec)
			if err != nil {
				t.Fatal(err)
			}
			made, err := Make(kind, Sow(14))
			if err != nil {
				t.Fatal(err)
			}
			if got := referenceDigest(t, made); got != tc.digest {
				t.Fatalf("geometry and materials: got %s, want %s", got, tc.digest)
			}
			for _, sample := range []struct {
				at  maths.Vec
				rgb maths.Light
			}{
				{maths.V(0, 0, 0), maths.RGB(.49, .32, .22)},
				{maths.V(.03, .04, -.01), maths.RGB(.506817938400206, .33098314344503244, .2275509111184598)},
				{maths.V(-.06, .18, .09), maths.RGB(.5040381651972219, .3291677813532878, .22630284968038533)},
			} {
				got := made.dressed[body].Texture.At(sample.at)
				if math.Abs(got.R-sample.rgb.R) > 1e-14 || math.Abs(got.G-sample.rgb.G) > 1e-14 || math.Abs(got.B-sample.rgb.B) > 1e-14 {
					t.Fatalf("paint at %v: got %v, want %v", sample.at, got, sample.rgb)
				}
			}
		})
	}
}
