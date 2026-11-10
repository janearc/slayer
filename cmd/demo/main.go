// Command demo lists and draws Slayer's numbered smokes and named scenes.
// Its embedded catalogue supplies modest first-look settings.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/scene/smoke"
)

var build, built, who, sessions string

// application owns the invocation lifetime used by scene subprocesses.
type application struct{ ctx context.Context }

// main runs the menu with cancellation from terminal signals.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(tool(ctx).Run(os.Args[one:], os.Stdout, os.Stderr))
}

// tool declares every demo verb and flag through Puzzlemouth.
func tool(ctx context.Context) *pzm.Tool {
	tool := pzm.New("demo", "see Slayer's scenes from a menu",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	destination := verb.WritesTo("the demo output directory")
	draws := verb.NewFacts(destination,
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	app := application{ctx: ctx}
	tool.Verb("list", verb.NoShort, "list smokes and named demos",
		"# list\n\nLists the smokes and shipped preview scenes.\n",
		"Example_list", reads, list)
	tool.Verb("show", verb.NoShort, "draw a smoke or named demo",
		"# show\n\nUse a smoke number or a demo name.\n"+
			"PNGs go to out/demos.\n",
		"Example_show", draws, app.show,
		pzm.Value("out", verb.NoShort,
			"output directory; default out/demos"),
		pzm.Value("width", verb.NoShort, "preview width in pixels"),
		pzm.Value("height", verb.NoShort, "preview height in pixels"),
		pzm.Value("rays", verb.NoShort,
			"samples per pixel along each axis"),
		pzm.Switch("terminal", verb.NoShort,
			"also show terminal cells"),
		pzm.Value("cells", verb.NoShort,
			"terminal dimensions; default 80x24"))
	return tool
}

// list shows stable smoke numbers, scene names and external data needs.
func list(call pzm.Call) error {
	if len(call.Args()) != nothing {
		return fmt.Errorf("list takes no arguments")
	}
	menu, err := readCatalogue()
	if err != nil {
		return err
	}
	for _, preset := range menu.Demos {
		_, err := fmt.Fprintf(call.Out(), "%s: %s\n",
			preset.Name, preset.About)
		if err != nil {
			return err
		}
	}
	for _, scene := range smoke.All() {
		needs := ""
		if len(scene.Inputs) != nothing {
			needs = fmt.Sprintf(" (data: %v)", scene.Inputs)
		}
		_, err := fmt.Fprintf(call.Out(), "%d: %s%s\n",
			scene.Number, scene.Name, needs)
		if err != nil {
			return err
		}
	}
	return nil
}
