//go:build darwin

package gpu

import (
	"encoding/binary"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// Deferred flame emission does not supply a paint grid. A real spatial
// texture is the positive control and must keep its separately sampled paint.
func TestFlamePaintMarkerDistinguishesDeferredAndSpatialColour(t *testing.T) {
	for _, painted := range []bool{false, true} {
		grid := fireGridFixture(t, 5, 0)
		var paint texture.Texture
		if painted {
			paint = banded{}
			grid.paint = paint
		}
		encoded := newScene()
		material := encoded.material(&thing.Material{
			Colour: maths.Grey(.2), Texture: paint, Shine: grid.shine,
		})
		encoded.boxedVolume(maths.Subspace{}, grid, material)
		if len(encoded.volumes.bytes) != 13*16 {
			t.Fatal("flame fixture did not produce one volume record")
		}
		// Volume's shone int4 is lane eleven: emission first, paint second.
		const shone = 11 * 16
		emission := int32(binary.LittleEndian.Uint32(encoded.volumes.bytes[shone:]))
		colour := int32(binary.LittleEndian.Uint32(encoded.volumes.bytes[shone+4:]))
		if emission < 0 {
			t.Fatal("flame fixture lost its emission range")
		}
		if painted {
			if len(encoded.flames) != 0 || colour < 0 {
				t.Fatal("spatial paint was deferred or lost its sampled grid")
			}
		} else if len(encoded.flames) != 1 || colour >= 0 {
			t.Fatal("unpainted deferred flame claimed a paint-grid range")
		}
	}
}
