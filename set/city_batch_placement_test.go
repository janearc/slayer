package set_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/set"
)

func TestCityBatchPlacementRecipeBuildsBothCatalogueCities(t *testing.T) {
	for _, name := range []string{"city-grid-10000", "city-clustered-10000"} {
		t.Run(name, func(t *testing.T) {
			runtime.GC()
			source, err := os.ReadFile("../doc/" + name + ".yaml")
			if err != nil {
				t.Fatal(err)
			}
			configured := cityPlacementSource(t, source, name, "nested")
			scene, err := set.Read(name+"-configured.yaml", configured)
			if err != nil {
				t.Fatal(err)
			}
			if len(scene.Things()) != 1 || scene.Things()[0].Kind() != name ||
				scene.Things()[0].Recipe() != "city placement" {
				t.Fatalf("resolved thing provenance: %+v", scene.Things())
			}
			recipe, found := scene.Recipe("city placement")
			if !found || recipe.Kind() != name || recipe.BatchPlacement() != manufacture.CityBatchNested {
				t.Fatalf("configured recipe: %+v, found %v", recipe, found)
			}
			row, err := recipe.Construction()
			if err != nil {
				t.Fatal(err)
			}
			made, err := manufacture.MakeWithComplexity(*row,
				manufacture.Sow(31), "quick")
			if err != nil {
				t.Fatal(err)
			}
			things, err := made.Things(maths.Local())
			if err != nil {
				t.Fatal(err)
			}
			meshes, faces := 0, 0
			for _, thing := range things {
				group, ok := thing.Shape.(*shape.MeshGroup)
				if !ok {
					t.Fatalf("city item is %T, want material mesh group", thing.Shape)
				}
				meshes += len(group.Members())
				faces += group.StoredFaces()
			}
			wantFaces := map[string]int{"city-grid-10000": 916758,
				"city-clustered-10000": 970316}[name]
			if len(things) != 9 || meshes != 21 || faces != wantFaces {
				t.Fatalf("configured %s: %d materials, %d batches, %d faces", name, len(things), meshes, faces)
			}
		})
	}
}

func TestCityPlacementSourceIsPartOfJobProvenance(t *testing.T) {
	source, err := os.ReadFile("../doc/city-grid-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	setPath := filepath.Join(directory, "city.yaml")
	jobSource, err := os.ReadFile("../doc/examples/jobs/city-grid-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(directory, "city.job.yaml")
	jobBytes := strings.Replace(string(jobSource), "../../../doc/city-grid-10000.yaml", "city.yaml", 1)
	if jobBytes == string(jobSource) {
		t.Fatal("job set reference was not replaced")
	}
	if err := os.WriteFile(jobPath, []byte(jobBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	write := func(mode string) {
		t.Helper()
		configured := cityPlacementSource(t, source, "city-grid-10000", mode)
		if err := os.WriteFile(setPath, configured, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("nested")
	nested, err := job.Read(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	resolved, has := nested.Scene().Set()
	if !has || resolved != setPath || nested.Inputs()["set"] != setPath {
		t.Fatalf("job set provenance: %q, %v, inputs %v", resolved, has, nested.Inputs())
	}
	nestedKey := nested.Key()
	write("fused")
	fused, err := job.Read(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	if nestedKey == fused.Key() {
		t.Fatal("changing the set's city batch mode did not change the job key")
	}
}

func TestCanonicalCitySetsAndJobsKeepDirectKinds(t *testing.T) {
	for _, name := range []string{"city-grid-10000", "city-clustered-10000"} {
		scenePath := "../doc/" + name + ".yaml"
		source, err := os.ReadFile(scenePath)
		if err != nil {
			t.Fatal(err)
		}
		scene, err := set.Read(scenePath, source)
		if err != nil {
			t.Fatal(err)
		}
		things := scene.Things()
		if len(things) != 1 || things[0].Kind() != name || things[0].Recipe() != "" {
			t.Fatalf("canonical %s thing changed: %+v", name, things)
		}
		picture, err := job.Read("../doc/examples/jobs/" + name + ".yaml")
		if err != nil {
			t.Fatal(err)
		}
		resolved, has := picture.Scene().Set()
		absolute, err := filepath.Abs(scenePath)
		if err != nil {
			t.Fatal(err)
		}
		if !has || resolved != absolute || picture.Inputs()["set"] != absolute {
			t.Fatalf("canonical %s job set reference: %q, %v, %v", name, resolved, has, picture.Inputs())
		}
	}
}

func TestCityBatchPlacementRecipeRefusesUnsupportedContexts(t *testing.T) {
	source, err := os.ReadFile("../doc/city-grid-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	valid := cityPlacementSource(t, source, "city-grid-10000", "nested")
	wrongKind := strings.Replace(string(valid), "kind: city-grid-10000\n      batch-placement:", "kind: boulder\n      batch-placement:", 1)
	unknownMode := strings.Replace(string(valid), "batch-placement: nested", "batch-placement: sideways", 1)
	directOption := strings.Replace(string(source), "        kind: city-grid-10000", "        kind: city-grid-10000\n        batch-placement: nested", 1)
	wrongContext := strings.Replace(string(valid),
		"    things:\n      city:\n        recipe: city placement\n        seed: 31\n        complexity: quick\n        east: 0\n        north: 0\n        height: {value: 0, unit: m}",
		"    stands:\n      city:\n        recipe: city placement\n        count: 1\n        seed: 31", 1)
	for name, raw := range map[string]string{
		"non-city recipe":  wrongKind,
		"unknown mode":     unknownMode,
		"direct thing key": directOption,
		"stand use":        wrongContext,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := set.Read("unsupported-city-mode.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted unsupported batch placement context")
			}
		})
	}
}

func cityPlacementSource(t *testing.T, source []byte, kind, mode string) []byte {
	t.Helper()
	raw := string(source)
	raw = strings.Replace(raw, "  set:\n", "  recipes:\n    city placement:\n      kind: "+kind+"\n      batch-placement: "+mode+"\n  set:\n", 1)
	raw = strings.Replace(raw, "        kind: "+kind, "        recipe: city placement", 1)
	if !strings.Contains(raw, "      batch-placement: "+mode) || !strings.Contains(raw, "        recipe: city placement") {
		t.Fatal("failed to configure city recipe in source fixture")
	}
	return []byte(raw)
}
