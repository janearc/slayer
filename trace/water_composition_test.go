package trace

import (
	"testing"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestClearMediumIdentityIgnoresIndependentWaterMetadata(t *testing.T) {
	firstObservation := materials.WaterObservation{Value: 31, Source: "first receipt"}
	secondObservation := materials.WaterObservation{Value: 31, Source: "second receipt"}
	firstWater, err := materials.NewWaterComposition(&firstObservation, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	secondWater, err := materials.NewWaterComposition(&secondObservation, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	base, err := thing.NewClearMedium("water", maths.RGB(.1, .2, .3), 1)
	if err != nil {
		t.Fatal(err)
	}
	first, err := base.WithWaterComposition(firstWater, "first optical source")
	if err != nil {
		t.Fatal(err)
	}
	second, err := base.WithWaterComposition(secondWater, "")
	if err != nil {
		t.Fatal(err)
	}
	if !sameClearMedium(first, second) || !sameClearMedium(first, base) {
		t.Fatal("metadata changed an otherwise identical optical medium")
	}
	different, err := thing.NewClearMedium("water", maths.RGB(.1, .2, .4), 1)
	if err != nil {
		t.Fatal(err)
	}
	if sameClearMedium(first, different) {
		t.Fatal("different optical coefficients compared equal")
	}
	if !sameClearMedium(nil, nil) || sameClearMedium(first, nil) {
		t.Fatal("nil identity changed")
	}
}
