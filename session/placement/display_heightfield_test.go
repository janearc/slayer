package placement

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestWorkingWorldDrawsHeightSamplesAsLabeledProxy(t *testing.T) {
	field, err := shape.NewHeightField([]float64{0, 1, 2, 3}, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field, Motion: thing.Still(maths.Subspace{Origin: maths.V(10, 5, 20)})})
	got, err := WorkingWorld(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Mesh.Indices) != 6 || !strings.Contains(got.Mesh.Notice, "approximation") {
		t.Fatalf("height field mesh has %d indices and notice %q", len(got.Mesh.Indices), got.Mesh.Notice)
	}
	want := map[maths.Vec]bool{maths.V(8, 5, 18): false, maths.V(12, 6, 18): false, maths.V(8, 7, 22): false, maths.V(12, 8, 22): false}
	for _, v := range got.Mesh.Vertices {
		if _, known := want[v.Position]; known {
			want[v.Position] = true
		}
	}
	for p, found := range want {
		if !found {
			t.Errorf("sample %v missing", p)
		}
	}
}

func TestWorkingWorldRefusesUnboundedHeightFieldMesh(t *testing.T) {
	const samples = 514
	field, err := shape.NewHeightField(make([]float64, samples*samples), samples, 10)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field})
	if _, err := WorkingWorld(world, 0); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("oversized field error = %v", err)
	}
}

func TestOlderViewRefusesUnlabeledHeightField(t *testing.T) {
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := newDisplayBuilder().thing(thing.Thing{Shape: field}); err == nil {
		t.Fatal("older display accepted a proxy it cannot label")
	}
}
