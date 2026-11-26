package branching

import (
	"errors"
	"math"
	"testing"
)

func TestSleevesCheckTheirJointsAndRetainTheirRole(t *testing.T) {
	joints := []Joint{{OfGirth: 1}, {OfGirth: .2}}
	skin, err := NewSleeve(joints, 6, 4)
	if err != nil || skin.Sides != 6 || skin.Role != 4 || skin.Flat {
		t.Fatalf("round sleeve: %+v, %v", skin, err)
	}
	blade, err := NewLeaf(joints, 5)
	if err != nil || !blade.Flat || blade.Role != 5 {
		t.Fatalf("blade: %+v, %v", blade, err)
	}
	if got := (Joint{OfGirth: .5, OfLength: .25}).Radius(4, 2); got != 2 {
		t.Fatalf("physical radius %v", got)
	}
	for _, bad := range [][]Joint{nil, {{OfGirth: 0}, {OfGirth: 1}}, {{OfGirth: math.NaN()}, {OfGirth: 1}}, {{OfLength: math.Inf(1)}, {}}} {
		if _, err := NewSleeve(bad, 6, 0); !errors.Is(err, ErrSleeve) {
			t.Fatalf("accepted bad joints: %v", err)
		}
		if _, err := NewLeaf(bad, 0); !errors.Is(err, ErrSleeve) {
			t.Fatalf("accepted bad blade joints: %v", err)
		}
	}
	if _, err := NewSleeve(joints, 2, 0); !errors.Is(err, ErrSleeve) {
		t.Fatalf("accepted two-sided sleeve: %v", err)
	}
}
