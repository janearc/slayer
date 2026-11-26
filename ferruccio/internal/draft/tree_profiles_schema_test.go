package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The draft editor checks its copied schema before the renderer opens the
// external book. Both supported schema versions must agree on the shape of
// tree-book recipes and refuse other kinds or malformed descriptors.
func TestTreeProfileRecipesInDraftSchemas(t *testing.T) {
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
			{"tower", "kind: tower\n      tree-profiles: {file: trees.yaml}", true},
			{"green city", "kind: city-clustered-green-10000\n      tree-profiles: {file: trees.yaml}", true},
			{"night city", "kind: city-clustered-night-10000\n      tree-profiles: {file: trees.yaml}", true},
			{"nonplant city", "kind: city-clustered-10000\n      tree-profiles: {file: trees.yaml}", false},
			{"boulder", "kind: boulder\n      tree-profiles: {file: trees.yaml}", false},
			{"null block", "kind: tower\n      tree-profiles: null", false},
			{"scalar block", "kind: tower\n      tree-profiles: trees.yaml", false},
			{"missing file", "kind: tower\n      tree-profiles: {}", false},
			{"null file", "kind: tower\n      tree-profiles: {file: null}", false},
			{"empty file", "kind: tower\n      tree-profiles: {file: ''}", false},
			{"blank file", "kind: tower\n      tree-profiles: {file: '  '}", false},
			{"extra option", "kind: tower\n      tree-profiles: {file: trees.yaml, fallback: true}", false},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				text := strings.Replace(source, "composition: {file: gate-composition.yaml}", tc.recipe, 1)
				if text == source {
					t.Fatal("recipe fixture changed")
				}
				doc, err := Open("tree-profiles.yaml", []byte(text))
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
