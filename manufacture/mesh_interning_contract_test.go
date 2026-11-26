package manufacture

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

func TestMeshInterningPreservesEqualityAndCollisions(t *testing.T) {
	original := drawnBox(t, 1, nil).Mesh
	var meshes sharedMeshes
	first, err := meshes.of(original)
	if err != nil {
		t.Fatal(err)
	}

	zero := *original
	zero.Triangles = slices.Clone(original.Triangles)
	for i := range zero.Triangles {
		for j := range zero.Triangles[i] {
			point := &zero.Triangles[i][j]
			if point.X == 0 {
				point.X = math.Copysign(0, -1)
			}
			if point.Y == 0 {
				point.Y = math.Copysign(0, -1)
			}
			if point.Z == 0 {
				point.Z = math.Copysign(0, -1)
			}
		}
	}
	same, err := meshes.of(&zero)
	if err != nil || same != first {
		t.Fatalf("signed-zero equal mesh: %p %v", same, err)
	}

	other := *original
	other.Triangles = slices.Clone(original.Triangles)
	other.Triangles[1], other.Triangles[2] = other.Triangles[2], other.Triangles[1]
	if other.Triangles[0] != original.Triangles[0] || other.Triangles[len(other.Triangles)-1] != original.Triangles[len(original.Triangles)-1] {
		t.Fatal("test meshes must share endpoint triangles")
	}
	second, err := meshes.of(&other)
	if err != nil || second == first {
		t.Fatalf("different middle triangles: %p %v", second, err)
	}
	again, err := meshes.of(original)
	if err != nil || again != first {
		t.Fatalf("original should retain its first mesh: %p %v", again, err)
	}
}

func TestMeshInterningPreservesInvalidMeshErrors(t *testing.T) {
	var meshes sharedMeshes
	empty := &cities.Mesh{}
	_, want := (surface(nil)).meshed()
	_, got := meshes.of(empty)
	if (want == nil) != (got == nil) || want != nil && want.Error() != got.Error() {
		t.Fatalf("empty mesh: got %v, want %v", got, want)
	}
	original := drawnBox(t, 1, nil).Mesh
	for _, index := range []int{0, len(original.Triangles) / 2} {
		invalid := *original
		invalid.Triangles = slices.Clone(original.Triangles)
		invalid.Triangles[index][0].X = math.NaN()
		skin := make(surface, len(invalid.Triangles))
		for i, triangle := range invalid.Triangles {
			skin[i] = face(triangle)
		}
		_, want = skin.meshed()
		first, got := meshes.of(&invalid)
		if (want == nil) != (got == nil) || want != nil && want.Error() != got.Error() {
			t.Fatalf("NaN mesh at triangle %d: got %v, want %v", index, got, want)
		}
		second, again := meshes.of(&invalid)
		if got != nil && (again == nil || again.Error() != got.Error()) || got == nil && (again != nil || first == second) {
			t.Fatalf("repeated NaN mesh at triangle %d changed equality or error: %v, %v", index, got, again)
		}
	}
}

func TestRenderedAssemblyKeepsInputsAndOwnsMeshes(t *testing.T) {
	look := plain("test material", .2, .3, .4)
	first, second := drawnBox(t, 1, look), drawnBox(t, 1, look)
	beforeFirst, beforeSecond := slices.Clone(first.Mesh.Triangles), slices.Clone(second.Mesh.Triangles)
	made, err := goodOfRendered([]cities.Rendered{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first.Mesh.Triangles, beforeFirst) || !slices.Equal(second.Mesh.Triangles, beforeSecond) {
		t.Fatal("assembly changed the rendered source triangles")
	}
	members := made.groups[body].Members()
	if len(members) != 2 || members[0].Mesh != members[1].Mesh {
		t.Fatal("assembly did not share equal source meshes")
	}
	bounds := members[0].Mesh.Bounds()
	first.Mesh.Triangles[0][0].X += 100
	if members[0].Mesh.Bounds() != bounds {
		t.Fatal("assembled mesh still depends on later source mutations")
	}
}
