package beachsurvey

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestWaterStudyUsesSuppliedExtinction(t *testing.T) {
	needsSurveys(t)
	want := maths.RGB(.31, .22, .17)
	location, _, _, err := NewWithWater("wategos", want)
	if err != nil {
		t.Fatal(err)
	}
	if err := location.Advance(0, nil); err != nil {
		t.Fatal(err)
	}
	ocean := location.World().OceanSurface
	if ocean == nil || ocean.Boundary.Material.ClearMedium.Extinction() != want {
		t.Fatal("supplied extinction was not staged on the water boundary")
	}
	nonfinite := maths.RGB(math.NaN(), .1, .1)
	if _, _, _, err := NewWithWater("wategos", nonfinite); err == nil {
		t.Fatal("accepted nonfinite water extinction")
	}
}

func TestWaterStudyRetainsIndependentObservations(t *testing.T) {
	needsSurveys(t)
	location, _, _, err := NewWithWater("aquatic-park", maths.RGB(.31, .22, .17))
	if err != nil {
		t.Fatal(err)
	}
	if err := location.Advance(0, nil); err != nil {
		t.Fatal(err)
	}
	material := location.World().OceanSurface.Boundary.Material
	composition, source, present := material.ClearMedium.WaterComposition()
	if !present || source == "" {
		t.Fatal("water composition or optical provenance was lost")
	}
	salinity, present := composition.Salinity()
	if !present || salinity.Value != 31 || salinity.Source == "" || salinity.At == "" {
		t.Fatalf("salinity observation lost: %+v", salinity)
	}
	sediment, present := composition.SuspendedSediment()
	if !present || sediment.Value != 13.9 || sediment.Qualifier == "" {
		t.Fatalf("sediment estimate lost: %+v", sediment)
	}
}
