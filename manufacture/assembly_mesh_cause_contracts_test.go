package manufacture

import (
	"errors"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

func TestPlacedObjectPreservesActualMeshConstructionCauses(t *testing.T) {
	material := &thing.Material{Name: "control", Colour: maths.Grey(.5)}
	for _, surface := range []fixedSurface{nil, {{maths.Vec{}, maths.Vec{}, maths.Vec{}}}} {
		made := good{parts: []part{{of: body, shape: surface}}, dressed: looks{body: material}}
		if _, err := made.Things(maths.Subspace{}); !errors.Is(err, mesh.ErrMesh) {
			t.Fatalf("object lost empty/degenerate mesh cause: %v", err)
		}
	}
}
