// Command azazel schedules Slayer jobs and retains the direct rendering adapter.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/azazel/engine"
	"github.com/janearc/slayer/session"
)

// build, built, who and sessions are set by the linker; they say which build
// this is.
var build, built, who, sessions string

// stamped is the tool that answers which build this is.
func stamped() *pzm.Tool {
	return pzm.New("azazel", "submit and schedule Slayer jobs across agents",
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

// main routes queue and rendering verbs. azazel --age and --who say
// which build this is.
func main() {
	if len(os.Args) > verbAt && queueEntry(os.Args[verbAt]) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		os.Exit(runQueue(ctx, os.Args[verbAt:], os.Stdout, os.Stderr))
	}

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
	stamp := session.Stamped(build, built, who, sessions)
	os.Exit(engine.Render(os.Args[fileAt], os.Args[settingsAt:], os.Stdout,
		os.Stderr, stamp))
}

// queueEntry preserves Pzm's verb aliases and flags before the verb.
func queueEntry(word string) bool {
	return strings.HasPrefix(word, "--") || slices.Contains([]string{
		"schema", "init", "submit", "status", "cancel", "pause", "resume", "tune",
		"work", "work-gpu", "recover", "execute", "execute-owned", "publish", "pages", "help",
		"agent-help",
	}, word)
}
