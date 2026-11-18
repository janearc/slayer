// Command searching prepares and previews the Something answers shoot.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/searching"
)

// Game fills these stamps when the preview command is built.
var build, built, who, sessions string

// main declares all shoot verbs and flags through Pzm.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	tool := pzm.New("searching", "Something answers: a song-driven shoot",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	tool.Verb("music", verb.NoShort, "render an authored score and its exact beat clock",
		"Writes a WAV, source documents, grid and receipt into a new directory.",
		"Example_music", verb.NewFacts(verb.WritesTo("the named new music directory"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light), music,
		pzm.Value("voicing", verb.NoShort, "checked Cheeptune voicing file"),
		pzm.Value("out", verb.NoShort, "new output directory"))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("progress", verb.NoShort, "watch completed frames and report to Barkeep",
		"Writes progress.json beside a take; optionally publishes a Barkeep status file or HTTP update.",
		"Example_progress", verb.NewFacts(verb.WritesTo("progress.json and the selected local Barkeep destination"),
			verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light),
		func(call pzm.Call) error { return progress(ctx, call) },
		pzm.Value("from", verb.NoShort, "first source frame, zero unsaid"),
		pzm.Value("frames", verb.NoShort, "expected frames, remaining song unsaid"),
		pzm.Value("pid", verb.NoShort, "producer PID; detects stopped incomplete jobs"),
		pzm.Value("progress-file", verb.NoShort, "atomic five-integer status file for the Barkeep side column"),
		pzm.Switch("follow", verb.NoShort, "watch every ten seconds until completion or interrupt"),
		pzm.Value("barkeep", verb.NoShort, "existing loopback Barkeep HTTP address"),
		pzm.Value("barkeep-kind", verb.NoShort, "configured message kind"),
		pzm.Value("barkeep-signal", verb.NoShort, "configured render signal"))
	tool.Verb("explain", verb.NoShort, "publish the synchronized making-of view",
		"Checks a completed render and binds its companion to the same movie, plan and score.",
		"Example_explain", verb.NewFacts(verb.WritesTo("companion files beside the completed take"),
			verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light), behind)
	writes := verb.NewFacts(verb.WritesTo("the named new output"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("plan", verb.NoShort, "print the checked story timing",
		"Reads SHOOT.yaml and its source-bound score; draws nothing.", "Example_plan", reads,
		func(call pzm.Call) error {
			edit, err := read(call)
			if err != nil {
				return err
			}
			return describe(call, edit)
		})
	tool.Verb("still", verb.NoShort, "draw one representative frame",
		"Draws a real scene at audio seconds, with optional site override.", "Example_still", writes,
		func(call pzm.Call) error { return still(ctx, call) },
		pzm.Value("at", verb.NoShort, "seconds into the original song"),
		pzm.Value("site", verb.NoShort, "optional site override"),
		pzm.Value("limit-seconds", verb.NoShort, "wall limit, 600 unsaid"),
		pzm.Value("out", verb.NoShort, "new PNG path"))
	tool.Verb("render", verb.NoShort, "render a stretch with the original music",
		"Writes actual PNG frames and an MP4 with source audio. No existing file is replaced.",
		"Example_render", writes, func(call pzm.Call) error { return render(ctx, call) },
		pzm.Value("from", verb.NoShort, "first frame, zero unsaid"),
		pzm.Value("frames", verb.NoShort, "frame count, remaining song unsaid"),
		pzm.Switch("defer-movie", verb.NoShort, "close scene resources before encoding saved PNGs and sound"),
		pzm.Switch("serial", verb.NoShort, "finish each frame before drawing the next; comparison control"),
		pzm.Value("limit-seconds", verb.NoShort, "wall limit, 600 unsaid"),
		pzm.Value("out", verb.NoShort, "output directory"))
	os.Exit(tool.RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// read refuses extra positional words before any renderer is constructed.
func read(call pzm.Call) (*searching.Edit, error) {
	if len(call.Args()) != 1 {
		return nil, fmt.Errorf("one SHOOT.yaml is required")
	}
	return searching.Read(call.Args()[0])
}

// describe exposes the editable story anchors in audio seconds.
func describe(call pzm.Call, edit *searching.Edit) error {
	return json.NewEncoder(call.Out()).Encode(map[string]any{
		"name": edit.Plan.Name, "width": edit.Plan.Width, "height": edit.Plan.Height,
		"fps": edit.Plan.FPS, "frames": edit.Frames(), "lock": edit.Lock,
		"radio-one": edit.RadioOne, "radio-two": edit.RadioTwo, "reveal": edit.Reveal,
		"coda": edit.Coda, "duration": edit.Grid.Score.Duration,
		"pointing": "authored shared direction", "spectra": "authored component views"})
}

// integer reads a declared optional frame count, never an environment parser.
func integer(call pzm.Call, name string, fallback int) (int, error) {
	if !call.Given(name) {
		return fallback, nil
	}
	return strconv.Atoi(call.Value(name))
}
