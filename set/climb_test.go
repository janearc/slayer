package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// steepCoast is the set of the steep coast, which says a climb, read from
// the sets directory.
func steepCoast(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../sets/terraces-on-a-steep-coast.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAStandOfTheContourSaysItsClimb(t *testing.T) {
	source := steepCoast(t)
	read, err := set.Read("coast.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	town := read.Stands()[0]
	climb, climbs := town.Climb()
	if !climbs || town.Habit() != "contour" {
		t.Fatalf("the town has the habit %q and a climb is %v", town.Habit(),
			climbs)
	}
	if climb.Rows != 4 || climb.Runs != 2 || climb.Lane != 4 || climb.Uphill {
		t.Errorf("the climb is %+v, not four rows of two runs, a 4 m lane "+
			"and doors downhill", climb)
	}
}

func TestADoorThatLooksUphillIsSaid(t *testing.T) {
	text := strings.Replace(steepCoast(t), "door: downhill", "door: uphill", 1)
	read, err := set.Read("coast.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if climb, _ := read.Stands()[0].Climb(); !climb.Uphill {
		t.Error("a door that looks uphill was read as one that looks down")
	}
}

func TestAClimbThatIsNotWholeOrNotSaidIsRefused(t *testing.T) {
	for name, edit := range map[string][2]string{
		"a door sideways": {"door: downhill", "door: sideways"},
		"half a row":      {"rows: 4", "rows: 2.5"},
		"no rows":         {"rows: 4", "rows: 0"},
		"no door":         {"          door: downhill\n", ""},
		"a lane in feet":  {"{value: 4, unit: m}", "{value: 4, unit: ft}"},
		"a key of no one": {"rows: 4", "rows: 4\n          storeys: 3"},
	} {
		text := strings.Replace(steepCoast(t), edit[0], edit[1], 1)
		if _, err := set.Read("coast.yaml", []byte(text)); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}

func TestTheContourAndTheClimbGoTogether(t *testing.T) {
	bare := strings.Replace(steepCoast(t), "        climb:\n"+
		"          rows: 4\n          runs: 2\n"+
		"          lane: {value: 4, unit: m}\n"+
		"          door: downhill\n", "", 1)
	if _, err := set.Read("coast.yaml", []byte(bare)); err == nil {
		t.Error("a stand of the contour with no climb was read")
	}
	lattice := strings.Replace(steepCoast(t), "habit: contour",
		"habit: lattice", 1)
	if _, err := set.Read("coast.yaml", []byte(lattice)); err == nil {
		t.Error("a climb on a lattice was read")
	}
}
