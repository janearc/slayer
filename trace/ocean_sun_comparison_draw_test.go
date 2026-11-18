//go:build draw

package trace

import (
	"os"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// TestOceanSunComparisonDraw is a small, explicit CPU source-path study. It
// compares the old straight shadow path with opt-in mean-plane sunlight.
func TestOceanSunComparisonDraw(t *testing.T) {
	dir := os.Getenv("SLAYER_OCEAN_SUN_COMPARISON_OUT")
	if dir == "" {
		t.Skip("set SLAYER_OCEAN_SUN_COMPARISON_OUT to retain the comparison")
	}
	const width, height = 161, 91
	water, err := thing.NewClearMedium("synthetic-study-water", maths.RGB(.08, .04, .02), 1)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 100)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: surface, Material: &thing.Material{
		Name: "synthetic flat water", Through: 1, Gloss: 1, Index: 1.333, ClearMedium: water}}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	for _, p := range []maths.Vec{maths.V(-2, -2, 0), maths.V(0, -2, 1), maths.V(2, -2, 2)} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: resting(p),
			Material: &thing.Material{Name: "white reference sphere", Colour: maths.Grey(.8)}})
	}
	sun, err := emit.NewDistant(maths.V(.35, 1, -1), maths.Grey(8))
	if err != nil {
		t.Fatal(err)
	}
	fill, err := emit.NewDistant(maths.Up, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps = []thing.Lamp{sun, fill}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary, ExtentMetres: 100,
		MetresPerUnit: 1, SunRoute: thing.OceanSunRouteGround, SunLampIndex: 0, FillLampIndex: 1}
	eye := mediumEye(maths.V(0, -2, -8), maths.V(0, 0, 1))
	draw := func(enabled bool) *frame.Frame {
		t.Helper()
		world.OceanSurface.DirectSunMeanPlane = enabled
		made, err := RenderWithInitialMedia(world, eye, film.Pinhole{Wide: 55 * maths.Degree},
			0, width, height, []InitialMedium{{Index: 1.333, ClearMedium: water}})
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	off, on := draw(false), draw(true)
	if off.At(width/2, height/2) == on.At(width/2, height/2) {
		t.Fatal("sun control and mean-plane center pixels are identical")
	}
	writeClearInscatterImage(t, dir, "legacy-shadow.png", off)
	writeClearInscatterImage(t, dir, "mean-plane-sun.png", on)
	t.Logf("flat water, synthetic RGB extinction .08/.04/.02 per metre, sun (0.35,1,-1); no scattering, no sky, no terrain; center pixels control=%+v transmitted=%+v", off.At(width/2, height/2), on.At(width/2, height/2))
}
