package manufactory_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// Every kind is found by its name, and a name that is no kind is
// refused with the names there are.
func TestEveryKindIsFoundByItsName(t *testing.T) {
	names := manufacture.KindNames()
	if len(names) == 0 {
		t.Fatal("the manufactory has no kinds")
	}
	_, err := manufacture.KindNamed("larch")
	if err == nil {
		t.Fatal("a larch is a kind, and there is no larch")
	}
	for _, name := range names {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal of a larch does not name the %s", name)
		}
	}
}

// finite is whether a point is somewhere.
func finite(point maths.Vec) bool {
	for _, part := range []float64{point.X, point.Y, point.Z} {
		if math.IsNaN(part) || math.IsInf(part, 0) {
			return false
		}
	}
	return true
}

// The reading of a skin can fail: a fern's fronds, open blades, are
// not closed, and a boulder wound the other way faces in.
func TestTheSkinReaderSeesAnOpenOrInsideOutSkin(t *testing.T) {
	fronds := made(t, "fern", 1)[1]
	if fronds.unclosed() == nil {
		t.Error("a fern's fronds read as closed")
	}
	stone := made(t, "boulder", 1)[0]
	if held := stone.reversed().volume(); held >= 0 {
		t.Errorf("a boulder wound the other way holds %g", held)
	}
}
