package set_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// mangroveKinds are the kinds the mangrove sets stand.
var mangroveKinds = map[string]bool{"kandelia-obovata": true,
	"aegiceras-corniculatum": true, "avicennia-marina": true,
	"bruguiera-gymnorhiza": true, "sonneratia-apetala": true}

// readMangroveSet reads one of the mangrove sets from the sets folder.
func readMangroveSet(t *testing.T, name string, change func(string) string) (
	set.Set, error) {
	t.Helper()
	path := filepath.Join("..", "sets", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := change(string(raw))
	return set.Read(path, []byte(changed))
}

// same leaves a set's text as it is.
func same(text string) string { return text }

func TestMangroveStudyStandsTwelveTreesOfFiveKinds(t *testing.T) {
	made, err := readMangroveSet(t, "mangrove-study.yaml", same)
	if err != nil {
		t.Fatal(err)
	}
	things := made.Things()
	if len(things) != 12 {
		t.Fatalf("the study stands %d trees, not twelve", len(things))
	}
	seeds, kinds := map[uint64]bool{}, map[string]bool{}
	for _, each := range things {
		if !mangroveKinds[each.Kind()] {
			t.Fatalf("%s is a %s, not a mangrove", each.Name(), each.Kind())
		}
		seeds[each.Seed()], kinds[each.Kind()] = true, true
	}
	if len(seeds) != len(things) || len(kinds) != len(mangroveKinds) {
		t.Fatalf("%d seeds and %d kinds among %d trees", len(seeds), len(kinds),
			len(things))
	}
}

func TestMangroveShoreStandsEachKindInItsBandOverTheSea(t *testing.T) {
	made, err := readMangroveSet(t, "mangrove-shore.yaml", same)
	if err != nil {
		t.Fatal(err)
	}
	seeds, total := map[uint64]bool{}, 0
	for _, each := range made.Stands() {
		if !mangroveKinds[each.Kind()] {
			t.Fatalf("%s stands %s, not a mangrove", each.Name(), each.Kind())
		}
		above, below := each.Band()
		if above < 0 || below <= above {
			t.Fatalf("%s grows between %v and %v m", each.Name(), above, below)
		}
		if each.Kind() == "sonneratia-apetala" && above < 1 {
			t.Fatalf("%s stands in the native band", each.Name())
		}
		if each.Kind() != "sonneratia-apetala" && below > 1 {
			t.Fatalf("%s stands in the planted band", each.Name())
		}
		count, seed := each.Count()
		if seeds[seed] {
			t.Fatalf("%s repeats a seed another stand has", each.Name())
		}
		seeds[seed], total = true, total+count
	}
	if total < 200 || total > 500 {
		t.Fatalf("a few hundred were asked for, and the shore stands %d", total)
	}
}

func TestMangroveShoreRefusesABandThatIsNowhere(t *testing.T) {
	_, err := readMangroveSet(t, "mangrove-shore.yaml", func(text string) string {
		return strings.Replace(text, "above: 0.1, below: 1,",
			"above: 1, below: 0.1,", 1)
	})
	if err == nil || !strings.Contains(err.Error(), "nowhere") {
		t.Fatalf("a band above one and below a tenth was read: %v", err)
	}
}

func TestMangroveShoreRefusesAKindThatIsNotOne(t *testing.T) {
	_, err := readMangroveSet(t, "mangrove-shore.yaml", func(text string) string {
		return strings.Replace(text, "kind: kandelia-obovata",
			"kind: rhizophora-apiculata", 1)
	})
	if err == nil {
		t.Fatal("a mangrove no profile grows was read")
	}
}

func TestSchemasNameEveryMangroveAndTheirCopiesAreLevel(t *testing.T) {
	for _, name := range []string{"ecosystem.schema.json",
		"ecosystem-v2.schema.json"} {
		own, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		copied, err := os.ReadFile(filepath.Join("..", "ferruccio", "internal",
			"assets", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(own, copied) {
			t.Fatalf("ferruccio's %s is not the set's", name)
		}
		for kind := range mangroveKinds {
			if !bytes.Contains(own, []byte("\""+kind+"\"")) {
				t.Fatalf("%s does not name %s", name, kind)
			}
		}
	}
}
