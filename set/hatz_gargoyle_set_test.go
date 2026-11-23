package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestHatzGargoyleSetPassesBothSchemas(t *testing.T) {
	raw, err := os.ReadFile("../sets/bioforms/hatz-gargoyle-study.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		source := strings.ReplaceAll(string(raw), "ecosystem-v2.json", "ecosystem-"+version+".json")
		if version == "v1" {
			source = strings.ReplaceAll(source, "  setting: space\n", "")
		}
		read, err := set.Read("gargoyle.yaml", []byte(source))
		if err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		if len(read.Things()) != 1 || read.Things()[0].Kind() != "hatz-gargoyle" {
			t.Fatalf("%s parsed the wrong ornament", version)
		}
		unknown := strings.ReplaceAll(source, "hatz-gargoyle", "hatz-gargoyles")
		if _, err := set.Read("unknown.yaml", []byte(unknown)); err == nil {
			t.Fatalf("%s accepted an unknown kind", version)
		}
	}
}

func TestHatzGargoyleIsAcceptedAsASuperblockRoofPart(t *testing.T) {
	raw, err := os.ReadFile("../sets/superblock-kowloon.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(raw), "        seed: 7\n", "        seed: 7\n        parts:\n          hatz-gargoyle: 1\n", 1)
	read, err := set.Read("roof.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	things := read.Things()
	if len(things) != 1 || things[0].Kind() != "superblock" || things[0].Parts()["hatz-gargoyle"] != 1 {
		t.Fatalf("roof mix was not read: %+v", things)
	}
}
