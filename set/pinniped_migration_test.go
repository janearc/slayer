package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestPinnipedScenesUseCheckedSpeciesAndOptions(t *testing.T) {
	for _, tc := range []struct{ file, species, option string }{
		{"bioforms/bay-seal.yaml", "harbor-seal", "study: bay"},
		{"bioforms/harbor-seal.yaml", "harbor-seal", "study: detailed-wet"},
		{"bioforms/harbor-seal-bones.yaml", "harbor-seal", "representation: bones"},
		{"bioforms/harbor-seal-hauled.yaml", "harbor-seal", "pose: hauled"},
		{"bioforms/harbor-seal-sea.yaml", "harbor-seal", "pose: swimming"},
		{"bioforms/harbor-seal-swimming.yaml", "harbor-seal", "pose: swimming"},
		{"bioforms/hawaiian-monk-seal.yaml", "hawaiian-monk-seal", "pose: hauled"},
		{"bioforms/walrus-fresh-5.yaml", "walrus", "pose: lying"},
		{"bioforms/walrus-fresh-5-hauled-out.yaml", "walrus", "pose: hauled-out"},
		{"bioforms/walrus-fresh-5-swimming.yaml", "walrus", "pose: swimming"},
		{"bioforms/walrus-fresh-5-skeleton.yaml", "walrus", "representation: bones"},
		{"bioforms/walrus-fresh-5-volumes.yaml", "walrus", "representation: volumes"},
		{"bioforms/walrus-outline.yaml", "walrus", "pose: lying"},
		{"bioforms/walrus-outline-raised.yaml", "walrus", "pose: hauled-out"},
		{"kelp-forest.yaml", "harbor-seal", "study: bay"},
		{"kelp-forest-bubbles.yaml", "harbor-seal", "study: bay"},
		{"gpu-tube-study.yaml", "harbor-seal", "study: bay"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			path := "../sets/" + tc.file
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), tc.option) {
				t.Fatalf("missing option %s", tc.option)
			}
			read, err := set.Read(path, raw)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, each := range read.Things() {
				if each.Recipe() == "" {
					continue
				}
				recipe, ok := read.Recipe(each.Recipe())
				if !ok || !recipe.IsAnimal() || recipe.AnimalSpecies() != tc.species {
					continue
				}
				if _, err := recipe.Construction(); err != nil {
					t.Fatal(err)
				}
				found = true
			}
			if !found {
				t.Fatal("missing checked pinniped recipe")
			}
			sources, err := read.ProfileSources()
			if err != nil || len(sources) == 0 {
				t.Fatalf("missing profile source: %v", err)
			}
		})
	}
}
