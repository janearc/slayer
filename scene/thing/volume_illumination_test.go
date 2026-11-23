package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv tables the illumination and grid tests read, each beside
// numbers.yaml.
const (
	volumeIlluminationAcceptedFile = "" +
		"testdata/volume-illumination-accepted-samples.csv"
	volumeIlluminationRefusedFile = "" +
		"testdata/volume-illumination-refused-samples.csv"
	volumeIlluminationKeptFile = "" +
		"testdata/volume-illumination-kept-nodes.csv"
	volumeIlluminationNodesFile = "" +
		"testdata/volume-illumination-refused-nodes.csv"
	volumeIlluminationBoxesFile = "" +
		"testdata/volume-illumination-refused-boxes.csv"
)

// volumeIlluminationAxes is how many axes a grid has nodes along.
const volumeIlluminationAxes = 3 // defined: the axes of space, x, y and z

// volumeIlluminationAxisColumns names the column of a table that holds
// each axis's node count, in the order of the axes.
var volumeIlluminationAxisColumns = [volumeIlluminationAxes]string{
	"x", "y", "z",
}

// volumeIlluminationNodes is the node counts a row of a table gives.
func volumeIlluminationNodes(
	t *testing.T, row map[string]string,
) [volumeIlluminationAxes]int {
	t.Helper()
	var nodes [volumeIlluminationAxes]int
	for axis, column := range volumeIlluminationAxisColumns {
		nodes[axis] = cellCount(t, row, column)
	}
	return nodes
}

// volumeIlluminationCube is a count of nodes that is the same each way.
func volumeIlluminationCube(side int) [volumeIlluminationAxes]int {
	return [volumeIlluminationAxes]int{side, side, side}
}

// volumeIlluminationSmall is the nodes of the small grid the tests make.
func volumeIlluminationSmall(t *testing.T) [volumeIlluminationAxes]int {
	t.Helper()
	side := pickedCount(t, "volume-illumination-small-nodes")
	return volumeIlluminationCube(side)
}

// volumeIlluminationBox is the box a row of a table gives, from its
// low corner to its high one, as it is written, sound or not.
func volumeIlluminationBox(t *testing.T, row map[string]string) maths.Box {
	t.Helper()
	low := maths.V(cell(t, row, "minx"), cell(t, row, "miny"),
		cell(t, row, "minz"))
	high := maths.V(cell(t, row, "maxx"), cell(t, row, "maxy"),
		cell(t, row, "maxz"))
	return maths.NewBox(low, high)
}

// Sky samples are even, from 2 to 64. The anti-tests: anything else is
// refused, an odd count between them too, and the zero illumination,
// the lamp-only one, has none.
func TestVolumeIlluminationTakesEvenSampleCountsFromTwoToSixtyFour(t *testing.T) {
	for _, row := range readSheet(t, volumeIlluminationAcceptedFile) {
		samples := cellCount(t, row, "samples")
		light, err := NewVolumeIllumination(samples)
		if err != nil || light.Samples() != samples {
			t.Errorf("%d samples: %d, %v", samples, light.Samples(), err)
		}
	}
	for _, row := range readSheet(t, volumeIlluminationRefusedFile) {
		samples := cellCount(t, row, "samples")
		if light, err := NewVolumeIllumination(samples); err == nil ||
			light.Samples() != nought {
			t.Errorf("%d samples were taken: %d, %v", samples,
				light.Samples(), err)
		}
	}
	if got := (VolumeIllumination{}).Samples(); got != nought {
		t.Errorf("the zero illumination has %d samples", got)
	}
}

// The directions are as many as the samples, each a unit long, in pairs
// that point opposite ways, so they sum to nothing; the first of each
// pair is in the upper half, at heights evenly spaced, which is what
// makes their solid angles equal; and they are the same every time.
func TestTheDirectionsAreOppositePairsEvenlySpacedInHeight(t *testing.T) {
	samples := pickedCount(t, "volume-illumination-direction-samples")
	tolerance := picked(t, "volume-illumination-tolerance")
	middle := picked(t, "volume-illumination-half")
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
		if !nearly(way.Len(), unity, tolerance) {
			t.Errorf("direction %d is %v long", at, way.Len())
		}
		sum = sum.Add(way)
	}
	if !nearly(sum.Len(), nought, tolerance) {
		t.Errorf("the directions sum to %v", sum)
	}
	for band := range samples / pair {
		up, down := ways[pair*band], ways[pair*band+unity]
		if down != up.Neg() {
			t.Errorf("pair %d is %v and %v", band, up, down)
		}
		want := (float64(band) + middle) / float64(samples/pair)
		if !nearly(up.Y, want, tolerance) {
			t.Errorf("pair %d is %v high, want %v", band, up.Y, want)
		}
	}
	again := light.Directions()
	for at := range ways {
		if again[at] != ways[at] {
			t.Errorf("direction %d moved from %v to %v",
				at, ways[at], again[at])
		}
	}
	// the anti-test: the lamp-only illumination looks no way at all
	if got := (VolumeIllumination{}).Directions(); len(got) != nought {
		t.Errorf("the zero illumination has %d directions", len(got))
	}
}

// The isotropic phase is one over the sphere's four pi steradians, so
// over the whole sphere it comes to one. The anti-test: over a
// hemisphere's two pi it comes to a half, and not to one, as a phase
// taken over the wrong solid angle would.
func TestTheIsotropicPhaseComesToOneOverTheSphere(t *testing.T) {
	tolerance := picked(t, "volume-illumination-phase-tolerance")
	sphere := picked(t, "volume-illumination-sphere-pi-multiple")
	hemisphere := picked(t, "volume-illumination-hemisphere-pi-multiple")
	whole := IsotropicPhase() * sphere * math.Pi
	if !nearly(whole, unity, tolerance) {
		t.Errorf("the isotropic phase over the sphere is %v", whole)
	}
	middle := picked(t, "volume-illumination-half")
	got := IsotropicPhase() * hemisphere * math.Pi
	if !nearly(got, middle, tolerance) {
		t.Errorf("the isotropic phase over a hemisphere is %v", got)
	}
}

// room is a box a metre each way from the world's zero.
func room() maths.Box {
	low := maths.V(-unity, -unity, -unity)
	high := maths.V(unity, unity, unity)
	return maths.NewBox(low, high)
}

// A grid is a box with room in it and 2 to 128 nodes each way, 262144
// in all at most: 128 by 128 by 16 is that many and is taken, and it
// gives back the box and the nodes it was made with. Its anti-test is
// the test below: every grid one step past these is refused.
func TestAnIncidentGridKeepsItsBoxAndItsNodes(t *testing.T) {
	for _, row := range readSheet(t, volumeIlluminationKeptFile) {
		nodes := volumeIlluminationNodes(t, row)
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
	small := volumeIlluminationSmall(t)
	for _, row := range readSheet(t, volumeIlluminationBoxesFile) {
		box := volumeIlluminationBox(t, row)
		if grid, err := NewIncidentGrid(box, small); err == nil || grid != nil {
			t.Errorf("a %s box was taken: %v", row["name"], err)
		}
	}
	for _, row := range readSheet(t, volumeIlluminationNodesFile) {
		nodes := volumeIlluminationNodes(t, row)
		grid, err := NewIncidentGrid(room(), nodes)
		if err == nil || grid != nil {
			t.Errorf("%v nodes were taken: %v", nodes, err)
		}
	}
}

// volumeIlluminationAttempt is a way of giving an illumination a grid
// that the illumination refuses.
type volumeIlluminationAttempt func(
	light VolumeIllumination, grid *IncidentGrid,
) (VolumeIllumination, error)

// volumeIlluminationLampOnly gives the lamp-only illumination a grid.
func volumeIlluminationLampOnly(
	_ VolumeIllumination, grid *IncidentGrid,
) (VolumeIllumination, error) {
	return VolumeIllumination{}.WithGrid(grid)
}

// volumeIlluminationNoGrid gives an illumination no grid at all.
func volumeIlluminationNoGrid(
	light VolumeIllumination, _ *IncidentGrid,
) (VolumeIllumination, error) {
	return light.WithGrid(nil)
}

// volumeIlluminationUnchecked gives an illumination a grid that was
// never made by NewIncidentGrid, so never checked.
func volumeIlluminationUnchecked(
	light VolumeIllumination, _ *IncidentGrid,
) (VolumeIllumination, error) {
	return light.WithGrid(&IncidentGrid{})
}

// An illumination with a grid is the same illumination holding it. The
// anti-tests: the one it was made from holds none; and a grid is
// refused by the lamp-only illumination, and so are no grid and a grid
// never checked.
func TestAnIlluminationHoldsTheGridItIsGiven(t *testing.T) {
	held := pickedCount(t, "volume-illumination-held-samples")
	light, err := NewVolumeIllumination(held)
	if err != nil {
		t.Fatal(err)
	}
	small := volumeIlluminationSmall(t)
	grid, err := NewIncidentGrid(room(), small)
	if err != nil {
		t.Fatal(err)
	}
	with, err := light.WithGrid(grid)
	if err != nil || with.Grid() != grid || with.Samples() != held {
		t.Errorf("with a grid: %v, %d samples, %v", with.Grid(),
			with.Samples(), err)
	}
	if light.Grid() != nil {
		t.Error("the illumination a grid was added to holds it too")
	}
	refused := map[string]volumeIlluminationAttempt{
		"the lamp-only illumination": volumeIlluminationLampOnly,
		"no grid":                    volumeIlluminationNoGrid,
		"a grid never checked":       volumeIlluminationUnchecked,
	}
	for name, try := range refused {
		got, err := try(light, grid)
		if err == nil || got.Grid() != nil || got.Samples() != nought {
			t.Errorf("%s was taken: %+v, %v", name, got, err)
		}
	}
}
