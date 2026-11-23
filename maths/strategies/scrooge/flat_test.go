package scrooge

import (
	"math"
	"testing"
)

// TestASurfaceIsFlatWhereItsHeightIsAFootprintWide: the distance is the one
// at which the footprint's width is the surface's height, crest to trough.
func TestASurfaceIsFlatWhereItsHeightIsAFootprintWide(t *testing.T) {
	cone := Footprint{Spread: 0.002}
	const reach = 2.5
	from := cone.FlatFrom(reach)
	if math.Abs(from-2500) > 1e-9 {
		t.Fatalf("flat from %v, want 2500", from)
	}
	if width := cone.WidthAt(from); math.Abs(width-2*reach) > 1e-9 {
		t.Fatalf("the footprint is %v wide there, want %v", width, 2*reach)
	}
}

// TestAFinerPictureKeepsItsWavesFurther: a third of the spread, as of a
// picture three times as many pixels across, is three times the distance.
func TestAFinerPictureKeepsItsWavesFurther(t *testing.T) {
	coarse := Footprint{Spread: 0.006}.FlatFrom(1)
	fine := Footprint{Spread: 0.002}.FlatFrom(1)
	if math.Abs(fine-3*coarse) > 1e-9 {
		t.Fatalf("fine %v, coarse %v, want three times", fine, coarse)
	}
}

// TestAFootprintAlreadyWideStartsFlat: one that starts as wide as the
// surface is tall is flat from the start, and never before it.
func TestAFootprintAlreadyWideStartsFlat(t *testing.T) {
	if from := (Footprint{Width: 5, Spread: 0.01}).FlatFrom(2.5); from != 0 {
		t.Fatalf("flat from %v, want 0", from)
	}
	if from := (Footprint{Width: 9, Spread: 0.01}).FlatFrom(2.5); from != 0 {
		t.Fatalf("flat from %v, want 0", from)
	}
}

// TestAFootprintThatStartsWideIsCounted: the width it starts with is taken
// off what the spread has to make up.
func TestAFootprintThatStartsWideIsCounted(t *testing.T) {
	from := (Footprint{Width: 1, Spread: 0.01}).FlatFrom(2.5)
	if math.Abs(from-400) > 1e-9 {
		t.Fatalf("flat from %v, want 400", from)
	}
}

// TestARayWithNoSpreadIsNeverFlat is the anti-test: a footprint that does
// not widen, as a ray with none has, never reaches the height, so nothing
// is taken flat by a caller that gave no footprint.
func TestARayWithNoSpreadIsNeverFlat(t *testing.T) {
	for _, cone := range []Footprint{{}, {Spread: -1}, {Spread: math.NaN()}} {
		if from := cone.FlatFrom(2.5); !math.IsInf(from, 1) {
			t.Fatalf("%+v is flat from %v, want never", cone, from)
		}
	}
}

// TestASurfaceWithNoHeightIsFlatEverywhere: a reach of nought is the plane.
func TestASurfaceWithNoHeightIsFlatEverywhere(t *testing.T) {
	if from := (Footprint{Spread: 0.002}).FlatFrom(0); from != 0 {
		t.Fatalf("flat from %v, want 0", from)
	}
}
