package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// tankNear is how near a measure of the tank is to the data's, in metres.
const tankNear = 1e-6

func TestTheCoreTankIsTheCoreStageOfTheSLSAlone(t *testing.T) {
	var core stackSection
	for _, row := range stacksRead {
		if row.Name != slsStackName {
			continue
		}
		for _, section := range row.Sections {
			if section.Part == slsCoreSection {
				core = section
			}
		}
	}
	if core.Length == 0 {
		t.Fatal("the SLS row has no core stage")
	}
	made := mustMake(t, slsCoreTankKind, 1)
	box := made.Bounds()
	if math.Abs(box.Max.Y-core.Length) > tankNear ||
		math.Abs(box.Max.X-core.Bottom/2) > tankNear ||
		math.Abs(box.Min.Y) > tankNear {
		t.Errorf("the tank spans %+v, want %v tall and %v across", box,
			core.Length, core.Bottom)
	}
	if got := len(made.slots()); got != 1 {
		t.Errorf("%d materials, want the core stage's one", got)
	}
	for _, look := range made.dressed {
		if look.Name != core.Part || look.Colour != core.made.Colour {
			t.Errorf("the tank is %q, not the core stage's paint", look.Name)
		}
	}
}

func TestTheCoreTankEnclosesAboutThePiRSquaredHOfItsSection(t *testing.T) {
	part, err := partOf(slsCoreTankKind, 1, maths.Identity, "quick")
	if err != nil {
		t.Fatal(err)
	}
	radius, length := 8.4/2, 64.6
	whole := math.Pi * radius * radius * length
	if got := part.piece.Volume; got > whole || got < whole*0.95 {
		t.Errorf("the tank encloses %v, a true cylinder %v", got, whole)
	}
}

func TestACoreTankNeedsAnSLSWithACoreStage(t *testing.T) {
	if _, err := coreTankOf(nil); err == nil {
		t.Error("a tank was made from no stacks")
	}
	bare := []stackRow{{Name: slsStackName,
		Sections: []stackSection{{Part: "upper stage"}}}}
	if _, err := coreTankOf(bare); err == nil {
		t.Error("a tank was made from an SLS with no core stage")
	}
}
