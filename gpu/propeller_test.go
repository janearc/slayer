package gpu

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"strings"
	"testing"
)

func propellerFixture(t *testing.T, blades int) medium.PropellerCavitation {
	t.Helper()
	field, err := medium.NewPropellerCavitation(medium.PropellerCavitationConfig{
		Blades: blades, Axis: maths.ZAxis, TipRadius: 1, RPM: 120, SlipstreamMPS: 4,
		LengthM: 8, CoreRadiusM: .15, DensityPerM: .8, PressureDeficitPa: 12000,
		RecoveryPaPerM: 2000, VisualOnset: true, FragmentScaleM: .3, FragmentContrast: 1,
		FragmentSeed: 0xdeadbeefcafe0705})
	if err != nil {
		t.Fatal(err)
	}
	return field
}

func TestPropellerEncodesOneAnalyticCardWithoutBaking(t *testing.T) {
	for _, blades := range []int{2, 5, 16} {
		world := thing.NewWorld()
		field := propellerFixture(t, blades)
		world.AddVolume(thing.Volume{Medium: field, Material: &thing.Material{Glow: maths.Grey(1)}})
		encoded, err := encode(world, maths.Subspace{}, film.Pinhole{Wide: 1}, 0, 2, 2)
		if err != nil {
			t.Fatal(err)
		}
		if encoded.counts.volumes != 1 || encoded.boxedCPU.Cells != 0 || len(encoded.boxes.bytes) != 4*len(propellerFields) {
			t.Fatalf("field unexpectedly baked/expanded: volumes=%d cells=%d cardbytes=%d", encoded.counts.volumes, encoded.boxedCPU.Cells, len(encoded.boxes.bytes))
		}
		card, err := propellerCardOf(field)
		if err != nil {
			t.Fatal(err)
		}
		if len(card) != len(propellerFields) {
			t.Fatal("variable card size")
		}
		for i, name := range propellerFields {
			if name == "blades" && card[i] != uint32(blades) {
				t.Fatal("blade count lost")
			}
			if name == "seed_low" && card[i] != 0xcafe0705 {
				t.Fatal("seed low lost")
			}
			if name == "seed_high" && card[i] != 0xdeadbeef {
				t.Fatal("seed high lost")
			}
		}
	}
}

func TestPropellerClearCloudStillRequiresSharedWaterTransport(t *testing.T) {
	water, err := thing.NewClearMedium("water", maths.Grey(.1), 1)
	if err != nil {
		t.Fatal(err)
	}
	cloud, err := medium.NewClearCloud(propellerFixture(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Through: 1, Index: 1.33, ClearMedium: water}})
	world.AddVolume(thing.NewVolume(cloud, nil))
	_, err = encodeKeptWithInitialMedia(world, nil, maths.Subspace{}, film.Pinhole{}, 0, 1, 1, nil, []trace.InitialMedium{{Index: 1.33, ClearMedium: water}})
	if err == nil || !strings.Contains(err.Error(), "local cloud volumes") {
		t.Fatalf("analytic kernel bypassed water/cloud refusal: %v", err)
	}
}

func TestPropellerRefusesCoreWhoseSquareUnderflowsMetal(t *testing.T) {
	config := propellerFixture(t, 5).Parameters()
	config.CoreRadiusM = 1e-30
	field, err := medium.NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = propellerCardOf(field); err == nil {
		t.Fatal("accepted a core that would create0/0 density inMetal")
	}
}
