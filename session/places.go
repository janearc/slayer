package session

// The places: where each thing of a set was stood, kept from the
// staging and written as a table of commas, the same table the old
// command's -places writes, so a tool that reads it (improv/walk) reads
// either.

import (
	"errors"
	"os"
	"path/filepath"
)

// placesFolderMode is the mode of a folder Places makes for its file;
// defined: a folder anyone may read.
const placesFolderMode = 0o755

// errUnplaced is the refusal of a table of places from a scene that
// stood nothing on a ground: a smoke, a named scene or a world-only set.
var errUnplaced = errors.New("hillside: this scene stands nothing on a " +
	"ground, so it has no places to write; a set of a world and no " +
	"ground, a smoke and a named scene have none")

// Places writes to path where each thing of the staged scene was stood:
// a header line of "of,kind,east,north,height", then a line for each
// thing, east and north on the terrain's grid and the ground's height
// there, in metres. The file's folder is made where it is not there. A
// scene that stood nothing on a ground is refused, and writes nothing.
func (session *Session) Places(path string) error {
	if session.closed {
		return errClosed
	}
	if !session.given.stoodKept {
		return errUnplaced
	}
	if err := os.MkdirAll(filepath.Dir(path), placesFolderMode); err != nil {
		return err
	}
	return writeStood(path, session.given.stood)
}
