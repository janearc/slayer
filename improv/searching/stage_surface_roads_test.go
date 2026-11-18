package searching

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func roadTestTerrain(t *testing.T, folded bool) *mesh.Mesh {
	t.Helper()
	y := 0.0
	if folded {
		y = 13
	}
	m, err := mesh.New([]maths.Vec{maths.V(-80, 0, -80), maths.V(80, 0, -80), maths.V(80, y, 80), maths.V(-80, 0, 80)}, [][3]int{{0, 2, 1}, {0, 3, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// A vertex-only check misses a road bridging a diagonal fold. Every interior
// sample must retain the same declared separation from the actual triangles.
func TestSummitRoadSurfaceFollowsActualTerrainAcrossFolds(t *testing.T) {
	terrain := roadTestTerrain(t, true)
	s := newSummitSurface([]*mesh.Mesh{terrain})
	s.ribbon([]maths.Vec{maths.V(-65, 0, -35), maths.V(-20, 0, 15), maths.V(25, 0, -15), maths.V(65, 0, 35)}, 8)
	road, err := s.roadMesh()
	if err != nil {
		t.Fatal(err)
	}
	if len(road.Faces) < 10 {
		t.Fatal("missing draped road")
	}
	for _, face := range road.Faces {
		a, b, c := road.Vertices[face[0]], road.Vertices[face[1]], road.Vertices[face[2]]
		for _, weights := range [][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1.0 / 3, 1.0 / 3, 1.0 / 3}, {.1, .45, .45}} {
			p := a.Scale(weights[0]).Add(b.Scale(weights[1])).Add(c.Scale(weights[2]))
			distance, hit := terrain.Hit(maths.Ray{At: maths.V(p.X, 1000, p.Z), Dir: maths.V(0, -1, 0)}, 0, 2000)
			if !hit || math.Abs(p.Y-(1000-distance)-summitRoadLift) > 1e-6 {
				t.Fatalf("road is detached/buried at %v, separation %g, hit %t", p, p.Y-(1000-distance), hit)
			}
		}
	}
}

// Crossing roads must form one surface without doubled junction area.
func TestSummitRoadJunctionHasNoOverlappingPavement(t *testing.T) {
	s := newSummitSurface([]*mesh.Mesh{roadTestTerrain(t, false)})
	s.ribbon([]maths.Vec{maths.V(-50, 0, 0), maths.V(50, 0, 0)}, 8)
	s.ribbon([]maths.Vec{maths.V(0, 0, -50), maths.V(0, 0, 50)}, 8)
	road, err := s.roadMesh()
	if err != nil {
		t.Fatal(err)
	}
	area := 0.0
	for _, f := range road.Faces {
		a, b, c := road.Vertices[f[0]], road.Vertices[f[1]], road.Vertices[f[2]]
		area += math.Abs(crossXZ(b.Sub(a), c.Sub(a))) / 2
	}
	if math.Abs(area-(2*100*8-8*8)) > 1e-4 {
		t.Fatalf("junction area %g, expected union area1536", area)
	}
	for _, p := range []maths.Vec{maths.V(0, 1, 0), maths.V(49, 1, 0), maths.V(0, 1, 49)} {
		if _, hit := road.Hit(maths.Ray{At: p, Dir: maths.V(0, -1, 0)}, 0, 2); !hit {
			t.Fatalf("gap at %v", p)
		}
	}
}

func TestSummitRoadCentrelineHasContinuousBends(t *testing.T) {
	path := smoothSummitPath([]maths.Vec{maths.V(-60, 0, 0), maths.V(0, 0, 0), maths.V(0, 0, 60)})
	maxTurn := 0.0
	for i := 1; i < len(path)-1; i++ {
		a, b := path[i].Sub(path[i-1]).Unit(), path[i+1].Sub(path[i]).Unit()
		angle := math.Acos(math.Max(-1, math.Min(1, a.Dot(b))))
		maxTurn = math.Max(maxTurn, angle)
	}
	if maxTurn > 10*maths.Degree {
		t.Fatalf("road retained a sharp bend: %g degrees", maxTurn/maths.Degree)
	}
}
