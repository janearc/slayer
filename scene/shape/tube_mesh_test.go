package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// What the tests of this file count in, each true by definition.
const (
	tubeMeshCornerCount  = 3         // defined: the corners of a face
	tubeMeshFirstCorner  = 0         // defined: the first corner of a face
	tubeMeshSecondCorner = 1         // defined: the second corner of a face
	tubeMeshThirdCorner  = 2         // defined: the third corner of a face
	tubeMeshThird        = 1.0 / 3.0 // defined: one third
)

// The places of the tubes in meshedTubes: a ring, a hair as thin as a
// whisker with no hole, and a cable.
const (
	tubeMeshRing  = 0 // defined: the place of the ring
	tubeMeshHair  = 1 // defined: the place of the hair
	tubeMeshCable = 2 // defined: the place of the cable
)

// The tables the tests of this file read: the tubes that are meshed, the
// tubes that are no tube, and the chord shares that are no share.
const (
	tubeMeshTubeFile    = "testdata/tube-mesh-tubes.csv"
	tubeMeshRefusedFile = "testdata/tube-mesh-refused.csv"
	tubeMeshShareFile   = "testdata/tube-mesh-shares.csv"
)

// tubeMeshOf reads every row of a csv table of tubes, with a radius, a wall
// and a length, as a tube.
func tubeMeshOf(t *testing.T, file string) []Tube {
	t.Helper()
	var tubes []Tube
	for _, each := range readSheet(t, file) {
		tubes = append(tubes, Tube{Radius: cell(t, each, "radius"),
			Thick:  cell(t, each, "wall"),
			Height: cell(t, each, "length")})
	}
	return tubes
}

// meshedTubes are tubes of different bores, walls and lengths: a ring,
// a hair as thin as a whisker with no hole, and a cable.
func meshedTubes(t *testing.T) []Tube {
	t.Helper()
	return tubeMeshOf(t, tubeMeshTubeFile)
}

// tubeMeshSlack is the room left for rounding when a distance is held to
// the chord share.
func tubeMeshSlack(t *testing.T) float64 {
	t.Helper()
	return picked(t, "tube-mesh-slack")
}

// meshOf grows a tube's mesh at the share slayer draws with.
func meshOf(t *testing.T, tube Tube) *Mesh {
	t.Helper()
	made, err := tube.Mesh(TubeChordShare())
	if err != nil {
		t.Fatalf("%+v: %v", tube, err)
	}
	return made
}

// tubeMeshCornersOf are the three corners of a face of made.
func tubeMeshCornersOf(made *Mesh,
	face [tubeMeshCornerCount]int) [tubeMeshCornerCount]maths.Vec {
	return [tubeMeshCornerCount]maths.Vec{
		made.Vertices[face[tubeMeshFirstCorner]],
		made.Vertices[face[tubeMeshSecondCorner]],
		made.Vertices[face[tubeMeshThirdCorner]]}
}

// farthestFromWall is the largest distance from tube's wall of any
// corner, edge middle or face middle of the mesh. The middle of a side
// is where a flat side lies furthest from the circle it stands for.
func farthestFromWall(made *Mesh, tube Tube) float64 {
	farthest := float64(nought)
	for _, face := range made.Faces {
		corners := tubeMeshCornersOf(made, face)
		first := corners[tubeMeshFirstCorner]
		second := corners[tubeMeshSecondCorner]
		third := corners[tubeMeshThirdCorner]
		points := []maths.Vec{first, second, third,
			first.Add(second).Scale(half),
			second.Add(third).Scale(half),
			third.Add(first).Scale(half),
			first.Add(second).Add(third).Scale(tubeMeshThird)}
		for _, point := range points {
			gap := math.Abs(tube.Distance(point))
			farthest = math.Max(farthest, gap)
		}
	}
	return farthest
}

// A mesh of a tube lies within the chord share of the tube's outer
// radius of the tube's wall, for every tube, whatever its radius.
func TestTubeMeshLiesWithinTheChordShare(t *testing.T) {
	slack := tubeMeshSlack(t)
	for _, tube := range meshedTubes(t) {
		limit := TubeChordShare()*(tube.Radius+tube.Thick/pair) +
			slack*tube.Height
		if got := farthestFromWall(meshOf(t, tube), tube); got > limit {
			t.Errorf("%+v: %v from the wall, share allows %v", tube, got, limit)
		}
	}
}

// Held against a tube of twice the radius the same mesh is far from the
// wall, so the check above is one that can fail. So is a mesh with
// too few sides held to the share.
func TestTubeMeshAntiTestFailsWhenTheTubeIsWrong(t *testing.T) {
	slack := tubeMeshSlack(t)
	for _, tube := range meshedTubes(t) {
		limit := TubeChordShare()*(tube.Radius+tube.Thick/pair) +
			slack*tube.Height
		wrong := Tube{Radius: tube.Radius * pair, Thick: tube.Thick,
			Height: tube.Height}
		if got := farthestFromWall(meshOf(t, tube), wrong); got <= limit {
			t.Errorf("%+v held to a tube of twice the radius: %v, within %v",
				tube, got, limit)
		}
		coarse, err := tube.Mesh(picked(t, "tube-mesh-quarter-share"))
		if err != nil {
			t.Fatal(err)
		}
		if got := farthestFromWall(coarse, tube); got <= limit {
			t.Errorf("%+v with a quarter share: %v, within %v", tube, got,
				limit)
		}
	}
}

// A hollow tube is two walls and two flat rings; one with no hole is
// a wall and two discs. Sides do not depend on the radius.
func TestTubeMeshFaceCounts(t *testing.T) {
	sides := tubeSides(TubeChordShare())
	hollow := pickedCount(t, "tube-mesh-hollow-faces")
	solid := pickedCount(t, "tube-mesh-solid-faces")
	tubes := meshedTubes(t)
	for _, row := range []struct {
		tube Tube
		want int
	}{
		{tubes[tubeMeshRing], hollow * sides},
		{tubes[tubeMeshHair], solid * sides},
		{tubes[tubeMeshCable], hollow * sides},
	} {
		if got := len(meshOf(t, row.tube).Faces); got != row.want {
			t.Errorf("%+v: %d faces, want %d", row.tube, got, row.want)
		}
	}
	if sides < tubeLeastSides || tubeSides(half) < tubeLeastSides {
		t.Errorf("sides %d and %d, fewer than a ring", sides, tubeSides(half))
	}
}

// windsAgainstNormals counts the faces whose winding disagrees with the normals
// of their corners.
func windsAgainstNormals(made *Mesh) int {
	against := nought
	for _, face := range made.Faces {
		corners := tubeMeshCornersOf(made, face)
		first := corners[tubeMeshFirstCorner]
		second := corners[tubeMeshSecondCorner]
		third := corners[tubeMeshThirdCorner]
		facing := made.Normals[face[tubeMeshFirstCorner]].
			Add(made.Normals[face[tubeMeshSecondCorner]]).
			Add(made.Normals[face[tubeMeshThirdCorner]])
		if second.Sub(first).Cross(third.Sub(first)).Dot(facing) <= nought {
			against++
		}
	}
	return against
}

// Every face winds the way its corners' normals face, and every normal
// is one long.
func TestTubeMeshWindsWithItsNormals(t *testing.T) {
	tolerance := picked(t, "tube-mesh-normal-tolerance")
	for _, tube := range meshedTubes(t) {
		made := meshOf(t, tube)
		if against := windsAgainstNormals(made); against != nought {
			t.Errorf("%+v: %d faces wound against their normals", tube, against)
		}
		for index, normal := range made.Normals {
			if math.Abs(normal.Len()-unity) > tolerance {
				t.Fatalf("%+v: normal %d is %v long", tube, index, normal.Len())
			}
		}
	}
}

// A mesh whose normals all face the other way is found by the check above:
// every face winds against them. A mesh is not made of a face turned over,
// alone or with all the rest: one alone, or all of a closed piece, is
// refused in words where the mesh is made (the winding tests of this
// package hold that), so the check is held to the normals turned.
func TestTubeMeshWindingCheckFindsFlippedNormals(t *testing.T) {
	made := meshOf(t, meshedTubes(t)[tubeMeshRing])
	turned := make([]maths.Vec, len(made.Normals))
	for place, normal := range made.Normals {
		turned[place] = normal.Scale(-unity)
	}
	flipped, err := NewMesh(made.Vertices, made.Faces, turned, true)
	if err != nil {
		t.Fatal(err)
	}
	if against := windsAgainstNormals(flipped); against != len(made.Faces) {
		t.Errorf("%d faces against their normals, want all %d",
			against, len(made.Faces))
	}
}

// A tube with no wall, no length or no outer radius, and a share that is
// not a share, are errors and not a mesh.
func TestTubeMeshRefusesWhatIsNotATube(t *testing.T) {
	for _, tube := range tubeMeshOf(t, tubeMeshRefusedFile) {
		if _, err := tube.Mesh(TubeChordShare()); err == nil {
			t.Errorf("%+v made a mesh", tube)
		}
	}
	ring := meshedTubes(t)[tubeMeshRing]
	for _, each := range readSheet(t, tubeMeshShareFile) {
		share := cell(t, each, "share")
		if _, err := ring.Mesh(share); err == nil {
			t.Errorf("a share of %v made a mesh", share)
		}
	}
}
