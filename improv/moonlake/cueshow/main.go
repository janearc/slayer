// Command cueshow is improv only: moonlake's cue compiler. From the
// measured score (score.yaml: the grid of beat times, the sections,
// the transcribed notes) and the cue sheet (cues.yaml: what the
// lights do, in bars), it says what the lights are doing at one
// moment, as key=value lines for lightshow to obey:
//
//	cueshow score.yaml cues.yaml seconds [fps]
//
// rig, beams, colour, sweep, white, magenta, mist, and bar for the
// eye. fps bounds the window a note must start in to fire a strobe;
// it is 12 unless given, matching lightshow's own frame clock.
//
// The reading and the fold are package show's; this command is its
// face on the command line.
package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/janearc/slayer/improv/moonlake/show"
)

// lightshowRate is the frames a second a strobe's window is counted in
// when none is given. Defined by lightshow's own frame clock.
const lightshowRate = 12.0

// main folds the sheet up to the moment and says the lights' state.
func main() {
	if len(os.Args) != 4 && len(os.Args) != 5 {
		fmt.Println("cueshow score.yaml cues.yaml seconds [fps]")
		return
	}
	seconds, err := finiteNumber(os.Args[3])
	if err != nil {
		fail(err)
	}
	fps := lightshowRate
	if len(os.Args) == 5 {
		if fps, err = finiteNumber(os.Args[4]); err != nil {
			fail(err)
		}
	}
	score, err := show.ReadScore(os.Args[1])
	if err != nil {
		fail(err)
	}
	sheet, err := show.ReadSheet(os.Args[2], score)
	if err != nil {
		fail(err)
	}
	moment, err := sheet.At(seconds, fps)
	if err != nil {
		fail(err)
	}
	fmt.Print(moment.Lines())
}

// fail says what went wrong on stderr and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// finiteNumber parses text as a number that is neither NaN nor infinite.
// It returns an error that names text when text is not a finite number.
func finiteNumber(text string) (float64, error) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("cueshow: %q is not a finite number", text)
	}
	return value, nil
}
