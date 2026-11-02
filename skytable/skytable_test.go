package skytable_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/skytable"
)

// lopsided is a sky that is different every way: red to the east, green
// overhead, blue to the north.
type lopsided struct{}

// Light is the sky's light along a way.
func (lopsided) Light(toward maths.Vec) maths.Light {
	toward = toward.Unit()
	return maths.RGB(0.5+0.5*toward.X, 0.5+0.5*toward.Y,
		0.5-0.5*toward.Z)
}

// the table reads a smooth sky back the way it lies: east red, west not,
// overhead green, north blue, south not, and every way between within
// a hundredth, so a table mirrored, turned or transposed is caught.
func TestTheTableReadsTheSkyTheWayItLies(t *testing.T) {
	table := skytable.Of(lopsided{})
	ways := []maths.Vec{maths.V(1, 0, 0), maths.V(-1, 0, 0),
		maths.V(0, 1, 0), maths.V(0, 0, -1), maths.V(0, 0, 1),
		maths.V(0.3, 0.5, -0.8), maths.V(-0.6, 0.2, 0.4),
		maths.V(0.1, -0.7, 0.2)}
	for _, way := range ways {
		got, want := table.Light(way), lopsided{}.Light(way)
		for _, pair := range [][2]float64{{got.R, want.R},
			{got.G, want.G}, {got.B, want.B}} {
			if !(math.Abs(pair[0]-pair[1]) <= 0.01) {
				t.Errorf("toward %v the table gives %v, the sky %v", way,
					got, want)
				break
			}
		}
	}
}

// with no sky the table is dark every way.
func TestNoSkyIsDark(t *testing.T) {
	table := skytable.Of(nil)
	if got := table.Light(maths.V(0.2, 0.9, 0.1)); got != (maths.Light{}) {
		t.Errorf("no sky gives %v", got)
	}
}
