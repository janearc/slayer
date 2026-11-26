package manufacture

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestCoralConstruction(t *testing.T) {
	kind, err := Coral(DefaultCoral())
	if err != nil {
		t.Fatal(err)
	}
	first, err := Make(kind, Sow(14))
	if err != nil {
		t.Fatal(err)
	}
	again, err := Make(kind, Sow(14))
	if err != nil {
		t.Fatal(err)
	}
	other, err := Make(kind, Sow(15))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.parts) != 121 || !reflect.DeepEqual(first.Bounds(), again.Bounds()) ||
		reflect.DeepEqual(first.Bounds(), other.Bounds()) {
		t.Fatal("branch count or seeded geometry is incorrect")
	}
	if _, err := first.Things(maths.Local()); err != nil {
		t.Fatalf("scene mesh: %v", err)
	}
}

func TestCoralRejectsInvalidDimensions(t *testing.T) {
	for _, spec := range []CoralSpec{
		{}, {Height: measure.Metres(math.NaN()), Radius: measure.Metres(.02), Levels: 3},
		{Height: measure.Metres(.2), Radius: measure.Metres(.1), Levels: 3},
		{Height: measure.Metres(.2), Radius: measure.Metres(.02), Levels: 5},
	} {
		if _, err := Coral(spec); err == nil {
			t.Fatalf("accepted invalid specification %+v", spec)
		}
	}
}

func TestDetailedCoralColonyFitsSceneMesh(t *testing.T) {
	spec := DefaultCoral()
	spec.Corallites, spec.FineSurface = true, true
	kind, err := Coral(spec)
	if err != nil {
		t.Fatal(err)
	}
	colony, err := Make(kind, Sow(14))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := colony.Things(maths.Local()); err != nil {
		t.Fatalf("detailed colony exceeds the scene mesh budget: %v", err)
	}
}
