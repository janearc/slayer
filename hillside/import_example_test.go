package hillside

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/mesh"
)

// TestOaklandExampleStagesTwoMetresBesideTheManufacturedLarson checks the
// shipped scene through the same admission and staging path as the app.
func TestOaklandExampleStagesTwoMetresBesideTheManufacturedLarson(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("..", "sets", "oakland-larson", "set.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	wanted, err := job.StillOf(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	source, ok := session.source.(stillFramer)
	if !ok {
		t.Fatal("example did not stage a still")
	}
	count, native := 0, 0
	low, high := math.Inf(1), math.Inf(-1)
	for _, part := range source.world.Things {
		if part.Material.Name != "mannequin" {
			native++
			continue
		}
		count++
		shape, ok := part.Shape.(*mesh.Mesh)
		if !ok {
			t.Fatalf("imported part has non-native shape %T", part.Shape)
		}
		for _, corner := range shape.Points() {
			y := part.At(0).Out(corner).Y
			low, high = min(low, y), max(high, y)
		}
	}
	if count != 8 || native <= 1 {
		t.Fatalf("wanted eight imported parts beside pad and craft, got %d/%d", count, native)
	}
	// Absolute metres: permits arithmetic rounding, far below authored precision.
	if math.Abs(low) > 1e-12 || math.Abs(high-2) > 1e-12 {
		t.Fatalf("mannequin spans %.17g to %.17g m, wanted 0 to 2", low, high)
	}
}
