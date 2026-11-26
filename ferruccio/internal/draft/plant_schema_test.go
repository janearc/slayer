package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetPlantSchemasAcceptExternalProfilesAndRefuseInvalidRecipes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "set", "testdata", "gate-composition-set.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.Replace(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json", 1)
		if version == "v1" {
			source = strings.Replace(source, "  setting: space\n", "", 1)
		}
		for _, tc := range []struct {
			name, recipe string
			valid        bool
		}{
			{"plant", "plant: {file: plants.yaml, species: valley-oak}", true},
			{"season", "plant: {file: plants.yaml, species: valley-oak}\n      season: autumn", true},
			{"seasonal family", "plant: {family: seasonal, file: plants.yaml, species: valley-oak}", true},
			{"conifer family", "plant: {family: conifer, file: plants.yaml, species: spruce}", true},
			{"conifer shoot", "plant: {family: conifer, file: plants.yaml, species: coast-redwood, representation: shoot}", true},
			{"shoot without family", "plant: {file: plants.yaml, species: valley-oak, representation: shoot}", false},
			{"fern shoot", "plant: {family: fern, file: plants.yaml, species: fern, representation: shoot}", false},
			{"shoot season", "plant: {family: conifer, file: plants.yaml, species: coast-redwood, representation: shoot}\n      season: autumn", false},
			{"unknown representation", "plant: {family: conifer, file: plants.yaml, species: coast-redwood, representation: trunk}", false},
			{"fern family", "plant: {family: fern, file: plants.yaml, species: fern}", true},
			{"cactus family", "plant: {family: cactus, file: plants.yaml, species: saguaro}", true},
			{"unknown family", "plant: {family: flower, file: plants.yaml, species: fern}", false},
			{"conifer season", "plant: {family: conifer, file: plants.yaml, species: spruce}\n      season: autumn", false},
			{"fern season", "plant: {family: fern, file: plants.yaml, species: fern}\n      season: autumn", false},
			{"cactus season", "plant: {family: cactus, file: plants.yaml, species: saguaro}\n      season: autumn", false},
			{"missing species", "plant: {file: plants.yaml}", false},
			{"empty file", "plant: {file: '', species: valley-oak}", false},
			{"unknown field", "plant: {file: plants.yaml, species: valley-oak, extra: yes}", false},
			{"mixed composition", "plant: {file: plants.yaml, species: valley-oak}\n      composition: {file: other.yaml}", false},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				text := strings.Replace(source, "composition: {file: gate-composition.yaml}", tc.recipe, 1)
				document, err := Open("gate.yaml", []byte(text))
				if err != nil {
					t.Fatal(err)
				}
				err = document.Check()
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v, check=%v", tc.valid, err)
				}
			})
		}
	}
}

func TestSetSchemasRejectFormerPlantKinds(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "set", "testdata", "gate-composition-set.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	kinds := []string{"spruce", "norway", "coast-redwood", "giant-sequoia", "coast-redwood-shoot", "giant-sequoia-shoot", "fern", "saguaro", "valley-oak", "alder", "poplar", "manzanita", "chaparral-brush", "joshua-tree", "kandelia-obovata", "aegiceras-corniculatum", "avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala", "pandanus-tectorius", "scaevola-taccada", "metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex", "erica-arborea", "salix-alba"}
	const original = "  set:\n    things:\n      gate:\n        recipe: gate\n        seed: 31\n        complexity: hull\n        east: 0\n        north: 0\n        height: {value: 0, unit: m}\n"
	if strings.Count(string(raw), original) != 1 {
		t.Fatal("fixture no longer has the expected scene object")
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.Replace(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json", 1)
		if version == "v1" {
			source = strings.Replace(source, "  setting: space\n", "", 1)
		}
		for _, kind := range append([]string{"can"}, kinds...) {
			for _, location := range []string{"recipe", "thing", "stand"} {
				t.Run(version+"/"+location+"/"+kind, func(t *testing.T) {
					text := source
					switch location {
					case "recipe":
						text = strings.Replace(text, "composition: {file: gate-composition.yaml}", "kind: "+kind, 1)
					case "thing":
						text = strings.Replace(text, "        recipe: gate\n", "        kind: "+kind+"\n", 1)
					case "stand":
						text = strings.Replace(text, original, "  set:\n    stands:\n      grove: {kind: "+kind+", count: 1, seed: 31}\n", 1)
					}
					doc, err := Open("old.yaml", []byte(text))
					if err != nil {
						t.Fatal(err)
					}
					err = doc.Check()
					if (err == nil) != (kind == "can") {
						t.Fatalf("kind %s: %v", kind, err)
					}
				})
			}
		}
	}
}
