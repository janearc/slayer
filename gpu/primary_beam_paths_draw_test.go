//go:build darwin && draw

package gpu

import (
	"math"
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
	base, primary := pictures[1].Pix[0].R, pictures[2].Pix[0].R
	if !reflectedAsWorked(base, primary) {
		t.Fatalf("reflected beam %g is not %g of the primary beam %g",
			base, glassReflectance*math.Exp(-puffDepth), primary)
	}
}

// What the base holds of a beam reflected in the glossy sphere, against
// what the primary pass holds of it, is worked from the scene: the pixel's
// ray goes straight down -z through the puff, strikes the sphere square on
// and comes straight back.
const (
	// glassReflectance is the share of light a glossy surface of index
	// 1.5 reflects at normal incidence: ((1.5 - 1) / (1.5 + 1))^2 = 0.04.
	glassReflectance = 0.04
	// puffDepth is the optical depth of the puff along a diameter: its
	// density is 0.3 times one less a smoothstep of the radius, and
	// the smoothstep's one less has a mean of a half, so two metres of
	// it are 0.3 * 2 * 0.5 = 0.3.
	puffDepth = 0.3
	// reflectedTolerance is how far the share may be from the worked
	// one: a thirtieth, for the 48-step march and float32 sums.
	reflectedTolerance = 0.03
)

// reflectedAsWorked is whether the base's red is the glossy share of the
// primary pass's. The beam and the puff are alike either side of the
// middle of the puff, so the leg back, through the same light and the same
// haze, gathers what the leg in did, kept by the leg in's depth and
// reflected by the sphere: base = reflectance * exp(-depth) * primary.
func reflectedAsWorked(base, primary float32) bool {
	want := glassReflectance * math.Exp(-puffDepth) * float64(primary)
	return primary > 0 && math.Abs(float64(base)-want) <=
		reflectedTolerance*want
}

// Anti: a matte sphere reflects no beam into the base, which the check
// on the glossy sphere reports as not the worked share.
func TestAMatteSphereIsReportedAsReflectingNoBeam(t *testing.T) {
	world := primaryBeamWorld(t)
	world.Things[0].Material = &thing.Material{}
	pictures := primaryBeamPictures(t, world, NativeMeshes)
	base, primary := pictures[1].Pix[0].R, pictures[2].Pix[0].R
	if !(primary > 0) {
		t.Fatal("the primary beam is missing", primary)
	}
	if reflectedAsWorked(base, primary) {
		t.Errorf("a matte sphere reads as reflecting: base %g, "+
			"primary %g", base, primary)
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
