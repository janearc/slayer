//go:build draw

package trace

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// TestOceanDiffuseSkyWaterOnlyControl renders a synthetic water-only control.
// It writes only when its output folder is explicitly supplied.
func TestOceanDiffuseSkyWaterOnlyControl(test *testing.T) {
	directory := os.Getenv("SLAYER_OCEAN_DIFFUSE_SKY_OUT")
	if directory == "" {
		test.Skip("set SLAYER_OCEAN_DIFFUSE_SKY_OUT to write the water-only control")
	}
	const (
		width, height = 160, 90
		index         = 1.333
	)
	core, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 80)
	if err != nil {
		test.Fatal(err)
	}
	continuation, err := seaocean.NewInfiniteMeanContinuation(core, 0, 20)
	if err != nil {
		test.Fatal(err)
	}
	base, err := thing.NewClearMediumWithScattering("synthetic-water",
		maths.RGB(.01, .015, .025), maths.RGB(.08, .1, .12), .25,
		1, 8, 12)
	if err != nil {
		test.Fatal(err)
	}
	sampled, err := base.WithDiffuseSkyInscatter(16)
	if err != nil {
		test.Fatal(err)
	}
	render := func(medium *thing.ClearMedium) *frame.Frame {
		test.Helper()
		world := thing.NewWorld()
		boundary := &thing.Thing{Shape: continuation,
			Material: &thing.Material{Name: "synthetic water", Through: 1,
				Gloss: 1, Index: index, ClearMedium: medium}}
		world.Things = append(world.Things, boundary)
		sun, err := emit.NewDistant(maths.V(.35, .93675, 0),
			maths.Light{})
		if err != nil {
			test.Fatal(err)
		}
		fill, err := emit.NewDistant(maths.Up, maths.Light{})
		if err != nil {
			test.Fatal(err)
		}
		world.Lamps = []thing.Lamp{sun, fill}
		world.Sky = sky.Gradient{Horizon: maths.RGB(.3, .5, .75),
			Zenith: maths.RGB(.5, .72, 1), Ground: maths.Light{}}
		world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
			Mean: maths.V(0, 0, 0), ExtentMetres: 80,
			MetresPerUnit: 1, DirectSunMeanPlane: true,
			InfiniteMeanPlane: true, SunRoute: thing.OceanSunRouteGround,
			SunLampIndex: 0, FillLampIndex: 1}
		eye := maths.Subspace{Origin: maths.V(0, -2, -2),
			Orientation: maths.LookAt(maths.V(0, -2, -2), maths.V(0, -10, -2))}
		picture, err := RenderWithInitialMedia(world, eye,
			film.Pinhole{Wide: 70 * maths.Degree}, time.Duration(0),
			width, height, []InitialMedium{{Index: index,
				ClearMedium: medium}})
		if err != nil {
			test.Fatal(err)
		}
		return picture
	}
	without := render(base)
	with := render(sampled)
	if without.At(width/2, height/2) == with.At(width/2, height/2) {
		test.Fatal("sky opt-in did not change the central water-only pixel")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	for name, picture := range map[string]*frame.Frame{
		"water-surface-control": without,
		"water-diffuse-sky":     with,
	} {
		stem := filepath.Join(directory, name)
		if err := raw.Write(stem+".pfm", picture); err != nil {
			test.Fatal(err)
		}
		file, err := os.OpenFile(stem+".png", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			test.Fatal(err)
		}
		if err := png.Encode(file, colour.Tone{Exposure: 1, Knee: .8,
			Top: 2.2}.Develop(picture)); err != nil {
			_ = file.Close()
			test.Fatal(err)
		}
		if err := file.Close(); err != nil {
			test.Fatal(err)
		}
	}
	test.Logf("rendered %dx%d water-only downward CPU control with zero-strength tagged sun; 8 camera steps × 16 sky directions = 128 sky-boundary evaluations per active underwater pixel", width, height)
}
