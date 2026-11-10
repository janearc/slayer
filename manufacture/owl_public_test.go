package manufacture_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

//go:embed testdata/owl-observatory.yaml
var owlContractRaw []byte

type owlContract struct {
	Seed               uint64
	Tolerance          float64
	MaximumFaces       int `yaml:"maximum-faces"`
	Axis, Tangent      maths.Vec
	PrimarySeat        float64 `yaml:"primary-seat"`
	Primary, Secondary struct {
		Diameter, Curvature float64
		Panels              int
	}
	Backing        float64
	CorrectorRoles []string `yaml:"corrector-roles"`
	Bounds         map[string]struct{ Minimum, Maximum maths.Vec }
}

func TestOWLPublicConstructionKeepsClosedComponentsInsideItsEnvelope(test *testing.T) {
	var data owlContract
	if err := yamlfile.Read("owl-observatory.yaml", owlContractRaw, &data); err != nil {
		test.Fatal(err)
	}
	for _, name := range []string{"owl", "owl-shelter"} {
		for _, detail := range manufacture.Complexities() {
			checkOWLPublic(test, data, name, detail)
		}
	}
}

func checkOWLPublic(test *testing.T, data owlContract, name, detail string) {
	test.Helper()
	kind, err := manufacture.KindNamed(name)
	if err != nil {
		test.Fatal(err)
	}
	object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(data.Seed), detail)
	if err != nil {
		test.Fatal(name, detail, err)
	}
	// This is the public boundary that rejected the ungrouped draft.
	things, err := object.Things(maths.Local())
	if err != nil || len(things) == 0 {
		test.Fatal(name, detail, "public Things refused the instrument", err)
	}
	bounds := object.Bounds()
	expected := data.Bounds[name]
	if bounds.Min.Sub(expected.Minimum).Len() > data.Tolerance ||
		bounds.Max.Sub(expected.Maximum).Len() > data.Tolerance {
		test.Fatal("declared envelope changed", name, detail, bounds)
	}
	var stored, expanded int
	for _, part := range things {
		switch solid := part.Shape.(type) {
		case *mesh.Mesh:
			checkOWLMesh(test, solid, maths.Local(), bounds, data.Tolerance)
			stored += solid.FaceCount()
			expanded += solid.FaceCount()
		case *shape.MeshGroup:
			for _, member := range solid.Members() {
				polygon, ok := member.Mesh.(*mesh.Mesh)
				if !ok {
					test.Fatal("OWL introduced a non-mesh group member")
				}
				checkOWLMesh(test, polygon, member.Space, bounds, data.Tolerance)
			}
			stored += solid.StoredFaces()
			expanded += solid.FaceCount()
		default:
			test.Fatal("OWL introduced a renderer-specific shape", name, detail)
		}
	}
	if stored > data.MaximumFaces || (name == "owl" && stored >= expanded) {
		test.Fatal("OWL components were expanded or exceeded their retained budget", stored, expanded)
	}
}

func checkOWLMesh(test *testing.T, polygon *mesh.Mesh,
	pose maths.Subspace, bounds maths.Box, tolerance float64) {
	test.Helper()
	if !polygon.Closed() {
		test.Fatal("OWL contains an open component")
	}
	for _, vertex := range polygon.Points() {
		point := pose.Out(vertex)
		if point.X < bounds.Min.X-tolerance || point.X > bounds.Max.X+tolerance ||
			point.Y < bounds.Min.Y-tolerance || point.Y > bounds.Max.Y+tolerance ||
			point.Z < bounds.Min.Z-tolerance || point.Z > bounds.Max.Z+tolerance {
			test.Fatal("actual OWL vertex escapes the declared envelope", point, bounds)
		}
	}
}
