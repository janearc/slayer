package set

import (
	"fmt"
	"testing"
)

func TestFormerPlantKindsAreRejected(t *testing.T) {
	// Cover every retired selector, both scene kinds and recipe kinds, with
	// schema validation and the schema-less reader. Non-plant controls work.
	kinds := []string{"spruce", "norway", "coast-redwood", "giant-sequoia", "coast-redwood-shoot", "giant-sequoia-shoot", "fern", "saguaro", "valley-oak", "alder", "poplar", "manzanita", "chaparral-brush", "joshua-tree", "kandelia-obovata", "aegiceras-corniculatum", "avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala", "pandanus-tectorius", "scaevola-taccada", "metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex", "erica-arborea", "salix-alba"}
	for _, schema := range []string{"", "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v1.json\n", "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\n"} {
		for _, kind := range append([]string{"can"}, kinds...) {
			for _, recipe := range []bool{false, true} {
				entry, recipes := "kind: "+kind, ""
				if recipe {
					entry, recipes = "recipe: specimen", "  recipes:\n    specimen: {kind: "+kind+"}\n"
				}
				source := schema + fmt.Sprintf(`scene:
  name: plant-refusal
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
%s  set:
    things:
      one: {%s, seed: 1, complexity: quick, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, recipes, entry)
				_, err := Read("old.yaml", []byte(source))
				if (err == nil) != (kind == "can") {
					t.Errorf("kind %s recipe=%t schema=%q: %v", kind, recipe, schema, err)
				}
			}
		}
	}
}
