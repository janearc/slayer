package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
	"math"
	"os"
	"testing"
)

// Foundations must meet the triangles that are actually drawn, rather than
// trusting a separate exact-height query when adaptive terrain is too coarse.
func TestMaunaKeaFoundationsMeetPreparedTerrain(t *testing.T) {
	path := os.Getenv("SLAYER_MAUNAKEA_TERRAIN")
	if path == "" {
		t.Skip("set SLAYER_MAUNAKEA_TERRAIN to the pinned NOAA CRM2023 raster")
	}
	raster, err := terrain.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	s := &heardStage{world: thing.NewWorld()}
	p := &stagePlan{Eye: [3]float64{-150, 46, -190}, Look: [3]float64{-35, 10, 32}, Wide: 72}
	if err := s.maunaKeaGround(p, raster); err != nil {
		t.Fatal(err)
	}
	for _, at := range []maths.Vec{maths.V(0, 0, 0), maths.V(100, 0, 100), maths.V(564.62, 0, 256.823), maths.V(582.511, 0, 95.551), maths.V(-267.436, 0, 367.332), maths.V(-370, 0, 205)} {
		ray := maths.Ray{At: maths.V(at.X, 1000, at.Z), Dir: maths.V(0, -1, 0)}
		near := 10000.0
		hit := false
		for _, part := range s.world.Things {
			if mesh, ok := part.Shape.(shape.Hitter); ok {
				distance, has := mesh.Hit(ray, 0, near)
				if has {
					near = distance
					hit = true
				}
			}
		}
		floor := s.ground(at.X, at.Z)
		if !hit || math.Abs((1000-near)-floor) > .05 {
			t.Errorf("foundation at %.3f %.3f: source %.6f, drawn %.6f, hit %t", at.X, at.Z, floor, 1000-near, hit)
		}
	}
}
