package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/launcher-placement.yaml
var launcherPlacementRaw []byte

type launcherPlacementContract struct {
	Length, Tolerance float64
	Seed, Index       uint64
	Launchers         int
	Boresight         maths.Vec
	MinimumDot        float64 `yaml:"minimum-dot"`
	Headings          []struct {
		Degrees float64
		Axis    maths.Vec
	}
}

func launcherPlacementData(test *testing.T) launcherPlacementContract {
	test.Helper()
	var data launcherPlacementContract
	if err := yamlfile.Read("launcher-placement.yaml", launcherPlacementRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	return data
}

func TestVisibleLaunchTubeEndsAtBeamAndFollowsItsAxis(test *testing.T) {
	data := launcherPlacementData(test)
	given := laserGiven(test)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	source := strings.Replace(string(raw), "    guide:\n",
		"    guide:\n      launcher: true\n", launcherUnit)
	sequenceWrite(test, given.set, []byte(source))
	world, made, _, err := staged(given)
	if err != nil {
		test.Fatal(err)
	}
	beam := actualLaser(test, world)
	found := false
	for _, object := range world.Things {
		if object.Material == nil || object.Material.Name != "optical tube" {
			continue
		}
		found = true
		pose := object.At(laserEpoch)
		exit := pose.Out(maths.YAxis.Scale(data.Length))
		if exit.Sub(beam.Point().At).Len() > data.Tolerance ||
			pose.Orientation.Y.Sub(beam.Point().Toward).Len() > data.Tolerance {
			test.Fatal("launch tube and beam disagree", exit, beam.Point())
		}
	}
	if !found || made.things == laserEpoch {
		test.Fatal("no visible launch geometry")
	}
}

func TestPlacementHeadingOverridesOnlyExplicitNamedThings(test *testing.T) {
	data := launcherPlacementData(test)
	for _, row := range data.Headings {
		source := "scene:\n  set:\n    things:\n      telescope:\n" +
			"        kind: elt\n        seed: 5\n        east: 1\n" +
			"        north: 1\n        heading: {value: HEADING, unit: deg}\n"
		angle := fmt.Sprint(row.Degrees)
		source = strings.ReplaceAll(source, "HEADING", angle)
		read, err := set.Read("placement.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		stage := staging{recipes: read}
		turn := stage.placementTurn("elt", "telescope", data.Seed, data.Index)
		direction := maths.Rotation(maths.YAxis, turn).Out(maths.ZAxis.Scale(-launcherUnit))
		if direction.Sub(row.Axis).Len() > data.Tolerance {
			test.Fatal("heading lost or reversed", row.Degrees, direction)
		}
		if stage.placementTurn("optical-observatory", "",
			data.Seed, data.Index) != laserEpoch {
			test.Fatal("unheaded optical recipe gained a seeded turn")
		}

	}
}
