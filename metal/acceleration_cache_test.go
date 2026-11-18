//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
	"time"
)

// refitsBeforeRebuild is bridge.m's refits_before_rebuild: how many
// refits in a row a deforming mesh takes before it is built again.
const refitsBeforeRebuild = 30

// longestBuild is far longer than any of these few triangles' builds
// takes on the device: a time beyond it is a clock read wrongly, as one
// with no start time on the device came back as the machine's uptime.
const longestBuild = 10 * time.Second

// identity is the order that takes each mesh into the node of its own
// number.
var identity = [levelMeshes]int{0, 1, 2, 3}

// leanedTriangle is the test triangle moved across by lean in its own
// vertices, so each lean is other bytes for the same shape.
func leanedTriangle(lean float32) []byte {
	var data []byte
	for _, corner := range [][2]float32{{-1, -1}, {1, -1}, {0, 1}} {
		for _, number := range []float32{corner[0] + lean, corner[1], 0, 0} {
			data = binary.LittleEndian.AppendUint32(data,
				math.Float32bits(number))
		}
	}
	return data
}

// cachedPlan is the levelled plan with mesh k leaned across by lean[k]
// in its own vertices, and node k holding mesh order[k]: two groups of
// two nodes each where levelAcross says, and a root shifting both.
func cachedPlan(t *testing.T, lean [levelMeshes]float32,
	order [levelMeshes]int, shift float32) (*Acceleration, int) {
	t.Helper()
	var vertices []byte
	for _, mesh := range order {
		vertices = append(vertices, leanedTriangle(lean[mesh])...)
	}
	plan, err := NewAcceleration(vertices, 16)
	requireDevice(t, err)
	var groups []Placement
	for half := range 2 {
		var places []Placement
		for side := range 2 {
			node, err := plan.Triangles(half*2+side, 1)
			requireDevice(t, err)
			place, err := Placed(node,
				translated(levelAcross[half*2+side], 0, 0), 3)
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

// cachedMeets checks that a ray down through each node's triangle, where
// its lean and the shift now put it, meets it, and that one beside
// every node meets nothing.
func cachedMeets(t *testing.T, kernel *Kernel, lean [levelMeshes]float32,
	order [levelMeshes]int, shift float32) {
	t.Helper()
	var across, want []float32
	for node, mesh := range order {
		middle := levelAcross[node] + lean[mesh] + shift
		across = append(across, middle, middle+1.4)
		want = append(want, 1, 0)
	}
	got, err := kernel.Grid(len(want), 1, 1, 1, [][]byte{downRays(across)})
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("lean %v order %v shift %v met %v (%v), want %v", lean,
			order, shift, got, err, want)
	}
}

// cachedStep is one build of the cached plan and the work it must do.
type cachedStep struct {
	name               string
	lean               [levelMeshes]float32
	order              [levelMeshes]int
	shift              float32
	kept, refit, built int
}

// runCached builds each step on one kernel, the first by Accelerate and
// the rest by Rebuild, and checks the work and the queries of each.
func runCached(t *testing.T, steps []cachedStep) {
	t.Helper()
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for index, step := range steps {
		plan, root := cachedPlan(t, step.lean, step.order, step.shift)
		build := kernel.Rebuild
		if index == 0 {
			build = kernel.Accelerate
		}
		before := kernel.GPUTime()
		if err := build(plan, root); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if spent := kernel.GPUTime() - before; spent < 0 ||
			spent > longestBuild {
			t.Fatalf("%s: the build took %v on the device", step.name,
				spent)
		}
		kept, refit, built := kernel.accelerationWork()
		if kept != step.kept || refit != step.refit || built != step.built {
			t.Fatalf("%s: kept %d refit %d built %d, want %d %d %d",
				step.name, kept, refit, built, step.kept, step.refit,
				step.built)
		}
		cachedMeets(t, kernel, step.lean, step.order, step.shift)
	}
}

// distinct leans every mesh differently, so no two share bytes.
var distinct = [levelMeshes]float32{0, .1, .2, .3}

// TestUnchangedBytesKeepEveryStructure builds a plan, then the same plan
// again: nothing is built twice. Moving only the root builds only the
// root. Two ranges of new bytes, the same in both, build one structure
// and share it, so only one is built.
func TestUnchangedBytesKeepEveryStructure(t *testing.T) {
	runCached(t, []cachedStep{
		{"first", distinct, identity, 0, 0, 0, 7},
		{"the same again", distinct, identity, 0, 7, 0, 0},
		{"the root moved", distinct, identity, 1.5, 6, 0, 1},
		{"two of the same new bytes", [levelMeshes]float32{.5, .5, .2, .3},
			identity, 1.5, 4, 0, 3},
	})
}

// TestMeshesAreKeptByContentNotPlace swaps two meshes between nodes:
// both are kept by their bytes, and only the group that places them,
// which now places other structures, and the root are built again.
func TestMeshesAreKeptByContentNotPlace(t *testing.T) {
	runCached(t, []cachedStep{
		{"first", distinct, identity, 0, 0, 0, 7},
		{"swapped", distinct, [levelMeshes]int{1, 0, 2, 3}, 0, 5, 0, 2},
		{"swapped back", distinct, identity, 0, 5, 0, 2},
	})
}

// TestADeformingMeshIsBuiltToRefitThenRefit leans one mesh frame after
// frame: the first move builds it so it may be refit, and each later
// move refits it; its group and the root are built every time.
func TestADeformingMeshIsBuiltToRefitThenRefit(t *testing.T) {
	steps := []cachedStep{{"first", distinct, identity, 0, 0, 0, 7}}
	lean := distinct
	for move := range 4 {
		lean[0] = -.05 * float32(move+1)
		step := cachedStep{"moved", lean, identity, 0, 4, 1, 2}
		if move == 0 {
			step.refit, step.built = 0, 3
		}
		steps = append(steps, step)
	}
	runCached(t, steps)
}

// TestRefitsStopAtTheLimit leans one mesh until it has been refit as
// many times as the limit allows: the next move builds it again, and
// the one after refits it once more.
func TestRefitsStopAtTheLimit(t *testing.T) {
	steps := []cachedStep{{"first", distinct, identity, 0, 0, 0, 7}}
	lean := distinct
	for move := range refitsBeforeRebuild + 3 {
		lean[0] = -.001 * float32(move+1)
		step := cachedStep{"refit", lean, identity, 0, 4, 1, 2}
		if move == 0 || move == refitsBeforeRebuild+1 {
			step.name, step.refit, step.built = "built", 0, 3
		}
		steps = append(steps, step)
	}
	runCached(t, steps)
}

// TestAMeshOfAnotherLengthIsBuiltNotRefit gives a deformed, refittable
// mesh's node a second triangle: a range of another length cannot be
// refit, so it is built.
func TestAMeshOfAnotherLengthIsBuiltNotRefit(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for _, lean := range []float32{0, .1} {
		plan, root := grownPlan(t, lean, 1)
		if err := kernel.Rebuild(plan, root); err != nil {
			t.Fatal(err)
		}
	}
	plan, root := grownPlan(t, .2, 2)
	if err := kernel.Rebuild(plan, root); err != nil {
		t.Fatal(err)
	}
	if _, refit, built := kernel.accelerationWork(); refit != 0 ||
		built != 2 {
		t.Fatalf("a longer range: refit %d built %d, want 0 and 2", refit,
			built)
	}
	got, err := kernel.Grid(2, 1, 1, 1,
		[][]byte{downRays([]float32{.2, 3.2})})
	if err != nil || !slices.Equal(got, []float32{1, 1}) {
		t.Fatalf("the longer range met %v (%v)", got, err)
	}
}

// grownPlan is one range of count leaned triangles, each three across
// from the last, placed once by a root.
func grownPlan(t *testing.T, lean float32, count int) (*Acceleration,
	int) {
	t.Helper()
	var vertices []byte
	for triangle := range count {
		vertices = append(vertices,
			leanedTriangle(lean+3*float32(triangle))...)
	}
	plan, err := NewAcceleration(vertices, 16)
	requireDevice(t, err)
	mesh, err := plan.Triangles(0, count)
	requireDevice(t, err)
	place, err := Placed(mesh, translated(0, 0, -5), 3)
	requireDevice(t, err)
	root, err := plan.Instances(place)
	requireDevice(t, err)
	return plan, root
}

// rowPlan is one triangle placed count times by a root, each three
// across from the last, from shift.
func rowPlan(t *testing.T, count int, shift float32) (*Acceleration, int) {
	t.Helper()
	plan, err := NewAcceleration(leanedTriangle(0), 16)
	requireDevice(t, err)
	mesh, err := plan.Triangles(0, 1)
	requireDevice(t, err)
	var places []Placement
	for index := range count {
		place, err := Placed(mesh,
			translated(3*float32(index)+shift, 0, -5), 3)
		requireDevice(t, err)
		places = append(places, place)
	}
	root, err := plan.Instances(places...)
	requireDevice(t, err)
	return plan, root
}

// TestARootRebuiltOverItsOldBuffersStaysRight moves and grows and
// shrinks a root frame after frame, so its builds write into the
// structures and instance buffers earlier builds gave up, where they
// fit, and into new ones where not: every frame meets each placement
// where it now is and nothing past the last.
func TestARootRebuiltOverItsOldBuffersStaysRight(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for step, count := range []int{1, 3, 3, 2, 5, 5, 1, 4} {
		shift := float32(step%3) * .2
		plan, root := rowPlan(t, count, shift)
		if err := kernel.Rebuild(plan, root); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		if _, _, built := kernel.accelerationWork(); step > 0 && built != 1 {
			t.Fatalf("step %d built %d nodes, want the root alone", step,
				built)
		}
		var across, want []float32
		for index := range count + 1 {
			across = append(across, 3*float32(index)+shift)
			want = append(want, float32(min(1, count-index)))
		}
		got, err := kernel.Grid(len(want), 1, 1, 1,
			[][]byte{downRays(across)})
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("step %d met %v (%v), want %v", step, got, err, want)
		}
	}
}
