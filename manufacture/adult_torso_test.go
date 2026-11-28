package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"testing"
)

func TestAdultTorsoProducesOneClosedNativePart(t *testing.T) {
	spec := primates.Adult()
	kind, err := AdultTorso(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Make(kind, seed{}); err == nil {
		t.Fatal("missing seed")
	}
	good, err := Make(kind, Sow(67))
	if err != nil {
		t.Fatal(err)
	}
	p, err := good.Things(maths.Local())
	if err != nil || len(p) != 1 || !p[0].Shape.(*mesh.Mesh).Closed() {
		t.Fatal(len(p), err)
	}
	spec.Left.Width = 0
	if _, err = AdultTorso(spec); err == nil {
		t.Fatal("invalid dimensions")
	}
}
