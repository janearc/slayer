package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Sky samples are even, from 2 to 64. The anti-tests: anything else is
// refused, an odd count between them too, and the zero illumination,
// the lamp-only one, has none.
func TestVolumeIlluminationTakesEvenSampleCountsFromTwoToSixtyFour(t *testing.T) {
	for _, samples := range []int{2, 16, 64} {
		light, err := NewVolumeIllumination(samples)
		if err != nil || light.Samples() != samples {
			t.Errorf("%d samples: %d, %v", samples, light.Samples(), err)
		}
	}
	for _, samples := range []int{-2, 0, 1, 3, 63, 65, 66} {
		if light, err := NewVolumeIllumination(samples); err == nil ||
			light.Samples() != 0 {
			t.Errorf("%d samples were taken: %d, %v", samples,
				light.Samples(), err)
		}
	}
	if got := (VolumeIllumination{}).Samples(); got != 0 {
		t.Errorf("the zero illumination has %d samples", got)
	}
}

// The directions are as many as the samples, each a unit long, in pairs
// that point opposite ways, so they sum to nothing; the first of each
// pair is in the upper half, at heights evenly spaced, which is what
// makes their solid angles equal; and they are the same every time.
func TestTheDirectionsAreOppositePairsEvenlySpacedInHeight(t *testing.T) {
	const samples = 16
	light, err := NewVolumeIllumination(samples)
	if err != nil {
		t.Fatal(err)
	}
	ways := light.Directions()
	if len(ways) != samples {
		t.Fatalf("%d directions for %d samples", len(ways), samples)
	}
	var sum maths.Vec
	for at, way := range ways {
		if !nearly(way.Len(), 1, 1e-12) {
			t.Errorf("direction %d is %v long", at, way.Len())
		}
		sum = sum.Add(way)
	}
	if !nearly(sum.Len(), 0, 1e-12) {
		t.Errorf("the directions sum to %v", sum)
	}
	for pair := range samples / 2 {
		up, down := ways[2*pair], ways[2*pair+1]
		if down != up.Neg() {
			t.Errorf("pair %d is %v and %v", pair, up, down)
		}
		if want := (float64(pair) + 0.5) / (samples / 2); !nearly(up.Y, want, 1e-12) {
			t.Errorf("pair %d is %v high, want %v", pair, up.Y, want)
		}
	}
	again := light.Directions()
	for at := range ways {
		if again[at] != ways[at] {
			t.Errorf("direction %d moved from %v to %v", at, ways[at], again[at])
		}
	}
	// the anti-test: the lamp-only illumination looks no way at all
	if got := (VolumeIllumination{}).Directions(); len(got) != 0 {
		t.Errorf("the zero illumination has %d directions", len(got))
	}
}

// The isotropic phase is one over the sphere's four pi steradians, so
// over the whole sphere it comes to one. The anti-test: over a
// hemisphere's two pi it comes to a half, and not to one, as a phase
// taken over the wrong solid angle would.
func TestTheIsotropicPhaseComesToOneOverTheSphere(t *testing.T) {
	if got := IsotropicPhase() * 4 * math.Pi; !nearly(got, 1, 1e-15) {
		t.Errorf("the isotropic phase over the sphere is %v", got)
	}
	if got := IsotropicPhase() * 2 * math.Pi; !nearly(got, 0.5, 1e-15) {
		t.Errorf("the isotropic phase over a hemisphere is %v", got)
	}
}

// room is a box a metre each way from the world's zero.
func room() maths.Box {
	return maths.Box{Min: maths.V(-1, -1, -1), Max: maths.V(1, 1, 1)}
}

// A grid is a box with room in it and 2 to 128 nodes each way, 262144
// in all at most: 128 by 128 by 16 is that many and is taken, and it
// gives back the box and the nodes it was made with. Its anti-test is
// the test below: every grid one step past these is refused.
func TestAnIncidentGridKeepsItsBoxAndItsNodes(t *testing.T) {
	for _, nodes := range [][3]int{{2, 2, 2}, {128, 128, 16}, {16, 128, 128}} {
		grid, err := NewIncidentGrid(room(), nodes)
		if err != nil {
			t.Fatalf("%v nodes: %v", nodes, err)
		}
		if box, kept := grid.Parameters(); box != room() || kept != nodes {
			t.Errorf("the grid keeps %v and %v", box, kept)
		}
	}
}

// A grid is refused with a node too few or too many on any axis, with
// more than 262144 nodes in all, and over a box that is flat, inside
// out, not a number, endless, or too wide for its width to be a number.
func TestAnIncidentGridThatCannotBeHeldIsRefused(t *testing.T) {
	huge := math.MaxFloat64
	boxes := map[string]maths.Box{
		"flat":         {Min: maths.V(-1, 1, -1), Max: maths.V(1, 1, 1)},
		"inside out":   {Min: maths.V(-1, -1, 1), Max: maths.V(1, 1, -1)},
		"not a number": {Min: maths.V(math.NaN(), -1, -1), Max: maths.V(1, 1, 1)},
		"no number up": {Min: maths.V(-1, -1, -1), Max: maths.V(1, math.NaN(), 1)},
		"endless low":  {Min: maths.V(-1, math.Inf(-1), -1), Max: maths.V(1, 1, 1)},
		"endless high": {Min: maths.V(-1, -1, -1), Max: maths.V(1, 1, math.Inf(1))},
		"too wide":     {Min: maths.V(-huge, -1, -1), Max: maths.V(huge, 1, 1)},
	}
	for name, box := range boxes {
		if grid, err := NewIncidentGrid(box, [3]int{4, 4, 4}); err == nil || grid != nil {
			t.Errorf("a %s box was taken: %v", name, err)
		}
	}
	for _, nodes := range [][3]int{{1, 4, 4}, {4, 0, 4}, {4, 4, -3}, {129, 4, 4},
		{4, 4, 129}, {128, 128, 17}, {65, 65, 65}} {
		if grid, err := NewIncidentGrid(room(), nodes); err == nil || grid != nil {
			t.Errorf("%v nodes were taken: %v", nodes, err)
		}
	}
}

// An illumination with a grid is the same illumination holding it. The
// anti-tests: the one it was made from holds none; and a grid is
// refused by the lamp-only illumination, and so are no grid and a grid
// never checked.
func TestAnIlluminationHoldsTheGridItIsGiven(t *testing.T) {
	light, err := NewVolumeIllumination(8)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := NewIncidentGrid(room(), [3]int{4, 4, 4})
	if err != nil {
		t.Fatal(err)
	}
	with, err := light.WithGrid(grid)
	if err != nil || with.Grid() != grid || with.Samples() != 8 {
		t.Errorf("with a grid: %v, %d samples, %v", with.Grid(),
			with.Samples(), err)
	}
	if light.Grid() != nil {
		t.Error("the illumination a grid was added to holds it too")
	}
	refused := map[string]func() (VolumeIllumination, error){
		"the lamp-only illumination": func() (VolumeIllumination, error) {
			return VolumeIllumination{}.WithGrid(grid)
		},
		"no grid": func() (VolumeIllumination, error) {
			return light.WithGrid(nil)
		},
		"a grid never checked": func() (VolumeIllumination, error) {
			return light.WithGrid(&IncidentGrid{})
		},
	}
	for name, try := range refused {
		if got, err := try(); err == nil || got.Grid() != nil || got.Samples() != 0 {
			t.Errorf("%s was taken: %+v, %v", name, got, err)
		}
	}
}
