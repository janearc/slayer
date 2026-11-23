package titan

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
)

// the relief the tests ask of a flat map
const (
	reliefSlope   = 0.05 // picked: a gentle plain
	reliefLongest = 50   // picked: tens of metres, under the chart's 100
	reliefPlaces  = 200  // picked: enough places for a mean square
)

// madeRelief is the root mean square of the made metres over a line of
// places on a flat chart, a centimetre fine, with the relief asked or not.
func madeRelief(t *testing.T, asked bool) float64 {
	t.Helper()
	world := ground(t, &chart{finest: 100, relief: 0, share: 0.6}, 3)
	if asked {
		if err := world.Relief(reliefSlope, reliefLongest); err != nil {
			t.Fatal(err)
		}
	}
	var squares float64
	for i := range reliefPlaces {
		lon := 20 + float64(i)*0.0137 // picked: a line
		lat := 10 + float64(i)*0.0071
		got, err := world.At(spot(t, lon, lat), inchworm)
		if err != nil {
			t.Fatal(err)
		}
		squares += got.Made() * got.Made()
	}
	return math.Sqrt(squares / reliefPlaces)
}

// A flat map with a relief asked has made relief, gentle: some, and no
// more than the steepness asked carried over the longest edge.
func TestAFlatMapWithReliefIsNotFlat(t *testing.T) {
	made := madeRelief(t, true)
	if !(made > 0) || made > reliefSlope*reliefLongest {
		t.Errorf("a flat map with relief was given %v m", made)
	}
}

// Anti: a flat map with no relief asked stays flat, as before.
func TestAFlatMapWithoutReliefStaysFlat(t *testing.T) {
	if made := madeRelief(t, false); made != 0 {
		t.Errorf("a flat map was given %v m of made relief", made)
	}
}

// Anti: a relief moves no corner the map has: at a corner of the mesh the
// ground is the map's height, measured, whatever relief is asked.
func TestReliefNeverMovesTheMap(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	world := ground(t, from, 1)
	if err := world.Relief(1, 1e4); err != nil { // picked: steep and wide
		t.Fatal(err)
	}
	corner, _, _ := mesh.Faces()[3].Corners()
	want, _, _ := from.Height(corner)
	got, err := world.At(&corner, 1000) // picked: coarser than the chart
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind() != measured || math.Abs(got.Height()-want) > 1e-6 {
		t.Errorf("at a corner the ground is %v %v, the map %v", got.Kind(),
			got.Height(), want)
	}
}

// A relief below nought, or with a longest that is no length, is refused.
func TestReliefRefusals(t *testing.T) {
	world := ground(t, &chart{finest: 100, relief: 0, share: 0.6}, 1)
	for _, asked := range [][2]float64{{-0.1, 10}, {0.1, 0},
		{math.NaN(), 10}, {0.1, math.Inf(1)}} {
		if err := world.Relief(asked[0], asked[1]); !errors.Is(err,
			errRefused) {
			t.Errorf("relief %v was not refused: %v", asked, err)
		}
	}
}
