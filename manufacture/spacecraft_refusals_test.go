package manufacture_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// unsown uses the public return type without naming its private Go type.
// Sow(0) is a valid seed; this instead clears the seed's sown state.
func unsown[Seed any](_ Seed) Seed {
	var empty Seed
	return empty
}

func TestSpacecraftRejectsAnUnsownSeedFromThePublicAPI(test *testing.T) {
	data := readSpacecraftContract(test)
	for _, profile := range data.Profiles {
		row, err := manufacture.Spacecraft(profile.Design,
			measure.Metres(profile.Aperture))
		if err != nil {
			test.Fatal(err)
		}
		seed := unsown(manufacture.Sow(data.Seed))
		for _, detail := range manufacture.Complexities() {
			made, err := manufacture.MakeWithComplexity(*row, seed, detail)
			if err == nil {
				test.Fatal("unsown seed made a spacecraft", profile.Design, detail)
			}
			parts, placementError := made.Things(maths.Local())
			if placementError == nil || len(parts) != spacecraftNear {
				test.Fatal("refused construction exposed geometry", parts)
			}
		}
	}
}

//go:embed testdata/spacecraft-refusals.yaml
var spacecraftRefusalsRaw []byte

func TestTinySpacecraftFailsCleanlyBeforeRendering(test *testing.T) {
	var data struct {
		Design   string
		Aperture float64
		Seed     uint64
	}
	if err := yamlfile.Read("spacecraft-refusals.yaml", spacecraftRefusalsRaw, &data); err != nil {
		test.Fatal(err)
	}
	row, err := manufacture.Spacecraft(data.Design, measure.Metres(data.Aperture))
	if err != nil {
		return // An earlier checked-constructor refusal is also valid.
	}
	for _, detail := range manufacture.Complexities() {
		made, err := manufacture.MakeWithComplexity(*row,
			manufacture.Sow(data.Seed), detail)
		if err != nil {
			continue
		}
		if _, err := made.Things(maths.Local()); err == nil {
			test.Fatal("numerically collapsed spacecraft exposed a mesh", detail)
		}
	}
}
