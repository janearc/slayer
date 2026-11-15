// Command beats makes an editable, explicitly estimated audio beat grid.
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// main delegates all words and flags to Pzm.
func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

// Build stamps are filled by Game; an unstamped development run says none.
var build, built, who, sessions string

// run is the beat analyser's declared CLI, separate for focused tests.
func run(args []string, in io.Reader, out, errs io.Writer) int {
	tool := pzm.New("beats", "an editable audio beat grid",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("the named score file"),
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("analyse", verb.NoShort, "estimate a track's beat grid",
		"Read stereo PCM through the movie reader. Write a Moonlake-compatible "+
			"score marked estimated; check its visual metronome before use.",
		"Example_analyse", facts, analyse,
		pzm.Value("out", verb.NoShort, "new YAML score path"),
		pzm.Value("drums", verb.NoShort, "aligned drum stem for analysis"),
		pzm.Value("tempo", verb.NoShort, "target BPM; estimate when omitted"),
		pzm.Value("meter", verb.NoShort, "authored beats per bar; default 4"),
		pzm.Value("first", verb.NoShort, "first bar's beat, audio seconds"))
	return tool.RunWith(args, in, out, errs)
}

// number reads a declared numeric flag, preserving the default if unsaid.
func number(call pzm.Call, flag string, fallback float64) (float64, error) {
	if !call.Given(flag) {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(call.Value(flag), 64)
	if err != nil {
		return 0, fmt.Errorf("--%s needs a number: %w", flag, err)
	}
	return value, nil
}
