package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files the sediment tests read their plumes and probe points from.
const (
	sedimentPlumesFile = "testdata/sediment-plumes.csv"
	sedimentProbesFile = "testdata/sediment-probes.csv"
)

// sedimentSilentProbes are the probes at which a plume has no density: the
// two ends along its length and the faces above, below and beside it.
var sedimentSilentProbes = []string{
	"upstream-end", "downstream-end", "surface", "floor", "edge",
	"beyond-edge",
}

// sedimentConfig is the plume configuration in the plumes file called name.
func sedimentConfig(t *testing.T, name string) SedimentPlumeConfig {
	t.Helper()
	row := readSheet(t, sedimentPlumesFile).row(t, sedimentPlumesFile, name)
	return SedimentPlumeConfig{
		Mouth: maths.V(cell(t, row, "mouth_x"), cell(t, row, "mouth_y"),
			cell(t, row, "mouth_z")),
		DirectionX:    cell(t, row, "direction_x"),
		DirectionZ:    cell(t, row, "direction_z"),
		Length:        cell(t, row, "length"),
		Depth:         cell(t, row, "depth"),
		MouthWidth:    cell(t, row, "mouth_width"),
		OffshoreWidth: cell(t, row, "offshore_width"),
		Extinction:    cell(t, row, "extinction"),
	}
}

// sedimentProbe is the point in the probes file called name.
func sedimentProbe(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, sedimentProbesFile).row(t, sedimentProbesFile, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// TestSedimentPlumeFadesAtItsBoundaries holds that a plume peaks at its
// midpoint and has no density at its ends or faces.
func TestSedimentPlumeFadesAtItsBoundaries(t *testing.T) {
	config := sedimentConfig(t, "fade")
	plume, err := NewSedimentPlume(config)
	if err != nil {
		t.Fatal(err)
	}
	peak := picked(t, "sediment-peak-extinction")
	tolerance := picked(t, "sediment-tolerance")
	got := plume.Density(sedimentProbe(t, "peak"))
	if math.Abs(got-peak) > tolerance {
		t.Fatalf("peak extinction %g, want .2", got)
	}
	for _, name := range sedimentSilentProbes {
		at := sedimentProbe(t, name)
		if got := plume.Density(at); got != nought {
			t.Errorf("density at %v = %g, want zero", at, got)
		}
	}
	if plume.Density(sedimentProbe(t, "inside")) <= nought {
		t.Fatal("expected suspended sediment inside the plume")
	}
}

// sedimentNoDirection takes the direction from a plume configuration.
func sedimentNoDirection(config *SedimentPlumeConfig) {
	config.DirectionX = nought
}

// sedimentNoDepth takes the depth from a plume configuration.
func sedimentNoDepth(config *SedimentPlumeConfig) {
	config.Depth = nought
}

// sedimentNegativeExtinction gives a plume configuration an extinction
// below nothing.
func sedimentNegativeExtinction(config *SedimentPlumeConfig) {
	config.Extinction = -unity
}

// sedimentInfiniteLength gives a plume configuration an endless length.
func sedimentInfiniteLength(config *SedimentPlumeConfig) {
	config.Length = math.Inf(unity)
}

// TestSedimentPlumeRejectsInvalidParameters holds that a plume is refused
// for no direction, no depth, a negative extinction or an infinite length.
func TestSedimentPlumeRejectsInvalidParameters(t *testing.T) {
	config := sedimentConfig(t, "valid")
	for _, invalid := range []func(*SedimentPlumeConfig){
		sedimentNoDirection, sedimentNoDepth, sedimentNegativeExtinction,
		sedimentInfiniteLength,
	} {
		candidate := config
		invalid(&candidate)
		if _, err := NewSedimentPlume(candidate); err == nil {
			t.Errorf("accepted invalid configuration %+v", candidate)
		}
	}
}

// TestSedimentPlumeCanJoinClearWater holds that a plume wrapped in a clear
// cloud keeps its density and its midpoint inside its bounds.
func TestSedimentPlumeCanJoinClearWater(t *testing.T) {
	plume, err := NewSedimentPlume(sedimentConfig(t, "join"))
	if err != nil {
		t.Fatal(err)
	}
	asymmetry := picked(t, "sediment-henyey-greenstein-asymmetry")
	wrapped, err := NewClearCloudWithHenyeyGreenstein(plume, asymmetry)
	if err != nil {
		t.Fatal(err)
	}
	point := sedimentProbe(t, "join-midpoint")
	joined := picked(t, "sediment-joined-extinction")
	tolerance := picked(t, "sediment-tolerance")
	if got := wrapped.Density(point); math.Abs(got-joined) > tolerance {
		t.Fatalf("wrapped density = %g, want .3", got)
	}
	box := wrapped.Bounds()
	if point.X < box.Min.X || point.X > box.Max.X || point.Y < box.Min.Y ||
		point.Y > box.Max.Y || point.Z < box.Min.Z || point.Z > box.Max.Z {
		t.Fatalf("bounds %v exclude plume midpoint %v", box, point)
	}
}
