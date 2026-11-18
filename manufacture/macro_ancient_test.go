package manufacture

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"math"
	"testing"
)

func TestMacroAncientNativeGeometryAtPlanetaryScale(t *testing.T) {
	for _, style := range []MacrostructureStyle{TerracedArcology, TwistedSpires, InhabitedBridge} {
		spec := DefaultMacrostructure(style)
		spec.Width = spec.Width.Times(100)
		spec.Depth = spec.Depth.Times(100)
		spec.Height = spec.Height.Times(100)
		spec.Wildness = 1
		spec.LivingCover = true
		kind, err := Macrostructure(spec)
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(kind, Sow(31))
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(style, err)
		}
		if len(parts) != 4 {
			t.Fatalf("%s has %d material batches", style, len(parts))
		}
		if made.faces() > 100000 {
			t.Fatal("growth exceeds native face budget")
		}
		for _, part := range parts {
			if part.Material.Texture == nil {
				t.Fatal("ancient surface has no patina")
			}
		}
		if made.Bounds().Max.Y < measure.Metres(100000).InMetres() {
			t.Fatal("specimen lost planetary dimensions")
		}
	}
}

func TestMacroAncientRejectsInvalidWildness(t *testing.T) {
	for _, value := range []float64{-1, 1.1, math.NaN(), math.Inf(1)} {
		spec := DefaultMacrostructure(TwistedSpires)
		spec.Wildness = value
		if _, err := Macrostructure(spec); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}

func TestMacroAncientRuggedTerrainNativeMeshes(t *testing.T) {
	for _, style := range []LandformStyle{MountainMassif, CalderaVolcano} {
		spec := DefaultLandform(style)
		spec.Ruggedness = 1
		spec.Radius = measure.Metres(300000)
		spec.Height = measure.Metres(24000)
		if style == CalderaVolcano {
			spec.CraterRadius = measure.Metres(45000)
			spec.CraterDepth = measure.Metres(9000)
		}
		kind, err := Landform(spec)
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(kind, Sow(35))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = made.Things(maths.Local()); err != nil {
			t.Fatal(err)
		}
		if made.faces() < 10000 || made.faces() > 100000 {
			t.Fatal("detailed terrain face budget", made.faces())
		}
	}
}

// Planetary bridges must remain closed across seed-dependent massing and
// unequal physical dimensions, rather than only the original seed31 specimen.
func TestMacroAncientBridgeSeededPlanetaryEnvelopes(t *testing.T) {
	for number := uint64(0); number < 64; number++ {
		t.Run(fmt.Sprint(number), func(t *testing.T) {
			spec := DefaultMacrostructure(InhabitedBridge)
			size := .65 + 1.1*noise.Hash(number, 1)
			spec.Width = measure.Metres(1000000 * size)
			spec.Depth = measure.Metres(180000 * size)
			spec.Height = measure.Metres(220000 * size)
			spec.Wildness = .85
			spec.LivingCover = true
			kind, err := Macrostructure(spec)
			if err != nil {
				t.Fatal(err)
			}
			made, err := Make(kind, Sow(number))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = made.Things(maths.Local()); err != nil {
				for index, piece := range made.parts {
					if _, problem := piece.skin().meshed(); problem != nil {
						t.Logf("part %d material %d: %v", index, piece.of, problem)
					}
				}
				t.Fatal(err)
			}
		})
		if t.Failed() {
			return
		}
	}
}
