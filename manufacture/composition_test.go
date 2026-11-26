package manufacture

import (
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

func TestCompositionKindKeepsItsCheckedParts(t *testing.T) {
	spec := cities.Composition{Version: 1, Name: "column", Parts: []cities.CompositionPart{
		{ID: "leg", Type: "leg", At: [3]float64{0, 0, 0}, Size: [3]float64{2, 3, 2}, Ground: true},
		{ID: "body", Type: "body", At: [3]float64{0, 3, 0}, Size: [3]float64{2, 4, 2}, RestsOn: []string{"leg"}},
	}}
	row, err := Composition(spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.Parts[0].Size[1] = 1
	spec.Parts[1].RestsOn[0] = "missing"
	if _, err := MakeWithComplexity(row, Sow(1), "hull"); err != nil {
		t.Fatal(err)
	}
	if _, err := Make(row, Sow(1)); err != nil {
		t.Fatal(err)
	}
	box, err := ExtentOf(row, Sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if box.Max.Y < 7 {
		t.Fatalf("extent lost body: %+v", box)
	}
	if _, err := Composition(spec); err == nil {
		t.Fatal("accepted detached body")
	}
}
