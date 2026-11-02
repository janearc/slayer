package manufacture_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// the manufactory from outside: what a command can reach, and nothing
// else. it names a kind, sows a seed, makes a thing and asks for what
// a scene holds. it cannot name the types it is handed, and need not.
func TestFromOutsideAKindAndASeedMakeAThing(t *testing.T) {
	names := manufacture.KindNames()
	if len(names) < 10 {
		t.Fatalf("only %d kinds: %v", len(names), names)
	}
	there := maths.Subspace{Origin: maths.V(4, 0, -2)}
	for _, name := range names {
		row, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.Make(row, manufacture.Sow(9))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		things, err := made.Things(there)
		if err != nil || len(things) == 0 {
			t.Fatalf("%s: %d things, %v", name, len(things), err)
		}
		if stood := things[0].At(0).Origin; stood != there.Origin {
			t.Errorf("%s stands at %v", name, stood)
		}
	}
	if _, err := manufacture.KindNamed("sofa"); err == nil {
		t.Error("a sofa was found among the kinds")
	}
}

// a thing's lights come out where the thing was stood, each answering
// what an emitter is made from: some light, a strength, a beam, and a
// share of itself at a time.
func TestFromOutsideAThingsLightsAreWhereItStands(t *testing.T) {
	row, err := manufacture.KindNamed("spotlight")
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.Make(row, manufacture.Sow(2))
	if err != nil {
		t.Fatal(err)
	}
	home := made.Lights(maths.Subspace{})
	quarter := maths.About(maths.YAxis, math.Pi/2)
	there := maths.Subspace{Origin: maths.V(10, 0, -5),
		Orientation: quarter}
	moved := made.Lights(there)
	if len(home) != 1 || len(moved) != 1 {
		t.Fatalf("%d lights at home and %d moved", len(home),
			len(moved))
	}
	if !moved[0].At().Near(there.Out(home[0].At()), 1e-9) ||
		!moved[0].Toward().Near(quarter.Out(home[0].Toward()), 1e-9) {
		t.Errorf("the light is at %v shining %v", moved[0].At(),
			moved[0].Toward())
	}
	lamp := moved[0].Light()
	half, soft := lamp.Angles()
	first, last := lamp.Bands().Span()
	if lamp.Strength() <= 0 || half.InRadians() <= 0 ||
		soft.InRadians() < 0 || last < first || lamp.Share(0) != 1 {
		t.Errorf("a light of strength %v, a beam of %v and %v, "+
			"bands %v to %v", lamp.Strength(), half.InRadians(),
			soft.InRadians(), first, last)
	}
}

// what a light answers from outside is its own: a spotlight's beam is
// the width its row allows with the soft edge its row gives, in that
// order, and a campfire's share of itself changes as it flickers.
// hippo's review swapped the two angles and held the share at one,
// and nothing noticed.
func TestFromOutsideALightAnswersForItself(t *testing.T) {
	lightOf := func(name string) interface {
		Angles() (measure.Angle, measure.Angle)
		Share(time.Duration) float64
	} {
		row, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.Make(row, manufacture.Sow(2))
		if err != nil {
			t.Fatal(err)
		}
		return made.Lights(maths.Subspace{})[0].Light()
	}
	half, soft := lightOf("spotlight").Angles()
	// the spotlight's row: a beam of 12 to 25 degrees, an edge of 6
	if half.InDegrees() < 12 || half.InDegrees() > 25 ||
		math.Abs(soft.InDegrees()-6) > 1e-9 {
		t.Errorf("a spotlight's beam is %v with an edge of %v",
			half.InDegrees(), soft.InDegrees())
	}
	fire := lightOf("campfire")
	shares := map[float64]bool{}
	for tick := range 20 {
		share := fire.Share(time.Duration(tick) * 90 * time.Millisecond)
		if share <= 0 || share > 1 {
			t.Fatalf("a fire is %v of itself", share)
		}
		shares[share] = true
	}
	if len(shares) < 10 {
		t.Errorf("a fire takes %d brightnesses in two seconds",
			len(shares))
	}
}
