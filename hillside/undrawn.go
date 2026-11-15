package hillside

import (
	"fmt"
	"os"

	"github.com/janearc/slayer/set"
)

// drawnFeatures are the features of a set that names schema version 2 that
// a stage draws, by the dotted name the reader gives each. Until a stage
// draws one, a set that holds it is refused. A stage that comes to draw a
// feature adds its name here in the same change.
//
// The stage for a scene in space (afloat.go) draws its setting, and a
// still reads a film's frames a second and has no use for them.
var drawnFeatures = map[string]bool{"scene.setting": true, "film.fps": true}

// refuseUndrawn is the refusal of a set that holds a feature of schema
// version 2 that the reader keeps and no stage draws, or nil when it holds
// none. It names the first such feature in the file: the file, the line and
// column of the key, the feature, and what the key says, if it says a word
// or a number.
func refuseUndrawn(read set.Set, file string, drawn map[string]bool) error {
	for _, each := range read.Undrawn() {
		if drawn[each.Feature()] {
			continue
		}
		said := ""
		if each.Says() != "" {
			said = ": " + each.Says()
		}
		return fmt.Errorf("%s:%d:%d: %s%s is read and not yet drawn", file,
			each.Line(), each.Column(), each.Feature(), said)
	}
	return nil
}

// readDrawable is the set a file holds, read, or the refusal of a file the reader
// will not take or that holds a feature of schema version 2 that no stage
// draws.
func readDrawable(path string) (set.Set, error) {
	written, err := os.ReadFile(path)
	if err != nil {
		return set.Set{}, err
	}
	read, err := set.Read(path, written)
	if err == nil {
		err = refuseUndrawn(read, path, drawnFeatures)
	}
	return read, err
}
