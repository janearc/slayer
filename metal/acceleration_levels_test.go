//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
)

// levelMeshes is how many triangle ranges the levelled plan holds, two
// to each of its two groups.
const levelMeshes = 4

// levelAcross is where each of the levelled plan's meshes stands across
// its group, before the group and the root move it.
var levelAcross = [levelMeshes]float32{-6, -3, 3, 6}

// levelMisses are places across where no mesh of the levelled plan
// stands at any shift: between and beyond them.
var levelMisses = []float32{0, 4.5, -4.5, 9}

// levelledPlan is four copies of one triangle as four ranges, two groups
// placing two each, and a root placing both groups shifted across: three
// levels with several builds in two of them, which share one command.
func levelledPlan(t *testing.T, shift float32) (*Acceleration, int) {
	t.Helper()
	var vertices []byte
	for range levelMeshes {
		vertices = append(vertices, triangleData()...)
	}
	plan, err := NewAcceleration(vertices, 16)
	requireDevice(t, err)
	var groups []Placement
	for half := range 2 {
		var places []Placement
		for side := range 2 {
			mesh := half*2 + side
			node, err := plan.Triangles(mesh, 1)
			requireDevice(t, err)
			place, err := Placed(node, translated(levelAcross[mesh], 0, 0), 3)
			requireDevice(t, err)
			places = append(places, place)
		}
		group, err := plan.Instances(places...)
		requireDevice(t, err)
		place, err := Placed(group, translated(shift, 0, -5), 3)
		requireDevice(t, err)
		groups = append(groups, place)
	}
	root, err := plan.Instances(groups...)
	requireDevice(t, err)
	return plan, root
}

// downRays are rays straight down from each place across, every mask,
// ten metres long.
func downRays(across []float32) []byte {
	var rays []byte
	for _, x := range across {
		for _, value := range []float32{x, 0, 0, 3, 0, 0, -1, 10} {
			rays = binary.LittleEndian.AppendUint32(rays,
				math.Float32bits(value))
		}
	}
	return rays
}

// levelProbe is the places across the levelled plan's rays start at for
// a shift, and what each must meet: every mesh, then every miss.
func levelProbe(shift float32) ([]float32, []float32) {
	var across, want []float32
	for _, x := range levelAcross {
		across, want = append(across, x+shift), append(want, 1)
	}
	for _, x := range levelMisses {
		across, want = append(across, x+shift), append(want, 0)
	}
	return across, want
}

// TestLevelsBuildInOneCommandAndMeetEveryMesh builds a hierarchy of
// three levels with several builds a level, first by Accelerate and then
// by Rebuild as it shifts: each query meets every mesh where it now is.
func TestLevelsBuildInOneCommandAndMeetEveryMesh(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for step, shift := range []float32{0, 1.5, -2, 0} {
		plan, root := levelledPlan(t, shift)
		build := kernel.Rebuild
		if step == 0 {
			build = kernel.Accelerate
		}
		if err := build(plan, root); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		across, want := levelProbe(shift)
		got, err := kernel.Grid(len(want), 1, 1, 1,
			[][]byte{downRays(across)})
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("shift %v met %v (%v), want %v", shift, got, err, want)
		}
	}
	if kernel.GPUTime() <= 0 {
		t.Fatal("the levelled builds reported no GPU time")
	}
}

// TestALevelOfOnlyPlacementsBuildsWithNoTriangles is a root over an empty
// group: two levels of placements with nothing at level nought, which
// meets nothing.
func TestALevelOfOnlyPlacementsBuildsWithNoTriangles(t *testing.T) {
	plan, err := NewAcceleration(nil, 16)
	requireDevice(t, err)
	group, err := plan.Instances()
	requireDevice(t, err)
	place, err := Placed(group, translated(0, 0, -5), 3)
	requireDevice(t, err)
	root, err := plan.Instances(place)
	requireDevice(t, err)
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	if err := kernel.Accelerate(plan, root); err != nil {
		t.Fatal(err)
	}
	got, err := kernel.Grid(1, 1, 1, 1, [][]byte{downRays([]float32{0})})
	if err != nil || len(got) != 1 || got[0] != 0 {
		t.Fatalf("an empty group met something: %v, %v", got, err)
	}
}
