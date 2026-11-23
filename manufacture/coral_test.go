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

func TestCoralliteReliefWrapsAndChangesTheBranchSkin(t *testing.T) {
	const radius = .02
	for _, height := range []float64{0, .013, .041} {
		left := coralliteRelief(0, height, radius)
		right := coralliteRelief(2*math.Pi, height, radius)
		if math.Abs(left-right) > 1e-12 {
			t.Fatalf("surface does not wrap at height %g: %g != %g", height, left, right)
		}
	}
	around := math.Round(2 * math.Pi * radius / .008)
	if coralliteRelief(0, 0, radius) >= coralliteRelief(.31*2*math.Pi/around, 0, radius) {
		t.Fatal("expected the cup centre to sit below its rim")
	}
	plain := coralBranch{length: .14, radius: radius, seed: 7}.skin(cut{})
	detailed := coralBranch{length: .14, radius: radius, seed: 7, corallites: true}.skin(cut{})
	if len(detailed) <= len(plain) {
		t.Fatal("the detailed surface needs enough faces to resolve its cups")
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
