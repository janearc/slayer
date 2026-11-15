//go:build draw

package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// oneBits is whether two pixels are one, bit for bit, in all five channels.
func oneBits(one, other frame.Pixel) bool {
	bits := func(p frame.Pixel) [5]uint32 {
		return [5]uint32{math.Float32bits(p.R), math.Float32bits(p.G),
			math.Float32bits(p.B), math.Float32bits(p.Cover),
			math.Float32bits(p.Depth)}
	}
	return bits(one) == bits(other)
}

// Every pixel replayed alone is the pixel Sampled made, bit for bit: of a sky
// and a smoking plane, mirrors, a puff, a gas, a field and a lit floor, at one,
// two and three rays per pixel each way.
func TestAReplayedPixelIsTheSampledPixelToTheBit(t *testing.T) {
	worlds := map[string]*thing.World{
		"smoking": smoking(), "mirrors": mirrors(true),
		"puffed": puffed(1), "earthlike": earthlike(),
		"hole": holeWorld(nil), "under red gas": underRedGas(true),
		"lit floor": litFloor(), "burning": burning(),
	}
	eye := lookingAt(maths.V(0, 2, 6), centre)
	lens := film.Pinhole{}
	const width, height = 12, 9
	for name, world := range worlds {
		for _, rays := range []int{1, 2, 3} {
			whole := sampled(world, eye, lens, 0, width, height, rays)
			for y := range height {
				for x := range width {
					alone, steps, err := Replay(world, eye, lens, 0,
						width, height, rays, x, y)
					if err != nil {
						t.Fatal(err)
					}
					if !oneBits(alone, whole.At(x, y)) {
						t.Fatalf("%s, %d rays, pixel %d, %d: replayed "+
							"%v, sampled %v", name, rays, x, y, alone,
							whole.At(x, y))
					}
					told(t, steps, rays)
				}
			}
		}
	}
}

// told holds a replay's steps to their shape: each ray from the lens,
// its light, and the pixel last.
func told(t *testing.T, steps []Step, rays int) {
	t.Helper()
	lenses, lights := 0, 0
	for _, step := range steps {
		switch step.What {
		case "a ray from the lens":
			lenses++
		case "the ray's light":
			lights++
		}
	}
	last := steps[len(steps)-1]
	if lenses != rays*rays || lights != rays*rays ||
		last.What != "the pixel" || last.Ray != -1 ||
		!strings.Contains(last.Said, "rays=") {
		t.Fatalf("steps: %d from the lens, %d lights, last %+v",
			lenses, lights, last)
	}
}
