// Command pan writes a film of stills moved slowly under a window, cut
// on a tune's bars, with the tune under it. See improv/pan/README.md.
//
//	pan write SCRIPT --out FILM.mp4 [--dry-run]
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/pan"
)

var build, built, who, sessions string

// argumentsAt is where the program's arguments start: its name is
// first. Defined.
const argumentsAt = 1

// main writes a film, a control-c throwing it away.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt,
		syscall.SIGTERM)
	code := tool(ctx).Run(os.Args[argumentsAt:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// writeHelp is write's page.
const writeHelp = `# write

pan write SCRIPT --out FILM.mp4 reads a script of stills, each moved
slowly under a window the film's size, cut on a tune's bars, and writes
the film with the tune under it. --dry-run says the film's size, length
and shots, and writes nothing.
`

// tool is pan's verbs.
func tool(ctx context.Context) *pzm.Tool {
	made := pzm.New("pan", "a film of stills moved slowly, cut on a tune",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	writes := verb.NewFacts(verb.WritesTo("the film named by --out"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	made.Verb("write", verb.NoShort, "write a film of stills cut on a tune",
		writeHelp, "Example_write", writes,
		func(call pzm.Call) error { return write(ctx, call) },
		pzm.Value("out", verb.NoShort, "the film to write, an mp4"))
	return made
}

// write reads the script and writes the film, or says it on a dry run.
func write(ctx context.Context, call pzm.Call) error {
	if len(call.Args()) != argumentsAt {
		return errors.New("write takes one script: pan write SCRIPT")
	}
	cut, err := pan.Read(call.Args()[0])
	if err != nil {
		return err
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "%d frames, %.2f s\n",
			cut.Frames(), cut.Length().Seconds())
		return err
	}
	if !call.Given("out") {
		return errors.New("say where the film goes: --out FILM.mp4")
	}
	return cut.Write(ctx, call.Value("out"), call.Out())
}
