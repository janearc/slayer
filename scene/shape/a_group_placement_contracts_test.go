package shape

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestGroupPlacementChecksEachIndependentAxis(t *testing.T) {
	valid := []maths.Subspace{{}, {Scale: math.SmallestNonzeroFloat64}, {Scale: 2},
		{Orientation: maths.Rotation(maths.V(1, 2, 3), .7)}}
	for _, space := range valid {
		if !ValidPlacement(space) {
			t.Fatalf("valid placement refused: %+v", space)
		}
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for axis := range 3 {
			var origin maths.Vec
			switch axis {
			case 0:
				origin.X = bad
			case 1:
				origin.Y = bad
			case 2:
				origin.Z = bad
			}
			if ValidPlacement(maths.Subspace{Origin: origin}) {
				t.Fatalf("accepted nonfinite origin axis %d", axis)
			}
		}
		if ValidPlacement(maths.Subspace{Scale: bad}) {
			t.Fatal("accepted nonfinite scale")
		}
	}
	if ValidPlacement(maths.Subspace{Scale: -math.SmallestNonzeroFloat64}) {
		t.Fatal("accepted negative scale")
	}
	for axis := range 3 {
		for _, size := range []float64{.5, 2} {
			turn := maths.Identity
			switch axis {
			case 0:
				turn.X = turn.X.Scale(size)
			case 1:
				turn.Y = turn.Y.Scale(size)
			case 2:
				turn.Z = turn.Z.Scale(size)
			}
			if ValidPlacement(maths.Subspace{Orientation: turn}) {
				t.Fatalf("accepted axis %d length %g", axis, size)
			}
		}
	}
	// Unit-length skew axes isolate orthogonality from length validation.
	for _, skew := range []float64{1e-9, math.Nextafter(1e-9, 0), 2e-9, .1} {
		for _, pair := range [][2]int{{0, 1}, {0, 2}, {1, 2}} {
			axes := [3]maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis}
			axes[pair[1]] = axes[pair[1]].Scale(math.Sqrt(1 - skew*skew)).Add(axes[pair[0]].Scale(skew))
			got := ValidPlacement(maths.Subspace{Orientation: maths.Orientation{X: axes[0], Y: axes[1], Z: axes[2]}})
			if got != (skew < 1e-9) {
				t.Fatalf("axis pair %v skew %.17g: accepted %t", pair, skew, got)
			}
		}
	}
	turn := maths.Identity
	turn.Z = turn.Z.Neg()
	if ValidPlacement(maths.Subspace{Orientation: turn}) {
		t.Fatal("accepted left-handed placement")
	}
}

func TestGroupBoxDistanceIsTheEuclideanDistanceToTheBox(t *testing.T) {
	box := maths.Box{Min: maths.V(-2, -3, -5), Max: maths.V(4, 7, 11)}
	for _, p := range []maths.Vec{maths.V(0, 0, 0), box.Min, box.Max} {
		if d := groupBoxDistance(box, p); d != 0 {
			t.Fatalf("point %v in box has distance %g", p, d)
		}
	}
	for mask := range 8 {
		p := maths.V(7, 11, 23)
		if mask&1 != 0 {
			p.X = -5
		}
		if mask&2 != 0 {
			p.Y = -7
		}
		if mask&4 != 0 {
			p.Z = -17
		}
		// Orthogonal offsets of 3, 4 and 12 give distance 13.
		if d := groupBoxDistance(box, p); d != 13 {
			t.Fatalf("corner %d distance %g, want 13", mask, d)
		}
	}
	for _, row := range []struct {
		p maths.Vec
		d float64
	}{
		{maths.V(-3, 0, 0), 1}, {maths.V(5, 0, 0), 1},
		{maths.V(0, -5, 0), 2}, {maths.V(0, 9, 0), 2},
		{maths.V(0, 0, -8), 3}, {maths.V(0, 0, 14), 3},
	} {
		if d := groupBoxDistance(box, row.p); d != row.d {
			t.Fatalf("point %v distance %g, want %g", row.p, d, row.d)
		}
	}
}

func TestGroupNearestTraversalFinishesForADeepHierarchy(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	solid, err := mesh.New(points, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var members []MeshMember
	for i := range 32 {
		members = append(members, MeshMember{Mesh: solid, Space: maths.Subspace{Origin: maths.V(float64(i)*3, 0, 0)}})
	}
	group, err := NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	hitDone := make(chan bool, 1)
	go func() {
		distance, hit := group.Hit(maths.Ray{At: maths.V(.2, .2, 1), Dir: maths.ZAxis.Neg()}, 0, 2)
		hitDone <- hit && math.Abs(distance-1) <= 1e-12
	}()
	select {
	case ok := <-hitDone:
		if !ok {
			t.Fatal("deep hierarchy lost its nearest ray hit")
		}
	case <-time.After(time.Second):
		t.Fatal("hit traversal did not finish within one second")
	}

	done := make(chan float64, 1)
	go func() { done <- group.Distance(maths.V(.2, .2, 1)) }()
	select {
	case distance := <-done:
		if !(math.Abs(distance-1) <= 1e-12) {
			t.Fatalf("nearest distance %g, want 1", distance)
		}
	case <-time.After(time.Second):
		t.Fatal("nearest traversal did not finish within one second")
	}
}
