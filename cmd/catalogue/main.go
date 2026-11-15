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

type application struct {
	context context.Context
	drawer  drawer
}

// main retains cancellation until the current specimen has finished.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(tool(ctx).Run(os.Args[executableWord:], os.Stdout, os.Stderr))
}

// tool declares the command's entire vocabulary through Puzzlemouth.
func tool(ctx context.Context) *pzm.Tool {
	return toolDrawing(ctx, nativeDrawer{})
}

// toolDrawing is the tool with the given drawer in place of Metal.
func toolDrawing(ctx context.Context, drawing drawer) *pzm.Tool {
	made := pzm.New("catalogue", "browse what the manufactory can make",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	destination := verb.WritesTo("a new catalogue directory, or a book to resume")
	draws := verb.NewFacts(destination,
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	made.Verb("list", verb.NoShort, "list the available specimens",
		"# list\n\nNames, descriptions and supported detail levels.\n",
		"Example_list", reads, list)
	app := application{context: ctx, drawer: drawing}
	made.Verb("render", verb.NoShort, "render a searchable specimen book",
		renderHelp, "Example_render", draws, app.render,
		pzm.Value("plan", verb.NoShort,
			"checked YAML or JSON render plan"),
		pzm.Value("out", verb.NoShort,
			"new directory, or a book to resume; default out/catalogue"))
	//ninja:vow blade each command says its own facts beside its verbs
	composes := verb.NewFacts(verb.WritesTo("a new PDF file"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	made.Verb("pdf", verb.NoShort, "compose a rendered book into one PDF",
		pdfHelp, "Example_pdf", composes, app.pdf,
		pzm.Value("book", verb.NoShort,
			"rendered book directory; default out/catalogue"),
		pzm.Value("out", verb.NoShort,
			"new PDF file; default out/catalogue.pdf"),
		pzm.Switch("with-failures", verb.NoShort,
			"give each card that did not render a page saying why"))
	return made
}

const renderHelp = `# render

Render all named kinds, or the names following render. A plan supplies
detail levels, camera, lighting and image budgets. The default is a small
quick-detail book. Native Metal rendering is required; there is no CPU
fallback. Open the destination's index.html to browse.

A specimen that fails is marked failed on its card with the reason, and
the run goes on to the next. At the end the run names every specimen that
failed and exits non-zero.

If the destination is not there, a new book is made. If it holds a book
(a catalogue.json this command wrote), a second run resumes it: cards
already rendered under the same plan are kept and not redrawn; failed and
pending cards are drawn; kinds added to the menu are added and drawn; kinds
the menu no longer has are dropped from the index and named. A card drawn
under a different plan is drawn again. Each card keeps the record of the
binary that drew it. If the destination exists and is not a book, the run
refuses it and changes nothing.
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
