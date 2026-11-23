//go:build darwin

package gpu

import (
	"testing"
	"unsafe"
)

// firstByte is where a slice's bytes begin.
func firstByte(held []byte) *byte { return unsafe.SliceData(held) }

// TestTheInputsAreWhereTheReleaseLooksForThem says that the places the
// release lets go are the scene's packed faces, bends and boxes, the
// very bytes and no others.
func TestTheInputsAreWhereTheReleaseLooksForThem(t *testing.T) {
	encoded := pooledBy(t, mixedTrees(), false)
	inputs := encoded.inputs()
	for name, place := range map[string]struct {
		at   int
		held []byte
	}{
		"triangles": {trianglesAt, encoded.triangles.bytes},
		"nodes":     {nodesAt, encoded.nodes.bytes},
		"bends":     {bendsAt, encoded.bends.bytes},
	} {
		if len(place.held) == 0 {
			t.Fatalf("%s: the fixture packed none", name)
		}
		if firstByte(inputs[place.at]) != firstByte(place.held) {
			t.Errorf("%s: input %d is not the scene's bytes", name, place.at)
		}
	}
}

// TestReleasingLetsGoOfTheFacesAndNothingElse says that the release
// empties the three inputs and the scene's hold on them, and leaves the
// header and every other input as they were.
func TestReleasingLetsGoOfTheFacesAndNothingElse(t *testing.T) {
	encoded := pooledBy(t, mixedTrees(), false)
	encoded.header.bytes = []byte{1, 2, 3, 4}
	inputs := encoded.inputs()
	encoded.releasePacked(inputs)()
	for _, at := range []int{trianglesAt, nodesAt, bendsAt} {
		if inputs[at] != nil {
			t.Errorf("input %d was not let go", at)
		}
	}
	if encoded.triangles.bytes != nil || encoded.nodes.bytes != nil ||
		encoded.bends.bytes != nil {
		t.Error("the scene still holds packed geometry")
	}
	if len(inputs[headerAt]) != 4 || len(encoded.header.bytes) != 4 {
		t.Error("the header was let go with the faces")
	}
}
