// Command slayer is Slayer's front door: package door's tool and its
// verbs. See door/README.md.
//
//	slayer draw JOB [--out PATH] [--width N] [--height N] [--rays N]
//	    [--at T] [--camera NAME] [--renderer NAME] [--quality NAME]
//	    [--vibes on|off] [--fastlight on|off] [--lamp-store on|off]
//	    [--sky-looks on|off] [--dry-run]
//	slayer list [HEADING] [--json]
//	slayer assess FILE [--judge] | --smokes [GROUP|N...]
//	slayer develop JOB RAW... [--sum] [--white K] [--out PATH]
//	slayer tunes DIR [--address HOST:PORT | --check | --films | --page]
//	slayer watch JOB [the draw flags]
//	slayer edit [JOB] [the draw flags]
//	slayer schema [--where]
//
// Bare slayer at a terminal is slayer edit on the newest job here.
//
// One older verb is read ahead of the door, in its own flags, until it
// moves onto it:
//
//	slayer render FILE [settings]
//
// render draws a set file, the whole of it, as hillside does: its things
// and its world together. its settings are hillside's, so -w, -h and -n
// say the size and the rays, and -out where the picture goes. see
// render.go.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/door"
	"github.com/janearc/slayer/improv/searching"
	"github.com/janearc/slayer/internal/observability"
)

// build, built, who and sessions are set by the linker; they say which build
// this is.
var build, built, who, sessions string

// where os.Args holds the verb, and the arguments after it: the program's
// name comes first.
const (
	verbAt = 1 // defined: the verb comes after the program's name
	askAt  = 2 // defined: its arguments come after the verb
)

// the numbers slayer exits with, and a count of none.
const (
	nothing    = 0 // defined: no amount at all
	exitFailed = 1 // defined: a verb failed
	exitUsage  = 2 // defined: a verb that slayer does not have
)

// older is the verbs read ahead of the door, each in its own flags, and
// the exit number it gives.
var older = map[string]func(args []string, said, wrong io.Writer) int{
	"render": render,
}

// moved is the verbs that went into the door's, each with what to type
// now; one of them says so and exits as a verb slayer does not have.
var moved = map[string]string{
	"set":    "set is now: slayer assess FILE --judge",
	"smokes": "smokes is now: slayer assess --smokes [GROUP|N...]",
}

// main runs an older verb by its word, or else the door, with a context
// that control-c and a terminate cancel.
func main() {
	os.Exit(runMain())
}

// runMain returns before main exits so the optional listeners close cleanly.
func runMain() int {
	defer startProfile(os.Stderr)()
	defer startHeapProfile(os.Stderr)()
	defer startFrameEvents()()
	defer startObservability(os.Stderr)()
	defer observability.StartSummary("slayer", os.Stderr)()
	if err := searching.RegisterSlayerScenes(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitFailed
	}
	// the build's digest is started now, from the binary as it starts: a
	// binary rebuilt while this one runs must not have its digest put on
	// what this one keeps (review of 2026-11-12); it is worked out beside
	// the verb, so a quick one does not wait for it
	go diskcache.Build() //nolint:errcheck // asked again, with its error, where used
	if len(os.Args) > verbAt && moved[os.Args[verbAt]] != "" {
		fmt.Fprintln(os.Stderr, "slayer: "+moved[os.Args[verbAt]])
		return exitUsage
	}
	if len(os.Args) > verbAt && older[os.Args[verbAt]] != nil {
		return older[os.Args[verbAt]](os.Args[askAt:], os.Stdout,
			os.Stderr)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt,
		syscall.SIGTERM)
	code := front(ctx, os.Args[verbAt:], os.Stdout, os.Stderr)
	stop()
	return code
}

// startFrameEvents leaves rendering untouched when the optional handoff fails.
func startFrameEvents() func() {
	stop, err := observability.StartFrameEventsFromEnv()
	if err != nil {
		return func() {}
	}
	return stop
}

// startObservability reports a listener failure without refusing the job.
func startObservability(wrong io.Writer) func() {
	stop, err := observability.Start("slayer")
	if err != nil {
		fmt.Fprintf(wrong, "slayer: observability unavailable: %v\n", err)
		return func() {}
	}
	return stop
}

// front runs the door's tool on args and gives its exit number.
func front(ctx context.Context, args []string, said, wrong io.Writer) int {
	here, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(wrong, "slayer: %v\n", err)
		return exitUsage
	}
	env, err := door.NewEnv(door.Given{Context: ctx, Build: build,
		Built: built, Who: who, Sessions: sessions,
		Environment: os.LookupEnv, Now: time.Now, Directory: here})
	if err != nil {
		fmt.Fprintf(wrong, "slayer: %v\n", err)
		return exitUsage
	}
	return door.Tool(env).Run(args, said, wrong)
}
