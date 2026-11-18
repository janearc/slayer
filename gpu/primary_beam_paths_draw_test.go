//go:build darwin && draw

package gpu

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// TestPrimaryBeamPassReconstructsNativeAntialiasing includes a real
// triangle mesh, several pixels and four samples per pixel on NativeMeshes.
func TestPrimaryBeamPassReconstructsNativeAntialiasing(t *testing.T) {
	world := primaryBeamWorld(t)
	world.Add(thing.Thing{Shape: floor(t, 5),
		Material: &thing.Material{Glow: maths.RGB(0, 0, 0.2)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -5),
			Orientation: maths.About(maths.XAxis, maths.Degree*90)})})
	pictures := primaryBeamGrid(t, world, NativeMeshes, 5, 3, 2)
	assertPrimaryBeamSum(t, pictures)
	energy := float32(0)
	for index, complete := range pictures[0].Pix {
		base := pictures[1].Pix[index]
		if base.Cover != complete.Cover || base.Depth != complete.Depth {
			t.Fatalf("base geometry differs at pixel %d", index)
		}
		energy += pictures[2].Pix[index].R
	}
	if !(energy > 0.001) {
		t.Fatal("native multisample reconstruction had no beam")
	}
}

// TestPrimaryBeamPassIsZeroWithoutBeams keeps a lit, emitting scene
// intact when no beam lamps exist, including on the native multisample grid.
func TestPrimaryBeamPassIsZeroWithoutBeams(t *testing.T) {
	world := primaryBeamWorld(t)
	world.Lamps = world.Lamps[1:]
	pictures := primaryBeamGrid(t, world, NativeMeshes, 5, 3, 2)
	assertPrimaryBeamSum(t, pictures)
	for _, pixel := range pictures[2].Pix {
		if pixel.R != 0 || pixel.G != 0 || pixel.B != 0 {
			t.Fatal("no-beam scene acquired beam light", pixel)
		}
	}
	if !(pictures[1].Pix[7].B > 0) {
		t.Fatal("no-beam base lost emission")
	}
}

// TestPrimaryBeamPassKeepsReflectedBeamInBase requires a reflected
// beam contribution to remain in the base, without primary double counting.
func TestPrimaryBeamPassKeepsReflectedBeamInBase(t *testing.T) {
	world := primaryBeamWorld(t)
	world.Things[0].Material = &thing.Material{Gloss: 1, Index: 1.5}
	pictures := primaryBeamPictures(t, world, NativeMeshes)
	assertPrimaryBeamSum(t, pictures)
	if !(pictures[1].Pix[0].R > 0.001) || !(pictures[2].Pix[0].R > 0.001) {
		t.Fatal("reflected or primary beam is missing",
			pictures[1].Pix[0], pictures[2].Pix[0])
	}
}

// TestPrimaryBeamPassKeepsRefractedBeamInBase places all scattering
// behind glass: only the refracted secondary path may bring it to the eye.
func TestPrimaryBeamPassKeepsRefractedBeamInBase(t *testing.T) {
	world := primaryBeamWorld(t)
	world.Things[0].Material = &thing.Material{Through: 1, Gloss: 1, Index: 1.5}
	world.Volumes[0].Motion = at(0, 0, -7)
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(0, -4, -7),
		Toward: maths.YAxis, Strength: maths.RGB(12, 0, 0)},
		measure.Metres(0.6), measure.Metres(8), measure.Degrees(0))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps[0] = beam
	pictures := primaryBeamPictures(t, world, NativeMeshes)
	assertPrimaryBeamSum(t, pictures)
	if !(pictures[1].Pix[0].R > 0.001) || pictures[2].Pix[0].R != 0 {
		t.Fatal("refracted beam missing or assigned to primary transport",
			pictures[1].Pix[0], pictures[2].Pix[0])
	}
}
