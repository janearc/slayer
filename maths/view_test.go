package maths_test

import (
	"github.com/janearc/slayer/maths"
	"testing"
)

func TestViewKeepsItsOriginAndFacesItsTarget(test *testing.T) {
	from, target := vector(test, "eye"), vector(test, "target")
	view, err := maths.View(from, target, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	exactly(test, "origin", view.Origin, from)
	want := target.Sub(from).Unit()
	near(test, "forward", view.Forward(), want)
	near(test, "side length", view.Orientation.X, maths.XAxis)
}

func TestViewRefusesAbsentAndUnmeasurableDirections(test *testing.T) {
	from, target := vector(test, "eye"), vector(test, "target")
	for _, name := range []string{"origin", "short", "nan axis", "infinite axis", "long"} {
		up := vector(test, name)
		view, err := maths.View(from, target, up)
		if err == nil || view != nil {
			test.Fatalf("accepted up %s", name)
		}
	}
	if view, err := maths.View(from, from, maths.Up); err == nil || view != nil {
		test.Fatal("accepted an absent look direction")
	}
	for _, name := range []string{"nan start", "infinite axis", "long"} {
		bad := vector(test, name)
		if view, err := maths.View(bad, target, maths.Up); err == nil || view != nil {
			test.Fatalf("accepted origin %s", name)
		}
		if view, err := maths.View(from, bad, maths.Up); err == nil || view != nil {
			test.Fatalf("accepted target %s", name)
		}
	}
}
