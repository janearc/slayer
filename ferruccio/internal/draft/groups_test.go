package draft

import (
	"strings"
	"testing"
)

func TestGroupsPreserveAuthoredIdentityWithoutMemberPlacements(t *testing.T) {
	source := strings.Replace(poseSource, "  world:\n", "    stands:\n      grove: {kind: cobble, count: 3, seed: 4}\n  world:\n", 1)
	document := poseDraft(t, source)
	groups, err := document.Groups()
	if err != nil || len(groups) != 1 || groups[0].ID != "scene.set.stands/grove" || groups[0].Name != "grove" {
		t.Fatalf("groups = %+v, %v", groups, err)
	}
	if err := document.ChangePose(PoseChange{ID: groups[0].ID, East: 1, LengthUnit: "m", AngleUnit: "deg"}); err == nil || document.Text() != source {
		t.Fatal("a generated group was treated as an editable individual")
	}
}

func TestTerrainGroupAreaEditIsAtomicAndPreservesNotes(t *testing.T) {
	source := strings.Replace(poseSource, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "  world:\n", "    stands:\n      grove:\n        kind: yurt\n        count: 3\n        seed: 4\n        habit: scatter\n        area: {west: 1, south: 2, east: 30, north: 40} # Keep region note.\n  world:\n", 1)
	document := poseDraft(t, source)
	id := "scene.set.stands/grove"
	if got, ok := document.GroupArea(id); !ok || got != [4]float64{1, 2, 30, 40} {
		t.Fatalf("area = %v, %v", got, ok)
	}
	if err := document.SetGroupArea(id, [4]float64{8, 9, 8, 50}); err == nil || document.Text() != source {
		t.Fatal("invalid region changed source")
	}
	want := [4]float64{10, 20, 80, 90}
	if err := document.SetGroupArea(id, want); err != nil {
		t.Fatal(err)
	}
	if got, ok := document.GroupArea(id); !ok || got != want {
		t.Fatalf("area = %v, %v", got, ok)
	}
	if !strings.Contains(document.Text(), "Keep region note") {
		t.Fatal("region comment lost")
	}
	reopened := poseDraft(t, document.Text())
	if got, ok := reopened.GroupArea(id); !ok || got != want {
		t.Fatalf("reopened area = %v, %v", got, ok)
	}
}

func TestGroupAreaOnlyForHabitsThatUseTerrainArea(t *testing.T) {
	source := strings.Replace(poseSource, "  world:\n", "    stands:\n      grove: {kind: yurt, count: 3, seed: 4, habit: scatter, area: {west: 1, south: 2, east: 30, north: 40}}\n  world:\n", 1)
	for _, text := range []string{source, strings.Replace(strings.Replace(source, "setting: space", "setting: ground", 1), "habit: scatter, ", "", 1)} {
		document := poseDraft(t, text)
		if _, ok := document.GroupArea("scene.set.stands/grove"); ok {
			t.Fatal("unsupported area offered")
		}
	}
}

func TestImplicitV2GroundOffersStandRegion(t *testing.T) {
	source := strings.Replace(poseSource, "  setting: space\n", "", 1)
	source = strings.Replace(source, "  world:\n", "    stands:\n      grove: {kind: yurt, count: 3, seed: 4, habit: scatter, area: {west: 1, south: 2, east: 30, north: 40}}\n  world:\n", 1)
	document := poseDraft(t, source)
	if _, ok := document.GroupArea("scene.set.stands/grove"); !ok {
		t.Fatal("default terrain did not expose its stand area")
	}
}

func TestSetGroupAreaReplacesHexadecimalCoordinates(t *testing.T) {
	source := strings.Replace(poseSource, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "  world:\n", `    stands:
      grove:
        kind: yurt
        count: 3
        seed: 4
        habit: scatter
        area: {west: 0x10, south: 2, east: 30, north: 40} # Keep region note.
  world:
`, 1)
	document := poseDraft(t, source)
	id := "scene.set.stands/grove"
	if err := document.Check(); err != nil {
		t.Fatalf("hexadecimal coordinate is valid YAML: %v", err)
	}
	want := [4]float64{10, 20, 80, 90}
	if err := document.SetGroupArea(id, want); err != nil {
		t.Fatal(err)
	}
	if got, ok := document.GroupArea(id); !ok || got != want {
		t.Fatalf("area = %v, %v", got, ok)
	}
	if !strings.Contains(document.Text(), "# Keep region note.") {
		t.Fatal("region comment lost")
	}
}
