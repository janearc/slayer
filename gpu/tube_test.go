package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// tubeWorld is a world with each tube in it, each stood a metre further
// along x than the one before.
func tubeWorld(tubes ...shape.Tube) *thing.World {
	world := thing.NewWorld()
	for index, each := range tubes {
		corner := maths.V(float64(index), 0, 0)
		at := maths.NewSubspace(corner, maths.Identity, 1)
		world.Add(thing.Thing{Shape: each, Motion: thing.Still(at)})
	}
	return world
}

// facesOfTube is how many triangles a tube has when it is stood.
func facesOfTube(t *testing.T, each shape.Tube) int {
	t.Helper()
	skin, err := each.Mesh(shape.TubeChordShare())
	if err != nil {
		t.Fatal(err)
	}
	return len(skin.Faces)
}

// A tube is drawn: the encoder takes it, stands it once, and packs the
// faces of its mesh. The faces are counted where they are packed, in
// counts.triangles: the pool packs each tree's faces from the tree and
// copies none into faces.triangles (pool_copy_test.go says so).
func TestATubeIsStoodAsItsMesh(t *testing.T) {
	cable := shape.Tube{Radius: .02, Thick: .004, Height: 3}
	encoded, err := encodeKept(tubeWorld(cable), nil, maths.Subspace{}, nil, 0, 8, 8, nil)
	if err != nil {
		t.Fatalf("a tube was refused: %v", err)
	}
	if len(encoded.standings) != 1 {
		t.Fatalf("%d standings, want 1", len(encoded.standings))
	}
	if got, want := encoded.counts.triangles, facesOfTube(t, cable); got != want {
		t.Errorf("%d faces packed, want %d", got, want)
	}
}

// packedForTubes is how many faces the encoder packs for a world of the
// tubes.
func packedForTubes(t *testing.T, tubes ...shape.Tube) int {
	t.Helper()
	encoded, err := encodeKept(tubeWorld(tubes...), nil, maths.Subspace{},
		nil, 0, 8, 8, nil)
	if err != nil {
		t.Fatal(err)
	}
	return encoded.counts.triangles
}

// Anti-test: the count the tests here read sees what is packed. A world
// of one tube packs the faces of one, and a world of that tube and a wider
// one packs more, so a count that stayed at nought, as faces.triangles
// does, or one that did not follow the tubes, would be reported.
func TestThePackedCountFollowsTheTubes(t *testing.T) {
	thin := shape.Tube{Radius: .02, Thick: .004, Height: 3}
	wide := shape.Tube{Radius: .04, Thick: .004, Height: 3}
	one, two := packedForTubes(t, thin), packedForTubes(t, thin, wide)
	if one == 0 {
		t.Fatal("a tube packed no faces")
	}
	if two <= one {
		t.Errorf("two tubes packed %d faces, one packed %d", two, one)
	}
}

// Tubes of the same size are grown once and stood each time they are
// placed.
func TestEqualTubesAreGrownOnce(t *testing.T) {
	cable := shape.Tube{Radius: .02, Thick: .004, Height: 3}
	encoded, err := encodeKept(tubeWorld(cable, cable, cable), nil,
		maths.Subspace{}, nil, 0, 8, 8, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.standings) != 3 {
		t.Fatalf("%d standings, want 3", len(encoded.standings))
	}
	if got, want := encoded.counts.triangles, facesOfTube(t, cable); got != want {
		t.Errorf("%d faces packed for three equal tubes, want %d", got, want)
	}
}

// Tubes that differ are not taken for each other: each is grown.
func TestDifferentTubesAreEachGrown(t *testing.T) {
	thin := shape.Tube{Radius: .02, Thick: .004, Height: 3}
	wide := shape.Tube{Radius: .04, Thick: .004, Height: 3}
	encoded, err := encodeKept(tubeWorld(thin, wide), nil, maths.Subspace{}, nil, 0, 8, 8, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := facesOfTube(t, thin) + facesOfTube(t, wide)
	if got := encoded.counts.triangles; got != want {
		t.Errorf("%d faces packed for two tubes, want %d", got, want)
	}
}

// A tube that cannot be meshed is refused as unsupported, and each
// placing of it is, not only the first.
func TestATubeWithNoLengthIsRefused(t *testing.T) {
	broken := shape.Tube{Radius: .02, Thick: .004, Height: 0}
	for count := 1; count <= 2; count++ {
		world := tubeWorld(broken)
		if count == 2 {
			world = tubeWorld(broken, broken)
		}
		_, err := encodeKept(world, nil, maths.Subspace{}, nil, 0, 8, 8, nil)
		var refused unsupported
		if !errors.As(err, &refused) {
			t.Errorf("%d placings: error %v, want unsupported", count, err)
		}
	}
}
