//go:build darwin

package gpu

import (
	"encoding/binary"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestApproximateHeightFieldNormalsPackSlopeGridForMetal(t *testing.T) {
	for _, mode := range []string{"bilinear-vertex-slopes", "spectral-slopes"} {
		t.Run(mode, func(t *testing.T) {
			core, err := shape.NewHeightField(
				[]float64{0, 0, 0, 0, 1, 0, 0, 0, 0}, 3, 2)
			if err != nil {
				t.Fatal(err)
			}
			var made *shape.HeightField
			if mode == "bilinear-vertex-slopes" {
				made, err = core.WithApproximateSmoothNormals()
			} else {
				made, err = core.WithApproximateSpectralNormals(
					[]float64{.1, .2, .3, .4, .5, .6, .7, .8, .9},
					[]float64{-.9, -.8, -.7, -.6, -.5, -.4, -.3, -.2, -.1})
			}
			if err != nil {
				t.Fatal(err)
			}
			world := thing.NewWorld()
			world.Add(thing.Thing{Shape: made, Material: &thing.Material{Name: "sea"}})
			encoded, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1)
			if err != nil {
				t.Fatalf("encoded shading mode %q: %v", mode, err)
			}
			if len(encoded.heights.bytes) != 27*4 {
				t.Fatalf("packed height/slope bytes = %d, want 27 float32 words",
					len(encoded.heights.bytes)/4)
			}
			// Field.normal follows the six existing 16-byte fields. Its x and
			// y words point at the x/z slope arrays in the shared heights input;
			// z opts the kernel into interpolating those shading-only slopes.
			at := 6 * 16
			got := [3]int32{
				int32(binary.LittleEndian.Uint32(encoded.fields.bytes[at:])),
				int32(binary.LittleEndian.Uint32(encoded.fields.bytes[at+4:])),
				int32(binary.LittleEndian.Uint32(encoded.fields.bytes[at+8:])),
			}
			if got != [3]int32{9, 18, 1} {
				t.Fatalf("packed normal offsets/flag = %v, want [9 18 1]", got)
			}
		})
	}
}
