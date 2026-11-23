package materials

import (
	"math"
	"testing"
)

func TestWaterCompositionCopiesIndependentObservations(t *testing.T) {
	salt := WaterObservation{Value: 12, Source: "salinity sample"}
	silt := WaterObservation{Value: 30, Source: "sediment sample"}
	water, err := NewWaterComposition(&salt, &silt, nil)
	if err != nil {
		t.Fatal(err)
	}
	salt.Value = 2
	silt.Source = "changed"
	gotSalt, hasSalt := water.Salinity()
	gotSilt, hasSilt := water.SuspendedSediment()
	if !hasSalt || !hasSilt || gotSalt.Value != 12 || gotSalt.Source != "salinity sample" ||
		gotSilt.Value != 30 || gotSilt.Source != "sediment sample" {
		t.Fatalf("water observations changed: %+v %+v", gotSalt, gotSilt)
	}
	var missing *WaterComposition
	if _, has := missing.Salinity(); has {
		t.Fatal("nil composition reports salinity")
	}
}

func TestWaterCompositionRejectsInvalidObservations(t *testing.T) {
	for _, sample := range []WaterObservation{
		{Value: -1, Source: "sample"},
		{Value: math.NaN(), Source: "sample"},
		{Value: math.Inf(1), Source: "sample"},
		{Value: 1, Source: "  "},
	} {
		if _, err := NewWaterComposition(&sample, nil, nil); err == nil {
			t.Errorf("accepted observation %+v", sample)
		}
	}
	if _, err := NewWaterComposition(nil, nil, nil); err == nil {
		t.Error("accepted empty composition")
	}
}
