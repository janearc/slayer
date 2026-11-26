package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestOrcaSetPassesBothSchemas(t *testing.T) {
	raw, err := os.ReadFile("../sets/bioforms/orca-clean-05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		read, err := set.Read("../sets/bioforms/orca-clean-05.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		if len(read.Things()) != 1 || read.Things()[0].Kind() != "animal-profile" {
			t.Fatalf("%s parsed the wrong thing", version)
		}
		unknown := strings.ReplaceAll(source, "species: orca", "species: unknown-whale")
		if _, err := set.Read("../sets/bioforms/orca-clean-05.yaml", []byte(unknown)); err == nil {
			t.Fatalf("%s accepted unknown whale profile", version)
		}
	}
}
