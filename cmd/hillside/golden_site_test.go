package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// Test the shipped launch rays against actual factory enclosure geometry.
// Reading the small scene does not load its terrain or forest.
func goldenStudy(test *testing.T) (set.Set, []thing.Thing) {
	test.Helper()
	path := filepath.Join("..", "..", "doc", "golden-hour-site.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		test.Fatal(err)
	}
	row, err := manufacture.KindNamed("elt")
	if err != nil {
		test.Fatal(err)
	}
	data := launcherPlacementData(test)
	made, err := manufacture.MakeWithComplexity(row,
		manufacture.Sow(data.Seed), "quick")
	if err != nil {
		test.Fatal(err)
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	return read, parts
}

func TestGoldenHourLaunchesClearTheMatchedELTEnclosure(test *testing.T) {
	read, parts := goldenStudy(test)
	data := launcherPlacementData(test)
	if len(read.Lasers()) != data.Launchers {
		test.Fatal("study lost its four launchers")
	}
	for _, laser := range read.Lasers() {
		_, x, y, z := laser.Origin()
		var ray maths.Ray
		ray.At = maths.V(x, y, z)
		ray.Dir = skyward(laser.Pointing())
		if ray.Dir.Dot(data.Boresight) < data.MinimumDot {
			test.Fatal("beam left the near-parallel telescope boresight", laser.Name())
		}
		_, reach, _ := laser.Envelope()
		if !laser.Launcher() {
			test.Fatal("study lost its physical launcher")
		}
		for _, part := range parts {
			if part.Material.Name != "observatory enclosure" {
				continue
			}
			polygon, ok := part.Shape.(*mesh.Mesh)
			if !ok {
				test.Fatal("unexpected enclosure geometry")
			}
			if distance, hit := polygon.Hit(ray, data.Tolerance, reach); hit {
				test.Fatal("guide beam strikes the enclosure", laser.Name(), distance)
			}
		}
	}
}
