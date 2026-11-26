package ferns

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/manufacture/branching"
)

func TestGrowthMakesFernAnatomyAndPreservesRefusals(t *testing.T) {
	grown, err := Growth(5, 9, 8, 13, specimen().Anatomy)
	if err != nil {
		t.Fatal(err)
	}
	if len(grown.Carries) != 1 || grown.Stem.Skins[0].Role != 8 {
		t.Fatalf("unexpected crown: %+v", grown)
	}
	frond := grown.Carries[0].What
	if len(frond.Carries) != 1 || frond.Stem.Skins[0].Role != 8 ||
		frond.Carries[0].What.Stem.Skins[0].Role != 13 || !frond.Carries[0].What.Stem.Flat() {
		t.Fatalf("unexpected frond: %+v", frond)
	}
	if _, err := Growth(0, 9, 8, 13, specimen().Anatomy); !errors.Is(err, branching.ErrRing) {
		t.Fatalf("zero fronds: %v", err)
	}
	if _, err := Growth(0, 0, 8, 13, specimen().Anatomy); !errors.Is(err, branching.ErrWhorls) {
		t.Fatalf("first refusal: %v", err)
	}
}
