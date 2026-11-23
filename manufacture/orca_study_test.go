package manufacture

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestOrcaStudyPosesRetainSkinAndFins(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	for _, stroke := range []float64{-1, 0, 1} {
		pieces, err := OrcaStudy(stroke, at)
		if err != nil {
			t.Fatalf("stroke %g: %v", stroke, err)
		}
		if len(pieces) != 6 {
			t.Fatalf("stroke %g: got %d pieces", stroke, len(pieces))
		}
	}
	for _, stroke := range []float64{-1.1, 1.1, math.NaN()} {
		if _, err := OrcaStudy(stroke, at); err == nil {
			t.Fatalf("accepted %g", stroke)
		}
	}
}

func TestOrcaStrokeMovesTailButNotHead(t *testing.T) {
	head, tail := maths.V(2, 0, 0), maths.V(-3, 0, 0)
	if orcaBend(head, 1) != orcaBend(head, -1) {
		t.Fatal("head moved with tail stroke")
	}
	if orcaBend(tail, 1).Y <= orcaBend(tail, -1).Y {
		t.Fatal("tail did not reverse vertically")
	}
}

func TestOrcaMarkingsStayInRestCoordinates(t *testing.T) {
	coat := orcaCoat{}
	eyePatch := coat.At(maths.V(2.02, .29, .55)).Luminance()
	darkBack := coat.At(maths.V(1.0, .5, .5)).Luminance()
	belly := coat.At(maths.V(.2, -.7, .1)).Luminance()
	if eyePatch < 3*darkBack || belly < 3*darkBack {
		t.Fatalf("post-ocular patch and belly should be pale: patch %g, belly %g, back %g", eyePatch, belly, darkBack)
	}
}
