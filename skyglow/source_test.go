package skyglow

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

// powerTolerance is how far a power may be from the sum a test works: the
// rounding of a few products.
const powerTolerance = 1e-12

// upward is a triangle of area two in the plane at the height of
// its place, turning counterclockwise seen from above, so it faces up.
func upward(place maths.Vec) (a, b, c maths.Vec) {
	return place, place.Add(maths.V(0, 0, 2)), place.Add(maths.V(2, 0, 0))
}

// A panel's power is its radiance times its area, the intensity it sends
// along its normal, and it faces the way its corners turn.
func TestAPanelsPowerIsItsRadianceTimesItsArea(t *testing.T) {
	cluster := NewCluster(25)
	first, second, third := upward(maths.V(3, 10, 3))
	cluster.Panel(first, second, third, maths.Grey(3))
	made := cluster.Sources()
	if len(made) != 1 {
		t.Fatalf("one panel made %d sources", len(made))
	}
	if made[0].Power != maths.Grey(6) || made[0].Normal != maths.Up {
		t.Errorf("panel of area 2 at radiance 3: power %v, normal %v",
			made[0].Power, made[0].Normal)
	}
}

// A flat emitter sends its light out of its front by Lambert's law: the
// full intensity along the normal, the cosine of the angle off it, and
// nothing out of the back or along the surface.
func TestAPanelShinesByLambertsLawFromItsFrontOnly(t *testing.T) {
	front := Source{Normal: maths.Up, Power: maths.Grey(8)}
	slant := maths.V(1, 1, 0).Unit()
	cases := map[string]struct {
		out  maths.Vec
		want maths.Light
	}{
		"along the normal": {maths.Up, maths.Grey(8)},
		"forty-five off":   {slant, maths.Grey(8 * math.Sqrt(0.5))},
		"along the face":   {maths.V(1, 0, 0), maths.Light{}},
		"out of the back":  {maths.Up.Neg(), maths.Light{}},
	}
	for name, one := range cases {
		got := front.toward(one.out)
		if math.Abs(got.R-one.want.R) > powerTolerance {
			t.Errorf("%s: %v, wanted %v", name, got, one.want)
		}
	}
}

// A point shines alike every way, whatever way is asked.
func TestAPointShinesAlikeEveryWay(t *testing.T) {
	point := Source{Power: maths.Grey(5)}
	for _, out := range []maths.Vec{maths.Up, maths.Up.Neg(),
		maths.V(1, 0, 0), maths.V(1, 1, 1).Unit()} {
		if got := point.toward(out); got != maths.Grey(5) {
			t.Errorf("a point sends %v out along %v", got, out)
		}
	}
}

// Panels in one bin facing one way are one source holding their power,
// from their middle weighted by power.
func TestPanelsInOneBinFacingOneWayAreOneSource(t *testing.T) {
	cluster := NewCluster(25)
	first, second, third := upward(maths.V(2, 10, 2))
	cluster.Panel(first, second, third, maths.Grey(1))
	first, second, third = upward(maths.V(12, 10, 2))
	cluster.Panel(first, second, third, maths.Grey(3))
	made := cluster.Sources()
	if len(made) != 1 || math.Abs(made[0].Power.R-8) > powerTolerance {
		t.Fatalf("two panels in a bin: %d sources, %v", len(made), made)
	}
	// the panels' middles are at 8/3 and 38/3 east, with powers 2 and 6
	weighted := (2*8.0/3 + 6*38.0/3) / 8
	if middle := made[0].At.X; math.Abs(middle-weighted) > powerTolerance {
		t.Errorf("the middle is %g east, wanted %g", middle, weighted)
	}
}

// Anti-test: panels in one bin facing opposite ways are not mixed; their
// light would cancel in a mean normal, and each goes its own way.
func TestPanelsFacingAwayFromEachOtherAreTwoSources(t *testing.T) {
	cluster := NewCluster(25)
	first, second, third := upward(maths.V(2, 10, 2))
	cluster.Panel(first, second, third, maths.Grey(1))
	cluster.Panel(first, third, second, maths.Grey(1))
	if made := cluster.Sources(); len(made) != 2 {
		t.Errorf("panels facing up and down made %d sources", len(made))
	}
}

// Anti-test: lamps the tracer scatters are kept apart from panels and
// from lamps it does not, since the glow adds one and not the other.
func TestTracedAndUntracedPointsAreKeptApart(t *testing.T) {
	cluster := NewCluster(25)
	cluster.Point(maths.V(1, 5, 1), maths.Grey(2), true)
	cluster.Point(maths.V(2, 5, 2), maths.Grey(2), false)
	made := cluster.Sources()
	if len(made) != 2 || made[0].Traced == made[1].Traced {
		t.Errorf("a traced and an untraced point: %v", made)
	}
}

// The same city makes the same list of sources, whatever order its
// emitters were found in.
func TestSourcesComeOutInAFixedOrder(t *testing.T) {
	places := []maths.Vec{maths.V(2, 10, 2), maths.V(60, 30, 2),
		maths.V(-40, 5, 90), maths.V(2, 80, 2)}
	forward, backward := NewCluster(25), NewCluster(25)
	for index := range places {
		first, second, third := upward(places[index])
		forward.Panel(first, second, third, maths.Grey(1))
		first, second, third = upward(places[len(places)-1-index])
		backward.Panel(first, second, third, maths.Grey(1))
	}
	if !reflect.DeepEqual(forward.Sources(), backward.Sources()) {
		t.Error("the same emitters made different lists in two orders")
	}
}

// Anti-test: what gives no light is left out, as is what cannot be a
// weight: a point of no power, of power that is not a number, and a
// triangle of no area.
func TestWhatGivesNoLightIsLeftOut(t *testing.T) {
	cluster := NewCluster(25)
	cluster.Point(maths.V(1, 1, 1), maths.Light{}, false)
	notANumber := maths.Grey(math.NaN())
	cluster.Point(maths.V(1, 1, 1), notANumber, false)
	endless := maths.Grey(math.Inf(1))
	cluster.Point(maths.V(1, 1, 1), endless, false)
	cluster.Panel(maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(2, 0, 0),
		maths.Grey(4))
	if made := cluster.Sources(); len(made) != 0 {
		t.Errorf("nothing shining made %d sources", len(made))
	}
}
