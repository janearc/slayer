package placement

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type travellingSphere struct{}

func (travellingSphere) At(at time.Duration) maths.Subspace {
	return maths.Subspace{Origin: maths.V(at.Seconds(), 0, 0)}
}

func TestWorkingWorldUsesStagedTimeAndReportsOmissions(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: travellingSphere{}})
	world.AddVolume(thing.Volume{})
	world.AddField(thing.Field{})
	result, err := WorkingWorld(world, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	left, right, ok := result.Mesh.Bounds()
	if !ok || left.X > 2.01 || left.X < 1.99 || right.X < 3.99 || right.X > 4.01 {
		t.Fatalf("time-three bounds = %v, %v, %v", left, right, ok)
	}
	if result.Volumes != 1 || result.Fields != 1 || result.Lamps != 0 {
		t.Fatalf("omissions = %+v", result)
	}
	other, err := WorkingWorld(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := other.Mesh.Bounds()
	if first.X > -0.99 || first.X < -1.01 {
		t.Fatalf("time-zero left = %v", first)
	}
}

func TestWorkingWorldRefusesUnsupportedAndEmptyGeometry(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	world.Add(thing.Thing{Shape: shape.Plane{}})
	if _, err := WorkingWorld(world, 0); err == nil || !strings.Contains(err.Error(), "Plane") {
		t.Fatalf("unsupported surface: %v", err)
	}
	if _, err := WorkingWorld(nil, 0); err == nil {
		t.Fatal("nil world was accepted")
	}
	empty := thing.NewWorld()
	empty.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Attrs: thing.Attrs{Hidden: true}})
	if _, err := WorkingWorld(empty, 0); err == nil {
		t.Fatal("hidden-only world was accepted")
	}
}
