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

func TestInstrumentAssemblyBoxAtUsesThePoseLocalFloorCorner(test *testing.T) {
	assembly := &instrumentAssembly{fine: cut{around: 12}}
	pose := tiltedFrame(maths.V(3, 4, 5), 90*maths.Degree)
	corner := maths.V(1, 2, 3)
	assembly.boxAt(2, 4, 6, corner, pose, spaceStructure)
	if len(assembly.object.parts) != 1 {
		test.Fatalf("parts = %d, want 1", len(assembly.object.parts))
	}
	part := assembly.object.parts[0]
	centre := corner.Add(maths.V(1, 0, 3))
	want := maths.NewSubspace(centre, maths.Identity, spaceOne).Within(pose)
	if part.space != want {
		test.Fatalf("space = %#v, want %#v", part.space, want)
	}
	if part.of != spaceStructure || part.fine != assembly.fine {
		test.Fatalf("part material/detail = %v/%v, want %v/%v", part.of, part.fine, spaceStructure, assembly.fine)
	}
}
