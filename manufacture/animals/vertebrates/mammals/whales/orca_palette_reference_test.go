//go:build darwin && arm64

package whales

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Captured on Darwin/arm64 with Go 1.27.3 from the unchanged 51aa4df1
// material and texture, before the palette moved into the whale profile.
func TestOrcaPaletteKeepsHistoricalMaterialAndTexture(t *testing.T) {
	profile, ok := Default().Get("orca")
	if !ok {
		t.Fatal("missing orca")
	}
	parts, err := profile.Swim(0, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	material := parts[0].Material
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00", material.Name)
	for _, v := range []float64{material.Colour.R, material.Colour.G, material.Colour.B, material.Gloss, material.Index, material.Satin} {
		_ = binary.Write(h, binary.LittleEndian, math.Float64bits(v))
	}
	sample := func(p maths.Vec) {
		c := material.Texture.At(p)
		for _, v := range []float64{c.R, c.G, c.B} {
			_ = binary.Write(h, binary.LittleEndian, math.Float64bits(v))
		}
	}
	for x := -15; x <= 15; x++ {
		for y := -4; y <= 4; y++ {
			for z := -4; z <= 4; z++ {
				sample(maths.V(float64(x)*.2, float64(y)*.15, float64(z)*.15))
			}
		}
	}
	points := profile.model.orcaRest().points
	for i := 0; i < len(points); i += max(1, len(points)/512) {
		sample(points[i])
	}
	const want = "e2ff3dbea34fd745fa9a39a0d3909e3c499548866d9a29a91bffac7e923e5d59"
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != want {
		t.Fatalf("material and texture %s, want %s", got, want)
	}
}
