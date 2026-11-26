package branching

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestLimbRunsKeepThePiecesJoinedAndRepeatable(t *testing.T) {
	skin, err := NewSleeve([]Joint{{OfGirth: 1}, {OfGirth: .8}, {OfGirth: .5}}, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	limb, err := NewLimb(2, 1, .15, skin)
	if err != nil {
		t.Fatal(err)
	}
	limb.Wander = .2
	space, err := SpaceAlong(maths.V(1, 2, 3), maths.V(1, .2, .3))
	if err != nil {
		t.Fatal(err)
	}
	space.Scale = 1.5
	on := Place{Space: space, Size: Size{Length: measure.Metres(4), Girth: measure.Metres(.2)}}
	runs, err := limb.Runs(on)
	if err != nil {
		t.Fatal(err)
	}
	again, err := limb.Runs(on)
	if err != nil || !reflect.DeepEqual(runs, again) {
		t.Fatalf("run changed: %v", err)
	}
	if len(runs) != 2 || runs[0].Size.Length.InMetres() != 2 || runs[1].Space.Scale != 1.5 {
		t.Fatalf("unexpected runs: %+v", runs)
	}
	end := runs[0].Space.Origin.Add(runs[0].Space.Orientation.Out(maths.YAxis).Scale(3))
	if !runs[1].Space.Origin.Near(end, 1e-9) {
		t.Fatalf("pieces do not join: %v, %v", end, runs[1].Space.Origin)
	}
}

func TestLimbRejectsInvalidSleevesAndDroop(t *testing.T) {
	skin, err := NewLeaf([]Joint{{OfGirth: 1}, {}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if limb, err := NewLimb(1, 1, 0, skin); err != nil || !limb.Flat() {
		t.Fatalf("flat limb: %+v, %v", limb, err)
	}
	if _, err := NewLimb(2, 1, 0, skin); !errors.Is(err, ErrLimb) {
		t.Fatalf("accepted mismatched joints: %v", err)
	}
	if _, err := NewLimb(1, 1, math.Inf(1), skin); !errors.Is(err, ErrLimb) {
		t.Fatalf("accepted infinite droop: %v", err)
	}
}
