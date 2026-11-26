package cacti

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestStemEmitsClosedStraightAndBentColumns(t *testing.T) {
	for _, bend := range []float64{0, 1} {
		s := Stem{Run: 2, Radius: 1, Bend: bend, Ribs: 4, Curve: 2, Cap: 2}
		faces := Skin[[3]maths.Vec](s, 4)
		first := faces[0]
		if first[0] != (maths.Vec{}) || math.Abs(first[1].Z+1) > 1e-12 ||
			(bend == 0 && first[2] != maths.V(1, 0, 0)) ||
			(bend != 0 && first[2] != maths.V(0, -1, 0)) {
			t.Fatalf("base winding: %+v", first)
		}
		// The bent shoulder has degenerate panels; the mesh omits their faces.
		want := 24
		if bend != 0 {
			want = 36
		}
		if len(faces) != want {
			t.Fatalf("bend %v: got %d faces, want %d", bend, len(faces), want)
		}
	}
}

func TestStemBoundsIncludeSpines(t *testing.T) {
	s := Stem{Run: 2, Radius: 1, Bend: 1}
	box := s.Bounds(.2)
	if box.Min != maths.V(-1.2, -1.2, -1.2) || box.Max != maths.V(2.2, 4.2, 1.2) {
		t.Fatalf("bounds: %+v", box)
	}
}

func TestStemSpinesFollowTheRibsWithinTheirBudget(t *testing.T) {
	prototype, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	stem := Stem{Run: 2, Radius: 1, Bend: 1, Ribs: 4}
	pose := maths.NewSubspace(maths.V(4, 5, 6), maths.Identity, 1)
	members, err := stem.Spines(nil, prototype, pose, .9, .5, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 8 {
		t.Fatalf("got %d spines, want 8", len(members))
	}
	want := pose.Out(maths.V(1.5, 1+2.0/3.0, 0))
	if members[0].Mesh != prototype || members[0].Space.Origin.Sub(want).Len() > 1e-12 {
		t.Fatalf("first spine: %+v, want origin %v", members[0], want)
	}
	if _, err := stem.Spines(nil, prototype, pose, .9, .5, 7); err == nil {
		t.Fatal("accepted spines past the budget")
	}
	if _, err := stem.Spines(nil, prototype, pose, 0, .5, 8); err == nil {
		t.Fatal("accepted zero spacing")
	}
	if _, err := stem.Spines(nil, prototype, pose, .01, .5, 8); err == nil {
		t.Fatal("accepted too many rows")
	}
	withoutRibs := stem
	withoutRibs.Ribs = 0
	if _, err := withoutRibs.Spines(nil, prototype, pose, .9, .5, 8); err == nil {
		t.Fatal("accepted a column with no ribs")
	}
	if _, err := stem.Spines(members, prototype, pose, .9, .5, 7); err == nil {
		t.Fatal("accepted an already overfull set of spines")
	}
}
