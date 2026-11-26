package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetAnimalSchemas(t *testing.T) {
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
			label, recipe string
			valid         bool
		}{
			{"fish", "animal: {family: fish, file: fish.yaml, species: pacific-sardine, pose: swim, count: 20}", true},
			{"bird", "animal: {family: bird, file: birds.yaml, species: grey-heron, pose: standing}", true},
			{"pinniped", "animal: {family: pinniped, file: pinnipeds.yaml, species: harbor-seal, pose: hauled}", true},
			{"theropod", "animal: {family: theropod, file: allosaurus.yaml, species: allosaurus, seconds: 0.5}", true},
			{"snake static", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica}", true},
			{"snake pose", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica, pose: lying}", false},
			{"snake count", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica, count: 1}", false},
			{"snake seconds", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica, seconds: 0}", false},
			{"snake phase", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica, phase: 0}", false},
			{"snake representation", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica, representation: skin}", false},
			{"snake study", "animal: {family: snake, file: snake.yaml, species: bitis-gabonica, study: bay}", false},
			{"sauropod static", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis}", true},
			{"sauropod pose", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis, pose: lying}", false},
			{"sauropod count", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis, count: 1}", false},
			{"sauropod seconds", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis, seconds: 0}", false},
			{"sauropod phase", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis, phase: 0}", false},
			{"sauropod representation", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis, representation: skin}", false},
			{"sauropod study", "animal: {family: sauropod, file: sauropod.yaml, species: alamosaurus-sanjuanensis, study: bay}", false},
			{"fish default", "animal: {family: fish, file: f.yaml, species: x}", true},
			{"fish bones count zero", "animal: {family: fish, file: f.yaml, species: x, pose: bones, count: 0}", true},
			{"fish school default pose", "animal: {family: fish, file: f.yaml, species: x, count: 20}", true},
			{"bird default", "animal: {family: bird, file: f.yaml, species: x}", true},
			{"bird swim", "animal: {family: bird, file: f.yaml, species: x, pose: swim}", false},
			{"bird count", "animal: {family: bird, file: f.yaml, species: x, count: 0}", false},
			{"bird seconds", "animal: {family: bird, file: f.yaml, species: x, seconds: 0}", false},
			{"pinniped gliding", "animal: {family: pinniped, file: f.yaml, species: x, pose: gliding}", false},
			{"pinniped seconds", "animal: {family: pinniped, file: f.yaml, species: x, seconds: 0}", false},
			{"pinniped count", "animal: {family: pinniped, file: f.yaml, species: x, count: 0}", false},
			{"theropod pose", "animal: {family: theropod, file: f.yaml, species: x, pose: standing}", false},
			{"theropod count", "animal: {family: theropod, file: f.yaml, species: x, count: 0}", false},
			{"fish time", "animal: {family: fish, file: f.yaml, species: x, seconds: 0}", false},
			{"fish standing", "animal: {family: fish, file: f.yaml, species: x, pose: standing}", false},
			{"fish school bones", "animal: {family: fish, file: f.yaml, species: x, pose: bones, count: 20}", false},
			{"fish school glide", "animal: {family: fish, file: f.yaml, species: x, pose: glide, count: 20}", false},
			{"pinniped bones", "animal: {family: pinniped, file: p.yaml, species: x, representation: bones}", true},
			{"pinniped volumes", "animal: {family: pinniped, file: p.yaml, species: x, representation: volumes, pose: lying}", true},
			{"pinniped study", "animal: {family: pinniped, file: p.yaml, species: x, study: bay}", true},
			{"study with pose", "animal: {family: pinniped, file: p.yaml, species: x, study: bay, pose: hauled}", false},
			{"study with representation", "animal: {family: pinniped, file: p.yaml, species: x, study: bay, representation: skin}", false},
			{"bones with pose", "animal: {family: pinniped, file: p.yaml, species: x, representation: bones, pose: lying}", false},
			{"volumes swimming", "animal: {family: pinniped, file: p.yaml, species: x, representation: volumes, pose: swimming}", false},
			{"empty study", "animal: {family: pinniped, file: p.yaml, species: x, study: ''}", false},
			{"null study", "animal: {family: pinniped, file: p.yaml, species: x, study: null}", false},
			{"bird study", "animal: {family: bird, file: p.yaml, species: x, study: bay}", false},
			{"fish representation", "animal: {family: fish, file: p.yaml, species: x, representation: bones}", false},
			{"missing species", "animal: {family: fish, file: fish.yaml}", false},
			{"unknown family", "animal: {family: imaginary, file: f.yaml, species: x}", false},
			{"mixed", "animal: {family: fish, file: f.yaml, species: x}\n      kind: fire", false},
		} {
			t.Run(version+"/"+tc.label, func(t *testing.T) {
				text := strings.Replace(source, "composition: {file: gate-composition.yaml}", tc.recipe, 1)
				doc, err := Open("gate.yaml", []byte(text))
				if err != nil {
					t.Fatal(err)
				}
				err = doc.Check()
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v check=%v", tc.valid, err)
				}
			})
		}
	}
}

func TestSetSchemasRejectFormerAnimalKinds(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "set", "testdata", "gate-composition-set.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
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
		} {
			text := strings.Replace(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json", 1)
			if version == "v1" {
				text = strings.Replace(text, "  setting: space\n", "", 1)
			}
			text = strings.Replace(text, "composition: {file: gate-composition.yaml}", "kind: "+tc.kind, 1)
			doc, err := Open("old.yaml", []byte(text))
			if err != nil {
				t.Fatal(err)
			}
			if err := doc.Check(); (err == nil) != tc.valid {
				t.Errorf("%s kind %s valid=%t: %v", version, tc.kind, tc.valid, err)
			}
		}
	}
}
