package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

func cutawayPlan(test *testing.T) *plan {
	test.Helper()
	raw := append([]byte("cutaway: true\n"), planRaw...)
	made, err := readPlan("cutaway.yaml", raw)
	if err != nil {
		test.Fatal(err)
	}
	if !made.Cutaway {
		test.Fatal("cutaway plan lost its checked setting")
	}
	return made
}

func TestCatalogueCutawayKeepsTheInstrumentAndPhysicalFraming(test *testing.T) {
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"rubin"})
	if err != nil {
		test.Fatal(err)
	}
	plan := defaultPlan(test)
	for _, entry := range entries {
		full, err := plan.construct(entry, "quick")
		if err != nil {
			test.Fatal(err)
		}
		var expected []*thing.Thing
		for _, part := range full.world.Things {
			if part.Material.Name != "observatory enclosure" {
				expected = append(expected, part)
			}
		}
		if len(expected) == len(full.world.Things) {
			test.Fatal("fixture has no enclosure to hide")
		}
		plan.Cutaway = true
		cut, err := plan.construct(entry, "quick")
		if err != nil {
			test.Fatal(err)
		}
		if !cut.cutaway || !reflect.DeepEqual(expected, cut.world.Things) {
			test.Fatal("cutaway altered the instrument")
		}
		if cut.bounds != full.bounds || !reflect.DeepEqual(cut.eye, full.eye) {
			test.Fatal("cutaway moved the physical framing")
		}
	}
}

func TestCatalogueDoesNotLabelAnUnenclosedObjectAsACutaway(test *testing.T) {
	data := readConstruction(test)
	plan := cutawayPlan(test)
	object, err := plan.construct(entry{Kind: data.Kind}, data.Detail)
	if err != nil {
		test.Fatal(err)
	}
	if object.cutaway || object.faces != data.Faces {
		test.Fatal("an unenclosed object was filtered or mislabelled")
	}
}

func TestCatalogueLabelsItsCutawayViews(test *testing.T) {
	book := book{Cards: []card{{Entry: entry{Kind: "rubin"}, Cutaway: true}}}
	root := test.TempDir()
	if err := book.write(root); err != nil {
		test.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		test.Fatal(err)
	}
	if !strings.Contains(string(raw), "Cutaway view: enclosure hidden.") {
		test.Fatal("cutaway looked like an unmodified manufactured object")
	}
}
