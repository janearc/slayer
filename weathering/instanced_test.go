package weathering

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// standings is how many places one design stands in, in the test that
// it is laid once: enough that a copy a place would show. picked.
const standings = 1000

// plateGroup is a roof plate as a group of one member, a design that
// can stand in many places.
func plateGroup(t *testing.T) *shape.MeshGroup {
	t.Helper()
	plate := roofPlate(plateHeight)
	made, err := shape.NewMesh(plate.Vertices, plate.Faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: made,
		Space: maths.Subspace{Scale: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	return group
}

// turnedPlace is an upright place: moved, turned about the vertical and
// twice as large as the design's own space.
var turnedPlace = maths.Subspace{Origin: maths.V(40, 1, -25),
	Orientation: maths.About(maths.YAxis, 0.5), Scale: 2}

// TestAGroupStoodUprightIsFoundAsIfLaid says that a look finds what
// stands over and under a point in a design stood at an upright place
// as it would in the same group laid triangle by triangle in the world.
func TestAGroupStoodUprightIsFoundAsIfLaid(t *testing.T) {
	group := plateGroup(t)
	stood, laid := NewShelter(shelterBlock), NewShelter(shelterBlock)
	stood.holdPlaced(group, turnedPlace, "roof")
	laid.hold(group, turnedPlace.Out, "roof")
	middle := turnedPlace.Out(maths.V(smallSide/2, 0, smallSide/2))
	for _, from := range []maths.Vec{middle, middle.Add(maths.V(0, 20, 0)),
		turnedPlace.Out(maths.V(smallSide+1, 0, smallSide/2))} {
		rise, name, found := stood.Above(from, 4*plateHeight)
		wantRise, wantName, wantFound := laid.Above(from, 4*plateHeight)
		if found != wantFound || name != wantName ||
			math.Abs(rise-wantRise) > 1e-9 {
			t.Errorf("above %v: %v %q %v stood, %v %q %v laid", from, rise,
				name, found, wantRise, wantName, wantFound)
		}
		fall, found := stood.Below(from, 40)
		wantFall, wantFound := laid.Below(from, 40)
		if found != wantFound || math.Abs(fall-wantFall) > 1e-9 {
			t.Errorf("below %v: %v %v stood, %v %v laid", from, fall,
				found, wantFall, wantFound)
		}
	}
	if len(stood.standing) != 1 || len(stood.all) != 0 {
		t.Errorf("%d standings and %d laid, want one and none",
			len(stood.standing), len(stood.all))
	}
}

// TestADesignStoodManyTimesIsLaidOnce says that a group stood in a
// thousand places costs its triangles once and a standing a place.
func TestADesignStoodManyTimesIsLaidOnce(t *testing.T) {
	group := plateGroup(t)
	shelter := NewShelter(shelterBlock)
	for index := range standings {
		origin := maths.V(float64(index)*3*smallSide, 0, 0)
		shelter.holdPlaced(group, maths.Subspace{Origin: origin}, "roof")
	}
	if _, _, found := shelter.Above(underRoof, 2*plateHeight); !found {
		t.Error("no roof over the first plate")
	}
	if len(shelter.designs) != 1 || len(shelter.standing) != standings ||
		len(shelter.all) != 0 {
		t.Errorf("%d designs, %d standings, %d laid; want 1, %d, 0",
			len(shelter.designs), len(shelter.standing), len(shelter.all),
			standings)
	}
	for _, made := range shelter.designs {
		if len(made.all) != facesOfPlate {
			t.Errorf("a design of %d triangles, want %d", len(made.all),
				facesOfPlate)
		}
	}
	if shelter.next != standings*facesOfPlate {
		t.Errorf("order runs to %d, want %d", shelter.next,
			standings*facesOfPlate)
	}
}

// TestAGroupNotUprightIsLaid says that a group whose place does not keep
// the world's up, here upside down, is laid in the world as before.
func TestAGroupNotUprightIsLaid(t *testing.T) {
	shelter := NewShelter(shelterBlock)
	over := maths.Subspace{Orientation: maths.About(maths.XAxis, math.Pi)}
	shelter.holdPlaced(plateGroup(t), over, "roof")
	shelter.Below(maths.V(0, 0, 0), 1)
	if len(shelter.standing) != 0 || len(shelter.all) != facesOfPlate {
		t.Errorf("%d standings and %d laid, want none and %d",
			len(shelter.standing), len(shelter.all), facesOfPlate)
	}
}

// TestALaterSurfaceWinsATieLaidOrStood says that of two roofs at one
// height, the one held later is found, whichever of them is laid and
// whichever stood, as when both were laid.
func TestALaterSurfaceWinsATieLaidOrStood(t *testing.T) {
	for _, standingFirst := range []bool{true, false} {
		shelter := NewShelter(shelterBlock)
		if standingFirst {
			shelter.holdPlaced(plateGroup(t), maths.Subspace{}, "first")
			shelter.hold(roofPlate(plateHeight), inPlace, "second")
		} else {
			shelter.hold(roofPlate(plateHeight), inPlace, "first")
			shelter.holdPlaced(plateGroup(t), maths.Subspace{}, "second")
		}
		_, name, found := shelter.Above(underRoof, 2*plateHeight)
		if !found || name != "second" {
			t.Errorf("standing first %v: found %q, want the second",
				standingFirst, name)
		}
	}
}

// TestUprightIsTurnedOnlyAboutTheVertical is the anti-test of upright:
// each way a place can lose the world's up is refused, and a turn about
// the vertical alone is kept.
func TestUprightIsTurnedOnlyAboutTheVertical(t *testing.T) {
	for _, each := range []struct {
		place maths.Subspace
		want  bool
	}{
		{maths.Subspace{}, true},
		{turnedPlace, true},
		{maths.Subspace{Orientation: maths.About(maths.XAxis, 0.1)}, false},
		{maths.Subspace{Orientation: maths.About(maths.ZAxis, 0.1)}, false},
		{maths.Subspace{Scale: -1}, false},
		{maths.Subspace{Scale: math.Inf(1)}, false},
	} {
		if got := upright(each.place); got != each.want {
			t.Errorf("upright(%+v) = %v, want %v", each.place, got,
				each.want)
		}
	}
}

// plateMesh is a roof plate as a mesh on its own, not in a group.
func plateMesh(t *testing.T) *shape.Mesh {
	t.Helper()
	plate := roofPlate(plateHeight)
	made, err := shape.NewMesh(plate.Vertices, plate.Faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// TestAMeshStoodManyTimesIsLaidOnce says that a mesh that is not a
// group, stood in a thousand places, is a design as a group is: its
// triangles kept once, and a standing a place.
func TestAMeshStoodManyTimesIsLaidOnce(t *testing.T) {
	plate := plateMesh(t)
	shelter := NewShelter(shelterBlock)
	for index := range standings {
		origin := maths.V(float64(index)*3*smallSide, 0, 0)
		shelter.holdPlaced(plate, maths.Subspace{Origin: origin}, "roof")
	}
	if _, _, found := shelter.Above(underRoof, 2*plateHeight); !found {
		t.Error("no roof over the first plate")
	}
	if len(shelter.designs) != 1 || len(shelter.standing) != standings ||
		len(shelter.all) != 0 {
		t.Errorf("%d designs, %d standings, %d laid; want 1, %d, 0",
			len(shelter.designs), len(shelter.standing), len(shelter.all),
			standings)
	}
}

// TestAMeshStoodOnceIsLaid is the anti-test: a mesh held in one place
// is laid in the world as before, with no design of its own.
func TestAMeshStoodOnceIsLaid(t *testing.T) {
	shelter := NewShelter(shelterBlock)
	shelter.holdPlaced(plateMesh(t), maths.Subspace{}, "roof")
	if _, _, found := shelter.Above(underRoof, 2*plateHeight); !found {
		t.Error("no roof over the plate")
	}
	if len(shelter.designs) != 0 || len(shelter.all) != facesOfPlate {
		t.Errorf("%d designs and %d laid, want none and %d",
			len(shelter.designs), len(shelter.all), facesOfPlate)
	}
}
