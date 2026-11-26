package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSquidPhaseInBothDraftSchemas(t *testing.T) {
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
			name, animal string
			valid        bool
		}{
			{"default", "family: squid, file: squid.yaml, species: humboldt-squid", true},
			{"zero phase", "family: squid, file: squid.yaml, species: humboldt-squid, phase: 0", true},
			{"negative phase", "family: squid, file: squid.yaml, species: humboldt-squid, phase: -3.14", true},
			{"phase boundary", "family: squid, file: squid.yaml, species: humboldt-squid, phase: 100000", true},
			{"phase too large", "family: squid, file: squid.yaml, species: humboldt-squid, phase: 100001", false},
			{"null phase", "family: squid, file: squid.yaml, species: humboldt-squid, phase: null", false},
			{"text phase", "family: squid, file: squid.yaml, species: humboldt-squid, phase: '0'", false},
			{"squid time", "family: squid, file: squid.yaml, species: humboldt-squid, seconds: 0", false},
			{"squid pose", "family: squid, file: squid.yaml, species: humboldt-squid, pose: swimming", false},
			{"squid count", "family: squid, file: squid.yaml, species: humboldt-squid, count: 0", false},
			{"squid study", "family: squid, file: squid.yaml, species: humboldt-squid, study: bay", false},
			{"squid representation", "family: squid, file: squid.yaml, species: humboldt-squid, representation: skin", false},
			{"fish phase", "family: fish, file: fish.yaml, species: pacific-sardine, phase: 0", false},
			{"theropod phase", "family: theropod, file: allosaurus.yaml, species: allosaurus, phase: 0", false},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				text := strings.Replace(source, "composition: {file: gate-composition.yaml}", "animal: {"+tc.animal+"}", 1)
				if text == source {
					t.Fatal("missing fixture anchor")
				}
				doc, err := Open("squid.yaml", []byte(text))
				if err != nil {
					t.Fatal(err)
				}
				if err := doc.Check(); (err == nil) != tc.valid {
					t.Fatalf("valid=%v: %v", tc.valid, err)
				}
			})
		}
	}
}
