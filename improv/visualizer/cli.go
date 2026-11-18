// Command visualizer draws a tune's instruments in time with its music,
// for placing a part by ear and by eye. It writes raw rgb24 frames to
// stdout for ffmpeg to put the song under, as the moonlake metronome
// does.
//
// At the top, the playhead's place as BAR.BEAT.SIXTEENTH, counted from
// one, with a square for each beat of the bar and a tick for each
// sixteenth. Under it, a lane for each instrument, its notes on a time
// axis that scrolls past a fixed line; a note glows while it sounds. A
// percussion lane has a row for each hit. Bar numbers run along the foot.
//
//	visualizer draw --fps 30 --width 1080 --height 1080 \
//	  'LEAD=lead.json#lead' 'DRUMS=drums.json#perc' | ffmpeg -f rawvideo \
//	  -pix_fmt rgb24 -s 1080x1080 -r 30 -i - -i song.wav ... out.mp4
//
// Each lane is NAME=SCORE#ROLE: the name shown, a cheeptune score, and
// the part of it to draw. The tempo and the meter come from the scores,
// which must agree; bar one, beat one is at the start of the song.
package main

import (
	"fmt"
	"os"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// main declares the visualizer through Pzm, like the metronome.
func main() {
	tool := pzm.New("visualizer", "draw a tune's instruments in time",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("raw RGB frames on stdout"),
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("draw", verb.NoShort, "write the visualizer's frames",
		"Writes RGB24 frames, one lane per NAME=SCORE#ROLE argument. "+
			"Mux the song beneath them.", "Example_draw", facts, drawVerb,
		pzm.Value("fps", verb.NoShort, "frames per second"),
		pzm.Value("width", verb.NoShort, "frame width"),
		pzm.Value("height", verb.NoShort, "frame height"),
		pzm.Value("seconds", verb.NoShort,
			"film length; the score's and two more if empty"))
	os.Exit(tool.RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// Build stamps are filled by Game.
var build, built, who, sessions string

// drawVerb checks for lanes and hands the call to render.
func drawVerb(call pzm.Call) error {
	if len(call.Args()) == 0 {
		return fmt.Errorf("draw needs at least one NAME=SCORE#ROLE lane")
	}
	return render(call)
}
