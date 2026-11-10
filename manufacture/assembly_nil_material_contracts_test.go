package manufacture

import (
	"errors"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

func TestAssemblyRefusesBothMissingAndExplicitlyNilMaterials(t *testing.T) {
	shape := crownShape{maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}}
	material := &thing.Material{Name: "control", Colour: maths.Grey(.5)}
	positive := good{parts: []part{{of: body, shape: shape}}, dressed: looks{body: material}}
	if _, err := positive.Things(maths.Subspace{}); err != nil {
		t.Fatal(err)
	}
	if _, err := gather(positive); err != nil {
		t.Fatal(err)
	}
	for _, dressed := range []looks{nil, {}, {body: nil}} {
		bad := positive
		bad.dressed = dressed
		if _, err := bad.Things(maths.Subspace{}); !errors.Is(err, errLook) {
			t.Fatalf("placed object accepted missing/nil material: %v", err)
		}
		if _, err := gather(bad); !errors.Is(err, errLook) {
			t.Fatalf("compound object accepted missing/nil material: %v", err)
		}
	}
}
