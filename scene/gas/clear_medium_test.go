package gas

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestVisibleTableClearMediumSnapshotsRGBExtinction(t *testing.T) {
	samples := []VisibleSample{
		{Nanometres: 465, Absorb: 0.4, Scatter: 0.1},
		{Nanometres: 550, Absorb: 0.2, Scatter: 0.3},
		{Nanometres: 610, Absorb: 0.1, Scatter: 0.9},
	}
	table, err := NewVisibleTable(samples, 0)
	if err != nil {
		t.Fatal(err)
	}
	medium, err := table.ClearMedium("test water", 2)
	if err != nil {
		t.Fatal(err)
	}
	samples[0].Absorb = 100
	owned := table.Samples()
	owned[0].Absorb = 200
	if got, want := medium.Extinction(), maths.RGB(1, 0.5, 0.5); got != want {
		t.Fatalf("snapshotted extinction = %v, want %v", got, want)
	}
	if medium.MetresPerUnit() != 2 || medium.Name() != "test water" {
		t.Fatalf("clear-medium identity or units changed: %q / %g",
			medium.Name(), medium.MetresPerUnit())
	}
}

func TestVisibleTableClearMediumRefusesMissingRGBRange(t *testing.T) {
	table, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 500, Absorb: 0.2},
		{Nanometres: 600, Absorb: 0.2},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.ClearMedium("partial", 1); err == nil {
		t.Fatal("table without all three RGB sample wavelengths was accepted")
	}
}
