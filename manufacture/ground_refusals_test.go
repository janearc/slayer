//ninja:vow dojo the private maker must preserve its assembly refusal
package manufacture

import (
	_ "embed"
	"errors"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/ground-refusals.yaml
var groundRefusalRaw []byte

func TestGroundObservatoryMakerReturnsItsAssemblyFailure(test *testing.T) {
	var data struct {
		Diameter, Height, Aperture float64
		Seed                       uint64
		Detail                     string
	}
	if err := yamlfile.Read("ground-refusals.yaml", groundRefusalRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	row := groundObservatory{diameter: data.Diameter, height: data.Height,
		aperture: data.Aperture}
	object, err := row.madeAt(Sow(data.Seed), data.Detail)
	if !errors.Is(err, errTubeFoot) {
		test.Fatal("lost the assembly's radius refusal", err)
	}
	parts, placementError := object.Things(maths.Local())
	if placementError == nil || len(parts) != spaceZero {
		test.Fatal("failed assembly exposed geometry", parts, placementError)
	}
}
