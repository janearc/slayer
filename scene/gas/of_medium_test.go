package gas

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// boxed is a medium with a box and a density and nothing else: no span of
// its own, no motion.
type boxed struct{ box maths.Box }

func (b boxed) Density(maths.Vec) float64 { return 1 }
func (b boxed) Bounds() maths.Box         { return b.box }

// spanned has a span that is not its box's: it gives back its near and far
// moved in by one, so that the span is the shape's and the arguments were
// handed on.
type spanned struct{ boxed }

func (spanned) Span(_ maths.Ray, near, far float64) (float64, float64, bool) {
	return near + 1, far - 1, true
}

// moving is a medium that is timed: its tick is a medium of another box,
// and which tick it was asked for rides in the box's edge.
type moving struct{ boxed }

func (m moving) Tick(t time.Duration) medium.Medium {
	return boxed{maths.Box{Max: maths.V(t.Seconds(), 0, 0)}}
}

var cube = maths.Box{Min: maths.V(-2, -2, -2), Max: maths.V(2, 2, 2)}

// a medium of a gas has the shape's box, whatever the gas.
func TestOfHasItsShapesBounds(t *testing.T) {
	o := Of{Shape: boxed{cube}, In: Haze(), Unit: 1000}
	if got := o.Bounds(); got != cube {
		t.Errorf("the bounds are %v, want %v", got, cube)
	}
}

// with a shape that spans, the span is the shape's own, handed the near and
// the far it was asked; with one that does not, it is the shape's box's.
// a ray from -5 along x meets the cube of two at 3 and 7; the shape that
// moves in by one gives 1 and 99 for a near of 0 and a far of 100.
//
// a ball of radius 2 with the same ray at a height of 1.5 is met by the
// sphere at 5 less and 5 more than the root of 1.75, not by its box at 3.
func TestOfSpansByTheShapeThenByTheBox(t *testing.T) {
	ray := maths.Ray{At: maths.V(-5, 0, 0), Dir: maths.V(1, 0, 0)}
	s := Of{Shape: spanned{boxed{cube}}, In: Haze()}
	enter, leave, ok := s.Span(ray, 0, 100)
	if !ok || enter != 1 || leave != 99 {
		t.Errorf("the shape's span is %v to %v (%v), want 1 to 99", enter, leave, ok)
	}
	b := Of{Shape: boxed{cube}, In: Haze()}
	enter, leave, ok = b.Span(ray, 0, 100)
	if !ok || enter != 3 || leave != 7 {
		t.Errorf("the box's span is %v to %v (%v), want 3 to 7", enter, leave, ok)
	}
	if _, _, ok := b.Span(maths.Ray{At: maths.V(-5, 3, 0), Dir: maths.V(1, 0, 0)}, 0, 100); ok {
		t.Error("a ray 3 above the cube meets it")
	}
	ball := Of{Shape: medium.Puff{Radius: 2, Thick: 1}, In: Haze()}
	high := maths.Ray{At: maths.V(-5, 1.5, 0), Dir: maths.V(1, 0, 0)}
	enter, leave, ok = ball.Span(high, 0, 100)
	root := math.Sqrt(1.75)
	wantClose(t, "the ball's entry", enter, 5-root)
	wantClose(t, "the ball's exit", leave, 5+root)
	if !ok {
		t.Error("a ray through the ball misses it")
	}
}

// a ticked medium of a gas is the shape ticked and the gas kept: the shape
// of the tick, the gas by name and its unit. one whose shape is not timed
// is the same medium.
func TestOfTicksItsShapeAndKeepsItsGas(t *testing.T) {
	o := Of{Shape: moving{boxed{cube}}, In: Haze(), Unit: 1000}
	got, ok := o.Tick(3 * time.Second).(Of)
	if !ok {
		t.Fatalf("a ticked medium of gas is %T", o.Tick(3*time.Second))
	}
	if want := (maths.Box{Max: maths.V(3, 0, 0)}); got.Bounds() != want {
		t.Errorf("the ticked shape has the box %v, want %v", got.Bounds(), want)
	}
	if got.In.Name != "haze" || got.Unit != 1000 {
		t.Errorf("the ticked medium has the gas %q and the unit %v", got.In.Name, got.Unit)
	}
	still := Of{Shape: boxed{cube}, In: Haze(), Unit: 1000}
	same, ok := still.Tick(3 * time.Second).(Of)
	if !ok || same.Bounds() != cube || same.In.Name != "haze" || same.Unit != 1000 {
		t.Errorf("a medium that does not move was ticked into %v", same)
	}
}

// a medium's gas is per its own unit: a unit of 1000 metres is the gas times
// 1000; one of a half is half the gas; and a unit of zero or less is not a unit
// and is taken as one metre, so the gas as it is.
func TestOfGasIsPerItsUnit(t *testing.T) {
	g := Gas{Ranges: map[radiation.Range]Optics{
		thing.Visible: {Absorb: 0.5, Scatter: 0.25}}}
	band := maths.Band{Nanometres: 550}
	for _, c := range []struct{ unit, per float64 }{
		{1000, 1000}, {2, 2}, {1, 1}, {0.5, 0.5}, {0, 1}, {-3, 1},
	} {
		got := Of{In: g, Unit: c.unit}.Gas()
		if a, s := got.Absorb(band), got.Scatter(band); a != 0.5*c.per || s != 0.25*c.per {
			t.Errorf("a unit of %v gives %v and %v, want %v and %v", c.unit, a, s,
				0.5*c.per, 0.25*c.per)
		}
	}
}
