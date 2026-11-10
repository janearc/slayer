//ninja:vow dojo first-error preservation belongs to the private assembly
package manufacture

import (
	_ "embed"
	"errors"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/instrument-refusal.yaml
var instrumentRefusalRaw []byte

type instrumentRefusal struct {
	From, To    maths.Vec
	BadRadius   float64 `yaml:"bad-radius"`
	ValidRadius float64 `yaml:"valid-radius"`
}

func TestInstrumentAssemblyKeepsTheFirstRefusalAndReturnsNoSpecimen(test *testing.T) {
	var data instrumentRefusal
	if err := yamlfile.Read("instrument-refusal.yaml", instrumentRefusalRaw, &data); err != nil {
		test.Fatal(err)
	}
	assembly := &instrumentAssembly{}
	assembly.rod(data.From, data.To, data.BadRadius, spaceStructure)
	// A second, different failure must not replace the bad-radius cause.
	assembly.rod(data.From, data.From, data.ValidRadius, spaceStructure)
	bounds := maths.SegmentBox(data.From, data.To, spaceZero)
	object, err := assembly.finish(bounds)
	if object != nil || !errors.Is(err, errTubeFoot) {
		test.Fatal("assembly exposed a specimen or lost its first cause", object, err)
	}
}
