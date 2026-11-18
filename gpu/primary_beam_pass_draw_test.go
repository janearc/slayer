//go:build darwin && draw

package gpu

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// primaryBeamWorld separates source colours so omitted light cannot hide
// behind a successful reconstruction of two wrong contributions.
func primaryBeamWorld(t *testing.T) *thing.World {
	t.Helper()
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(0, -4, 0),
		Toward: maths.YAxis, Strength: maths.RGB(12, 0, 0)},
		measure.Metres(0.6), measure.Metres(8), measure.Degrees(0))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Lamp(beam)
	world.Lamp(film.Flood{Strength: maths.RGB(0, 2, 0)})
	world.AddVolume(thing.Volume{
		Medium: medium.Puff{Radius: 1, Thick: 0.3},
		Material: &thing.Material{Colour: maths.Grey(0.8),
			Glow: maths.RGB(0, 0, 0.2)}})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Glow: maths.RGB(0, 0, 0.4)},
		Motion:   at(0, 0, -4)})
	return world
}

// primaryBeamPictures requires actual device work for every contribution.
func primaryBeamPictures(t *testing.T, world *thing.World,
	mode Mode) [3]*frame.Frame {
	t.Helper()
	return primaryBeamGrid(t, world, mode, 1, 1, 1)
}

// primaryBeamGrid draws matching complete, base and beam sample grids.
func primaryBeamGrid(t *testing.T, world *thing.World,
	mode Mode, width, height, side int) [3]*frame.Frame {
	t.Helper()
	var pictures [3]*frame.Frame
	for index, pass := range []CompositePass{CompletePicture,
		WithoutPrimaryBeams, PrimaryBeams} {
		drawer, err := NewCompositeFrames(mode, pass)
		if err != nil {
			t.Fatal(err)
		}
		picture, profile, err := drawer.Draw(world, 1,
			maths.Subspace{Origin: maths.V(0, 0, 3)},
			film.Pinhole{Wide: 35 * maths.Degree}, 0, width, height, side)
		drawer.Close()
		if err != nil {
			t.Fatal(err)
		}
		if profile.Device == "" || profile.TraceGPU <= 0 {
			t.Fatalf("pass %d did not execute on Metal: %+v", pass, profile)
		}
		pictures[index] = picture
	}
	return pictures
}

// assertPrimaryBeamSum checks linear reconstruction before display mapping.
func assertPrimaryBeamSum(t *testing.T, pictures [3]*frame.Frame) {
	t.Helper()
	for index, full := range pictures[0].Pix {
		base, beam := pictures[1].Pix[index], pictures[2].Pix[index]
		for _, pair := range [][2]float32{{full.R, base.R + beam.R},
			{full.G, base.G + beam.G}, {full.B, base.B + beam.B}} {
			if math.Abs(float64(pair[0]-pair[1])) > 3e-6 {
				t.Fatalf("pixel %d: base plus beam %g, complete %g",
					index, pair[1], pair[0])
			}
		}
	}
}

// TestPrimaryBeamPassKeepsSourcesSeparate also exercises the per-lamp
// visibility store, keeping its emission and flood out of the beam.
func TestPrimaryBeamPassKeepsSourcesSeparate(t *testing.T) {
	for _, mode := range []Mode{Software, FasterLighting} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			pictures := primaryBeamPictures(t, primaryBeamWorld(t), mode)
			assertPrimaryBeamSum(t, pictures)
			base, beam := pictures[1].Pix[0], pictures[2].Pix[0]
			if !(beam.R > 0.001) || beam.G != 0 || beam.B != 0 {
				t.Fatalf("beam absent or nonbeam light leaked: %+v", beam)
			}
			if base.R != 0 || !(base.G > 0) || !(base.B > 0) {
				t.Fatalf("base lost its sources or contains beam: %+v", base)
			}
		})
	}
}

// TestPrimaryBeamPassKeepsSkyAndGasSeparate covers the gas transport
// branch and the painted medium's explicitly enabled sky illumination.
func TestPrimaryBeamPassKeepsSkyAndGasSeparate(t *testing.T) {
	for _, air := range []bool{false, true} {
		world := primaryBeamWorld(t)
		world.Sky = sky.Gradient{Horizon: maths.RGB(0, 0.4, 0),
			Zenith: maths.RGB(0, 0.4, 0), Ground: maths.RGB(0, 0.4, 0)}
		light, err := thing.NewVolumeIllumination(4)
		if err != nil {
			t.Fatal(err)
		}
		world.VolumeIllumination = light
		if air {
			volume := puffOf(dusk(0.4), 1, 0.3)
			world.Volumes = []*thing.Volume{&volume}
		}
		pictures := primaryBeamPictures(t, world, FasterLighting)
		assertPrimaryBeamSum(t, pictures)
		beam := pictures[2].Pix[0]
		if !(beam.R > 0.001) || beam.G != 0 || beam.B != 0 {
			t.Fatalf("air=%v: missing beam or leaking sky: %+v", air, beam)
		}
		if !(pictures[1].Pix[0].G > 0) {
			t.Fatal("base lost nonbeam illumination")
		}
	}
}

// TestPrimaryBeamPassKeepsOverlappingExtinction requires the late beam
// to dim behind an overlapping, coloured absorber without counting it twice.
func TestPrimaryBeamPassKeepsOverlappingExtinction(t *testing.T) {
	world := primaryBeamWorld(t)
	clear := primaryBeamPictures(t, world, Software)
	absorber := puffOf(gas.Gas{Name: "authored coloured absorber",
		Visible: func(nm float64) (float64, float64) { return nm / 550, 0 }},
		1, 0.8)
	absorber.Motion = at(0, 0, 0.8)
	world.AddVolume(absorber)
	attenuated := primaryBeamPictures(t, world, Software)
	assertPrimaryBeamSum(t, attenuated)
	before, after := clear[2].Pix[0].R, attenuated[2].Pix[0].R
	if !(after > 0 && after < before*0.9) {
		t.Fatalf("overlapping extinction missing: %g -> %g", before, after)
	}
}

// TestPrimaryBeamPassStopsAtForegroundAndHasNoVacuumSkin rejects a
// screen-space overlay that ignores scene depth or invents visible light.
func TestPrimaryBeamPassStopsAtForegroundAndHasNoVacuumSkin(t *testing.T) {
	for _, vacuum := range []bool{false, true} {
		world := primaryBeamWorld(t)
		if vacuum {
			world.Volumes = nil
		} else {
			world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.3},
				Material: &thing.Material{Glow: maths.RGB(0, 0, 0.7)},
				Motion:   at(0, 0, 2)})
		}
		pictures := primaryBeamPictures(t, world, FasterLighting)
		assertPrimaryBeamSum(t, pictures)
		beam := pictures[2].Pix[0]
		if beam.R != 0 || beam.G != 0 || beam.B != 0 {
			t.Fatalf("vacuum=%v: forbidden beam light %+v", vacuum, beam)
		}
		if !(pictures[1].Pix[0].B > 0) {
			t.Fatal("base emission was lost")
		}
	}
}

// TestPrimaryBeamPassRetainsSampledLampWeights checks more lamps than
// direct selection can ask: both passes must use the complete distribution.
func TestPrimaryBeamPassRetainsSampledLampWeights(t *testing.T) {
	world := primaryBeamWorld(t)
	// Flood selection with many lamps is explicitly unsupported by the GPU.
	world.Lamps = world.Lamps[:1]
	for index := range 24 {
		world.Lamp(emit.Lit{At: maths.V(float64(index)-12, 3, 2),
			Strength: maths.RGB(0, 2, 0)})
	}
	pictures := primaryBeamPictures(t, world, Software)
	assertPrimaryBeamSum(t, pictures)
	if !(pictures[2].Pix[0].R > 0) || pictures[2].Pix[0].G != 0 {
		t.Fatal("sampled beam missing or contaminated", pictures[2].Pix[0])
	}
}

// TestPrimaryBeamPassAcceptsConstructedPinhole uses the public camera
// constructor, as searching does, instead of a hand-built lens value.
func TestPrimaryBeamPassAcceptsConstructedPinhole(t *testing.T) {
	lens, err := film.PinholeOf(45 * maths.Degree)
	if err != nil {
		t.Fatal(err)
	}
	drawer, err := NewCompositeFrames(Software, PrimaryBeams)
	if err != nil {
		t.Fatal(err)
	}
	defer drawer.Close()
	picture, profile, err := drawer.Draw(primaryBeamWorld(t), 1,
		maths.Subspace{Origin: maths.V(0, 0, 3)}, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Device == "" || profile.TraceGPU <= 0 || !(picture.Pix[0].R > 0) {
		t.Fatal("constructed pinhole did not draw the beam on Metal")
	}
}
