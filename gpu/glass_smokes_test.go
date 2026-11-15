//go:build draw

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/smoke"
	slayer "github.com/janearc/slayer/trace"
)

// glassSmokes are the smokes that stand one clear thing inside another:
// a glass of water on a table, a ball of water with bubbles of air in
// it, and a glass of cola with its fizz. The glass is a mesh, turned as
// a potter turns one, and outranks what is poured into it.
var glassSmokes = []int{49, 51, 52}

// the gpu draws each of the glass smokes as slayer's tracer does, the
// whole smoke as its builder makes it, through the smoke's own lens.
// The plain balls of nested_test.go hold the kernel's bends; these hold
// them in the pictures they were made for
func TestTheGPUDrawsTheGlassSmokesAsSlayerDoes(t *testing.T) {
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	for _, number := range glassSmokes {
		made, known := smoke.Get(number)
		if !known {
			t.Fatalf("there is no smoke %d", number)
		}
		t.Run(made.Name, func(t *testing.T) {
			world, eye := made.Scene()
			lens := film.Pinhole{Wide: made.Wide}
			drawn, err := Sampled(world, eye, lens, 0, 96, 64, 2)
			if errors.Is(err, metal.ErrNoDevice) {
				t.Skip("no gpu here")
			}
			if err != nil {
				t.Fatal(err)
			}
			truth, err := slayer.Sampled(world, eye, lens, 0, 96, 64, 2)
			if err != nil {
				t.Fatal(err)
			}
			found := compare(truth, drawn, 96, 64)
			count, furthest := strays(truth, drawn, exact.largest)
			t.Logf("apart: %+v; %d pixels past %v, the furthest %v", found,
				count, exact.largest, furthest)
			if !(found.mean <= exact.mean) {
				t.Errorf("the gpu's light is %v off slayer's on average, "+
					"want %v or less", found.mean, exact.mean)
			}
			if count > fewStrays || !(furthest <= farthestStray) {
				t.Errorf("%d pixels are past %v and the furthest is %v "+
					"off; want %d or fewer and none past %v", count,
					exact.largest, furthest, fewStrays, farthestStray)
			}
		})
	}
}

// A glass smoke's picture is held to the exact tolerance on average, and
// pixel for pixel but for a stray or two. The cola's fizz is hundreds of
// balls a few thousandths of a unit across, and a ray that grazes the
// rim of one lands on it in the tracer's doubles and misses it in the
// kernel's floats: measured on 2026-11-12, one pixel of the cola's 6,144
// is 0.0027 off, and none of the other two smokes' is past the limit.
const (
	fewStrays     = 2    // picked: twice what was measured in the worst of the three
	farthestStray = 0.01 // picked: about four times the one stray measured; a bend at the wrong surface is tenths
)

// strays is how many pixels of two pictures are further apart, in any
// colour, than a limit, and how far apart the furthest of them is.
func strays(one, other *frame.Frame, limit float64) (count int, furthest float64) {
	for at := range one.Pix {
		left, right := one.Pix[at], other.Pix[at]
		gap := math.Max(math.Abs(float64(left.R-right.R)),
			math.Max(math.Abs(float64(left.G-right.G)),
				math.Abs(float64(left.B-right.B))))
		if gap > limit {
			count++
		}
		furthest = math.Max(furthest, gap)
	}
	return count, furthest
}
