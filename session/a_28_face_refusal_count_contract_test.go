package session

import (
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the faces a tally says it keeps are the faces of the real geometry
// that was stood, counted here independently of the command's tally.
func TestTheTallyKeepsTheTotalRealGeometryStood(t *testing.T) {
	given := mustParse(t)
	// Count real public geometry independently of the command's tally.
	counts := [2]int{}
	for index, name := range []string{"can", "cobble"} {
		kind, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		sown := manufacture.Sow(stir(given.seed, index))
		object, err := manufacture.MakeWithComplexity(kind,
			sown, "quick")
		if err != nil {
			t.Fatal(err)
		}
		parts, err := object.Things(maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range parts {
			switch solid := part.Shape.(type) {
			case *mesh.Mesh:
				counts[index] += solid.FaceCount()
			case *shape.Mesh:
				counts[index] += len(solid.Faces)
			case *shape.MeshGroup:
				counts[index] += solid.StoredFaces()
			default:
				t.Fatalf("fixture has uncounted geometry %T", solid)
			}
		}
		if counts[index] < 1 {
			t.Fatal("fixture lost its actual faces", name)
		}
	}
	world, have := thing.NewWorld(), stock{}
	var made tally
	for index, name := range []string{"can", "cobble"} {
		err := stand(world, &made, given, have, name, index,
			maths.Subspace{})
		if err != nil {
			t.Fatal("real object stood", name, err)
		}
	}
	if made.kept != counts[0]+counts[1] || made.things != 2 {
		t.Fatalf("the tally keeps %d faces of %d things, the real "+
			"geometry has %d faces of 2", made.kept, made.things,
			counts[0]+counts[1])
	}
}

// Count public generated shapes, independently of the command's tally.
// Repeated group placements cost logical faces; each local prototype is
// stored once within its material group.
func publicGeometryCounts(t *testing.T, parts []thing.Thing) (stored, logical int) {
	t.Helper()
	for _, part := range parts {
		switch solid := part.Shape.(type) {
		case *mesh.Mesh:
			stored += solid.FaceCount()
			logical += solid.FaceCount()
		case *shape.Mesh:
			stored += len(solid.Faces)
			logical += len(solid.Faces)
		case *shape.MeshGroup:
			stored += solid.StoredFaces()
			logical += solid.FaceCount()
		default:
			t.Fatalf("reference has uncounted public geometry %T", solid)
		}
	}
	return stored, logical
}
