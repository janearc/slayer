package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// strength is all a flame's sources' light together, by channel.
func strength(sources []Source) maths.Light {
	var sum maths.Light
	for _, source := range sources {
		sum = sum.Add(source.Strength)
	}
	return sum
}

// a flame's sources stand where it has soot, and their light together
// comes to the flame's whole light as the cells grow fine: its
// absorption times its glow, summed here over a fine grid of its box
// with its own volume to a point. sources 12 and 24 cells a side each
// come within five parts in a hundred of it.
func TestAFlamesSourcesAddUpToItsLight(t *testing.T) {
	fire := body(t, campfire).Tick(time.Second).(*Body)
	tint := fire.Tint()
	box := fire.Bounds()
	const fine = 60
	size := box.Max.Sub(box.Min).Scale(1.0 / fine)
	var whole maths.Light
	for i := range fine * fine * fine {
		x, y, z := i%fine, i/fine%fine, i/(fine*fine)
		at := box.Min.Add(maths.V((float64(x)+0.5)*size.X,
			(float64(y)+0.5)*size.Y, (float64(z)+0.5)*size.Z))
		glow := fire.glows.at(fire.Heat(at)).Times(tint)
		whole = whole.Add(glow.Scale(fire.Density(at) * size.X * size.Y *
			size.Z))
	}
	for _, cells := range []int{12, 24} {
		sources := fire.Sources(cells)
		for _, source := range sources {
			if !(fire.Density(source.At) > 0) {
				t.Fatalf("a source at %v, where there is no soot",
					source.At)
			}
		}
		got := strength(sources)
		if share := math.Abs(got.R/whole.R - 1); share > 0.05 {
			t.Errorf("%d cells a side come to %v of the flame's red",
				cells, got.R/whole.R)
		}
	}
	if fire.Sources(0) != nil {
		t.Error("no cells gave sources")
	}
}
