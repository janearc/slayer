//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// Keep the fragmented field in an actual image test, alongside the isolated
// strict test. This independently reviewed fixture has a local float32 error
// envelope; it does not change the general renderer tolerance.
func TestPropellerMetalCoupledDispersalVolumeImageMatchesCPU(t *testing.T) {
	p := propellerFixture(t, 5).Parameters()
	p.WanderM, p.WanderScaleM, p.SpreadStartM, p.SpreadPerM = .8, 1.5, 1, .12
	field, err := medium.NewPropellerCavitation(p)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: field,
		Material: &thing.Material{Glow: maths.RGB(.5, .7, 1)}})
	origin := maths.V(4, 3, 9)
	eye := maths.Subspace{Origin: origin,
		Orientation: maths.LookAt(origin, maths.V(0, 0, 2.5))}
	const side = 1
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	cpu, err := trace.Sampled(world, eye, lens, 0, 97, 65, side)
	if err != nil {
		t.Fatal(err)
	}
	gpu, profile, err := SampledWithInitialMedia(Software, world, eye,
		lens, 0, 97, 65, side, nil)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	apart := compare(cpu, gpu, 97, 65)
	maxAbs, totalAbs, total := 0.0, 0.0, 0.0
	for y := range 65 {
		for x := range 97 {
			a, b := cpu.At(x, y), gpu.At(x, y)
			for _, pair := range [][2]float32{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}} {
				d := math.Abs(float64(pair[0]) - float64(pair[1]))
				maxAbs = math.Max(maxAbs, d)
				totalAbs += d
				total += float64(pair[0])
			}
		}
	}
	t.Logf("device=%q side%d apart%+v maxabs%g L1%g signal%g",
		profile.Device, side, apart, maxAbs, totalAbs, total)
	for _, problem := range findings(apart, tolerance{2e-5, exact.largest}) {
		t.Error(problem)
	}
	if !(maxAbs <= 3e-6) {
		t.Errorf("coupled absolute RGB error %g exceeds 3e-6", maxAbs)
	}
}
