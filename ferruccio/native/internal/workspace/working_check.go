package workspace

import (
	"fmt"

	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// CheckWorking says whether the working display would take the file at path,
// without opening a window, starting a GPU device or drawing. It runs the
// steps Open runs, in their order: the file is read as a draft (load), the
// draft's schema accepts it (validate), the text reads as a set and the
// display takes the set (updateWorking).
//
// It returns nil when the display would take it and otherwise the refusal,
// worded as the display words it; a schema refusal says so.
//
// Without meshes the display's checks that cost nothing are run. With them
// the display's own scene is built on a cache of its own, so a shape that
// has no mesh or no hit test is found here and not in the window.
func CheckWorking(path string, meshes bool) error {
	document, absolute, err := ReadDocument(path)
	if err != nil {
		return err
	}
	if err := document.Check(); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	read, err := set.Read(absolute, []byte(document.Text()))
	if err != nil {
		return err
	}
	if !meshes {
		return placement.CheckDisplay(read)
	}
	_, err = placement.NewDisplayCache().Scene(read)
	return err
}
