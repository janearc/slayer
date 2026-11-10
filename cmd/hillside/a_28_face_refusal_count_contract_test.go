package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestFaceRefusalNamesTheTotalRealGeometryItWouldKeep(t *testing.T) {
	given := mustParse(t)
	// Count real public geometry independently of the command's tally.
	counts := [2]int{}
	for index, name := range []string{"can", "cobble"} {
		kind, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		object, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(stir(given.seed, index)), "quick")
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
	given.faces = counts[0] + counts[1] - 1
	if err := stand(world, &made, given, have, "can", 0, maths.Subspace{}); err != nil {
		t.Fatal("first real object should fit", err)
	}
	before, things := made, len(world.Things)
	err := stand(world, &made, given, have, "cobble", 1, maths.Subspace{})
	want := fmt.Sprintf("it would keep %d faces, past the -faces of %d",
		counts[0]+counts[1], given.faces)
	if !errors.Is(err, errTooMany) || !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal lost the actual combined face count: %v, want %q", err, want)
	}
	if made != before || len(world.Things) != things {
		t.Fatal("refused geometry was added or counted")
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
