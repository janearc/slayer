package main

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/spacecraft-block.yaml
var spacecraftSceneBlock string

//go:embed testdata/spacecraft-placement.yaml
var spacecraftPlacementRaw []byte

type spacecraftPlacementContract struct {
	Replacement                   string
	Lift, Epoch, Later, Tolerance float64
	Aperture                      float64
	Delta                         maths.Vec
}

func spacecraftGiven(test *testing.T, data spacecraftPlacementContract,
	lift float64,
) asked {
	test.Helper()
	given := mustParse(test)
	given.set = onFixture(test, "  terrain:", spacecraftSceneBlock+"  terrain:")
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	replacement := strings.ReplaceAll(data.Replacement, "$LIFT", fmt.Sprint(lift))
	text := strings.ReplaceAll(string(raw), "kind: spruce", replacement)
	text = strings.ReplaceAll(text, "alight: true", "alight: false")
	text = strings.ReplaceAll(text, "object: the stone", "object: the tree")
	text = strings.ReplaceAll(text, "standoff_x: 0", "standoff_x: 3")
	sequenceWrite(test, given.set, []byte(text))
	return given
}

func spacecraftPrimary(test *testing.T, world *thing.World) *thing.Thing {
	test.Helper()
	for index := range world.Things {
		part := world.Things[index]
		if part.Material != nil && part.Material.Name == "primary mirror" {
			return part
		}
	}
	test.Fatal("spacecraft recipe did not produce a primary")
	return nil
}

func TestSpacecraftYAMLHeightReachesGeometryCameraAndRetainedFrames(test *testing.T) {
	var data spacecraftPlacementContract
	if err := yamlfile.Read("testdata/spacecraft-placement.yaml", spacecraftPlacementRaw, &data); err != nil {
		test.Fatal(err)
	}
	low, _, lowGiven, err := staged(spacecraftGiven(test, data, data.Epoch))
	if err != nil {
		test.Fatal(err)
	}
	high, _, given, err := staged(spacecraftGiven(test, data, data.Lift))
	if err != nil {
		test.Fatal(err)
	}
	oldPrimary := spacecraftPrimary(test, low)
	primary := spacecraftPrimary(test, high)
	epoch := time.Duration(data.Epoch * float64(time.Second))
	lowAt, highAt := oldPrimary.Motion.At(epoch), primary.Motion.At(epoch)
	delta := highAt.Origin.Sub(lowAt.Origin)
	if delta.Sub(data.Delta).Len() > data.Tolerance {
		test.Fatal("YAML lift did not reach the actual mesh", delta)
	}
	if math.Abs(given.eye.Origin.Y-lowGiven.eye.Origin.Y-data.Lift) > data.Tolerance {
		test.Fatal("relative camera did not follow the lifted object", lowGiven.eye, given.eye)
	}
	bounds := primary.Shape.Bounds()
	if math.Abs(bounds.Max.X-bounds.Min.X-data.Aperture) > data.Tolerance {
		test.Fatal("lost authored spacecraft aperture", bounds)
	}
	elapsed := time.Duration(data.Later * float64(time.Second))
	later, _, err := given.live.frameAt(elapsed)
	if err != nil {
		test.Fatal(err)
	}
	if spacecraftPrimary(test, later).Shape != primary.Shape {
		test.Fatal("static spacecraft geometry was rebuilt")
	}
}
