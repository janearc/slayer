package trace

import (
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// The picture the refusals are tried on: four pixels by three, one ray
// a pixel, and the pixel replayed.
const (
	parityWidth, parityHeight = 4, 3
	parityX, parityY          = 2, 1
)

// skyLitSea is an eye in the air over the tagged endless sea, whose
// water scatters the sky's light so much a metre, under a sky so bright.
func skyLitSea(t *testing.T, scatters, bright float64) (*thing.World,
	maths.Subspace) {
	t.Helper()
	world, _, _, _ := diffuseSkyOceanFixture(t, waterIndex, 8)
	base, err := thing.NewClearMediumWithScattering("parity-water",
		maths.Light{}, maths.Grey(scatters), 0, 1, 8, 12)
	if err != nil {
		t.Fatal(err)
	}
	water, err := base.WithDiffuseSkyInscatter(8)
	if err != nil {
		t.Fatal(err)
	}
	world.OceanSurface.Boundary.Material.ClearMedium = water
	world.Sky = sky.Gradient{Horizon: maths.Grey(bright),
		Zenith: maths.Grey(bright), Ground: maths.Grey(bright)}
	return world, lookingAt(maths.V(0, 2, 0), maths.V(0, 0, 0.5))
}

// threeDraws is the error of each of Sampled, SampledEach over the whole
// frame, and Replay of one pixel, on a world from an eye.
func threeDraws(world *thing.World, eye maths.Subspace) (sampled, each,
	replay error) {
	lens := film.Pinhole{}
	_, sampled = Sampled(world, eye, lens, 0, parityWidth, parityHeight, 1)
	whole := []image.Rectangle{image.Rect(0, 0, parityWidth, parityHeight)}
	_, each = SampledEach(world, eye, lens, 0, parityWidth, parityHeight,
		1, whole)
	_, _, replay = Replay(world, eye, lens, 0, parityWidth, parityHeight,
		1, parityX, parityY)
	return sampled, each, replay
}

// What Sampled refuses to draw, SampledEach and Replay refuse too, with
// the same words: they are Sampled's picture by sections and by one
// pixel, and a picture that leaves light out and says nothing is not
// that picture.
//
// The world is one that is found out only as it is drawn: water that
// scatters 1e300 a metre under a sky of 1e300 makes a light no pixel
// can hold, and Sampled says the in-scattering produced an
// unrepresentable source.
func TestSampledEachAndReplayRefuseWhatSampledRefuses(t *testing.T) {
	sampled, each, replay := threeDraws(skyLitSea(t, 1e300, 1e300))
	if sampled == nil || !strings.Contains(sampled.Error(),
		"unrepresentable") {
		t.Fatalf("the test's own world is not refused by Sampled as it "+
			"is drawn: %v", sampled)
	}
	for name, got := range map[string]error{"SampledEach": each,
		"Replay": replay} {
		if got == nil || got.Error() != sampled.Error() {
			t.Errorf("%s gave %v, and Sampled refuses with: %v", name, got,
				sampled)
		}
	}
}

// Anti: a world Sampled draws is drawn by all three. The same sea with
// water that scatters a tenth a metre under a sky of one is refused by
// none.
func TestSampledEachAndReplayDrawWhatSampledDraws(t *testing.T) {
	sampled, each, replay := threeDraws(skyLitSea(t, 0.1, 1))
	for name, got := range map[string]error{"Sampled": sampled,
		"SampledEach": each, "Replay": replay} {
		if got != nil {
			t.Errorf("%s refused a sea under a sky: %v", name, got)
		}
	}
}
