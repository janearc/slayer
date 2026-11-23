package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// walled is the number of wall parts drawn at the demo level that have a
// texture of the type a test asks about.
func walled[T any](drawn []Rendered) int {
	found := 0
	for _, one := range drawn {
		if _, is := one.Material.Texture.(T); is && one.Mesh.Name == "wall" {
			found++
		}
	}
	return found
}

func TestGroningenIsTimberLapAndMidwestIsVinylAndNeitherIsStucco(
	t *testing.T) {
	for culture, want := range map[string]string{"groningen": "timber",
		"midwest": "vinyl"} {
		drawn := terraceAt(t, culture, 1, "demo")
		if walled[lap](drawn) == 0 {
			t.Errorf("%s (%s): no wall in courses of boards", culture, want)
		}
		for _, one := range drawn {
			if one.Mesh.Name == "wall" && one.Material.Crinkle != nil {
				t.Fatalf("%s: a wall with a stucco bump", culture)
			}
		}
	}
}

func TestACondominiumMayBeBrickAndAPortofinoWallIsStillRender(t *testing.T) {
	bricked := 0
	for seed := range Seed(12) {
		design, err := CondominiumAt(seed, nil, "demo")
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := design.Rendered("demo")
		if err != nil {
			t.Fatal(err)
		}
		bricked += walled[brick](drawn)
	}
	if bricked == 0 {
		t.Error("no condominium of twelve has a brick wall")
	}
	if walled[lap](terraceAt(t, "portofino", 3, "demo")) != 0 {
		t.Error("a Portofino wall is in courses of boards")
	}
}

func TestBrickIsInCoursesWithMortarBetweenAndEachCourseOffsetByHalf(
	t *testing.T) {
	brick := brick{Brick: maths.RGB(0.5, 0.2, 0.1), Mortar: maths.RGB(0.4, 0.4,
		0.4), Length: 0.2, Height: 0.06, Joint: 0.01}
	if got := brick.At(maths.V(0.1, 0.03, 0)); got == brick.Mortar {
		t.Error("the middle of a brick is mortar")
	}
	if got := brick.At(maths.V(0.205, 0.03, 0)); got != brick.Mortar {
		t.Error("a joint between two bricks is not mortar")
	}
	if got := brick.At(maths.V(0.1, 0.065, 0)); got != brick.Mortar {
		t.Error("a joint between two courses is not mortar")
	}
	if got := brick.At(maths.V(0.205, 0.10, 0)); got == brick.Mortar {
		t.Error("the course above is not offset by half a brick")
	}
	if brick.At(maths.V(0.1, 0.03, 0)) != brick.At(maths.V(0, 0.03, 0.1)) {
		t.Error("a wall facing along x reads differently from one along z")
	}
}

func TestBrickSaysWhereItsJointsAre(t *testing.T) {
	brick := brick{Brick: maths.RGB(0.5, 0.2, 0.1), Mortar: maths.RGB(0.4, 0.4,
		0.4), Length: 0.2, Height: 0.06, Joint: 0.01}
	for _, at := range []maths.Vec{maths.V(0.1, 0.03, 0),
		maths.V(0.205, 0.03, 0), maths.V(0.1, 0.065, 0),
		maths.V(0.205, 0.10, 0)} {
		if brick.InJoint(at) != (brick.At(at) == brick.Mortar) {
			t.Errorf("at %v InJoint disagrees with At", at)
		}
	}
}

func TestLapHasAShadowLineAtTheFootOfEachCourse(t *testing.T) {
	lap := lap{Board: maths.RGB(0.6, 0.6, 0.6), Height: 0.16, Shadow: 0.5,
		Share: 0.2}
	foot, face := lap.At(maths.V(0, 0.01, 0)), lap.At(maths.V(0, 0.1, 0))
	if !(foot.R < face.R) || math.Abs(foot.R-0.5*face.R) > tolerance {
		t.Errorf("shadow line %v, board face %v", foot, face)
	}
}

func TestAGarageDoorIsSectionsWithRaisedPanelsAtTheDemoLevel(t *testing.T) {
	sections, panels := 0, 0
	for seed := range Seed(detailSeeds) {
		drawn := terraceAt(t, "midwest", seed, "demo")
		sections += len(named(drawn, "garage section"))
		panels += len(named(drawn, "garage panel"))
		for _, level := range []Detail{"hull", "quick"} {
			terrace := terraceAt(t, "midwest", seed, level)
			if len(named(terrace,
				"garage section")) != 0 {
				t.Fatalf("a garage section at %s", level)
			}
		}
	}
	if sections == 0 || panels != 2*sections {
		t.Errorf("%d sections and %d panels, two to a section wanted",
			sections, panels)
	}
}
