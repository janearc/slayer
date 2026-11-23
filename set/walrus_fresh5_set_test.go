package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestWalrusSetPassesBothSchemas(t *testing.T) {
	raw, err := os.ReadFile("../sets/bioforms/walrus-fresh-5.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json", "ecosystem-"+v+".json")
		if v == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		got, err := set.Read("walrus.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		if len(got.Things()) != 1 || got.Things()[0].Kind() != "walrus-fresh-5" {
			t.Fatalf("%s parsed wrong specimen", v)
		}
		unknown := strings.ReplaceAll(source, "walrus-fresh-5", "walrus-fresh-55")
		if _, err = set.Read("unknown.yaml", []byte(unknown)); err == nil {
			t.Fatalf("%s accepted unknown kind", v)
		}
	}
}
