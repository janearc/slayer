// Command compose writes one of the quasar shorts as a film, from its
// script in the walk package's form and a folder of its pictures, with
// no sound until a tune is chosen. With -still it writes one frame as a
// PNG instead, to look at before a whole film is made.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"time"

	"github.com/janearc/slayer/cheeptune"

	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/interfaces/movie"
)

// The numbers the command is made of: defined.
const (
	firstBar = 1 // bars count from one
	stereo   = 2 // samples a moment, left and right
	ok       = 0
	failed   = 1
	noStill  = -1
)

// asked is the command line, read.
type asked struct {
	script, pictures, short, out string
	score, voicing               string
	still, bar                   int
}

// main runs the command on its arguments and exits with the code run gives.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run reads the command line and writes the short, or its one frame.
func run(args []string, out, problems io.Writer) int {
	given, err := read(args, problems)
	if err == nil {
		err = write(given, out)
	}
	if err != nil {
		fmt.Fprintln(problems, err)
		return failed
	}
	return ok
}

// read is the command line: the script, its pictures, the short's name,
// where it goes, and a frame to write alone.
func read(args []string, problems io.Writer) (asked, error) {
	flags := flag.NewFlagSet("compose", flag.ContinueOnError)
	flags.SetOutput(problems)
	var given asked
	flags.StringVar(&given.script, "script", "", "the short's script")
	flags.StringVar(&given.pictures, "pictures", "", "its pictures' folder")
	flags.StringVar(&given.short, "short", "", "the short's name")
	flags.StringVar(&given.out, "out", "", "the film, or the frame's PNG")
	flags.IntVar(&given.still, "still", noStill, "write this frame alone")
	flags.StringVar(&given.score, "score", "", "a tune's score, for "+
		"under the short; none is silence")
	flags.StringVar(&given.voicing, "voicing", "", "the tune's voicing")
	flags.IntVar(&given.bar, "bar", firstBar, "the tune's bar the short "+
		"starts on")
	if err := flags.Parse(args); err != nil {
		return asked{}, err
	}
	if given.script == "" || given.short == "" || given.out == "" {
		return asked{}, errors.New("compose: give -script, -short and " +
			"-out")
	}
	return given, nil
}

// write writes the asked short, or its one frame.
func write(given asked, out io.Writer) error {
	script, err := walk.ReadScript(given.script, given.pictures)
	if err != nil {
		return err
	}
	short, found := script.Short(given.short)
	if !found {
		return fmt.Errorf("compose: the script has no short called %q",
			given.short)
	}
	if given.still > noStill {
		return still(short, given)
	}
	sound, err := soundFor(given, short.Seconds(), out)
	if err != nil {
		return err
	}
	frames, err := short.Write(context.Background(), given.out, sound)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s: %d frames, %.1f s\n", given.out, frames,
		short.Seconds())
	return nil
}

// soundFor is the tune's span under the short, from its bar, as the walk
// package cuts and lowers it, and says each bar's level under the short;
// with no score, silence.
func soundFor(given asked, seconds float64, out io.Writer) (movie.Sound,
	error) {
	if given.score == "" {
		return movie.Sound{}, nil
	}
	if given.bar < firstBar {
		return movie.Sound{}, fmt.Errorf("compose: bar %d is before "+
			"the first: bars count from one", given.bar)
	}
	piece, err := cheeptune.Open(given.score, given.voicing)
	if err != nil {
		return movie.Sound{}, err
	}
	meter := float64(piece.Score.Meter)
	from := piece.Score.SecondsAt(float64(given.bar-firstBar) * meter)
	samples := make([]float32, stereo*len(piece.Left))
	for i := range piece.Left {
		samples[stereo*i], samples[stereo*i+1] = piece.Left[i],
			piece.Right[i]
	}
	song, err := movie.NewSound(piece.Rate, samples)
	if err != nil {
		return movie.Sound{}, err
	}
	for bar := given.bar; ; bar++ {
		at := piece.Score.SecondsAt(float64(bar-firstBar) * meter)
		if at >= from+seconds {
			break
		}
		_, _, _, level, _ := piece.At(time.Duration(at *
			float64(time.Second)))
		fmt.Fprintf(out, "bar %d at %.3f s: level %.2f\n", bar, at, level)
	}
	return walk.SongSpan(song, from, seconds)
}

// still writes one frame of a short as a PNG.
func still(short walk.Short, given asked) error {
	frame, err := short.Frame(given.still, walk.NewStills())
	if err != nil {
		return err
	}
	file, err := os.Create(given.out)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := png.Encode(file, frame); err != nil {
		return err
	}
	return file.Close()
}
