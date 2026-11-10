package manufacture_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/yard-equipment.yaml
var yardContractRaw []byte

type yardContract struct {
	Seed         uint64
	Tolerance    float64
	MaximumFaces int `yaml:"maximum-faces"`
	Profiles     []yardProfile
}

type yardProfile struct {
	Kind             string
	Minimum, Maximum maths.Vec
}

type yardProbe struct {
	profile  yardProfile
	contract yardContract
}

// The fixture declares the physical space the scene can reserve for each kind.
func TestYardEquipmentKeepsClosedPartsInsideItsWorkingEnvelope(test *testing.T) {
	var data yardContract
	if err := yamlfile.Read("yard-equipment.yaml", yardContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	for _, profile := range data.Profiles {
		probe := yardProbe{profile: profile, contract: data}
		test.Run(profile.Kind, probe.check)
	}
}

func (probe yardProbe) check(test *testing.T) {
	test.Helper()
	row, err := manufacture.KindNamed(probe.profile.Kind)
	if err != nil {
		test.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		object, err := manufacture.MakeWithComplexity(row,
			manufacture.Sow(probe.contract.Seed), detail)
		if err != nil {
			test.Fatal(detail, err)
		}
		bounds := object.Bounds()
		if bounds.Min != probe.profile.Minimum ||
			bounds.Max != probe.profile.Maximum {
			test.Fatal("yard envelope changed", detail, bounds)
		}
		parts, err := object.Things(maths.Local())
		if err != nil {
			test.Fatal(detail, err)
		}
		limits := meshContract{Tolerance: probe.contract.Tolerance,
			MaximumFaces: probe.contract.MaximumFaces}
		limits.assertMeshes(test, parts, bounds)
	}
}
