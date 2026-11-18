// Command atelier-capture exports the actual Fyne widget tree for visual review.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/ferruccio/internal/capture"
	"github.com/janearc/slayer/ferruccio/internal/studio"
)

var build, built, who, sessions string

// main registers the diagnostic command through Puzzlemouth.
func main() {
	os.Exit(command().RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// command exposes the diagnostic contract without creating a canvas.
func command() *pzm.Tool {
	tool := pzm.New("atelier-capture", "Capture the real prototype canvas", pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("one new PNG and its receipt"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	tool.Verb("snapshot", verb.NoShort, "export a Fyne canvas", "Software-rendered actual widgets; not a native-window screenshot or HTML mockup.", "Example_snapshot", facts, snapshot,
		pzm.Value("out", verb.NoShort, "new PNG path"), pzm.Value("study", verb.NoShort, "venus or titan"), pzm.Value("area", verb.NoShort, "set, cameras, sequence or music"), pzm.Value("size", verb.NoShort, "wide or compact"), pzm.Value("pane", verb.NoShort, "yaml or settings"))
	return tool
}

// snapshot paints actual widgets with Fyne's public software canvas driver.
func snapshot(call pzm.Call) error {
	if len(call.Args()) != 0 || call.Value("out") == "" {
		return fmt.Errorf("snapshot needs --out and no positional arguments")
	}
	study, area, sizeName, pane := call.Value("study"), call.Value("area"), call.Value("size"), call.Value("pane")
	if study == "" {
		study = "venus"
	}
	if area == "" {
		area = "set"
	}
	if sizeName == "" {
		sizeName = "wide"
	}
	if pane == "" {
		pane = "yaml"
	}
	sizes := map[string]fyne.Size{"wide": fyne.NewSize(1800, 1120), "compact": fyne.NewSize(1440, 1000)}
	size, ok := sizes[sizeName]
	if !ok {
		return fmt.Errorf("size must be wide or compact")
	}
	if area != "set" && area != "cameras" && area != "sequence" && area != "music" {
		return fmt.Errorf("unknown workspace area")
	}
	if study != "venus" && study != "titan" {
		return fmt.Errorf("study must be venus or titan")
	}
	if pane != "yaml" && pane != "settings" {
		return fmt.Errorf("pane must be yaml or settings")
	}
	receipt := map[string]any{"build": build, "renderer": "Fyne software canvas", "native_window": false, "study": study, "area": area, "pane": pane, "width": size.Width, "height": size.Height}
	if call.DryRun() {
		return json.NewEncoder(call.Out()).Encode(receipt)
	}
	application := test.NewApp()
	defer application.Quit()
	application.Settings().SetTheme(studio.Theme())
	window := application.NewWindow("Atelier capture")
	workspace, err := studio.New(studio.Options{Window: window, Study: study})
	if err != nil {
		return err
	}
	workspace.ShowArea(area)
	workspace.ShowInspector(pane)
	canvas := software.NewCanvas()
	canvas.SetPadded(false)
	canvas.SetContent(workspace.Content())
	canvas.Resize(size)
	if err := capture.Write(call.Value("out"), canvas.Capture(), receipt); err != nil {
		return err
	}
	return json.NewEncoder(call.Out()).Encode(receipt)
}
