package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/ferruccio/native/internal/look"
)

// lookHelp is the one line --help gives for look.
const lookHelp = "print the newest look at a set, or all of them"

// lookPage is the longer page for look.
const lookPage = `Print what the window said it was showing, in text, without a window.

The Describe button in the window, or the D key on the picture, appends a
look to a file beside the set: neon-street.yaml has neon-street.looks.txt.
An example that has no place of its own keeps its looks in the evidence
directory, which --evidence names. Each look is one block that starts with
three dashes and holds the time, the SHA-256 of the draft it describes, the
camera, the selected thing, what was under the pointer and the things in
view. What the window could not tell, a look says it does not know.

look FILE prints the newest look at the set FILE. --all prints every one,
oldest first. It reads the file and writes nothing.`

// registerLook declares look through Puzzlemouth beside open and check.
func registerLook(tool *pzm.Tool) {
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("look", verb.NoShort, lookHelp, lookPage, "Example_look", facts, runLook,
		pzm.Switch("all", verb.NoShort, "print every look, oldest first"),
		pzm.Value("evidence", verb.NoShort, "where an unsaved example keeps its looks"))
}

// runLook prints the newest look at the set the call names, or all of them.
func runLook(call pzm.Call) error {
	if len(call.Args()) != 1 {
		return errors.New("look needs one set file")
	}
	path := look.PathFor(call.Args()[0])
	blocks, err := look.Read(path)
	if err != nil && call.Value("evidence") != "" {
		path = filepath.Join(call.Value("evidence"), filepath.Base(path))
		blocks, err = look.Read(path)
	}
	if err != nil {
		return err
	}
	if !call.On("all") {
		blocks = blocks[len(blocks)-1:]
	}
	for _, block := range blocks {
		if _, err := fmt.Fprint(call.Out(), block); err != nil {
			return err
		}
	}
	return nil
}
