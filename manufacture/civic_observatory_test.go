package manufacture_test

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/civic-observatory.yaml
var civicContractRaw []byte

type civicContract struct {
	Kind               string
	Seed               uint64
	Minimum, Maximum   maths.Vec
	Tolerance          float64
	MaximumFaces       int       `yaml:"maximum-faces"`
	ObjectiveCentre    maths.Vec `yaml:"objective-centre"`
	ObjectiveAxis      maths.Vec `yaml:"objective-axis"`
	ObjectiveDiameter  float64   `yaml:"objective-diameter"`
	ObjectiveThickness float64   `yaml:"objective-thickness"`
	Roofs              []struct {
		Ray      maths.Ray
		Distance float64
	}
}

func TestCivicObservatoryKeepsClosedRoofsAndNominalObjective(test *testing.T) {
	var data civicContract
	if err := yamlfile.Read("civic-observatory.yaml", civicContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	row, err := manufacture.KindNamed(data.Kind)
	if err != nil {
		test.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		object, err := manufacture.MakeWithComplexity(row,
			manufacture.Sow(data.Seed), detail)
		if err != nil {
			test.Fatal(detail, err)
		}
		bounds := object.Bounds()
		if bounds.Min != data.Minimum || bounds.Max != data.Maximum {
			test.Fatal("civic envelope changed", detail, bounds)
		}
		parts, err := object.Things(maths.Local())
		if err != nil {
			test.Fatal(detail, err)
		}
		limits := meshContract{Tolerance: data.Tolerance,
			MaximumFaces: data.MaximumFaces}
		limits.assertMeshes(test, parts, bounds)
		data.checkRoofs(test, parts)
		data.checkObjective(test, parts)
	}
}

func (data civicContract) checkRoofs(test *testing.T, parts []thing.Thing) {
	test.Helper()
	found := false
	for _, part := range parts {
		if part.Material.Name != "copper dome proxy" {
			continue
		}
		found = true
		for _, probe := range data.Roofs {
			hit, ok := part.Shape.(*mesh.Mesh).Hit(probe.Ray,
				spacecraftNear,
				math.Inf(spacecraftPositiveSign))
			error := math.Abs(hit - probe.Distance)
			if !ok || error > data.Tolerance {
				test.Fatal("lost a civic roof or its height",
					probe.Ray.At, hit, ok)
			}
		}
	}
	if !found {
		test.Fatal("missing civic roofs")
	}
}

func (data civicContract) checkObjective(test *testing.T, parts []thing.Thing) {
	test.Helper()
	found := false
	for _, part := range parts {
		if part.Material.Name != "Zeiss objective proxy" {
			continue
		}
		found = true
		polygon := part.Shape.(*mesh.Mesh)
		radius := float64(spacecraftNear)
		near := math.Inf(spacecraftPositiveSign)
		far := math.Inf(-spacecraftPositiveSign)
		for _, point := range polygon.Points() {
			offset := point.Sub(data.ObjectiveCentre)
			along := offset.Dot(data.ObjectiveAxis)
			near, far = min(near, along), max(far, along)
			cross := offset.Sub(data.ObjectiveAxis.Scale(along))
			radius = max(radius, cross.Len())
		}
		diameterError := math.Abs(radius + radius - data.ObjectiveDiameter)
		if diameterError > data.Tolerance ||
			math.Abs(near) > data.Tolerance ||
			math.Abs(far-data.ObjectiveThickness) > data.Tolerance {
			test.Fatal("wrong objective aperture or pose",
				radius, near, far)
		}
	}
	if !found {
		test.Fatal("missing nominal Zeiss objective")
	}
}
