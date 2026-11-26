//go:build draw && darwin && arm64

package manufacture

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// Hash real geometry: sphere/tube primitives remain analytic and meshes keep
// the established reference encoding. Texture samples are a separate, small
// witness rather than a claim to fingerprint every point on a material.
func pinnipedWitness(t *testing.T, made good) string {
	t.Helper()
	things, err := made.Things(maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	bounds := made.Bounds()
	referenceVector(h, bounds.Min)
	referenceVector(h, bounds.Max)
	referenceCount(h, len(things))
	seen := map[shape.Shape]int{}
	for _, part := range things {
		referenceSpace(h, part.At(0))
		switch solid := part.Shape.(type) {
		case shape.Sphere:
			referenceWrite(h, byte(10))
			referenceWrite(h, solid.Radius)
		case shape.Tube:
			referenceWrite(h, byte(11))
			referenceWrite(h, [3]float64{solid.Radius, solid.Thick, solid.Height})
		default:
			referenceWrite(h, byte(12))
			referenceGeometry(h, solid, seen)
		}
		if part.Material == nil {
			referenceWrite(h, byte(0))
			continue
		}
		referenceWrite(h, byte(1))
		m := part.Material
		referenceCount(h, len(m.Name))
		if _, err := h.Write([]byte(m.Name)); err != nil {
			t.Fatal(err)
		}
		referenceWrite(h, [8]float64{m.Colour.R, m.Colour.G, m.Colour.B, m.Glow.R, m.Glow.G, m.Glow.B, m.Gloss, m.Index})
		if m.Texture == nil {
			referenceWrite(h, byte(0))
			continue
		}
		referenceWrite(h, byte(1))
		for _, p := range []maths.Vec{maths.V(0, 0, 0), maths.V(.1, .1, .1), maths.V(1, .2, .1)} {
			c := m.Texture.At(p)
			referenceWrite(h, [3]float64{c.R, c.G, c.B})
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// These are outputs actually captured on Darwin/ARM64 from commit 324771b1,
// before the pinniped profile and rig changes, at seed 5 and quick detail.
// Texture samples are witnesses, not full texture or rendered-image hashes.
func TestPinnipedKeepsPreProfileOutput(t *testing.T) {
	for _, row := range []struct{ name, digest string }{
		{"harbor-seal-hauled", "52af606ebe10c4c36bbf3c2314da6f459963aa138d85ffeb37589708b8ee589d"},
		{"harbor-seal-swimming", "4a9fa6cfa00648d6ecb30a520d47bb0e30c64c4e9b5cbfa4634134d34071218f"},
		{"hawaiian-monk-seal-hauled", "4a19bed20b7d785451877722219506fa0c2edb7e443bac28dd7bb337ffcd7ce4"},
		{"walrus-fresh-5", "1ee2ffe1a1097de486b0b2c13826a3281ab713a3ba76ef3f2c0545a8cff0699b"},
	} {
		t.Run(row.name, func(t *testing.T) {
			kind, err := referencePinnipedKind(row.name)
			if err != nil {
				t.Fatal(err)
			}
			made, err := MakeWithComplexity(kind, Sow(5), "quick")
			if err != nil {
				t.Fatal(err)
			}
			if got := pinnipedWitness(t, made); got != row.digest {
				t.Fatalf("got %s, want pre-profile %s", got, row.digest)
			}
		})
	}
}
