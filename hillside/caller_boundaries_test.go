package hillside

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestLegacyStandPropagatesManufactureLampAndSmokeFailures(t *testing.T) {
	given := mustParse(t)
	if err := stand(thing.NewWorld(), &tally{}, given, stock{}, "missing kind", 0, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "missing kind") {
		t.Fatalf("maker refusal: %v", err)
	}
	given.alight, given.floor = 1, 1e9
	if err := stand(thing.NewWorld(), &tally{}, given, stock{}, "spruce", 0, maths.Subspace{}); err == nil {
		t.Fatal("smoke refusal lost")
	}
	given = mustParse(t)
	given.exposure = math.NaN()
	if err := stand(thing.NewWorld(), &tally{}, given, stock{}, "spotlight", 0, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "exposure") {
		t.Fatalf("lamp refusal lost: %v", err)
	}
}

func TestBareGroundRefusesANonfiniteSpan(t *testing.T) {
	if _, err := bare(mustParse(t), math.NaN()); err == nil || !strings.Contains(err.Error(), "forest: a ground") {
		t.Fatalf("nonfinite ground span: %v", err)
	}
}
