package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// tinted is a medium thicker to some light than other: its density
// times tint, channel by channel, as trace.Tinted asks.
type tinted struct {
	medium.Medium
	tint maths.Light
}

// Tint is how many times its density each channel meets.
func (slab tinted) Tint() maths.Light { return slab.tint }

// a tinted medium is a door of the tracer's
var _ trace.Tinted = tinted{}

// soot's tilt at the eye's three wavelengths, 610, 550 and 465 nm, by
// hottel and broughton's 1.39, worked out here: thickest to blue.
var sooty = maths.RGB(math.Pow(550.0/610, 1.39), 1,
	math.Pow(550.0/465, 1.39))

// rgb is a light's three channels, red, green and blue.
func rgb(pixel maths.Light) [3]float64 {
	return [3]float64{pixel.R, pixel.G, pixel.B}
}

// Kirchhoff, channel by channel: a slab of density k and depth d,
// tinted t, glowing g before a sky of s, is in each channel
// g (1 - exp(-k t d)) + s exp(-k t d). a frame keeps float32.
func TestATintedGlowingSlabIsKirchhoffsInEachChannel(t *testing.T) {
	const density, depth, sky = 0.4, 2.0, 0.25
	glow := maths.RGB(3, 2, 1)
	slab := &thing.Volume{Medium: tinted{fake.NewSlab(
		maths.V(-1, -1, -depth/2), maths.V(1, 1, depth/2), density), sooty},
		Material: &thing.Material{Glow: glow}}
	world := &thing.World{Volumes: []*thing.Volume{slab},
		Sky: fake.NewSky(maths.Grey(sky))}
	ahead := maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}
	picture, err := trace.Render(world, maths.Subspace{},
		fake.NewAimed(ahead), 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	pixel := picture.At(0, 0)
	got := rgb(maths.RGB(float64(pixel.R), float64(pixel.G),
		float64(pixel.B)))
	glows, tints := rgb(glow), rgb(sooty)
	for c := range got {
		kept := math.Exp(-density * tints[c] * depth)
		want := glows[c]*(1-kept) + sky*kept
		if math.Abs(got[c]/want-1) > 1e-6 {
			t.Errorf("channel %d: %.9f through the slab, kirchhoff says "+
				"%.9f", c, got[c], want)
		}
	}
}

// a lamp's light through a tinted medium keeps exp(-k t l) in each
// channel, l the part of its way to the ground inside the medium: the
// plane's ground drawn with a tinted slab between and without it.
func TestALampThroughATintedMediumKeepsEachChannelsShare(t *testing.T) {
	render := func(volumes []*thing.Volume) []maths.Light {
		world := plane.world()
		world.Volumes = volumes
		picture, err := trace.Render(world, maths.Subspace{},
			plane.sensor(), 0, len(plane.xs), 1)
		if err != nil {
			t.Fatal(err)
		}
		var drawn []maths.Light
		for x := range plane.xs {
			pixel := picture.At(x, 0)
			drawn = append(drawn, maths.RGB(float64(pixel.R),
				float64(pixel.G), float64(pixel.B)))
		}
		return drawn
	}
	bare := render(nil)
	dimmed := render([]*thing.Volume{{Medium: tinted{fake.NewSlab(
		maths.V(-slabHalf, slabFloor, -slabHalf),
		maths.V(slabHalf, slabTop, slabHalf), slabDensity), sooty}}})
	tints := rgb(sooty)
	for at, x := range plane.xs {
		way := math.Hypot(x, plane.height)
		inside := way * (plane.height - slabFloor) / plane.height
		lit, kept := rgb(bare[at]), rgb(dimmed[at])
		for c := range tints {
			want := math.Exp(-slabDensity * tints[c] * inside)
			if got := kept[c] / lit[c]; math.Abs(got/want-1) > 1e-3 {
				t.Errorf("at x %.1f channel %d keeps %.5f; the tinted "+
					"medium between says %.5f", x, c, got, want)
			}
		}
	}
}
