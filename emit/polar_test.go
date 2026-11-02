package emit

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// a beam pointing up, narrow, still: lit along it, dark across it
func upright() Polar {
	return Polar{Light: sunlike(), Width: 5 * maths.Degree,
		Soft: 2 * maths.Degree, Axis: maths.V(0, 1, 0)}
}

func TestABeamLightsAlongItsPoleAndNotAcross(t *testing.T) {
	clock := time.Duration(0)
	beams := upright().Beams(&clock)
	_, _, along := beams.Light(maths.V(0, 10, 0), maths.Vec{})
	_, _, across := beams.Light(maths.V(10, 0, 0), maths.Vec{})
	if along.G <= 0 || across.G != 0 {
		t.Errorf("along the pole %v, across it %v", along, across)
	}
}

func TestTheSecondPoleIsOppositeUntilOffset(t *testing.T) {
	north, south := upright().Poles(0)
	if !near(north.Dot(south), -1) {
		t.Errorf("poles %v and %v are not opposed", north, south)
	}
	turned := upright()
	turned.Offset = 20 * maths.Degree
	north, south = turned.Poles(0)
	apart := math.Acos(north.Dot(south)) / maths.Degree
	if math.Abs(apart-160) > 1e-6 {
		t.Errorf("offset poles are %.3f degrees apart, want 160", apart)
	}
}

// tilted off its spin axis, a pole goes round it: a lighthouse
func TestATiltedPoleSweepsRoundTheSpin(t *testing.T) {
	sweeping := upright()
	sweeping.Tilt, sweeping.Spin = 30*maths.Degree, 4*time.Second
	start, _ := sweeping.Poles(0)
	half, _ := sweeping.Poles(2 * time.Second)
	whole, _ := sweeping.Poles(4 * time.Second)
	if !near(start.Y, math.Cos(30*maths.Degree)) {
		t.Errorf("the pole is %v, want 30 degrees off the axis", start)
	}
	if near(start.X, half.X) && near(start.Z, half.Z) {
		t.Errorf("half a turn later the pole has not moved: %v", half)
	}
	if !near(start.X, whole.X) || !near(start.Z, whole.Z) {
		t.Errorf("a whole turn later the pole is %v, want %v", whole,
			start)
	}
}

func TestTheBeamsReadTheClock(t *testing.T) {
	sweeping := upright()
	sweeping.Tilt, sweeping.Spin = 60*maths.Degree, 4*time.Second
	clock := time.Duration(0)
	beams := sweeping.Beams(&clock)
	north, _ := sweeping.Poles(0)
	_, _, before := beams.Light(north.Scale(10), maths.Vec{})
	clock = 2 * time.Second
	_, _, after := beams.Light(north.Scale(10), maths.Vec{})
	if before.G <= 0 || after.G != 0 {
		t.Errorf("where the pole was, %v then %v", before, after)
	}
}
