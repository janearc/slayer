package set

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestAnimalRecipeReadsExternalProfiles(t *testing.T) {
	dir := t.TempDir()
	for family, bookFamily := range map[string]string{"fish": "fish", "birds": "bird", "pinniped_profiles": "pinniped", "whale-profiles": "whale"} {
		_, raw, err := manufacture.EmbeddedAnimalBook(bookFamily)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, family+".yaml"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		family, file, species, pose string
		count                       int
		valid                       bool
	}{
		{"fish", "fish.yaml", "pacific-bluefin-tuna", "glide", 0, true},
		{"fish", "fish.yaml", "pacific-sardine", "swim", 20, true},
		{"bird", "birds.yaml", "grey-heron", "standing", 0, true},
		{"pinniped", "pinniped_profiles.yaml", "harbor-seal", "hauled", 0, true},
		{"pinniped", "pinniped_profiles.yaml", "walrus", "gliding", 0, false},
		{"whale", "whale-profiles.yaml", "orca", "swim", 0, true},
		{"whale", "whale-profiles.yaml", "orca", "spy-hop", 0, true},
		{"whale", "whale-profiles.yaml", "orca", "glide", 0, false},
		{"bird", "fish.yaml", "grey-heron", "standing", 0, false},
		{"fish", "fish.yaml", "absent", "swim", 0, false},
		{"bird", "birds.yaml", "grey-heron", "swim", 0, false},
		{"fish", "fish.yaml", "pacific-sardine", "bones", 20, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s-%s-%s-%d", tc.family, tc.species, tc.pose, tc.count), func(t *testing.T) {
			count := ""
			if tc.family == "fish" {
				count = fmt.Sprintf(", count: %d", tc.count)
			}
			source := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: animal
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: %s, file: %s, species: %s, pose: %s%s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.family, tc.file, tc.species, tc.pose, count)
			read, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
			if !tc.valid {
				if err == nil {
					t.Fatal("accepted invalid animal recipe")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			recipe, ok := read.Recipe("specimen")
			if !ok || !recipe.IsAnimal() || recipe.AnimalSpecies() != tc.species {
				t.Fatalf("recipe %#v, found %v", recipe, ok)
			}
			if _, err := recipe.Construction(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFormerAnimalKindsAreRejected(t *testing.T) {
	for _, tc := range []struct {
		kind  string
		valid bool
	}{
		{"can", true}, {"bird-grey-heron-standing", false}, {"pacific-sardine-school-500", false},
		{"bay-seal", false},
		{"harbor-seal", false},
		{"harbor-seal-dry", false},
		{"harbor-seal-bones", false},
		{"harbor-seal-hauled", false},
		{"harbor-seal-swimming", false},
		{"hawaiian-monk-seal-hauled", false},
		{"walrus-fresh-5", false},
		{"walrus-fresh-5-skeleton", false},
		{"walrus-fresh-5-volumes", false},
		{"walrus-fresh-5-hauled-out", false},
		{"walrus-fresh-5-swimming", false},
		{"orca-clean-05", false}, {"orca-clean-05-bones", false}, {"orca-clean-05-spy-hop", false},
	} {
		for _, schema := range []string{"", "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\n"} {
			source := schema + fmt.Sprintf(`scene:
  name: animal
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  set:
    things:
      one: {kind: %s, seed: 1, complexity: quick, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, tc.kind)
			if _, err := Read("old.yaml", []byte(source)); (err == nil) != tc.valid {
				t.Errorf("kind %s valid=%t: %v", tc.kind, tc.valid, err)
			}
		}
	}
}

func TestSchemaLessTheropodSecondsMustBeNumeric(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "manufacture", "animals", "vertebrates", "dinosaurs", "theropods", "allosaurus-profiles.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "allosaurs.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"0", true}, {"0.5", true}, {"null", false}, {"~", false}, {"'0.5'", false}, {".nan", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			source := fmt.Sprintf(`scene:
  name: animal
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: theropod, file: allosaurs.yaml, species: allosaurus, seconds: %s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, tc.value)
			_, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t: %v", tc.valid, err)
			}
		})
	}
}

func TestPinnipedRecipeSelectsRepresentationOrStudy(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "manufacture", "animals", "vertebrates", "mammals", "pinnipeds", "profiles.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pinnipeds.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, species, options string
		valid                  bool
	}{
		{"harbor bones", "harbor-seal", ", representation: bones", true},
		{"walrus bones", "walrus", ", representation: bones", true},
		{"walrus volumes", "walrus", ", representation: volumes, pose: lying", true},
		{"harbor wet", "harbor-seal", ", study: detailed-wet", true},
		{"harbor dry", "harbor-seal", ", study: detailed-dry", true},
		{"bay", "harbor-seal", ", study: bay", true},
		{"study with pose", "harbor-seal", ", study: bay, pose: hauled", false},
		{"study with representation", "harbor-seal", ", study: bay, representation: skin", false},
		{"skeleton with pose", "walrus", ", representation: bones, pose: lying", false},
		{"volumes swimming", "walrus", ", representation: volumes, pose: swimming", false},
		{"study absent from species", "walrus", ", study: bay", false},
		{"unsupported representation", "harbor-seal", ", representation: volumes", false},
		{"null study", "harbor-seal", ", study: null", false},
		{"empty study", "harbor-seal", ", study: ''", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// No schema label: the reader itself must enforce this contract.
			source := fmt.Sprintf(`scene:
  name: animal
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: pinniped, file: pinnipeds.yaml, species: %s%s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, tc.species, tc.options)
			read, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
			if err == nil {
				recipe, found := read.Recipe("specimen")
				if !found || !recipe.IsAnimal() {
					t.Fatal("lost checked animal recipe")
				}
				if _, err := recipe.Construction(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
