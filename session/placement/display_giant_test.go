package placement

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

func TestGiantBoundaryUsesProducerFrameAndScale(t *testing.T) {
	world := thing.NewWorld()
	orbit := thing.Orbit{Centre: maths.V(0, 0, -4e8),
		Radius: 5e7, Axis: maths.YAxis, Period: 20 * time.Second, Scale: 2}
	volume := world.AddVolume(thing.Volume{Medium: medium.Giant{Radius: 7e7}, Motion: orbit})
	for _, frame := range []time.Duration{0, 5 * time.Second} {
		boundaries := WorkingGiantBoundaries(world, frame)
		at := volume.At(frame)
		if len(boundaries) != 1 || boundaries[0].Centre != at.Origin || boundaries[0].Radius != 7e7*at.Scale {
			t.Fatalf("boundary differs from producer pose: %v", boundaries)
		}
	}
}

func TestGiantBoundaryProjectsPinholeTangentRays(t *testing.T) {
	scene := lookScene()
	scene.Width, scene.Height = 1000, 800
	bound := DisplayGiantBoundary{Centre: maths.V(5e7, -8e7, -4e8), Radius: 7e7}
	edges := scene.ProjectGiantBoundary(bound)
	if len(edges) != 96 {
		t.Fatalf("wanted the full off-axis limb, got %d segments", len(edges))
	}
	half, aspect := scene.projectionShape()
	for _, edge := range edges {
		for _, point := range edge {
			ray := maths.V((2*point[0]-1)*half, (1-2*point[1])*half/aspect, -1).Unit()
			if delta := math.Abs(ray.Cross(bound.Centre).Len()/bound.Radius - 1); delta > 1e-12 {
				t.Fatalf("projected limb vertex is off the producer tangent by %.4g radii", delta)
			}
		}
	}
}

func TestNearGiantBoundaryLeavesDistantGlobePolicyAlone(t *testing.T) {
	scene := lookScene()
	bound := DisplayGiantBoundary{Centre: maths.V(0, 0, -120), Radius: 35}
	if edges := scene.ProjectGiantBoundary(bound); len(edges) != 96 {
		t.Fatalf("near giant should show its full outline, got %d segments", len(edges))
	}
	if edges := scene.ProjectDistantSphere(bound); len(edges) != 0 {
		t.Fatalf("near globe should have no distant outline, got %d segments", len(edges))
	}
}

func TestGiantBoundaryClipsAndRefusesSpheresAtTheEyePlane(t *testing.T) {
	scene := lookScene()
	bound := DisplayGiantBoundary{Centre: maths.V(4e8, 0, -4e8), Radius: 7e7}
	edges := scene.ProjectGiantBoundary(bound)
	if len(edges) == 0 || len(edges) >= 96 {
		t.Fatalf("partially visible outline has %d segments", len(edges))
	}
	touchesEdge := false
	for _, edge := range edges {
		for _, point := range edge {
			if point[0] < 0 || point[0] > 1 || point[1] < 0 || point[1] > 1 {
				t.Fatalf("outline escaped clip: %v", point)
			}
			touchesEdge = touchesEdge || point[0] == 1
		}
	}
	if !touchesEdge {
		t.Fatal("partial outline failed to reach the side of the frame")
	}
	for _, position := range []maths.Vec{
		maths.V(0, 0, -100),
		maths.V(7e7, 0, -7e7),
		maths.V(7e7, 0, -6e7),
		maths.V(0, 0, 4e8),
		maths.V(8e8, 0, -4e8),
	} {
		bound.Centre = position
		if edges := scene.ProjectGiantBoundary(bound); len(edges) > 0 {
			t.Fatalf("out-of-scope boundary at %v produced %d edges", position, len(edges))
		}
	}
}

func TestAzureOrbitShowsAProducerGiantBoundary(t *testing.T) {
	const file = "../../sets/caprica-binary/working-system.yaml"
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(file, data)
	if err != nil {
		t.Fatal(err)
	}
	shot, err := read.Shot("azure-orbit", nil)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world := thing.NewWorld()
	if err := stage.Add(world, said, maths.Vec{}); err != nil {
		t.Fatal(err)
	}
	scene := DisplayScene{Eye: shot.Eye(), Lens: shot.Lens(), Width: 1280, Height: 800}
	for _, boundary := range WorkingGiantBoundaries(world, 0) {
		if !boundary.Centre.Near(shot.Focus(), 1e-6) {
			continue
		}
		if edges := scene.ProjectGiantBoundary(boundary); len(edges) == 0 {
			t.Fatal("Azure has no visible boundary at its authored camera")
		}
		return
	}
	t.Fatal("no producer giant boundary at the camera's Azure focus")
}
