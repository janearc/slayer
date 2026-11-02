package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// chalk is a floor that gives back a share of each colour.
var chalk = maths.RGB(0.9, 0.6, 0.3)

// floor is a floor of chalk and nothing else.
func floor() *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: chalk}})
	return w
}

// straightDown is a ray from above the middle of the floor, going down.
func straightDown() maths.Ray {
	return maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, -1, 0)}
}

// a floor under a sun gives back the colour times the sun times the
// cosine of the angle off straight on. on a floor that gives back 0.9,
// 0.6 and 0.3, under a sun of 1, 0.8 and 0.5 at theta from the floor's
// up, that is 0.9 by 1, 0.6 by 0.8 and 0.3 by 0.5, times cos theta.
//
// worked at 0, 30, 60 and 85 degrees, where the cosine is 1, root 3 over
// 2, a half, and the sine of 5 degrees. a sun below the floor lights
// nothing from above, and lights the underside square on.
func TestAFloorUnderASunIsLitByTheCosine(t *testing.T) {
	sun := maths.RGB(1, 0.8, 0.5)
	for _, deg := range []float64{0, 30, 60, 85} {
		w := floor()
		th := deg * maths.Degree
		w.Lamp(thing.Sun(maths.V(math.Sin(th), math.Cos(th), 0), sun))
		cos := map[float64]float64{0: 1, 30: math.Sqrt(3) / 2, 60: 0.5, 85: math.Sin(5 * maths.Degree)}[deg]
		colours(t, "a sun off the vertical", Trace(w, straightDown(), 0), chalk.Times(sun).Scale(cos), 2e-6)
	}
	w := floor()
	w.Lamp(thing.Sun(maths.V(0, -1, 0), sun))
	colours(t, "a sun under the floor, from above", Trace(w, straightDown(), 0), maths.Light{}, 0)
	below := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.V(0, 1, 0)}
	colours(t, "a sun under the floor, from below", Trace(w, below, 0), chalk.Times(sun), 2e-6)
	w.Lamps = nil
	w.Lamp(thing.Sun(maths.V(0, 1, 0), sun))
	colours(t, "a sun over the floor, from below", Trace(w, below, 0), maths.Light{}, 0)
}

// a lamp from every direction at once lights a surface by what it has
// weighed for that surface, and not by the cosine again: a flood of 1,
// 0.8 and 0.5 lights a floor facing up by all of it, and the underside,
// which its dome does not reach, not at all.
//
// a thing with no material is made of grey, 0.8, and under a sun
// straight above is 0.8 of the sun.
func TestAFloodLightsAFloorByWhatItWeighed(t *testing.T) {
	sun := maths.RGB(1, 0.8, 0.5)
	w := floor()
	w.Lamp(film.Flood{Strength: sun})
	colours(t, "a flood from above", Trace(w, straightDown(), 0), chalk.Times(sun), 2e-6)
	below := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.V(0, 1, 0)}
	colours(t, "a flood from below", Trace(w, below, 0), maths.Light{}, 0)

	bare := thing.NewWorld()
	bare.Add(thing.Thing{Shape: shape.Plane{}})
	bare.Lamp(thing.Sun(maths.V(0, 1, 0), sun))
	colours(t, "a thing with no material", Trace(bare, straightDown(), 0), sun.Scale(0.8), 2e-6)
}

// a spot gives light by the inverse square of its distance and the cosine
// at the floor: of strength 2 at 3 across and 4 up it is 5 away, and
// lights a floor by 2 over 25 times the cosine, 4 over 5, 0.064 of a
// floor that gives back all.
//
// straight overhead at 2 and at 4 it is a quarter and a sixteenth of its
// strength. the aim and the cone are where the light is, and a floor
// outside the cone has none.
func TestAFloorUnderASpotIsLitByTheInverseSquare(t *testing.T) {
	white := func() *thing.World {
		w := thing.NewWorld()
		w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: maths.Grey(1)}})
		return w
	}
	for _, c := range []struct {
		name string
		at   maths.Vec
		want float64
	}{
		{"3 across and 4 up", maths.V(3, 4, 0), 2.0 / 25 * 0.8},
		{"2 overhead", maths.V(0, 2, 0), 2.0 / 4},
		{"4 overhead", maths.V(0, 4, 0), 2.0 / 16},
	} {
		w := white()
		w.Lamp(film.Spot{At: c.at, Aim: maths.V(0, -1, 0), Strength: maths.Grey(2), Cone: 1.5})
		// the eye looks from the side, so that the ball above does not
		// stand in the way
		r := maths.Ray{At: maths.V(0, 1, -6), Dir: maths.V(0, -1, 6).Unit()}
		colours(t, c.name, Trace(w, r, 0), maths.Grey(c.want), 2e-6)
	}
	w := white()
	w.Lamp(film.Spot{At: maths.V(0, 2, 0), Aim: maths.V(0, -1, 0), Strength: maths.Grey(2), Cone: 0.1})
	colours(t, "outside the cone", Trace(w, maths.Ray{At: maths.V(5, 1, -6), Dir: maths.V(0, -1, 6).Unit()}, 0),
		maths.Light{}, 0)
}

// light is stopped by what stands in its way, and by that alone. a small
// ball with its underside 0.05 over the floor stops the sun from the
// floor under it, however near, and does not touch a floor it is not
// over.
//
// a lamp that is 2 up is not stopped by a ball behind it at 5; and a ball
// scaled by two is stopped at twice its own size, not at half or four
// times.
func TestWhatStandsBetweenTheFloorAndALampStopsIt(t *testing.T) {
	// the eye looks slantwise at the floor point under the ball, and
	// misses the ball by 0.02
	aslant := maths.Ray{At: maths.V(4, 2, 0), Dir: maths.V(-4, -2, 0).Unit()}
	w := floor()
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.2}, Motion: at(maths.V(0, 0.25, 0), 1)})
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	colours(t, "under a ball just above the floor", Trace(w, aslant, 0), maths.Light{}, 0)
	away := maths.Ray{At: maths.V(7, 2, 0), Dir: maths.V(-4, -2, 0).Unit()}
	colours(t, "away from the ball", Trace(w, away, 0), chalk, 2e-6)

	spot := func() *thing.World {
		w := floor()
		w.Lamp(film.Spot{At: maths.V(0, 2, 0), Aim: maths.V(0, -1, 0), Strength: maths.Grey(1), Cone: 1.5})
		return w
	}
	look := maths.Ray{At: maths.V(0, 1, -6), Dir: maths.V(0, -1, 6).Unit()}
	behind := spot()
	behind.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: at(maths.V(0, 5, 0), 1)})
	colours(t, "a ball behind the lamp", Trace(behind, look, 0), chalk.Scale(0.25), 2e-6)
	large := spot()
	large.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: at(maths.V(0, 5, 0), 2)})
	colours(t, "a ball behind the lamp, doubled", Trace(large, look, 0), chalk.Scale(0.25), 2e-6)
	between := spot()
	between.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.4}, Motion: at(maths.V(0, 1, 0), 2)})
	colours(t, "a ball between, doubled", Trace(between, look, 0), maths.Light{}, 0)
}

// a thing scaled is as far away as its size says: a ball of radius 1
// doubled has a radius of 2, so a ray from 20 meets it at 18.
func TestAScaledThingIsMetWhereItsSizeSaysItIs(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: at(maths.Vec{}, 2)})
	got := Trace(w, maths.Ray{At: maths.V(0, 0, 20), Dir: maths.V(0, 0, -1)}, 0)
	if got.Depth != 18 {
		t.Errorf("a doubled ball is met at %v, want 18", got.Depth)
	}
}

// wall is the plane z = 0 found only by its distance: it can be marched
// to and not hit, and its bounds are a slab 2 thick about it.
type wall struct {
	calls *int
	// offset moves the box off the wall, to a place a ray does not go
	offset float64
	// flush ends the box at the wall, and does not carry it past
	flush bool
}

func (w wall) Bounds() maths.Box {
	low := -1.0
	if w.flush {
		low = 0
	}
	return maths.Box{Min: maths.V(-10+w.offset, -10, low), Max: maths.V(10+w.offset, 10, 1)}
}

func (w wall) Distance(p maths.Vec) float64 {
	if w.calls != nil {
		*w.calls++
	}
	return p.Z
}

// crawl is a wall found by distance that says it is 0.01 away until the
// ray is within half that of the wall at z = 0, so that a march takes
// steps of 0.01, and a box that holds the last n of them.
type crawl struct{ n int }

func (c crawl) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-10, -10, -1), Max: maths.V(10, 10, float64(c.n)*0.01)}
}

func (c crawl) Distance(p maths.Vec) float64 {
	if p.Z > 0.005 {
		return 0.01
	}
	return 0
}

// a march takes at most 256 steps to find a surface, the number it began with, and
// counts the step it begins on: a wall found on the 256th place it tries,
// after 255 steps of 0.01, is met, and one that wants the 257th is not.
// this holds the cutoff it began with, not a truth about walls.
func TestAMarchGivesUpAfterTwoHundredAndFiftySixSteps(t *testing.T) {
	for _, c := range []struct {
		n    int
		meet bool
	}{{255, true}, {256, false}} {
		w := thing.NewWorld()
		w.Add(thing.Thing{Shape: crawl{c.n}})
		got := Trace(w, maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}, 0)
		if (got.Cover == 1) != c.meet {
			t.Errorf("a wall %d steps off: covered %v, want met %v", c.n, got.Cover, c.meet)
		}
	}
}

// a march lands on a wall when it is nearer than a millionth of the
// wall's reach, and not when it is exactly that near. the wall's box is
// 20 by 20 by 2, so its reach is its diagonal, the root of 804, worked
// here by hand and not by the tracer's own measure.
//
// from exactly that near, one step reaches the wall itself, that far
// along; from half as near, the march is there where it starts.
func TestAMarchTouchesWithinAndNotAt(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: wall{}})
	near := 1e-6 * math.Sqrt(20*20+20*20+2*2)
	down := maths.V(0, 0, -1)
	got := Trace(w, maths.Ray{At: maths.V(0, 0, near), Dir: down}, 0)
	if got.Cover != 1 || got.Depth != float32(near) {
		t.Errorf("from %v the wall is met at %v (covered %v), want at %v",
			near, got.Depth, got.Cover, near)
	}
	got = Trace(w, maths.Ray{At: maths.V(0, 0, near/2), Dir: down}, 0)
	if got.Cover != 1 || got.Depth != 0 {
		t.Errorf("from %v the wall is met at %v (covered %v), want at 0",
			near/2, got.Depth, got.Cover)
	}
}

// a march finds a wall that lies exactly on the far side of its box: a
// ray from 8 comes into a slab 1 thick whose far face is the wall, at 7,
// takes a step of 1 and is on the wall at 8, which is the end of the
// box, and is met there.
func TestAMarchReachesTheFarSideOfItsBox(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: wall{flush: true}})
	got := Trace(w, maths.Ray{At: maths.V(0, 0, 8), Dir: maths.V(0, 0, -1)}, 0)
	if got.Cover != 1 || got.Depth != 8 {
		t.Errorf("the wall is met at %v (covered %v), want at 8", got.Depth, got.Cover)
	}
}

// a shape is not marched on a ray that misses its box: the box is the
// first question, and when it says no the shape is not asked its distance
// once.
func TestARayThatMissesTheBoxIsNotMarched(t *testing.T) {
	var calls int
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: wall{calls: &calls, offset: 100}})
	got := Trace(w, maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}, 0)
	if got.Cover != 0 || calls != 0 {
		t.Errorf("a ray that misses the box covers %v and asked %d distances", got.Cover, calls)
	}
}

// of two things as far away as each other, the nearest the ray finds is
// the one listed first: two walls at the same place, red and then blue.
func TestOfTwoThingsEquallyNearTheFirstListedIsMet(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: wall{}, Material: &thing.Material{Glow: maths.RGB(1, 0, 0)}})
	w.Add(thing.Thing{Shape: wall{}, Material: &thing.Material{Glow: maths.RGB(0, 0, 1)}})
	colours(t, "two walls", Trace(w, maths.Ray{At: maths.V(0, 0, 8), Dir: maths.V(0, 0, -1)}, 0),
		maths.RGB(1, 0, 0), 0)
}
