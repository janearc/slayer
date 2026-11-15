// Command azazel is the render queue's command. For now it has one verb,
// and no queue.
//
//	azazel render FILE [settings]
//
// render draws a set file through the library slayer render uses, and
// writes the picture where -out says. the queue's admission, locking and
// publication are not wired to it; see azazel/engine.
package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/azazel/engine"
	"github.com/janearc/slayer/hillside"
)

// build, built, who and sessions are set by the linker; they say which build
// this is.
var build, built, who, sessions string

// stamped is the tool that answers which build this is.
func stamped() *pzm.Tool {
	return pzm.New("azazel", "render a set file through the queue's engine",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
}

// where os.Args holds the verb, the file and the settings: the program's
// name comes first.
const (
	verbAt     = 1 // defined: the verb comes after the program's name
	fileAt     = 2 // defined: the file comes after the verb
	settingsAt = 3 // defined: the settings come after the file
)

// exitUsage is the number azazel exits with for a call it does not take.
const exitUsage = 2 // defined: hillside's number for what it could not take

// main runs the one verb, and gives its number. azazel --age and --who say
// which build this is.
func main() {
	if slices.Contains(os.Args, "--age") || slices.Contains(os.Args, "--who") {
		os.Exit(stamped().Run(os.Args[verbAt:], os.Stdout, os.Stderr))
	}
	if len(os.Args) > verbAt && os.Args[verbAt] == "composite" {
		if len(os.Args) <= fileAt {
			fmt.Fprintln(os.Stderr, "usage: azazel composite SETTINGS [settings]; direct starts unavailable")
			os.Exit(exitUsage)
		}
		fmt.Fprintln(os.Stderr, engine.ErrCompositeAdmissionUnavailable)
		os.Exit(1)
	}
	if len(os.Args) <= fileAt || os.Args[verbAt] != "render" {
		fmt.Fprintln(os.Stderr, "usage: azazel render FILE [settings]")
		os.Exit(exitUsage)
	}
	stamp := hillside.Stamped(build, built, who, sessions)
	os.Exit(engine.Render(os.Args[fileAt], os.Args[settingsAt:], os.Stdout,
		os.Stderr, stamp))
}
