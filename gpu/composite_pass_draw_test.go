//go:build darwin && draw

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// TestCompositePassTransport checks a closed-form central chord as well as
// reconstruction of the ordinary trace. A front surface must stop the fog.
func TestCompositePassTransport(t *testing.T) {
	for _, mode := range []Mode{Software, NativeMeshes} {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 0.2}, Material: &thing.Material{Glow: maths.RGB(0.2, 0.1, 0.05)}})
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Glow: maths.RGB(0.3, 0.5, 0.8)}, Motion: at(0, 0, -4)})
		eye := maths.Subspace{Origin: maths.V(0, 0, 3)}
		var pictures [3]*frame.Frame
		var trans *frame.Frame
		for _, pass := range []CompositePass{CompletePicture, SurfacePlate, VolumeTransport} {
			drawer, err := NewCompositeFrames(mode, pass)
			mustDevice(t, err)
			pictures[pass], _, err = drawer.Draw(world, 1, eye, film.Pinhole{}, 0, 1, 1, 1)
			mustDevice(t, err)
			if pass == VolumeTransport {
				trans = drawer.Transmission()
			}
			drawer.Close()
			if drawer.Transmission() != nil {
				t.Fatal("closed drawer retained transmission")
			}
		}
		got := trans.Pix[0]
		// Puff tapers by 1-smoothstep(r). Each radial half integrates
		// to half the radius, so this central chord's optical depth is
		// Thick*Radius, not Thick times the full diameter.
		want := math.Exp(-0.2)
		for _, value := range []float32{got.R, got.G, got.B} {
			if math.Abs(float64(value)-want) > 2e-5 {
				t.Fatalf("transmission %g, want %g", value, want)
			}
		}
		p, s, r := pictures[SurfacePlate].Pix[0], pictures[VolumeTransport].Pix[0], pictures[CompletePicture].Pix[0]
		for _, pair := range [][2]float32{{s.R + got.R*p.R, r.R}, {s.G + got.G*p.G, r.G}, {s.B + got.B*p.B, r.B}} {
			if math.Abs(float64(pair[0]-pair[1])) > 2e-6 {
				t.Fatalf("decomposed %g != ordinary %g", pair[0], pair[1])
			}
		}
		if p.R != 0.3 || s.R <= 0 {
			t.Fatalf("plate or scattering missing: %+v %+v", p, s)
		}
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25}, Material: &thing.Material{}, Motion: at(0, 0, 2)})
		drawer, err := NewCompositeFrames(mode, VolumeTransport)
		mustDevice(t, err)
		scatter, _, err := drawer.Draw(world, 2, eye, film.Pinhole{}, 0, 1, 1, 1)
		mustDevice(t, err)
		if scatter.Pix[0].R != 0 || drawer.Transmission().Pix[0].R != 1 {
			t.Fatal("fog leaked through the foreground occluder")
		}
		drawer.Close()
	}
}
