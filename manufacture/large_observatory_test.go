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

//go:embed testdata/large-observatory.yaml
var largeContractRaw []byte

type largeContract struct {
	Kind                                  string
	Seed                                  uint64
	Diameter, Height, Aperture, Tolerance float64
	Panels                                int
	MinimumClearance                      float64 `yaml:"minimum-clearance"`
	MaximumFaces                          int     `yaml:"maximum-faces"`
	Centre, Tangent                       maths.Vec
	Probe                                 maths.Ray
	Mirrors                               []string
}

func readLargeContract(test *testing.T) largeContract {
	test.Helper()
	var data largeContract
	if err := yamlfile.Read("large-observatory.yaml", largeContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	return data
}

func TestGroundObservatoryLargeHasItsAperturePanelsAndClosedMirrors(
	test *testing.T,
) {
	data := readLargeContract(test)
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
		bounds := object.Bounds()
		if bounds.Max.X-bounds.Min.X != data.Diameter ||
			bounds.Min.Y != spacecraftNear || bounds.Max.Y != data.Height {
			test.Fatal("large observatory lost its envelope", detail, bounds)
		}
		parts, err := object.Things(maths.Local())
		if err != nil {
			test.Fatal(detail, err)
		}
		limits := meshContract{Tolerance: data.Tolerance,
			MaximumFaces: data.MaximumFaces}
		limits.assertMeshes(test, parts, bounds)
		data.checkMirrors(test, parts)
	}
}

func (data largeContract) checkMirrors(test *testing.T, parts []thing.Thing) {
	test.Helper()
	for _, name := range data.Mirrors {
		found := false
		for _, part := range parts {
			if part.Material.Name != name {
				continue
			}
			if found {
				test.Fatal("duplicate mirror role", name)
			}
			found = true
			if name == "primary mirror" {
				data.checkPrimary(test, part.Shape.(*mesh.Mesh))
			}
		}
		if !found {
			test.Fatal("missing mirror role", name)
		}
	}
}

func (data largeContract) checkPrimary(test *testing.T, polygon *mesh.Mesh) {
	test.Helper()
	if connectedPanels(polygon) != data.Panels {
		test.Fatal("wrong installed primary inventory")
	}
	outer, inner := float64(spacecraftNear), math.Inf(spacecraftPositiveSign)
	for _, point := range polygon.Points() {
		local := point.Sub(data.Centre)
		radius := math.Hypot(local.X, local.Dot(data.Tangent))
		outer, inner = max(outer, radius), min(inner, radius)
	}
	if math.Abs(outer+outer-data.Aperture) > data.Tolerance ||
		inner < data.MinimumClearance {
		test.Fatal("wrong primary extent or central clearance", outer, inner)
	}
	if _, hit := polygon.Hit(data.Probe, spacecraftNear,
		math.Inf(spacecraftPositiveSign)); hit {
		test.Fatal("primary filled its central opening")
	}
}

func TestGroundObservatoryLargeRefusesAnUnsownSeed(test *testing.T) {
	data := readLargeContract(test)
	row, err := manufacture.KindNamed(data.Kind)
	if err != nil {
		test.Fatal(err)
	}
	seed := manufacture.Sow(data.Seed)
	object, err := manufacture.MakeWithComplexity(row, unsown(seed), "quick")
	if err == nil {
		test.Fatal("unsown seed constructed a large telescope")
	}
	if parts, err := object.Things(maths.Local()); err == nil ||
		len(parts) != spacecraftNear {
		test.Fatal("refusal exposed geometry")
	}
}
