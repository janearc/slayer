package manufacture_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

//go:embed testdata/guide-launcher.yaml
var launcherContractRaw []byte

func TestGuideLauncherKeepsAnOpenAxisAndClosedWalls(test *testing.T) {
	var data struct {
		Seed             uint64
		Tolerance        float64
		MaximumFaces     int `yaml:"maximum-faces"`
		Minimum, Maximum maths.Vec
		Ray              maths.Ray
		Far              float64
	}
	if err := yamlfile.Read("guide-launcher.yaml", launcherContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	row, err := manufacture.KindNamed("guide-star-launcher")
	if err != nil {
		test.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		made, err := manufacture.MakeWithComplexity(row,
			manufacture.Sow(data.Seed), detail)
		if err != nil {
			test.Fatal(detail, err)
		}
		bounds := made.Bounds()
		if bounds.Min != data.Minimum || bounds.Max != data.Maximum {
			test.Fatal("launcher envelope changed", bounds)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			test.Fatal(err)
		}
		probe := summitProbe{contract: summitContract{
			Tolerance: data.Tolerance, MaximumFaces: data.MaximumFaces}}
		for _, part := range parts {
			polygon, ok := part.Shape.(*mesh.Mesh)
			if !ok {
				test.Fatal("launcher is not an ordinary mesh")
			}
			probe.polygon(test, polygon, maths.Local(), bounds)
			ray := data.Ray
			if _, hit := polygon.Hit(ray, spacecraftNear, data.Far); hit {
				test.Fatal("launcher blocks its own outgoing axis", detail)
			}
		}
	}
}
