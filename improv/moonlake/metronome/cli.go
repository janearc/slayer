package main

import (
	"fmt"
	"os"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// main declares the visual metronome through Pzm, like the renderer.
func main() {
	tool := pzm.New("metronome", "check the score against its audio",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("raw RGB frames on stdout"),
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("draw", verb.NoShort, "write the visual metronome",
		"Writes RGB24 frames. Mux original audio beneath them; estimates "+
			"stay labelled until checked.", "Example_draw", facts, drawVerb,
		pzm.Value("score", verb.NoShort, "score YAML"),
		pzm.Value("cues", verb.NoShort, "cue sheet YAML"),
		pzm.Value("fps", verb.NoShort, "frames per second"),
		pzm.Value("width", verb.NoShort, "frame width"),
		pzm.Value("height", verb.NoShort, "frame height"))
	os.Exit(tool.RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// Build stamps are filled by Game.
var build, built, who, sessions string

// drawVerb passes Pzm selections to the existing visual metronome.
func drawVerb(call pzm.Call) error {
	if len(call.Args()) != 0 || call.Value("score") == "" ||
		call.Value("cues") == "" {
		return fmt.Errorf("draw needs score and cues, no arguments")
	}
	return render(call)
}
