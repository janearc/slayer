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

//go:embed testdata/survey-observatory.yaml
var surveyContractRaw []byte

type surveySurface struct {
	Material string
	Span     float64
	Hit      bool
}

type surveyContract struct {
	Kind      string
	Seed      uint64
	Tolerance float64
	Tangent   maths.Vec
	Ray       maths.Ray
	Surfaces  []surveySurface
}

func TestGroundObservatorySurveyHasThreeMirrorsAndOneBacking(test *testing.T) {
	var data surveyContract
	if err := yamlfile.Read("survey-observatory.yaml", surveyContractRaw,
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
			test.Fatal(err)
		}
		parts, err := object.Things(maths.Local())
		if err != nil {
			test.Fatal(err)
		}
		for _, surface := range data.Surfaces {
			surface.check(test, data, parts)
		}
	}
}

func (surface surveySurface) check(test *testing.T, data surveyContract,
	parts []thing.Thing,
) {
	test.Helper()
	for _, part := range parts {
		if part.Material.Name != surface.Material {
			continue
		}
		polygon := part.Shape.(*mesh.Mesh)
		low := math.Inf(spacecraftPositiveSign)
		high := math.Inf(-spacecraftPositiveSign)
		for _, point := range polygon.Points() {
			along := point.Dot(data.Tangent)
			low, high = min(low, along), max(high, along)
		}
		if math.Abs(high-low-surface.Span) > data.Tolerance ||
			connectedPanels(polygon) != spacecraftPositiveSign {
			test.Fatal("wrong mirror or housing extent", surface, high-low)
		}
		_, hit := polygon.Hit(data.Ray, spacecraftNear,
			math.Inf(spacecraftPositiveSign))
		if hit != surface.Hit {
			test.Fatal("wrong axial opening", surface, hit)
		}
		return
	}
	test.Fatal("missing survey component", surface.Material)
}
