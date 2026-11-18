package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestScaleHumanStandsExactlyTwoMetresTall(t *testing.T) {
	row, err := manufacture.KindNamed("human-scale-2m")
	if err != nil {
		t.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(1), detail)
		if err != nil {
			t.Fatal(detail, err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(detail, err)
		}
		low, high := math.Inf(1), math.Inf(-1)
		for _, part := range parts {
			for _, point := range part.Shape.(*mesh.Mesh).Points() {
				low, high = min(low, point.Y), max(high, point.Y)
			}
			if part.Material.Metal == nil {
				t.Fatal("scale mannequin has no material optics")
			}
		}
		if math.Abs(low) > 1e-12 || math.Abs(high-2) > 1e-12 {
			t.Fatalf("%s: actual geometry spans %g to %g metres", detail, low, high)
		}
		bounds := made.Bounds()
		if bounds.Min.Y != 0 || bounds.Max.Y != 2 {
			t.Fatalf("incorrect scale bounds: %v", bounds)
		}
	}
}

// anti-test of TestScaleHumanStandsExactlyTwoMetresTall
func TestScaleHumanRefusesAnUnsownSeed(t *testing.T) {
	row, err := manufacture.KindNamed("human-scale-2m")
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(row,
		unsown(manufacture.Sow(1)), "quick")
	if err == nil || made.PartCount() != 0 {
		t.Fatal("unsown construction returned a scale human")
	}
}
