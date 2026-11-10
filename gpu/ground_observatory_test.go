//go:build darwin

package gpu_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/ground-observatory.yaml
var groundGPURaw []byte

type groundGPUCheck struct {
	plan instrumentGPUPlan
	view instrumentGPUView
	lens *film.Pinhole
}

func TestGroundObservatoryMeshesAndMaterialsAgreeOnMetal(test *testing.T) {
	var plan instrumentGPUPlan
	if err := yamlfile.Read("ground-observatory.yaml", groundGPURaw,
		&plan); err != nil {
		test.Fatal(err)
	}
	lens, err := film.PinholeOf(plan.Wide * maths.Degree)
	if err != nil {
		test.Fatal(err)
	}
	for _, view := range plan.Views {
		name := view.Design
		if view.Cutaway {
			name += " cutaway"
		}
		check := groundGPUCheck{plan: plan, view: view, lens: lens}
		test.Run(name, check.compare)
	}
}

// compare lets a merge check select one instrument without rendering its peers.
func (check groundGPUCheck) compare(test *testing.T) {
	eye, err := maths.View(check.view.Eye, check.view.Target, maths.YAxis)
	if err != nil {
		test.Fatal(err)
	}
	world := check.view.ground(test, check.plan.Seed)
	check.plan.compare(test, test.Name(), world, eye, check.lens)
}

// ground sends the public factory's ordinary meshes through both GPU paths.
func (view instrumentGPUView) ground(test *testing.T,
	seed uint64,
) *thing.World {
	test.Helper()
	row, err := manufacture.KindNamed(view.Design)
	if err != nil {
		test.Fatal(err)
	}
	object, err := manufacture.MakeWithComplexity(row,
		manufacture.Sow(seed), "quick")
	if err != nil {
		test.Fatal(err)
	}
	parts, err := object.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	for _, part := range parts {
		if view.Cutaway && part.Material.Name == "observatory enclosure" {
			continue
		}
		world.Add(part)
	}
	world.Lamp(view.Source)
	return world
}
