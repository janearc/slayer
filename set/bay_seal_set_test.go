package set_test

import (
	"github.com/janearc/slayer/set"
	"os"
	"strings"
	"testing"
)

func TestBaySealSetPassesBothSchemas(t *testing.T) {
	raw, err := os.ReadFile("../sets/bioforms/bay-seal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		read, err := set.Read("seal.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		if len(read.Things()) != 1 || read.Things()[0].Kind() != "bay-seal" {
			t.Fatalf("%s parsed wrong specimen", version)
		}
		unknown := strings.ReplaceAll(source, "bay-seal", "bay-seals")
		if _, err := set.Read("unknown.yaml", []byte(unknown)); err == nil {
			t.Fatalf("%s accepted unknown kind", version)
		}
	}
}
