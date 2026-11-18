// Command atelier opens the separate Fyne authoring prototype.
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/internal/studio"
)

var build, built, who, sessions string

// main keeps every public verb and flag at the Puzzlemouth boundary.
func main() {
	os.Exit(command().RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// command exposes the declared interface for dry-run and refusal tests.
func command() *pzm.Tool {
	tool := pzm.New("atelier", "A place to build worlds for Slayer", pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("only files explicitly saved in the editor"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("open", verb.NoShort, "open the authoring window", "Opens a Fyne interface prototype; no render is started.", "Example_open", facts, func(call pzm.Call) error {
		if len(call.Args()) > 1 {
			return fmt.Errorf("open accepts at most one job file")
		}
		if call.DryRun() {
			_, err := fmt.Fprintln(call.Out(), "open the Atelier interface; no render submission")
			return err
		}
		application := app.NewWithID("org.slayerscenes.atelier.prototype")
		application.Settings().SetTheme(studio.Theme())
		window := application.NewWindow("Atelier · Slayer")
		workspace, err := studio.New(studio.Options{Window: window, Study: "venus", Tools: toolsFor(call), Examples: call.Value("examples")})
		if err != nil {
			return err
		}
		if len(call.Args()) == 1 {
			if err := workspace.LoadFile(call.Args()[0]); err != nil {
				return err
			}
		} else if call.Value("examples") != "" {
			if err := workspace.LoadExample("first-picture.yaml"); err != nil {
				return err
			}
		}
		window.SetContent(workspace.Content())
		window.Resize(fyne.NewSize(1800, 1120))
		window.ShowAndRun()
		return nil
	}, runtimeFlags()...)
	tool.Bare("open")
	proofFacts := verb.NewFacts(verb.WritesTo("one new PNG and receipt"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("proof", verb.NoShort, "capture the native window and close", "Opens only Atelier, captures its canvas, then exits after two seconds. Use the shared GPU lane.", "Example_proof", proofFacts, nativeProof,
		append([]pzm.Flag{pzm.Value("out", verb.NoShort, "new PNG path"), pzm.Value("file", verb.NoShort, "existing YAML to open"), pzm.Switch("render", verb.NoShort, "render the loaded YAML before capture")}, runtimeFlags()...)...)
	return tool
}

// runtimeFlags names existing programs and data without a hidden host dependency.
func runtimeFlags() []pzm.Flag {
	return []pzm.Flag{pzm.Value("slayer", verb.NoShort, "Slayer executable"), pzm.Value("game", verb.NoShort, "Game5 executable"), pzm.Value("game-home", verb.NoShort, "scoped Game configuration home"), pzm.Value("shelf", verb.NoShort, "optional Slayer source shelf"), pzm.Value("examples", verb.NoShort, "directory of existing YAML examples")}
}

// toolsFor passes declared settings to the isolated preview process.
func toolsFor(call pzm.Call) preview.Tools {
	return preview.Tools{Slayer: call.Value("slayer"), Game: call.Value("game"), GameHome: call.Value("game-home"), Shelf: call.Value("shelf")}
}
