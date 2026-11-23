package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// highest is how far up the highest corner of any drawn part stands,
// after each is placed on the building.
func highest(drawn []Rendered) float64 {
	top := math.Inf(-1)
	for _, one := range drawn {
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				top = math.Max(top, one.Place.Out(corner).Y)
			}
		}
	}
	return top
}

// mustChoose is a named choice, or the test's end.
func mustChoose(t *testing.T, name string) float64 {
	t.Helper()
	value, err := choice(name)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAYurtIsAWallARoofACrownAndADoorEachOfItsRole(t *testing.T) {
	design, err := yurt()
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	roles := []partRole{roleFrame, roleCover, roleCrown, roleDoor}
	if len(drawn) != len(roles) {
		t.Fatalf("%d parts, want %d", len(drawn), len(roles))
	}
	for index, role := range roles {
		want, err := material(role)
		if err != nil {
			t.Fatal(err)
		}
		if drawn[index].Material != want {
			t.Errorf("part %d is not made of %s", index, role)
		}
	}
}

func TestAYurtsSizesAreTheDatas(t *testing.T) {
	design, _ := yurt()
	drawn, err := design.Rendered("demo")
	if err != nil {
		t.Fatal(err)
	}
	crown := mustChoose(t, "roof-top") + mustChoose(t, "crown-rise")
	if got := highest(drawn); math.Abs(got-crown) > tolerance {
		t.Errorf("the crown stands %v high, data says %v", got, crown)
	}
	wall := drawn[0]
	for _, triangle := range wall.Mesh.Triangles {
		for _, corner := range triangle {
			reach := math.Hypot(corner.X, corner.Z)
			if reach > mustChoose(t, "wall-radius")+tolerance {
				t.Fatalf("the wall reaches %v out", reach)
			}
		}
	}
}

func TestTheDoorStandsOnTheWallFacingSouth(t *testing.T) {
	design, _ := yurt()
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	door := drawn[len(drawn)-1]
	west, east := math.Inf(1), math.Inf(-1)
	south := math.Inf(-1)
	for _, triangle := range door.Mesh.Triangles {
		for _, corner := range triangle {
			placed := door.Place.Out(corner)
			west, east = math.Min(west, placed.X), math.Max(east, placed.X)
			south = math.Max(south, placed.Z)
		}
	}
	farthest := mustChoose(t, "wall-radius") + mustChoose(t, "door-thickness")/2
	if math.Abs(west+east) > tolerance || math.Abs(south-farthest) > tolerance {
		t.Errorf("the door spans %v to %v and reaches %v south, want "+
			"centred and %v", west, east, south, farthest)
	}
}

func TestTheFeltIsOpaqueDiffuseCloth(t *testing.T) {
	felt, err := material(roleCover)
	if err != nil {
		t.Fatal(err)
	}
	if felt.Through != 0 || felt.Glow != (maths.Light{}) || felt.Gloss != 0 {
		t.Errorf("the felt transmits, glows or shines: %+v", felt)
	}
}

func TestAMaterialIsTheSameEachTimeItIsAskedFor(t *testing.T) {
	one, _ := material(roleCover)
	other, _ := material(roleCover)
	if one != other || one == nil {
		t.Error("two asks for the cover gave two materials")
	}
}

func TestARoleNothingIsMadeOfIsRefused(t *testing.T) {
	if _, err := material("thatch"); err == nil {
		t.Error("a role with no data had a material")
	}
}

func TestDataThatIsNotWholeIsRefused(t *testing.T) {
	const provenance = "  kind: picked\n  by: test\n"
	for name, raw := range map[string]string{
		"a part of a side": "numbers:\n- name: circle-sides-quick\n  value: 2.5\n" +
			"  says: a ring\n" + provenance,
		"too few sides": "numbers:\n- name: circle-sides-quick\n  value: 2\n" +
			"  says: a ring\n" + provenance,
		"no one who picked": "numbers:\n- name: wall-radius\n  value: 3\n" +
			"  says: a wall\n  kind: picked\n",
	} {
		if _, err := readChoices("test.yaml", []byte(raw)); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}

func TestARoleWithoutAGlossIsRefused(t *testing.T) {
	const red = "numbers:\n- name: cover-red\n  value: 0.5\n  says: red\n" +
		"  kind: picked\n  by: test\n"
	book, err := readChoices("test.yaml", []byte(red))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readMaterials(book); err == nil {
		t.Error("a role with a red and nothing else was made")
	}
}

func TestADesignWithAnElementThatFailsFails(t *testing.T) {
	design := Design{Name: "bare", Elements: []placedElement{{Element: door{}}}}
	if _, err := design.Rendered("quick"); err == nil {
		t.Error("a door with no size was drawn")
	}
}

func TestADesignWithAMassOfNoRoleFails(t *testing.T) {
	design := Design{Name: "bare", Massing: []mass{{Name: "slab",
		Form: Box{1, 1, 1}, Role: "thatch"}}}
	if _, err := design.Rendered("quick"); err == nil {
		t.Error("a mass of a role nothing is made of was drawn")
	}
}
