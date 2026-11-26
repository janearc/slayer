package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetCompositionSchemasAcceptFileAndInlineAndRejectAmbiguity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "set", "testdata", "gate-composition-set.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			source := strings.Replace(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json", 1)
			if version == "v1" {
				source = strings.Replace(source, "  setting: space\n", "", 1)
			}
			cases := []struct {
				name, replacement string
				valid             bool
			}{
				{"file", "composition: {file: gate-composition.yaml}", true},
				{"inline", "composition: {inline: {version: 1, name: gate, parts: []}}", true},
				{"both", "composition: {file: gate-composition.yaml, inline: {}}", false},
				{"unknown", "composition: {other: gate-composition.yaml}", false},
				{"empty file", "composition: {file: ''}", false},
			}
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					text := strings.Replace(source, "composition: {file: gate-composition.yaml}", c.replacement, 1)
					document, err := Open("gate.yaml", []byte(text))
					if err != nil {
						t.Fatal(err)
					}
					err = document.Check()
					if (err == nil) != c.valid {
						t.Fatalf("valid=%v, check=%v", c.valid, err)
					}
				})
			}
		})
	}
}
