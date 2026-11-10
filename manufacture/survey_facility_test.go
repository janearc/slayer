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

//go:embed testdata/survey-facility.yaml
var facilityContractRaw []byte

type facilityContract struct {
	Kind             string
	Seed             uint64
	Tolerance        float64
	MaximumFaces     int `yaml:"maximum-faces"`
	Minimum, Maximum maths.Vec
	Ray              maths.Ray
	Roofs            []facilityRoof
	Clearance        facilityRoof
}

type facilityRoof struct{ East, South, Height float64 }

func TestSurveyFacilityKeepsItsServiceWingAndSeparation(test *testing.T) {
	var data facilityContract
	if err := yamlfile.Read("survey-facility.yaml", facilityContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		data.check(test, detail)
	}
}

func (data facilityContract) check(test *testing.T, detail string) {
	test.Helper()
	row, err := manufacture.KindNamed(data.Kind)
	if err != nil {
		test.Fatal(err)
	}
	object, err := manufacture.MakeWithComplexity(row,
		manufacture.Sow(data.Seed), detail)
	if err != nil {
		test.Fatal(err)
	}
	bounds := object.Bounds()
	if bounds.Min != data.Minimum || bounds.Max != data.Maximum {
		test.Fatal("wrong facility envelope", detail, bounds)
	}
	parts, err := object.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	limits := meshContract{Tolerance: data.Tolerance,
		MaximumFaces: data.MaximumFaces}
	limits.assertMeshes(test, parts, bounds)
	data.checkRoofs(test, parts, detail)
}

// checkRoofs requires actual tier surfaces and an open separation joint.
func (data facilityContract) checkRoofs(test *testing.T,
	parts []thing.Thing, detail string,
) {
	test.Helper()
	for _, roof := range data.Roofs {
		distance := data.distance(parts, roof)
		if math.Abs(data.Ray.At.Y-distance-roof.Height) > data.Tolerance {
			test.Fatal("missing service roof tier", detail, roof, distance)
		}
	}
	if !math.IsInf(data.distance(parts, data.Clearance), spacecraftPositiveSign) {
		test.Fatal("service building closed the enclosure gap", detail)
	}
}

// distance probes actual surfaces, so a declared empty envelope cannot pass.
func (data facilityContract) distance(parts []thing.Thing,
	probe facilityRoof,
) float64 {
	ray := data.Ray
	ray.At.X, ray.At.Z = probe.East, probe.South
	closest := math.Inf(spacecraftPositiveSign)
	for _, part := range parts {
		distance, found := part.Shape.(*mesh.Mesh).Hit(ray, spacecraftNear, closest)
		if found {
			closest = distance
		}
	}
	return closest
}
