// Command catalogue lists the manufactory and renders a local specimen book.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

var build, built, who, sessions string

// executableWord is defined by the executable before its command arguments.
const executableWord = 1

// noEntries is defined as an empty selection.
const noEntries = 0

type application struct{ context context.Context }

// main retains cancellation until the current specimen has finished.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(tool(ctx).Run(os.Args[executableWord:], os.Stdout, os.Stderr))
}

// tool declares the command's entire vocabulary through Puzzlemouth.
func tool(ctx context.Context) *pzm.Tool {
	made := pzm.New("catalogue", "browse what the manufactory can make",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	destination := verb.WritesTo("a new catalogue directory")
	draws := verb.NewFacts(destination,
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	made.Verb("list", verb.NoShort, "list the available specimens",
		"# list\n\nNames, descriptions and supported detail levels.\n",
		"Example_list", reads, list)
	app := application{context: ctx}
	made.Verb("render", verb.NoShort, "render a searchable specimen book",
		renderHelp, "Example_render", draws, app.render,
		pzm.Value("plan", verb.NoShort,
			"checked YAML or JSON render plan"),
		pzm.Value("out", verb.NoShort,
			"new output directory; default out/catalogue"))
	return made
}

const renderHelp = `# render

Render all named kinds, or the names following render. A plan supplies
detail levels, camera, lighting and image budgets. The default is a small
quick-detail book. Native Metal rendering is required; there is no CPU
fallback. The destination must be new. Open its index.html to browse.
`

// list reads the same checked menu that supplies the rendered book.
func list(call pzm.Call) error {
	menu, err := readMenu()
	if err != nil {
		return err
	}
	entries, err := menu.selectEntries(call.Args())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		_, err := fmt.Fprintf(call.Out(), "%s: %s %v\n",
			entry.Kind, entry.About, entry.Details)
		if err != nil {
			return err
		}
	}
	return nil
}
