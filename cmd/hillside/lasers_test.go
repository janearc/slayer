package main

import (
	_ "embed"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"os"
	"strings"
	"testing"
	"time"
)

// laserEpoch is defined as elapsed time zero for a static object pose.
const laserEpoch = 0

//go:embed testdata/laser-block.yaml
var laserBlock string

//go:embed testdata/laser-contract.yaml
var laserAdapterRaw []byte

type laserAdapterContract struct {
	Offset     maths.Vec `json:"offset"`
	Direction  maths.Vec `json:"direction"`
	Radius     float64   `json:"radius"`
	Reach      float64   `json:"reach"`
	Probe      float64   `json:"probe"`
	Wavelength float64   `json:"wavelength"`
	Strength   float64   `json:"strength"`
	Tolerance  float64   `json:"tolerance"`
	Later      float64   `json:"later"`
}

func laserGiven(test *testing.T) asked {
	test.Helper()
	given := mustParse(test)
	given.set = onFixture(test, "  terrain:", laserBlock+"  terrain:")
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "alight: true", "alight: false")
	raw = []byte(text)
	sequenceWrite(test, given.set, raw)
	return given
}

func actualLaser(test *testing.T, world *thing.World) *emit.Beam {
	test.Helper()
	var beam *emit.Beam
	for _, lamp := range world.Lamps {
		if candidate, ok := lamp.(*emit.Beam); ok {
			if beam != nil {
				test.Fatal("duplicate launch")
			}
			beam = candidate
		}
	}
	if beam == nil {
		test.Fatal("no actual beam in the scene")
	}
	return beam
}

func TestLaserYAMLReachesTheSceneAndRetainsItsLaunch(test *testing.T) {
	var want laserAdapterContract
	if err := yamlfile.Read("testdata/laser-contract.yaml", laserAdapterRaw, &want); err != nil {
		test.Fatal(err)
	}
	world, made, given, err := staged(laserGiven(test))
	if err != nil {
		test.Fatal(err)
	}
	beam := actualLaser(test, world)
	if beam.Point().Toward.Sub(want.Direction).Len() > want.Tolerance || beam.Radius().InMetres() != want.Radius || beam.Reach().InMetres() != want.Reach {
		test.Fatal("lost pointing or dimensions", beam)
	}
	want.assertOrigin(test, world, beam)
	if made.lamps != len(world.Lamps) {
		test.Fatal("beam missing from lamp tally")
	}
	at := beam.Point().At.Add(want.Direction.Scale(want.Probe))
	var band maths.Band
	band.Nanometres = want.Wavelength
	if beam.Lands(at, band) <= laserEpoch || beam.Point().Strength.Luminance() <= laserEpoch {
		test.Fatal("lost actual sodium spectrum")
	}
	elapsed := time.Duration(want.Later * float64(time.Second))
	later, _, err := given.live.frameAt(elapsed)
	if err != nil {
		test.Fatal(err)
	}
	if actualLaser(test, later) != beam {
		test.Fatal("static launch was rebuilt or lost between frames")
	}
}

func (want laserAdapterContract) assertOrigin(test *testing.T, world *thing.World, beam *emit.Beam) {
	test.Helper()
	found := false
	for _, object := range world.Things {
		if object.Material != nil && object.Material.Name == "granite" {
			found = true
			origin := object.At(laserEpoch).Origin.Add(want.Offset)
			if beam.Point().At != origin {
				test.Fatal("laser did not use placed object plus world offset", beam.Point().At, origin)
			}
		}
	}
	if !found {
		test.Fatal("missing actual named-origin geometry")
	}
}

func TestLaserRefusesAnObjectThatWasNotPlaced(test *testing.T) {
	given := laserGiven(test)
	read := readBoundarySet(test, given.set)
	stage := staging{world: thing.NewWorld()}
	err := stage.lasers(read)
	if err == nil || !strings.Contains(err.Error(), "guide") || !strings.Contains(err.Error(), "the stone") || !strings.Contains(err.Error(), "has no placed origin") {
		test.Fatal("unplaced launch was not explained", err)
	}
}
