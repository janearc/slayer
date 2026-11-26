package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/mesh"
)

func TestSquidAnimalRecipeKeepsGeometryAndBounds(t *testing.T) {
	name, raw, err := EmbeddedAnimalBook("squid")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadAnimalBook("squid", name, raw)
	if err != nil {
		t.Fatal(err)
	}
	phase := 1.1
	row, err := book.New(AnimalOptions{Species: "humboldt-squid", Phase: &phase})
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(row, Sow(7))
	if err != nil {
		t.Fatal(err)
	}
	want, err := NewHumboldtSquid(phase)
	if err != nil {
		t.Fatal(err)
	}
	if got, reference := birdAllosaurusWitness(made.native), birdAllosaurusWitness(want); got != reference {
		t.Fatal("profile recipe changed the authored specimen")
	}
	bounds := made.Bounds()
	extent, err := ExtentOf(row, Sow(7))
	if err != nil || extent != bounds {
		t.Fatalf("extent %v, bounds %v, error %v", extent, bounds, err)
	}
	for i, part := range made.native {
		actual := transformed(part.Shape.Bounds(), part.At(0))
		if actual.Min.X < bounds.Min.X || actual.Min.Y < bounds.Min.Y || actual.Min.Z < bounds.Min.Z ||
			actual.Max.X > bounds.Max.X || actual.Max.Y > bounds.Max.Y || actual.Max.Z > bounds.Max.Z {
			t.Fatalf("part %d escapes native bounds", i)
		}
	}
	zero := 0.0
	other, err := book.New(AnimalOptions{Species: "humboldt-squid", Phase: &zero})
	if err != nil {
		t.Fatal(err)
	}
	still, err := Make(other, Sow(7))
	if err != nil {
		t.Fatal(err)
	}
	a, b := made.native[0].Shape.(*mesh.Mesh).Points(), still.native[0].Shape.(*mesh.Mesh).Points()
	var movement float64
	for i := range a {
		movement = math.Max(movement, a[i].Sub(b[i]).Len())
	}
	if movement <= .01 {
		t.Fatalf("phase did not change the actual skin: %g", movement)
	}
}

func TestSquidAnimalRecipeRefusesWrongOptions(t *testing.T) {
	name, raw, _ := EmbeddedAnimalBook("squid")
	book, err := ReadAnimalBook("squid", name, raw)
	if err != nil {
		t.Fatal(err)
	}
	zero, tooLarge, notFinite := 0.0, 100001.0, math.NaN()
	count, pose, bones := 0, "swimming", "bones"
	for _, option := range []AnimalOptions{
		{Species: "missing"}, {Species: "humboldt-squid", Phase: &tooLarge},
		{Species: "humboldt-squid", Phase: &notFinite}, {Species: "humboldt-squid", Seconds: &zero},
		{Species: "humboldt-squid", Count: &count}, {Species: "humboldt-squid", Pose: &pose},
		{Species: "humboldt-squid", Representation: &bones},
	} {
		if _, err := book.New(option); err == nil {
			t.Fatalf("accepted invalid option %+v", option)
		}
	}
	fishName, fishRaw, _ := EmbeddedAnimalBook("fish")
	fish, err := ReadAnimalBook("fish", fishName, fishRaw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fish.New(AnimalOptions{Species: "pacific-sardine", Phase: &zero}); err == nil {
		t.Fatal("fish accepted squid phase")
	}
}
