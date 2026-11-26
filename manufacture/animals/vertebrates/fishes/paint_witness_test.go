package fishes

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

// Captured from the root fish builder before the family move. These samples
// exercise the seeded spots as well as the species' other skin colours.
func TestFishSeededPaintRetainsItsSamples(t *testing.T) {
	for _, tc := range []struct {
		name string
		seed uint64
		want string
	}{
		{"pacific-sardine", 1, "5437f73054891bab8a023426dc0ad2c6b6930fde429c088bd0470ff39a7f60e7"},
		{"pacific-sardine", 5, "4a294a2d2a7535deb7849235c76afad9fec241df6411bd7af2db9bcdea3823f2"},
		{"dusky-grouper", 1, "299bf9220fc60999509667d6010bbf5303b1f013774c1edd2579a747893735d5"},
		{"dusky-grouper", 5, "abfea017bde3d1eeba167aba09a818bd365e05300c51c8087717ac903fba2227"},
	} {
		plan := planFish(tc.name, sow(tc.seed))
		out := plan.outline()
		paint := plan.paint()
		h := sha256.New()
		for i := 1; i < 100; i++ {
			s := .10 + .80*float64(i)/100
			for k := 0; k < 64; k++ {
				p := out.surface(s, 2*math.Pi*float64(k)/64)
				color := paint.At(p)
				for _, value := range []float64{color.R, color.G, color.B} {
					var b [8]byte
					binary.LittleEndian.PutUint64(b[:], math.Float64bits(value))
					h.Write(b[:])
				}
			}
		}
		if got := fmt.Sprintf("%x", h.Sum(nil)); got != tc.want {
			t.Errorf("%s seed %d: got %s, want %s", tc.name, tc.seed, got, tc.want)
		}
	}
}
