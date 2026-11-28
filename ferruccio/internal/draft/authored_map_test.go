package draft

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMapStandMarksUseSourceCentresAndKnownAnchors(t *testing.T) {
	var tree yaml.Node
	err := yaml.Unmarshal([]byte(`
rectangle: {area: {west: 10, south: 20, east: 50, north: 80}}
referenced: {within: {of: inlet}}
unknown: {within: {of: missing}}
unspecified: {}
`), &tree)
	if err != nil {
		t.Fatal(err)
	}
	anchors := map[string]MapMark{"inlet": {East: 300, North: 400}}
	got := appendStandMarks(nil, tree.Content[0], anchors, &MapMark{East: 200, North: 320})
	area := MapRectangle{West: 210, South: 340, East: 250, North: 400}
	if len(got) == 0 || got[0].Area == nil || *got[0].Area != area {
		t.Fatalf("stand area = %v, want %v", got, area)
	}
	got[0].Area = nil
	want := []MapMark{
		{ID: "scene.set.stands/rectangle", Name: "rectangle", East: 230, North: 370, Group: true},
		{ID: "scene.set.stands/referenced", Name: "referenced", East: 300, North: 400, Group: true},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("map stand marks = %v, want %v", got, want)
	}
	withoutOrigin := appendStandMarks(nil, tree.Content[0], anchors, nil)
	if len(withoutOrigin) != 1 || withoutOrigin[0] != want[1] {
		t.Fatalf("map stand marks without known window = %v", withoutOrigin)
	}
}

func TestMapTerrainRectangleUsesSampledExtent(t *testing.T) {
	for _, trial := range []struct {
		yaml string
		want *MapRectangle
	}{
		{"{west: 600, north: 900, across: 65, down: 105, step: 10}", &MapRectangle{600, 800, 660, 900}},
		{"{lat: 22, lon: 114, across: 65, down: 105, step: 10}", &MapRectangle{0, 5, 60, 105}},
		{"{west: 0x258, north: 0x384, across: 0xFA0, down: 0x69, step: 0xA}", &MapRectangle{600, 800, 4600, 900}},
		{"{lat: 22, lon: 114, across: .inf, down: 105, step: 10}", nil},
		{"{lat: .nan, lon: 114, across: 65, down: 105, step: 10}", nil},
		{"{west: null, north: 900, across: 65, down: 105, step: 10}", nil},
		{"{west: 600, north: 900, across: 0x186A1, down: 105, step: 10}", nil},
		{"{lat: 22, lon: 114, across: 8, down: 105, step: 10}", nil},
	} {
		var tree yaml.Node
		if err := yaml.Unmarshal([]byte(trial.yaml), &tree); err != nil {
			t.Fatal(err)
		}
		terrain := tree.Content[0]
		got := mapTerrainRectangle(terrain, mapTerrainOrigin(terrain))
		if (got == nil) != (trial.want == nil) || got != nil && *got != *trial.want {
			t.Fatalf("terrain %s gives %+v, want %+v", trial.yaml, got, trial.want)
		}
	}
}

func TestMapTerrainOriginMatchesStagedLastRow(t *testing.T) {
	for _, trial := range []struct {
		yaml        string
		east, north float64
	}{
		{"{west: 600, north: 900, down: 105, step: 10}", 600, 800},
		{"{lat: 22, lon: 114, down: 105, step: 10}", 0, 5},
	} {
		var tree yaml.Node
		if err := yaml.Unmarshal([]byte(trial.yaml), &tree); err != nil {
			t.Fatal(err)
		}
		origin := mapTerrainOrigin(tree.Content[0])
		if origin == nil || origin.East != trial.east || origin.North != trial.north {
			t.Fatalf("terrain %s gives %+v, want %v east and %v north", trial.yaml, origin, trial.east, trial.north)
		}
	}
}
