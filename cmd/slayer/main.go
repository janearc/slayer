// Command slayer draws the set files we write, from outside, the way a
// stranger would, and keeps a note of every place it has to reach in.
//
//	slayer set FILE [-camera NAME] [-at T] [-k K] [-samples N] [-out DIR]
//
// set draws a set file as the terminal shows it and as a real picture,
// judges the one against the other with the comparison, and writes the
// real one again through the set's print. see draw.go, and doc/sets.md
// for what a set file says.
//
// slayer is not smokecheck. smokecheck is the gate on the smokes
// (scene/smoke); slayer is ours, and it grows with whatever the next
// picture needs. where slayer's own packages make that hard, the
// difficulty is the finding.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/janearc/slayer/render/compare"
	"github.com/janearc/slayer/scene/thing"
)

// the terminal slayer pretends to be, as smokecheck does: 179 by 50 cells of
// sextants, each three real pixels across and four down
const (
	textW, textH = 358, 150
	across, down = 3, 4
)

// main runs a verb.
func main() {
	verbs := map[string]func([]string) (bool, error){
		"set": draw,
	}
	if len(os.Args) < 2 || verbs[os.Args[1]] == nil {
		fmt.Fprintln(os.Stderr, "usage: slayer set FILE [-camera NAME] "+
			"[-at T] [-k K] [-samples N] [-out DIR]")
		os.Exit(2)
	}
	honest, err := verbs[os.Args[1]](os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "slayer %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	if !honest {
		os.Exit(1)
	}
}

// Fineness is the least -k a world may be judged at, as smokecheck has it: a
// world with a field is never judged by a reference coarser than twice.
func Fineness(w *thing.World) int {
	if len(w.Fields) > 0 {
		return 2
	}
	return 1
}

// verdict is a comparison as it is kept: the numbers, and whether they
// make it honest.
type verdict struct {
	compare.Report
	Honest bool
}

// writeVerdict keeps a comparison's numbers beside its pictures.
func writeVerdict(path string, r compare.Report) error {
	b, err := json.MarshalIndent(verdict{r, r.Honest()}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
