package island

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/terrain"
)

// span is the transition length the motion tests use.
const span = 10 * time.Second

// newMotion is a motion over the terrain with room for every sample.
func newMotion(t *testing.T, ground *Terrain) *Motion {
	t.Helper()
	motion, err := NewMotion(ground, span, 64, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return motion
}

// TestNewMotionRefusals checks each bad control and the payload that cannot
// hold one three by three lattice of transition vertices.
func TestNewMotionRefusals(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	cases := []struct {
		name    string
		ground  *Terrain
		span    time.Duration
		patches int
		bytes   int64
	}{
		{"no terrain", nil, span, 1, 1 << 20},
		{"no source", &Terrain{}, span, 1, 1 << 20},
		{"no span", ground, 0, 1, 1 << 20},
		{"no patches", ground, span, 0, 1 << 20},
		{"no bytes", ground, span, 1, 0},
		{"bytes under a lattice", ground, span, 1, 8 * transitionValueBytes},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			if motion, err := NewMotion(each.ground, each.span, each.patches, each.bytes); err == nil || motion != nil {
				t.Fatalf("got %v, %v", motion, err)
			}
		})
	}
	if _, err := NewMotion(ground, span, 1, 9*transitionValueBytes); err != nil {
		t.Fatalf("exactly one lattice of bytes was refused: %v", err)
	}
}

// TestTransitionValueBytesIsTheSizeOfOneVertex checks the payload unit:
// four vectors of three float64 and one pair of ints.
func TestTransitionValueBytesIsTheSizeOfOneVertex(t *testing.T) {
	if transitionValueBytes != 4*24+16 {
		t.Fatalf("%d bytes", transitionValueBytes)
	}
}

// TestMotionPrepareRefusals checks the controls that Prepare checks before
// and after it asks for a view.
func TestMotionPrepareRefusals(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	view := lookDown(t, 1e6)
	origin := originAt(t, maths.V(0, 1e6, 0))
	var nobody *Motion
	if _, err := nobody.Prepare(context.Background(), view, origin, 0); err == nil {
		t.Error("a nil motion prepared")
	}
	var none context.Context
	if _, err := newMotion(t, ground).Prepare(none, view, origin, 0); err == nil {
		t.Error("a nil context prepared")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newMotion(t, ground).Prepare(cancelled, view, origin, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: %v", err)
	}
	motion := newMotion(t, ground)
	if _, err := motion.Prepare(context.Background(), view, origin, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := motion.Prepare(context.Background(), view, origin, 4*time.Second); err == nil ||
		!strings.Contains(err.Error(), "moved backwards") {
		t.Errorf("backwards clock: %v", err)
	}
	motion.Reset()
	if _, err := motion.Prepare(context.Background(), view, origin, time.Second); err != nil {
		t.Errorf("a reset motion refused an earlier time: %v", err)
	}
}

// TestMotionPrepareReportsViewFailures checks a refused view and a failing
// source on the first call.
func TestMotionPrepareReportsViewFailures(t *testing.T) {
	sentinel := errors.New("source gone")
	ground := testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) { return 0, false, sentinel }), testDetail())
	origin := originAt(t, maths.V(0, 1e6, 0))
	if _, err := newMotion(t, ground).Prepare(context.Background(), lookDown(t, 1e6), origin, 0); !errors.Is(err, sentinel) {
		t.Errorf("source failure: %v", err)
	}
	good := newMotion(t, testTerrain(t, flat(0), testDetail()))
	if _, err := good.Prepare(context.Background(), footprintZero(), origin, 0); err == nil {
		t.Error("an unchecked view prepared")
	}
}

// TestMotionKeepsAnIdenticalSelection checks that a second view that selects
// the same patches reuses the retained geometry and reads no source.
func TestMotionKeepsAnIdenticalSelection(t *testing.T) {
	reads := 0
	ground := testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) {
		reads++
		return 0, true, nil
	}), testDetail())
	motion := newMotion(t, ground)
	origin := originAt(t, maths.V(0, 1e6, 0))
	first, err := motion.Prepare(context.Background(), lookDown(t, 1e6), origin, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Patches) != 4 || first.Transition != nil || reads != 25 {
		t.Fatalf("first: %d patches, %d reads", len(first.Patches), reads)
	}
	second, err := motion.Prepare(context.Background(), lookDown(t, 9e5), origin, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if reads != 25 || second.Transition != nil || len(second.Patches) != 4 ||
		second.Samples != 25 || second.Surfaces[0] != first.Surfaces[0] {
		t.Fatalf("second: %d reads, transition %v", reads, second.Transition)
	}
}

// TestMotionBlendsBetweenTwoViews checks a transition from the four-patch
// view to the sixteen-patch view, with points worked by hand from the lattice:
// the old triangle at the start, the average half way, the new one at the end.
func TestMotionBlendsBetweenTwoViews(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	motion := newMotion(t, ground)
	ctx := context.Background()
	farOrigin, nearOrigin := maths.V(0, 1e6, 0), maths.V(0, 3e5, 0)
	if _, err := motion.Prepare(ctx, lookDown(t, 1e6), originAt(t, farOrigin), 0); err != nil {
		t.Fatal(err)
	}
	// Key (2,2) is the middle of the first sixteen-patch cell. The old
	// four-patch mesh cuts that cell along the diagonal from key (0,0) to
	// key (4,4), so its triangle puts the point halfway between them.
	old := midway(world(t, 0, 0, 0), world(t, 4, 4, 0))
	exact := world(t, 2, 2, 0)
	start := 2 * time.Second
	checks := []struct {
		name  string
		at    time.Duration
		want  maths.Vec
		share float64
	}{
		{"at the start", start, old.Sub(nearOrigin), 0},
		{"half way", start + span/2, midway(old, exact).Sub(nearOrigin), 0.5},
		{"just before the end", start + span - 1, exact.Sub(nearOrigin), 1},
	}
	for _, each := range checks {
		got, err := motion.Prepare(ctx, lookDown(t, 3e5), originAt(t, nearOrigin), each.at)
		if err != nil {
			t.Fatalf("%s: %v", each.name, err)
		}
		record := got.Transition
		if record == nil || math.Abs(record.Share-each.share) > 1e-9 ||
			record.FromPatches != 4 || record.TargetPatches != 16 ||
			record.PayloadBytes != 81*transitionValueBytes || got.Samples != 81 {
			t.Fatalf("%s: record %+v, %d samples", each.name, record, got.Samples)
		}
		if len(got.Patches) != 16 || len(got.Surfaces) != 16 || !got.Limited {
			t.Fatalf("%s: %d patches, limited %v", each.name, len(got.Patches), got.Limited)
		}
		if point := got.Surfaces[0].Points()[4]; !near(point, each.want, 1e-6) {
			t.Errorf("%s: point %v, wanted %v", each.name, point, each.want)
		}
	}
	done, err := motion.Prepare(ctx, lookDown(t, 3e5), originAt(t, nearOrigin), start+span)
	if err != nil {
		t.Fatal(err)
	}
	if done.Transition != nil || len(done.Patches) != 16 || done.Samples != 81 {
		t.Fatalf("after the end: %+v", done.Transition)
	}
	if point := done.Surfaces[0].Points()[4]; !near(point, exact.Sub(nearOrigin), 1e-6) {
		t.Errorf("after the end: point %v, wanted %v", point, exact.Sub(nearOrigin))
	}
}

// TestMotionReportsNormalCountsOnce checks that the target's normal counts
// appear on the first frame of a transition only, and that a frame which
// reuses its selection reports none.
func TestMotionReportsNormalCountsOnce(t *testing.T) {
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	motion := newMotion(t, testTerrain(t, flat(0), detail))
	ctx := context.Background()
	if _, err := motion.Prepare(ctx, lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)), 0); err != nil {
		t.Fatal(err)
	}
	low := originAt(t, maths.V(0, 3e5, 0))
	first, err := motion.Prepare(ctx, lookDown(t, 3e5), low, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if first.NormalStencils != 81 || first.NormalHeightQueries != 324 || !first.Surfaces[0].Smooth() {
		t.Fatalf("first frame: %d stencils, %d queries", first.NormalStencils, first.NormalHeightQueries)
	}
	second, err := motion.Prepare(ctx, lookDown(t, 3e5), low, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if second.NormalStencils != 0 || second.NormalHeightQueries != 0 {
		t.Fatalf("second frame: %d stencils, %d queries", second.NormalStencils, second.NormalHeightQueries)
	}
	if second.Transition == nil || second.Transition.Share != maths.Smooth(0, 1, 0.1) {
		t.Fatalf("second frame transition %+v", second.Transition)
	}
	retained, err := motion.Prepare(ctx, lookDown(t, 3e5), low, 11*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	again, err := motion.Prepare(ctx, lookDown(t, 3e5), low, 12*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if again.NormalStencils != 0 || again.NormalHeightQueries != 0 || retained.Transition != nil || again.Transition != nil {
		t.Fatalf("retained frames reported %d stencils", again.NormalStencils)
	}
}

// TestMotionStartsAgainFromAnEmptyView checks that a view with no ground
// is replaced, not blended, when ground comes into view, and that a later
// target blends from it once it has ground.
func TestMotionStartsAgainFromAnEmptyView(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	motion := newMotion(t, ground)
	ctx := context.Background()
	away, err := footprintFor(maths.Subspace{Origin: maths.V(0, 1e6, 0), Orientation: maths.Orientation{
		X: maths.V(1, 0, 0), Y: maths.V(0, 0, 1), Z: maths.V(0, -1, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	origin := originAt(t, maths.V(0, 1e6, 0))
	empty, err := motion.Prepare(ctx, away, origin, 0)
	if err != nil || len(empty.Patches) != 0 {
		t.Fatalf("empty view: %+v, %v", empty, err)
	}
	seen, err := motion.Prepare(ctx, lookDown(t, 1e6), origin, time.Second)
	if err != nil || seen.Transition != nil || len(seen.Patches) != 4 {
		t.Fatalf("first ground: transition %v, %d patches, %v", seen.Transition, len(seen.Patches), err)
	}
	empty, err = motion.Prepare(ctx, away, origin, 2*time.Second)
	if err != nil || empty.Transition != nil || len(empty.Patches) != 0 {
		t.Fatalf("ground leaving view: %+v, %v", empty, err)
	}
}

// TestMotionStartsANewTransitionWhenOneEnds checks that a view that arrives
// after the old interval has passed blends from the finished target.
func TestMotionStartsANewTransitionWhenOneEnds(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	motion := newMotion(t, ground)
	ctx := context.Background()
	if _, err := motion.Prepare(ctx, lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := motion.Prepare(ctx, lookDown(t, 3e5), originAt(t, maths.V(0, 3e5, 0)), time.Second); err != nil {
		t.Fatal(err)
	}
	at := time.Second + span + time.Second
	got, err := motion.Prepare(ctx, lookDown(t, 2e7), originAt(t, maths.V(0, 2e7, 0)), at)
	if err != nil {
		t.Fatal(err)
	}
	if got.Transition == nil || got.Transition.FromPatches != 16 || got.Transition.TargetPatches != 1 ||
		got.Transition.Share != 0 {
		t.Fatalf("transition %+v", got.Transition)
	}
}

// TestMotionRefusesATransitionOverItsLimits checks the common patch
// allowance and the payload allowance, and that a failed transition leaves
// the earlier geometry in place for another try.
func TestMotionRefusesATransitionOverItsLimits(t *testing.T) {
	ctx := context.Background()
	for name, motion := range map[string]func(*testing.T) *Motion{
		"patch allowance": func(t *testing.T) *Motion {
			made, err := NewMotion(testTerrain(t, flat(0), testDetail()), span, 4, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			return made
		},
		"payload allowance": func(t *testing.T) *Motion {
			made, err := NewMotion(testTerrain(t, flat(0), testDetail()), span, 64, 80*transitionValueBytes)
			if err != nil {
				t.Fatal(err)
			}
			return made
		},
	} {
		owner := motion(t)
		if _, err := owner.Prepare(ctx, lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)), 0); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		_, err := owner.Prepare(ctx, lookDown(t, 3e5), originAt(t, maths.V(0, 3e5, 0)), time.Second)
		if err == nil || !strings.Contains(err.Error(), "allowance") {
			t.Errorf("%s: error %v", name, err)
		}
		if owner.active != nil {
			t.Errorf("%s: a refused transition stayed active", name)
		}
	}
}

// TestMotionStopsWhenTheContextEndsInATransition cancels the context during
// the last source read of the target, so the transition's own check refuses.
func TestMotionStopsWhenTheContextEndsInATransition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reads, stopAt := 0, math.MaxInt
	ground := testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) {
		reads++
		if reads == stopAt {
			cancel()
		}
		return 0, true, nil
	}), testDetail())
	motion := newMotion(t, ground)
	if _, err := motion.Prepare(ctx, lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)), 0); err != nil {
		t.Fatal(err)
	}
	stopAt = reads + 81
	_, err := motion.Prepare(ctx, lookDown(t, 3e5), originAt(t, maths.V(0, 3e5, 0)), time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
}

// TestTransitionPrepareStopsOnACancelledContext builds a transition and then
// asks it for a frame with a context that has ended.
func TestTransitionPrepareStopsOnACancelledContext(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	motion := newMotion(t, ground)
	origin := originAt(t, maths.V(0, 1e6, 0))
	from, err := ground.Prepare(context.Background(), lookDown(t, 1e6), origin)
	if err != nil {
		t.Fatal(err)
	}
	target, err := ground.Prepare(context.Background(), lookDown(t, 3e5), origin)
	if err != nil {
		t.Fatal(err)
	}
	made, err := motion.transition(context.Background(), from, target, 0)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := made.prepare(cancelled, origin, 0.5); !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
}

// TestSamplerRefusesIrregularEndpoints checks the two ways an endpoint can
// fail the layout check.
func TestSamplerRefusesIrregularEndpoints(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	patch := terrain.Patch{MaxColumn: 16, MaxRow: 16}
	if _, err := ground.sampler(Prepared{Patches: []terrain.Patch{patch}}); err == nil ||
		!strings.Contains(err.Error(), "inconsistent") {
		t.Errorf("one patch and no mesh: %v", err)
	}
	triangle, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 0, 1)},
		[][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ground.sampler(Prepared{Patches: []terrain.Patch{patch}, Surfaces: []*mesh.Mesh{triangle}}); err == nil ||
		!strings.Contains(err.Error(), "regular lattice") {
		t.Errorf("one triangle: %v", err)
	}
}

// TestSamplerFindsPointsOnTheStitchedTriangles checks values worked by hand
// on the four-patch endpoint: a point in the lower triangle of a cell, one
// in the upper triangle, a knot, a boundary point shared by two patches and
// the region's far corner.
func TestSamplerFindsPointsOnTheStitchedTriangles(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	origin := maths.V(0, 1e6, 0)
	prepared, err := ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, origin))
	if err != nil {
		t.Fatal(err)
	}
	sample, err := ground.sampler(prepared)
	if err != nil {
		t.Fatal(err)
	}
	corner := world(t, 0, 0, 0)
	lower := corner.Add(world(t, 4, 4, 0).Sub(corner).Scale(0.25)).Add(world(t, 4, 0, 0).Sub(corner).Scale(0.5))
	upper := corner.Add(world(t, 0, 4, 0).Sub(corner).Scale(0.5)).Add(world(t, 4, 4, 0).Sub(corner).Scale(0.25))
	for _, each := range []struct {
		name string
		key  lattice
		want maths.Vec
	}{
		{"lower triangle", lattice{3, 1}, lower},
		{"upper triangle", lattice{1, 3}, upper},
		{"knot", lattice{4, 4}, world(t, 4, 4, 0)},
		{"shared boundary", lattice{8, 8}, world(t, 8, 8, 0)},
		{"far corner", lattice{16, 16}, world(t, 16, 16, 0)},
		{"top edge", lattice{12, 0}, world(t, 12, 0, 0)},
	} {
		got, normal, err := sample.at(each.key)
		if err != nil || !near(got, each.want, 1e-6) || normal != (maths.Vec{}) {
			t.Errorf("%s: got %v, %v, normal %v; wanted %v", each.name, got, err, normal, each.want)
		}
	}
}

// TestSamplerLeafAssignsSharedBoundariesUpward checks which leaf owns a key
// on a seam, and that a key outside the region owns none.
func TestSamplerLeafAssignsSharedBoundariesUpward(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	prepared, err := ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if err != nil {
		t.Fatal(err)
	}
	sample, err := ground.sampler(prepared)
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range []struct {
		key  lattice
		want terrain.Patch
	}{
		{lattice{8, 8}, terrain.Patch{MinColumn: 8, MaxColumn: 16, MinRow: 8, MaxRow: 16, Level: 1}},
		{lattice{16, 0}, terrain.Patch{MinColumn: 8, MaxColumn: 16, MinRow: 0, MaxRow: 8, Level: 1}},
		{lattice{0, 16}, terrain.Patch{MinColumn: 0, MaxColumn: 8, MinRow: 8, MaxRow: 16, Level: 1}},
		{lattice{3, 3}, terrain.Patch{MinColumn: 0, MaxColumn: 8, MinRow: 0, MaxRow: 8, Level: 1}},
	} {
		got, _, err := sample.leaf(each.key)
		if err != nil || got != each.want {
			t.Errorf("leaf%v = %+v, %v; wanted %+v", each.key, got, err, each.want)
		}
	}
	for _, key := range []lattice{{17, 3}, {3, 17}, {-1, 0}} {
		if _, _, err := sample.leaf(key); err == nil || !strings.Contains(err.Error(), "does not cover") {
			t.Errorf("leaf%v: error %v", key, err)
		}
		if _, _, err := sample.at(key); err == nil {
			t.Errorf("at%v: no error", key)
		}
	}
	if _, _, err := (surfaceSampler{root: terrain.Patch{}}).leaf(lattice{0, 0}); err == nil ||
		!strings.Contains(err.Error(), "invalid native patch") {
		t.Errorf("invalid root: %v", err)
	}
	if _, _, err := (surfaceSampler{root: terrain.Patch{MaxColumn: 1, MaxRow: 1}}).leaf(lattice{0, 0}); err == nil {
		t.Error("a root with no children and no leaf found a leaf")
	}
}

// TestSamplerInterpolatesNormalsOnSmoothMeshes checks that a smooth
// endpoint returns the corner normal at a knot, which over sea level points
// away from the sphere's centre.
func TestSamplerInterpolatesNormalsOnSmoothMeshes(t *testing.T) {
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	ground := testTerrain(t, flat(0), detail)
	prepared, err := ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if err != nil {
		t.Fatal(err)
	}
	sample, err := ground.sampler(prepared)
	if err != nil {
		t.Fatal(err)
	}
	_, normal, err := sample.at(lattice{4, 4})
	want := world(t, 4, 4, 0).Sub(maths.V(0, -1e6, 0)).Unit()
	if err != nil || !near(normal, want, 1e-6) {
		t.Fatalf("normal %v, %v; wanted %v", normal, err, want)
	}
}

// TestCommonPatchesTakesTheFinerLeaf checks the common refinement of a
// coarse and a fine cover, and its three refusals.
func TestCommonPatchesTakesTheFinerLeaf(t *testing.T) {
	root := terrain.Patch{MaxColumn: 16, MaxRow: 16}
	children, err := root.Children()
	if err != nil {
		t.Fatal(err)
	}
	coarse := map[terrain.Patch]int{root: 0}
	fine := map[terrain.Patch]int{}
	for index, child := range children {
		fine[child] = index
	}
	got, err := commonPatches(root, coarse, fine, 4)
	if err != nil || len(got) != 4 || got[0] != children[0] || got[3] != children[3] {
		t.Fatalf("got %v, %v", got, err)
	}
	same, err := commonPatches(root, coarse, coarse, 1)
	if err != nil || len(same) != 1 || same[0] != root {
		t.Fatalf("same cover: %v, %v", same, err)
	}
	if _, err := commonPatches(root, coarse, fine, 3); err == nil || !strings.Contains(err.Error(), "allowance") {
		t.Errorf("limit of three: %v", err)
	}
	if _, err := commonPatches(root, coarse, map[terrain.Patch]int{}, 4); err == nil ||
		!strings.Contains(err.Error(), "incomplete") {
		t.Errorf("a cover with a hole: %v", err)
	}
	if _, err := commonPatches(terrain.Patch{}, coarse, fine, 4); err == nil {
		t.Error("an invalid root was accepted")
	}
}

// TestTransitionRefusesAnIrregularEndpoint gives a transition a start and
// then a target whose meshes do not match their patches.
func TestTransitionRefusesAnIrregularEndpoint(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	motion := newMotion(t, ground)
	good, err := ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if err != nil {
		t.Fatal(err)
	}
	bad := Prepared{Patches: good.Patches}
	if _, err := motion.transition(context.Background(), bad, good, 0); err == nil ||
		!strings.Contains(err.Error(), "inconsistent") {
		t.Errorf("bad start: %v", err)
	}
	if _, err := motion.transition(context.Background(), good, bad, 0); err == nil ||
		!strings.Contains(err.Error(), "inconsistent") {
		t.Errorf("bad target: %v", err)
	}
}

// TestPrepareRefusesARootBoxPastTheRange checks that a ground whose height
// overflows the sphere's radius is refused when the view is selected.
func TestPrepareRefusesARootBoxPastTheRange(t *testing.T) {
	vast, err := frame.NewSphere(measure.Metres(1e295), frame.Location{})
	if err != nil {
		t.Fatal(err)
	}
	detail := testDetail()
	detail.MinimumHeight, detail.MaximumHeight = math.MaxFloat64, math.MaxFloat64
	ground, err := New(vast, flat(math.MaxFloat64), testRegion, detail)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, maths.Vec{}))
	if err == nil || !strings.Contains(err.Error(), "invalid spherical location") {
		t.Fatalf("error %v", err)
	}
}

// zeroTransition is a transition over one patch whose every endpoint value
// is the zero vector, so any mesh built from it has faces with no area.
func zeroTransition() *terrainTransition {
	faces := [][3]int{{0, 4, 1}, {0, 3, 4}, {1, 5, 2}, {1, 4, 5}, {3, 7, 4}, {3, 6, 7}, {4, 8, 5}, {4, 7, 8}}
	return &terrainTransition{patches: []terrain.Patch{{MaxColumn: 4, MaxRow: 4, Level: 1}},
		values: map[lattice]transitionVertex{}, faces: faces, side: 3}
}

// TestTransitionPrepareRefusesWhatCannotBeMeshed checks that a transition
// whose points have no area is refused, directly and through a motion that
// holds it, and that a render origin which overflows is refused.
func TestTransitionPrepareRefusesWhatCannotBeMeshed(t *testing.T) {
	origin := originAt(t, maths.Vec{})
	if _, err := zeroTransition().prepare(context.Background(), origin, 0.5); err == nil ||
		!strings.Contains(err.Error(), "no area") {
		t.Errorf("direct: %v", err)
	}
	motion := newMotion(t, testTerrain(t, flat(0), testDetail()))
	motion.active = zeroTransition()
	if _, err := motion.Prepare(context.Background(), lookDown(t, 1e6), origin, time.Second); err == nil ||
		!strings.Contains(err.Error(), "no area") {
		t.Errorf("through a motion: %v", err)
	}
	far := zeroTransition()
	far.values[lattice{0, 0}] = transitionVertex{from: maths.V(-1e300, 0, 0), to: maths.V(-1e300, 0, 0)}
	edge := originAt(t, maths.V(math.MaxFloat64, 0, 0))
	if _, err := far.prepare(context.Background(), edge, 1); err == nil ||
		!strings.Contains(err.Error(), "overflow") {
		t.Errorf("overflowing origin: %v", err)
	}
}

// TestSurfaceRefusesARenderOriginThatOverflows checks that a surface on a
// very large sphere, seen from an origin at the far end of the number
// range, is refused instead of returning infinite points.
func TestSurfaceRefusesARenderOriginThatOverflows(t *testing.T) {
	vast, err := frame.NewSphere(measure.Metres(1e295), frame.Location{})
	if err != nil {
		t.Fatal(err)
	}
	ground, err := New(vast, flat(0), testRegion, testDetail())
	if err != nil {
		t.Fatal(err)
	}
	patch := terrain.Patch{MinColumn: 4, MaxColumn: 6, MinRow: 0, MaxRow: 2, Level: 2}
	book := bookFor(ground, []terrain.Patch{patch})
	_, err = book.surface(patch, originAt(t, maths.V(math.MaxFloat64, 0, 0)))
	if err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("error %v", err)
	}
}
