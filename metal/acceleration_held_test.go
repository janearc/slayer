//go:build draw && darwin

package metal

import (
	"slices"
	"syscall"
	"testing"
)

// passesHeld is bridge.m's passes_held: a context holds the meshes this
// pass and the one before placed, and lets the rest go.
const passesHeld = 2

// changingPasses is how many builds the bound is watched over, and
// changingFirst the triangles of the first: each build is a triangle
// longer than the last, so no mesh can be kept or refit. the shape is
// the reviewer's probe in the 116 review.
const (
	changingPasses = 60
	changingFirst  = 40000
)

// fixedLean is where pairedPlan's fixed triangle stands across, apart
// from every triangle of its growing range.
const fixedLean = -10

// bytesPerMegabyte turns maxrss, which darwin gives in bytes, into
// megabytes for the log.
const bytesPerMegabyte = 1 << 20

// rebuildCounted rebuilds a plan on the kernel and checks the work its
// build did.
func rebuildCounted(t *testing.T, kernel *Kernel, plan *Acceleration,
	root int, name string, kept, refit, built int) {
	t.Helper()
	if err := kernel.Rebuild(plan, root); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	gotKept, gotRefit, gotBuilt := kernel.accelerationWork()
	if gotKept != kept || gotRefit != refit || gotBuilt != built {
		t.Fatalf("%s: kept %d refit %d built %d, want %d %d %d", name,
			gotKept, gotRefit, gotBuilt, kept, refit, built)
	}
}

// TestAMeshNotPlacedForTwoPassesIsBuiltAgain places one triangle, then
// ranges of two and three: the first is placed by neither of the last
// two passes, so it is let go, and placed again it is built, not kept.
func TestAMeshNotPlacedForTwoPassesIsBuiltAgain(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for _, count := range []int{1, 2, 3} {
		plan, root := grownPlan(t, 0, count)
		rebuildCounted(t, kernel, plan, root, "grown", 0, 0, 2)
	}
	plan, root := grownPlan(t, 0, 1)
	rebuildCounted(t, kernel, plan, root, "the first again", 0, 0, 2)
}

// TestAMeshPlacedThePassBeforeIsKept is the other side of the bound:
// one triangle, then a range of two, then the one triangle again, which
// the pass before last placed, so it is still held and kept.
func TestAMeshPlacedThePassBeforeIsKept(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for _, count := range []int{1, 2} {
		plan, root := grownPlan(t, 0, count)
		rebuildCounted(t, kernel, plan, root, "grown", 0, 0, 2)
	}
	plan, root := grownPlan(t, 0, 1)
	rebuildCounted(t, kernel, plan, root, "the first again", 1, 0, 1)
}

// pairedPlan is one fixed triangle and a range of count triangles, each
// three across from the last, both placed by a root.
func pairedPlan(t *testing.T, count int) (*Acceleration, int) {
	t.Helper()
	vertices := leanedTriangle(fixedLean)
	for triangle := range count {
		vertices = append(vertices,
			leanedTriangle(3*float32(triangle))...)
	}
	plan, err := NewAcceleration(vertices, 16)
	requireDevice(t, err)
	var places []Placement
	for _, span := range [][2]int{{0, 1}, {1, count}} {
		mesh, err := plan.Triangles(span[0], span[1])
		requireDevice(t, err)
		place, err := Placed(mesh, translated(0, 0, -5), 3)
		requireDevice(t, err)
		places = append(places, place)
	}
	root, err := plan.Instances(places...)
	requireDevice(t, err)
	return plan, root
}

// TestAMeshPlacedEveryPassStaysKept keeps one triangle in every pass
// while the range beside it grows: the triangle is kept every time, long
// after the meshes beside it were let go, and both are still met.
func TestAMeshPlacedEveryPassStaysKept(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for count := 1; count <= 6; count++ {
		plan, root := pairedPlan(t, count)
		kept, built := 1, 2
		if count == 1 {
			kept, built = 0, 3
		}
		rebuildCounted(t, kernel, plan, root, "paired", kept, 0, built)
	}
	got, err := kernel.Grid(3, 1, 1, 1,
		[][]byte{downRays([]float32{fixedLean, 0, fixedLean / 2})})
	if err != nil || !slices.Equal(got, []float32{1, 1, 0}) {
		t.Fatalf("the paired plan met %v (%v), want 1 1 0", got, err)
	}
}

// peakResident is the process's peak resident set in bytes.
func peakResident(t *testing.T) int64 {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	return usage.Maxrss
}

// TestHeldMeshesStayBoundedAcrossChangingPasses rebuilds sixty times, a
// triangle longer each time, as a mesh that changes every frame does:
// the context never holds more than the last two passes' meshes, and
// the first pass's bytes, placed again, are built, not kept.
func TestHeldMeshesStayBoundedAcrossChangingPasses(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	before, most := peakResident(t), 0
	for pass := range changingPasses {
		plan, root := grownPlan(t, 0, changingFirst+pass)
		rebuildCounted(t, kernel, plan, root, "longer", 0, 0, 2)
		most = max(most, kernel.accelerationHeld())
	}
	after := peakResident(t)
	t.Logf("held at most %d mesh structures over %d passes; maxrss %d MB "+
		"before, %d MB after", most, changingPasses,
		before/bytesPerMegabyte, after/bytesPerMegabyte)
	if most > passesHeld {
		t.Fatalf("held %d mesh structures, want %d at most", most,
			passesHeld)
	}
	plan, root := grownPlan(t, 0, changingFirst)
	rebuildCounted(t, kernel, plan, root, "the first pass's bytes", 0, 0, 2)
}

// failedSteps are the cached plan's leans before a failed build: the
// first mesh moves until it has been refit twice, so the build after
// writes into a spare and into its group's kept instances.
var failedSteps = []float32{0, -.05, -.1, -.15}

// TestAFailedRebuildLeavesTheHeldHierarchy fails a build after every
// structure is described and, apart, after its command has run: the
// hierarchy held before still serves, and the same build again does
// the work it would have done and meets where it should.
func TestAFailedRebuildLeavesTheHeldHierarchy(t *testing.T) {
	for _, fault := range []accelerationFault{faultAllocation, faultCommand} {
		kernel, err := Open(acceleratedProbe, "probe")
		requireDevice(t, err)
		lean := distinct
		for _, step := range failedSteps {
			lean[0] = step
			plan, root := cachedPlan(t, lean, identity, 0)
			if err := kernel.Rebuild(plan, root); err != nil {
				t.Fatalf("fault %d, lean %v: %v", fault, step, err)
			}
		}
		held := lean
		lean[0] = -.2
		plan, root := cachedPlan(t, lean, identity, 0)
		kernel.failNextBuild(fault)
		if err := kernel.Rebuild(plan, root); err == nil {
			t.Fatalf("fault %d: the build did not fail", fault)
		}
		cachedMeets(t, kernel, held, identity, 0)
		rebuildCounted(t, kernel, plan, root, "after the fault", 4, 1, 2)
		cachedMeets(t, kernel, lean, identity, 0)
		kernel.Close()
	}
}
