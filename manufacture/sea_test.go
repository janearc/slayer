package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/thing"
)

// testSea is a sheet of a fresh breeze's sea, a hundred metres a side on
// sixty-four samples, standing at the world's origin.
func testSea() Sea {
	return Sea{Model: oceantex.Sea{Wind: 8, Fetch: 50e3},
		Seed: 1, Sheet: &Sheet{Extent: 100, Samples: 64},
		At:     maths.Subspace{Origin: maths.V(0, 2, 0), Scale: 1},
		Optics: thing.OceanSurface{SunLampIndex: -1, FillLampIndex: -1}}
}

// waterWear is the wear of a sea that wears nothing in particular.
func waterWear(open *arid.Sea) (*thing.Material, error) {
	return &thing.Material{Name: "water"}, nil
}

func TestASheetsBoxIsItsSquareAboutItsMeanAndTheCrestsReach(t *testing.T) {
	box, err := testSea().Extent()
	if err != nil {
		t.Fatal(err)
	}
	reach := crestOverSignificant * oceantex.Sea{Wind: 8, Fetch: 50e3}.Height()
	if math.Abs(box.Max.X-50) > 1e-9 || math.Abs(box.Min.Z+50) > 1e-9 ||
		math.Abs(box.Max.Y-(2+reach)) > 1e-9 ||
		math.Abs(box.Min.Y-(2-reach)) > 1e-9 {
		t.Errorf("a sheet's box %+v, not 100 across and %v either side of 2",
			box, reach)
	}
}

func TestACollarWidensASheetsBoxOnEverySide(t *testing.T) {
	collared := testSea()
	collared.Sheet = &Sheet{Extent: 100, Samples: 64,
		Mean: &MeanPlane{Collar: 30}}
	box, err := collared.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(box.Max.X-80) > 1e-9 || math.Abs(box.Min.X+80) > 1e-9 {
		t.Errorf("a collar of 30 left the box %+v, not 80 out", box)
	}
}

func TestASeaWithNoEdgeIsAskedItsBoxWithinTheGround(t *testing.T) {
	open := testSea()
	open.Sheet = nil
	if _, err := open.Extent(); err == nil {
		t.Fatal("a sea with no edge was given a box with no ground to lie in")
	}
	open.Within = maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(400, 90, 300)}
	box, err := open.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.X != 0 || box.Max.X != 400 || box.Max.Z != 300 {
		t.Errorf("an open sea's box %+v is not the ground's plane", box)
	}
	if box.Max.Y >= 90 || box.Min.Y > 2 {
		t.Errorf("an open sea's box %+v is not a slab about its mean", box)
	}
}

func TestASheetContinuedForeverIsHeldToTheGroundToo(t *testing.T) {
	endless := testSea()
	endless.Sheet = &Sheet{Extent: 100, Samples: 64,
		Mean: &MeanPlane{Infinite: true}}
	if _, err := endless.Extent(); !errors.Is(err, errSea) {
		t.Errorf("an endless plane was given a box with no ground: %v", err)
	}
}

func TestASheetStagesAsOneBoundaryAndTheWorldsOcean(t *testing.T) {
	world := thing.NewWorld()
	if err := testSea().Staged(world, waterWear); err != nil {
		t.Fatal(err)
	}
	ocean := world.OceanSurface
	if len(world.Things) != 1 || ocean == nil ||
		ocean.Boundary != world.Things[0] {
		t.Fatalf("%d things and ocean %+v, not one boundary", len(world.Things),
			ocean)
	}
	if ocean.ExtentMetres != 100 || ocean.Mean != maths.V(0, 2, 0) ||
		ocean.SunLampIndex != -1 || ocean.WaveMotion != nil {
		t.Errorf("the world's ocean %+v does not say what the sea was", ocean)
	}
}

func TestASeaWhoseWearRefusesAddsNothing(t *testing.T) {
	world := thing.NewWorld()
	refuse := func(*arid.Sea) (*thing.Material, error) {
		return nil, errors.New("no")
	}
	if err := testSea().Staged(world, refuse); err == nil ||
		len(world.Things) != 0 || world.OceanSurface != nil {
		t.Errorf("a refused wear left %d things and ocean %v (%v)",
			len(world.Things), world.OceanSurface, err)
	}
}

func TestASeaWithNoWindAndNoBandsIsRefusedItsSheet(t *testing.T) {
	still := testSea()
	still.Sheet = &Sheet{Extent: 0, Samples: 0}
	if err := still.Staged(thing.NewWorld(), waterWear); err == nil {
		t.Error("a sheet of no extent was staged")
	}
}
