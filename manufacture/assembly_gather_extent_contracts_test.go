package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

func TestGatherRetainsPrescribedPhysicalExtentBeyondItsDisplayHull(t *testing.T) {
	display := maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}
	physical := maths.Box{Min: maths.V(-2, -1, -2), Max: maths.V(2, 3, 2)}
	material := &thing.Material{Name: "coarse hull", Colour: maths.RGB(.2, .4, .6)}
	one := good{parts: []part{{of: body, shape: crownShape{display}}}, dressed: looks{body: material}, extent: &physical}
	// A coarse representation can retain the larger specimen's physical box.
	// Gathering one or several such members must retain that separate extent.
	for _, members := range [][]good{{one}, {one, one}} {
		combined, err := gather(members...)
		if err != nil {
			t.Fatal(err)
		}
		if combined.Bounds() != physical {
			t.Fatalf("gather lost prescribed extent: %v want %v", combined.Bounds(), physical)
		}
		things, err := combined.Things(maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		if len(things) != len(members) {
			t.Fatal("gather lost a represented material")
		}
		for _, thing := range things {
			if thing.Material != material {
				t.Fatal("gather changed its member's material")
			}
		}
	}
}
