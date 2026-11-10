package main

import (
	"github.com/janearc/puzzlemouth/yamlfile"
	"os"
	"strings"
	"testing"
)

// TestSharedLaserPointingReachesTheRetainedBeam holds the real scene adapter.
func TestSharedLaserPointingReachesTheRetainedBeam(test *testing.T) {
	var want laserAdapterContract
	if err := yamlfile.Read("laser-contract.yaml", laserAdapterRaw,
		&want); err != nil {
		test.Fatal(err)
	}
	given := laserGiven(test)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	old := "pointing: {azimuth: 30, altitude: 70}"
	pointing := "pointing: shared science"
	text := strings.ReplaceAll(string(raw), old, pointing)
	if text == string(raw) {
		test.Fatal("shared-pointing fixture did not apply")
	}
	text += "\n  laser-pointings:\n" +
		"    shared science: {azimuth: 30, altitude: 70}\n"
	sequenceWrite(test, given.set, []byte(text))
	world, _, retained, err := staged(given)
	if err != nil {
		test.Fatal(err)
	}
	beam := actualLaser(test, world)
	if beam.Point().Toward.Sub(want.Direction).Len() > want.Tolerance {
		test.Fatal("shared direction changed in scene assembly",
			beam.Point().Toward, want.Direction)
	}
	later, _, err := retained.live.frameAt(laserEpoch)
	if err != nil {
		test.Fatal(err)
	}
	if actualLaser(test, later) != beam {
		test.Fatal("common static direction lost its retained beam")
	}
}
