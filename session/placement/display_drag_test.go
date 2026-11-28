package placement

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

func TestDragViewDepthKeepsDirectionAcrossCameraPitches(t *testing.T) {
	for _, test := range []struct {
		name  string
		pitch float64
	}{
		{"down", -25},
		{"level", 0},
		{"up", 25},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene := lookScene()
			scene.Eye.Orientation = maths.About(maths.XAxis, test.pitch*maths.Degree)
			anchor := scene.Eye.Forward().Scale(3)
			east, north, err := scene.DragViewDepth(anchor, .45, .4, .55, .4)
			if err != nil || math.Abs(east-.6) > near || math.Abs(north) > near {
				t.Fatalf("horizontal drag = %g, %g (%v)", east, north, err)
			}
			east, north, err = scene.DragViewDepth(anchor, .45, .45, .45, .55)
			wantNorth := .6 * math.Sin(test.pitch*maths.Degree)
			if err != nil || math.Abs(east) > near || math.Abs(north-wantNorth) > near {
				t.Fatalf("vertical drag = %g, %g (%v), want north %g", east, north, err, wantNorth)
			}
		})
	}
}

func TestDragViewDepthRejectsUnstableProjection(t *testing.T) {
	scene := lookScene()
	for _, test := range []struct {
		name   string
		anchor maths.Vec
		endX   float64
	}{
		{"behind eye", maths.V(0, 0, 2), .6},
		{"captured pointer too far outside", maths.V(0, 0, -2), 30},
		{"bad pointer", maths.V(0, 0, -2), math.NaN()},
	} {
		t.Run(test.name, func(t *testing.T) {
			east, north, err := scene.DragViewDepth(test.anchor, .5, .5, test.endX, .5)
			if err == nil || east != 0 || north != 0 {
				t.Fatalf("invalid projection yielded %g, %g (%v)", east, north, err)
			}
		})
	}
}

func TestDragViewDepthAllowsWidePortraitEdgeAndNearbyCapture(t *testing.T) {
	scene := lookScene()
	scene.Width, scene.Height = 60, 100
	scene.Lens = film.Pinhole{Wide: 179 * maths.Degree}
	anchor := maths.V(0, 0, -2)
	ray, err := scene.displayRay(.95, .95)
	if err != nil || ray.Dir.Dot(scene.Eye.Forward()) >= .05 {
		t.Fatal("test needs a valid wide edge ray rejected by the former angular threshold")
	}
	east, north, err := scene.DragViewDepth(anchor, .95, .95, 1.03, .95)
	wantEast := 4 * math.Tan(179*maths.Degree/2) * 4 / 60
	if err != nil || math.Abs(east-wantEast) > near || math.Abs(north) > near {
		t.Fatalf("wide edge drag = %g, %g (%v), want east %g", east, north, err, wantEast)
	}
}

func TestDragViewDepthRefusesNearlyParallelRays(t *testing.T) {
	ray := maths.Ray{Dir: maths.V(1, 0, -1e-14)}
	if point, ok := dragViewPoint(ray, maths.V(0, 0, -2), maths.V(0, 0, -1)); ok {
		t.Fatalf("grazing camera ray yielded %v", point)
	}
}
