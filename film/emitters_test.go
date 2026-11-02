package film

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// a radar beams its strength in its range, from its low end up to but not its
// high one, and zero outside: x-ray's range is 1e-3 to 10 nm.
func TestARadarBeamsInItsRangeAlone(t *testing.T) {
	r := Radar{Range: radiation.XRay(), Strength: 7}
	for band, want := range map[float64]float64{1e-4: 0, 1e-3: 7, 5: 7,
		10: 0, 20: 0} {
		if got := r.Beams(maths.Band{Nanometres: band}); got != want {
			t.Errorf("at %v nm it beams %v, want %v", band, got, want)
		}
	}
	toward, dist, light := Radar{At: maths.V(0, 3, 4)}.Light(maths.V(0, 0, 0),
		maths.V(0, 1, 0))
	if !toward.Near(maths.V(0, 0.6, 0.8), 1e-15) || dist != 5 ||
		light != (maths.Light{}) {
		t.Errorf("a radar 5 away: %v, %v, %v", toward, dist, light)
	}
}

// the sun is its direction, one unit long, infinitely far, at full
// strength; a flood is no direction, infinitely far, all of its strength
// facing its up, half on its side and none facing down, with the world's
// up when it is given none.
func TestTheSunAndTheFlood(t *testing.T) {
	toward, dist, light := Sun{Toward: maths.V(0, 3, 4),
		Strength: maths.Grey(2)}.Light(maths.Vec{}, maths.Vec{})
	if !toward.Near(maths.V(0, 0.6, 0.8), 1e-15) || !math.IsInf(dist, 1) ||
		light != maths.Grey(2) {
		t.Errorf("the sun: %v, %v, %v", toward, dist, light)
	}
	for _, flood := range []Flood{{Strength: maths.Grey(1)},
		{Up: maths.V(0, 1, 0), Strength: maths.Grey(1)}} {
		for n, want := range map[maths.Vec]float64{maths.V(0, 1, 0): 1,
			maths.V(1, 0, 0): 0.5, maths.V(0, -1, 0): 0} {
			toward, dist, light := flood.Light(maths.Vec{}, n)
			if toward != (maths.Vec{}) || !math.IsInf(dist, 1) ||
				light != maths.Grey(want) {
				t.Errorf("%+v facing %v: %v, %v, %v", flood, n, toward,
					dist, light)
			}
		}
	}
}

// a spot lights a point one unit down its aim at its full strength, and a
// point at its own place with nothing, from nowhere.
func TestASpotAtAndFromItsPlace(t *testing.T) {
	s := Spot{Aim: maths.V(0, -1, 0), Strength: maths.Grey(3), Cone: 0.5}
	if toward, dist, light := s.Light(maths.V(0, -1, 0), maths.Vec{}); toward !=
		maths.V(0, 1, 0) || dist != 1 || light != maths.Grey(3) {
		t.Errorf("one below: %v, %v, %v", toward, dist, light)
	}
	if toward, dist, light := s.Light(maths.Vec{}, maths.Vec{}); toward !=
		(maths.Vec{}) || dist != 0 || light != (maths.Light{}) {
		t.Errorf("at the lamp: %v, %v, %v", toward, dist, light)
	}
}

// where the dot of the way to a point and the aim rounds past one, or
// below minus one: straight down the aim is all the light, and straight
// behind none, and neither is NaN.
func TestASpotAimedWhereTheDotRoundsPastOne(t *testing.T) {
	// down (1, 0, 5) the dot is 1.0000000000000002, found by search
	ahead := Spot{Aim: maths.V(1, 0, 5), Strength: maths.Grey(26), Cone: 0.5}
	if _, _, got := ahead.Light(maths.V(1, 0, 5), maths.Vec{}); math.IsNaN(
		got.R) || math.Abs(got.R-1) > 1e-12 {
		t.Errorf("down the aim: %v, want 26 over 26", got)
	}
	// behind (1, 1, 1) at (-2, -2, -2) it is -1.0000000000000002
	s := Spot{Aim: maths.V(1, 1, 1), Strength: maths.Grey(12), Cone: 0.5}
	_, _, behind := s.Light(maths.V(-2, -2, -2), maths.Vec{})
	if behind != (maths.Light{}) {
		t.Errorf("behind: %v", behind)
	}
}

// a spot's edge, a cone of half a radian softened over a fifth: all of the
// light inside, none at the cone and past it, and the smooth step between,
// a half at the middle and 0.15625 a quarter of the way in. a hard edge is
// all of it short of the cone.
func TestASpotsEdge(t *testing.T) {
	s := Spot{Cone: 0.5, Soft: 0.2}
	for off, want := range map[float64]float64{0.1: 1, 0.3: 1, 0.4: 0.5,
		0.45: 0.15625, 0.5: 0, 0.6: 0} {
		if got := s.edge(off); math.Abs(got-want) > 1e-12 {
			t.Errorf("%v off the aim: %v, want %v", off, got, want)
		}
	}
	hard := Spot{Cone: 0.5}
	if hard.edge(0.49) != 1 || hard.edge(0.5) != 0 {
		t.Errorf("a hard edge: %v, %v", hard.edge(0.49), hard.edge(0.5))
	}
}

// a laser of no aim points down minus z and reaches a thousand units;
// one given a reach of half a unit reaches half a unit.
func TestALasersAimAndReach(t *testing.T) {
	l := Laser{Width: 1, Strength: maths.Grey(1)}
	for along, lit := range map[float64]bool{5: true, 999: true,
		1000.5: false} {
		_, _, light := l.Light(maths.V(0, 0, -along), maths.Vec{})
		if (light.R > 0) != lit {
			t.Errorf("%v down minus z: %v", along, light)
		}
	}
	short := Laser{Aim: maths.V(0, 0, -1), Width: 1, Length: 0.5,
		Strength: maths.Grey(1)}
	if _, _, light := short.Light(maths.V(0, 0, -0.7), maths.Vec{}); light.R !=
		0 {
		t.Errorf("past a reach of a half: %v", light)
	}
}

// along a beam: nothing at its source, light from just past it out to the
// end of its reach, at the end too; across it: none from a beam of no
// width, a last glimmer at three widths out, e to the minus eighteen, and
// none past.
func TestALasersBeamAlongAndAcross(t *testing.T) {
	l := Laser{Aim: maths.V(1, 0, 0), Width: 1, Length: 10,
		Strength: maths.Grey(1)}
	for along, lit := range map[float64]bool{0: false, 0.5: true,
		10: true} {
		_, _, light := l.Light(maths.V(along, 0, 0), maths.Vec{})
		if (light.R > 0) != lit {
			t.Errorf("%v along: %v", along, light)
		}
	}
	_, _, edge := l.Light(maths.V(5, 3, 0), maths.Vec{})
	if math.Abs(edge.R/math.Exp(-18)-1) > 1e-9 {
		t.Errorf("three widths out: %v, want e^-18", edge.R)
	}
	_, _, one := l.Light(maths.V(5, 1, 0), maths.Vec{})
	if math.Abs(one.R/math.Exp(-2)-1) > 1e-9 {
		t.Errorf("a width out: %v, want e^-2, as the doc says", one.R)
	}
	if _, _, past := l.Light(maths.V(5, 3.5, 0), maths.Vec{}); past.R != 0 {
		t.Errorf("three and a half widths out: %v", past)
	}
	thin := l
	thin.Width = 0
	if _, _, none := thin.Light(maths.V(5, 0, 0), maths.Vec{}); none !=
		(maths.Light{}) {
		t.Errorf("a beam of no width: %v", none)
	}
}

// a beam's box is its length and three widths round it: from (1, 2, 3)
// along (2, -2, 1) for nine units, half a unit wide, it ends at (7, -4, 6),
// worked by hand.
func TestALasersBox(t *testing.T) {
	b := Laser{At: maths.V(1, 2, 3), Aim: maths.V(2, -2, 1), Width: 0.5,
		Length: 9}.Bounds()
	if !b.Min.Near(maths.V(-0.5, -5.5, 1.5), 1e-12) ||
		!b.Max.Near(maths.V(8.5, 3.5, 7.5), 1e-12) {
		t.Errorf("the box is %v", b)
	}
}
