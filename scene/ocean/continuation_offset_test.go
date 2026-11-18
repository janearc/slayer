package ocean

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"testing"
)

func TestInfiniteMeanContinuationOffsetStaysInsideCaptured1024Crease(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{Sea: oceantex.Sea{Wind: 7, Fetch: 50000}, Extent: 512, Samples: 1024, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	// The captured TIR departure lies just right of this collar grid seam.
	// A normal lift enters the neighboring left cell and incorrectly exits water.
	point := maths.V(590*512.0/1023-256+.0001, 0, 291.5181)
	height, _, _ := surface.finite.sample(point.X, point.Z)
	point.Y = height
	offset := surface.Normal(point).Scale(-.0015)
	legacy := point.Add(offset)
	height, _, _ = surface.finite.sample(legacy.X, legacy.Z)
	if gap := legacy.Y - height; gap <= .001 {
		t.Fatalf("fixture did not cross crease: raw gap%g", gap)
	}
	got := surface.OffsetToSide(point, offset)
	height, _, _ = surface.finite.sample(got.X, got.Z)
	if gap := got.Y - height; gap >= 0 {
		t.Fatalf("departure is not water-side: raw gap%g at%v", gap, got)
	}
	if got.Sub(point).Len() >= offset.Len() {
		t.Fatal("crossing lift was not reduced")
	}
	if got.Sub(point).Unit().Sub(offset.Unit()).Len() > 1e-8 {
		t.Fatal("lift changed its direction")
	}
}

func TestInfiniteMeanContinuationOffsetStaysOutsideValleyCrease(t *testing.T) {
	core, err := shape.NewHeightField([]float64{2, 0, 2, 2, 0, 2, 2, 0, 2}, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(.0001, .0002, 0)
	offset := surface.Normal(point).Scale(.0015)
	if gap := surface.Distance(point.Add(offset)); gap >= 0 {
		t.Fatalf("fixture did not cross into water: %g", gap)
	}
	if got := surface.OffsetToSide(point, offset); surface.Distance(got) <= 0 {
		t.Fatalf("departure did not remain air-side: %v", got)
	}
}

func TestInfiniteMeanContinuationOffsetPreservesSafeSides(t *testing.T) {
	surface, err := NewInfiniteMeanContinuation(continuationCore(t, true), 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(1, 0, 5)
	for _, offset := range []maths.Vec{maths.V(.001, -.002, .001), maths.V(.001, .002, .001)} {
		if got := surface.OffsetToSide(point, offset); got != point.Add(offset) {
			t.Fatalf("safe offset changed: got%v want%v", got, point.Add(offset))
		}
	}
}

func TestInfiniteMeanContinuationOffsetFallsBackAtCoordinatePrecision(t *testing.T) {
	surface, err := NewInfiniteMeanContinuation(continuationCore(t, true), 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []float64{-1, 1} {
		// A near-root residual can exceed an offset too small to represent.
		point := maths.V(1, -side*1e-12, 5)
		got := surface.OffsetToSide(point, maths.V(0, side*1e-30, 0))
		if gap := surface.Distance(got); gap*side <= 0 {
			t.Fatalf("precision fallback lost side %g: point=%v gap=%g", side, got, gap)
		}
	}
}
