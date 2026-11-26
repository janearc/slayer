package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/session/placement"
)

func TestSessionNamesEveryStaticPartOfAnAuthoredThing(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	source := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: a small space scene
  setting: space
  seed: 19
  light: none
  set:
    things:
      visitor: {kind: human-scale-2m, seed: 7, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  film: {width: 32, height: 24, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -6, 2]}
      orientation: {look_at: [0, 0, 1]}
`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	wanted, err := job.StillFrom(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	world, err := opened.inspected(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) == 0 {
		t.Fatal("fixture has no staged surfaces")
	}
	for _, part := range world.Things {
		author, ok := world.AuthorOf(part)
		if !ok || author.Collection != "scene.set.things" || author.Name != "visitor" {
			t.Fatalf("part identity = %+v %v", author, ok)
		}
	}
}

func TestGroundedPlacementsKeepDistinctAuthoredGroups(t *testing.T) {
	first := `        position:
          east: {value: 120, unit: m}
          north: {value: 180, unit: m}
          up: {value: 50, unit: m}`
	second := `        position:
          east: {value: 130, unit: m}
          north: {value: 180, unit: m}
          up: {value: 50, unit: m}`
	read := readPositionSet(t, positionedSetText("ground", first, second))
	stage := testStaging()
	stage.recipes = read
	stage.west, stage.north, stage.lowest = 100, 200, 30
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, part := range stage.world.Things {
		author, ok := stage.world.AuthorOf(part)
		if !ok || author.Collection != "scene.set.things" {
			t.Fatalf("grounded part identity = %+v %v", author, ok)
		}
		counts[author.Name]++
	}
	if counts["near"] == 0 || counts["far"] == 0 || len(counts) != 2 {
		t.Fatalf("grounded authors = %v", counts)
	}
}

func TestGeneratedStandPartsNameTheGroupNotIndividuals(t *testing.T) {
	read := readPositionSet(t, standSetText(""))
	stage := testStaging()
	stage.recipes = read
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, part := range stage.world.Things {
		author, ok := stage.world.AuthorOf(part)
		if !ok {
			t.Fatal("static stand part has no authored group")
		}
		switch author.Collection {
		case "scene.set.things":
			if author.Name != "ger" {
				t.Fatalf("unexpected thing %+v", author)
			}
		case "scene.set.stands":
			if author.Name != "the district" {
				t.Fatalf("invented stand member %+v", author)
			}
		default:
			t.Fatalf("unexpected authored collection %+v", author)
		}
		counts[author.Collection]++
	}
	if counts["scene.set.things"] == 0 || counts["scene.set.stands"] == 0 {
		t.Fatalf("groups: %v", counts)
	}
	shared, err := placement.WorkingInstances(stage.world, 0)
	if err != nil {
		t.Fatal(err)
	}
	foundStand := false
	for _, batch := range shared.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.stands" {
			if batch.Author.Name != "the district" {
				t.Fatalf("shared batch invented a stand member: %+v", batch.Author)
			}
			foundStand = true
		}
	}
	if !foundStand {
		t.Fatal("shared view lost the generated stand's authored group")
	}
}
