package manufacture_test

import (
	_ "embed"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// spacecraftNear is defined as the ray origin bound.
const spacecraftNear = 0

// spacecraftPositiveSign is defined as the positive sign of infinity.
const spacecraftPositiveSign = 1

//go:embed testdata/spacecraft.yaml
var spacecraftContractRaw []byte

type spacecraftProbe struct {
	Material string
	Ray      maths.Ray
	Hit      bool
}
type spacecraftProfile struct {
	Design      string
	Aperture    float64
	PrimaryAxis string `json:"primary-axis" yaml:"primary-axis"`
	Probes      []spacecraftProbe
}
type spacecraftContract struct {
	Seed             uint64
	Tolerance        float64
	MaximumFaces     int `json:"maximum-faces" yaml:"maximum-faces"`
	Profiles         []spacecraftProfile
	InvalidApertures []struct {
		Value   float64
		Message string
	} `json:"invalid-apertures" yaml:"invalid-apertures"`
}

func readSpacecraftContract(test *testing.T) spacecraftContract {
	test.Helper()
	var contract spacecraftContract
	if err := yamlfile.Read("testdata/spacecraft.yaml", spacecraftContractRaw, &contract); err != nil {
		test.Fatal(err)
	}
	return contract
}

func makeSpacecraft(test *testing.T, profile spacecraftProfile,
	seed uint64, detail string,
) ([]thing.Thing, maths.Box) {
	test.Helper()
	row, err := manufacture.Spacecraft(profile.Design, measure.Metres(profile.Aperture))
	if err != nil {
		test.Fatal(err)
	}
	specimen, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(seed), detail)
	if err != nil {
		test.Fatal(err)
	}
	things, err := specimen.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	return things, specimen.Bounds()
}

func TestSpacecraftProfilesKeepClosedBoundedGeometryAtEveryDetail(test *testing.T) {
	contract := readSpacecraftContract(test)
	for _, profile := range contract.Profiles {
		var first maths.Box
		for index, detail := range manufacture.Complexities() {
			things, bounds := makeSpacecraft(test, profile, contract.Seed, detail)
			if index == spacecraftNear {
				first = bounds
			}
			if bounds != first {
				test.Fatal("physical envelope changed with detail", profile.Design, detail)
			}
			limits := meshContract{Tolerance: contract.Tolerance,
				MaximumFaces: contract.MaximumFaces}
			limits.assertMeshes(test, things, bounds)
			profile.assertPrimaryAndProbes(test, things, contract.Tolerance)
		}
	}
}

func (profile spacecraftProfile) assertPrimaryAndProbes(test *testing.T,
	things []thing.Thing, tolerance float64,
) {
	test.Helper()
	for _, object := range things {
		if object.Material.Name == "primary mirror" {
			bounds := object.Shape.Bounds()
			span := bounds.Max.X - bounds.Min.X
			if profile.PrimaryAxis == "y" {
				span = bounds.Max.Y - bounds.Min.Y
			}
			if !(math.Abs(span-profile.Aperture) <= tolerance) {
				test.Fatal("wrong primary extent", profile.Design, span, profile.Aperture)
			}
		}
	}
	for _, probe := range profile.Probes {
		found := false
		present := false
		for _, object := range things {
			if object.Material.Name != probe.Material {
				continue
			}
			present = true
			polygon, ok := object.Shape.(*mesh.Mesh)
			if !ok {
				test.Fatal("expected a spacecraft mesh")
			}
			_, hit := polygon.Hit(probe.Ray, spacecraftNear, math.Inf(spacecraftPositiveSign))
			found = found || hit
		}
		if !present || found != probe.Hit {
			test.Fatal("wrong spacecraft silhouette", profile.Design, probe, found, present)
		}
	}
}

func TestSpacecraftConstructorRefusesInvalidSizesAndUnknownDesigns(test *testing.T) {
	contract := readSpacecraftContract(test)
	for _, invalid := range contract.InvalidApertures {
		_, err := manufacture.Spacecraft("spitzer", measure.Metres(invalid.Value))
		if err == nil || !strings.Contains(err.Error(), invalid.Message) {
			test.Fatal("wrong spacecraft aperture refusal", invalid, err)
		}
	}
	if _, err := manufacture.Spacecraft("unknown", measure.Metres(contract.Profiles[spacecraftNear].Aperture)); err == nil {
		test.Fatal("accepted unknown spacecraft")
	}
}
