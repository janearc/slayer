package embers

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// a burst's contract: thrown as a sphere, every firebrand leaves at
// between half and all of the spread, up and down alike; thrown as a box,
// the corners leave faster than the spread, which is how a burst showed
// as a square.

// thrownSpeeds is each living firebrand's launch speed, read back from
// where it is: with no air and no gravity it stands its launch velocity
// times the drag's reach, tau (1 - e^(-age/tau)), from where it was born.
func thrownSpeeds(t *testing.T, sphere bool) (speeds []float64, up, down int) {
	t.Helper()
	spec := Spec{Rate: 200, Life: 2 * time.Second, Drag: time.Second,
		Radius: measure.Metres(0.1), Jitter: maths.V(10, 10, 10),
		Hot: measure.Kelvin(1800), Cool: measure.Kelvin(900), Seed: 7,
		Sphere: sphere}
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, firebrand := range source.At(time.Second) {
		age := firebrand.Age().Seconds()
		reach := 1 - math.Exp(-age)
		if reach < 0.01 {
			continue
		}
		at := firebrand.Position()
		speeds = append(speeds, at.Len()/reach)
		if at.Y > 0 {
			up++
		} else {
			down++
		}
	}
	if len(speeds) < 100 {
		t.Fatalf("only %d firebrands to read", len(speeds))
	}
	return speeds, up, down
}

func TestASphereThrowsEveryFirebrandAtBetweenHalfAndAllTheSpread(t *testing.T) {
	speeds, up, down := thrownSpeeds(t, true)
	for i, speed := range speeds {
		if speed < 5-1e-9 || speed > 10+1e-9 {
			t.Fatalf("firebrand %d thrown at %.3f m/s, not 5 to 10", i, speed)
		}
	}
	if up < len(speeds)/4 || down < len(speeds)/4 {
		t.Fatalf("%d thrown up and %d down: not a sphere", up, down)
	}
}

func TestABoxThrowsItsCornersFasterThanTheSpread(t *testing.T) {
	speeds, _, _ := thrownSpeeds(t, false)
	fastest := 0.0
	for _, speed := range speeds {
		fastest = math.Max(fastest, speed)
	}
	if fastest <= 10 {
		t.Fatalf("the box's fastest is %.3f m/s; its corners reach 17", fastest)
	}
}
