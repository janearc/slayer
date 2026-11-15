//go:build draw

package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
)

func TestSampledFlameRetainsTerminalImageQuality(t *testing.T) {
	world := burning()
	hot, err := flame.NewBody(flame.Fire{Radius: .5, Height: 1.2,
		Absorption: 3, Hottest: measure.Kelvin(1800),
		Edge: measure.Kelvin(1100), Seed: 3}, 1e3)
	if err != nil {
		t.Fatal(err)
	}
	world.Volumes[0].Medium = hot
	world.Volumes[0].Material.Shine = hot.Shine()

	world.Lamp(thing.Sun(maths.V(0, 1, 1), maths.Grey(1)))
	eye := lookingAt(maths.V(0, 1, 4), maths.V(0, .4, 0))
	lens := film.Pinhole{Pixel: terminal.SextantPixel}
	tone := colour.Tone{Knee: .8, Top: 16}
	before := terminal.Picture(terminal.Chop(tone.Develop(
		sampled(world, eye, lens, 0, 168, 84, 1))))
	kept := world.Volumes
	world.Volumes = nil
	bare := terminal.Picture(terminal.Chop(tone.Develop(
		sampled(world, eye, lens, 0, 168, 84, 1))))
	world.Volumes = kept
	source := world.Volumes[0].Medium.(*flame.Body)

	field, err := flame.NewSampled(source, .03)
	if err != nil {
		t.Fatal(err)
	}
	world.Volumes[0].Medium = field
	after := terminal.Picture(terminal.Chop(tone.Develop(
		sampled(world, eye, lens, 0, 168, 84, 1))))
	errorSum, count := 0.0, 0
	for y := range 84 {
		for x := range 168 {
			a, b := before.RGBAAt(x, y), after.RGBAAt(x, y)
			clear := bare.RGBAAt(x, y)
			difference := math.Abs(float64(a.R)-float64(clear.R)) +
				math.Abs(float64(a.G)-float64(clear.G)) +
				math.Abs(float64(a.B)-float64(clear.B))
			if difference < 48 {

				continue
			}
			errorSum += math.Abs(float64(a.R)-float64(b.R)) +
				math.Abs(float64(a.G)-float64(b.G)) +
				math.Abs(float64(a.B)-float64(b.B))
			count += 3
		}
	}
	t.Logf("mean RGB error on visible fire: %.3f/255", errorSum/float64(max(1, count)))
	if count == 0 || errorSum/float64(count) > 6 {
		t.Fatalf("sampled flame lost terminal image quality: mean error %.3f", errorSum/float64(max(1, count)))
	}
}
