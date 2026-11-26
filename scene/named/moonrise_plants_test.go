package named

import (
	"bytes"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestMoonrisePlantSelectors(t *testing.T) {
	// This is the complete set of plant choices offered before registry removal.
	names := []string{"spruce", "norway", "coast-redwood", "giant-sequoia", "coast-redwood-shoot", "giant-sequoia-shoot", "fern", "saguaro", "valley-oak", "alder", "poplar", "manzanita", "chaparral-brush", "joshua-tree", "kandelia-obovata", "aegiceras-corniculatum", "avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala", "pandanus-tectorius", "scaevola-taccada", "metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex", "erica-arborea", "salix-alba"}
	scene := moonriseScene()
	for _, name := range names {
		if err := scene.Check(map[string]any{"kind": name}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if _, ok := moonrisePlants[name]; !ok {
			t.Errorf("%s is not routed through a checked plant book", name)
		}
	}
	for _, tc := range []struct {
		selector, family, species string
		shoot                     bool
	}{
		{"norway", "conifer", "norway", false},
		{"coast-redwood-shoot", "conifer", "coast-redwood", true},
		{"fern", "fern", "fern", false},
		{"saguaro", "cactus", "saguaro", false},
		{"valley-oak", "seasonal", "valley-oak", false},
	} {
		t.Run(tc.selector, func(t *testing.T) {
			filename, raw, err := manufacture.EmbeddedPlantBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			book, err := manufacture.ReadPlantBook(tc.family, filename, raw)
			if err != nil {
				t.Fatal(err)
			}
			options := manufacture.PlantOptions{Species: tc.species}
			if tc.shoot {
				shoot := "shoot"
				options.Representation = &shoot
			}
			row, err := book.New(options)
			if err != nil {
				t.Fatal(err)
			}
			made, err := manufacture.Make(row, manufacture.Sow(3))
			if err != nil {
				t.Fatal(err)
			}
			want, err := made.Things(maths.Subspace{})
			if err != nil {
				t.Fatal(err)
			}
			world := thing.NewWorld()
			if err := planted(world, tc.selector, 3); err != nil {
				t.Fatal(err)
			}
			if len(world.Things) != len(want) {
				t.Fatalf("got %d parts, want %d", len(world.Things), len(want))
			}
			for i, got := range world.Things {
				if got.Shape.Bounds() != want[i].Shape.Bounds() || got.At(0) != want[i].At(0) || got.Material.Name != want[i].Material.Name {
					t.Fatalf("part %d changed shape, placement or material", i)
				}
			}
		})
	}
	if err := scene.Check(map[string]any{"kind": "Norway"}); err == nil {
		t.Fatal("accepted unknown plant selector")
	}
	if err := planted(thing.NewWorld(), "Norway", 3); err == nil {
		t.Fatal("planted unknown selector")
	}
	if err := planted(thing.NewWorld(), "can", 3); err != nil {
		t.Fatalf("ordinary kind refused: %v", err)
	}
}

func TestMoonriseRefusesAmbiguousPlantSelectors(t *testing.T) {
	load := func(family string, rename string) moonriseBook {
		t.Helper()
		name, raw, err := manufacture.EmbeddedPlantBook(family)
		if err != nil {
			t.Fatal(err)
		}
		if rename != "" {
			old, replacement := []byte("kind: valley-oak"), []byte("kind: "+rename)
			if !bytes.Contains(raw, old) {
				t.Fatal("seasonal fixture lost valley-oak")
			}
			raw = bytes.Replace(raw, old, replacement, 1)
		}
		book, err := manufacture.ReadPlantBook(family, name, raw)
		if err != nil {
			t.Fatal(err)
		}
		return moonriseBook{family, book}
	}
	seasonal, conifer, fern := load("seasonal", ""), load("conifer", ""), load("fern", "")
	if _, err := collectMoonrisePlants([]moonriseBook{seasonal, conifer, fern}, []string{"can"}); err != nil {
		t.Fatalf("distinct choices refused: %v", err)
	}
	for _, tc := range []struct {
		name   string
		books  []moonriseBook
		public []string
	}{
		{"public species", []moonriseBook{seasonal}, []string{"valley-oak"}},
		{"public shoot", []moonriseBook{conifer}, []string{"coast-redwood-shoot"}},
		{"species in two books", []moonriseBook{load("seasonal", "fern"), fern}, nil},
		{"species and shoot", []moonriseBook{load("seasonal", "coast-redwood-shoot"), conifer}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := collectMoonrisePlants(tc.books, tc.public); err == nil {
				t.Fatal("ambiguous selector accepted")
			}
		})
	}
}
