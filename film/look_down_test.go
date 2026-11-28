package film

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestACameraLookingStraightDownHasNorthAtTheTop stands an eye straight
// over the world's origin, looking down, and asks which way the top and the
// right of its picture look. North is the world's minus z and east its plus
// x (set/shots.go, fromMap), so a map seen from above has north at the top
// and east at the right, as a map is read. A worker on Caprica measured
// straight-down frames rolled a quarter turn (2026-11-26); this says
// whether the roll is LookAt's.
func TestACameraLookingStraightDownHasNorthAtTheTop(t *testing.T) {
	from := maths.V(0, 10, 0)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
	const side = 101
	top := Look(eye, Pinhole{}, side/2, 0, side, side, maths.Band{}).Dir
	right := Look(eye, Pinhole{}, side-1, side/2, side, side,
		maths.Band{}).Dir
	if top.Z >= 0 || abs(top.X) > 1e-9 {
		t.Errorf("the top of the picture looks toward %v; want north, "+
			"minus z", top)
	}
	if right.X <= 0 || abs(right.Z) > 1e-9 {
		t.Errorf("the right of the picture looks toward %v; want east, "+
			"plus x", right)
	}
}

// TestACameraLookingDownOnASphereHasItsNorthAtTheTop stands an eye over
// a sphere at forty-five degrees north, looking at the sphere's centre, and
// asks that the top of its picture lean toward the sphere's north pole,
// the world's up, and the right toward the east: a planet's local map,
// seen from above, is read as a map is. East on a sphere is its pole
// crossed with the place, which on the meridian through minus z is
// minus x.
func TestACameraLookingDownOnASphereHasItsNorthAtTheTop(t *testing.T) {
	from := maths.V(0, 10, -10)
	east := maths.Up.Cross(from).Unit()
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
	const side = 101
	middle := Look(eye, Pinhole{}, side/2, side/2, side, side,
		maths.Band{}).Dir
	top := Look(eye, Pinhole{}, side/2, 0, side, side, maths.Band{}).Dir
	right := Look(eye, Pinhole{}, side-1, side/2, side, side,
		maths.Band{}).Dir
	if up := top.Sub(middle); up.Y <= 0 {
		t.Errorf("the top of the picture leans %v from its middle; want "+
			"toward the pole, plus y", up)
	}
	if across := right.Sub(middle); across.Dot(east) <= 0 {
		t.Errorf("the right of the picture leans %v from its middle; want "+
			"east, %v", across, east)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
