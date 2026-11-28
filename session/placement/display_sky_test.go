package placement

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
)

func TestDisplaySkySamplesProducerLightThroughTheWorkingLens(t *testing.T) {
	scene := lookScene()
	scene.Width, scene.Height = 80, 120
	scene.Lens.Pixel = 1.2
	scene.Eye.Orientation = maths.About(maths.ZAxis, 20*maths.Degree)
	source := sky.Gradient{
		Horizon: maths.RGB(.3, .4, .5), Zenith: maths.RGB(.7, .8, 2), Ground: maths.RGB(.1, .2, .3),
		Up: maths.V(1, 2, 1),
	}
	scene.SkyGradient = WorkingSkyGradient(source)
	for _, pixel := range [][2]int{{0, 0}, {41, 23}, {79, 0}, {42, 70}, {79, 119}} {
		ray := film.Look(scene.Eye, scene.Lens, pixel[0], pixel[1], scene.Width, scene.Height, maths.Band{})
		want := source.Light(ray.Dir)
		got := scene.SampleSky((float64(pixel[0])+.5)/float64(scene.Width), (float64(pixel[1])+.5)/float64(scene.Height))
		if math.Abs(want.R-got.R)+math.Abs(want.G-got.G)+math.Abs(want.B-got.B) > 1e-12 {
			t.Fatalf("pixel %v sampled %+v, want %+v", pixel, got, want)
		}
	}
}

func TestDisplaySkyRefusesOtherModelsAndInvalidLight(t *testing.T) {
	if WorkingSkyGradient(sky.Stars{}) != nil || WorkingSkyGradient(nil) != nil ||
		WorkingSkyGradient(sky.Gradient{Zenith: maths.Grey(math.Inf(1))}) != nil ||
		WorkingSkyGradient(sky.Gradient{Ground: maths.Grey(-1)}) != nil {
		t.Fatal("an unsupported sky reached the gradient representation")
	}
}
