// Command moonrise draws a full moon rising behind a tree on the
// earth, through a long lens, as a raw: the linear light of every
// pixel, for developing after.
//
//	moonrise -out x.pfm [-preview x.png] [-stops 0] [-kind norway]
//	         [-seed 3] [-away 1100] [-moon 0.3] [-sun 2] [-wide 2]
//	         [-w 1430] [-h 1000] [-n 2]
//
// the moon is its own size at its own distance, by nasa's moon fact
// sheet, and the sun is distant, low behind the eye. the frame is
// imax's, 1.43 to 1; two degrees wide is about a 2000 mm lens on 15/70.
//
// picked, and said so: where the moon stands, straight ahead at the
// altitude asked; the earth's colour; and white light, since reddening
// is the atmosphere's and the tracer has none. the preview is linear
// light by stops through srgb's transfer, with no film curve.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

// errAsked is a command asked for what it cannot draw.
var errAsked = errors.New("moonrise: cannot draw what was asked")

// asked is what the command was asked to draw, and how.
type asked struct {
	out, preview, kind     string
	seed                   uint64
	stops, away, moon, sun float64
	wide                   float64
	width, height, samples int
}

// parse is what the arguments ask, or why they cannot be drawn.
func parse(arguments []string, say io.Writer) (asked, error) {
	given := asked{}
	flags := flag.NewFlagSet("moonrise", flag.ContinueOnError)
	flags.SetOutput(say)
	flags.StringVar(&given.out, "out", "", "where the raw goes, a .pfm")
	flags.StringVar(&given.preview, "preview", "", "a plain png beside it")
	flags.Float64Var(&given.stops, "stops", 0, "the preview's exposure")
	flags.StringVar(&given.kind, "kind", "norway", "the tree's kind")
	flags.Uint64Var(&given.seed, "seed", 3, "the tree's seed")
	flags.Float64Var(&given.away, "away", 1100, "metres to the tree")
	flags.Float64Var(&given.moon, "moon", 0.3, "the moon's altitude, deg")
	flags.Float64Var(&given.sun, "sun", 2, "the sun's altitude, deg")
	flags.Float64Var(&given.wide, "wide", 2, "the lens's width, deg")
	flags.IntVar(&given.width, "w", 1430, "width in pixels")
	flags.IntVar(&given.height, "h", 1000, "height in pixels")
	flags.IntVar(&given.samples, "n", 2, "rays per pixel along each side")
	if err := flags.Parse(arguments); err != nil {
		return asked{}, err
	}
	return given, given.sound()
}

// sound is what is wrong with what was asked, by flag, or nothing.
func (given asked) sound() error {
	for flag, good := range map[string]bool{
		"-out":  given.out != "",
		"-wide": given.wide > 0 && given.wide < 180,
		"-sun":  given.sun >= 0 && given.sun <= 90,
		"-moon": given.moon >= -5 && given.moon <= 90,
		"-away": given.away > 0,
		"-w":    given.width >= 1, "-h": given.height >= 1,
		"-n": given.samples >= 1,
	} {
		if !good {
			return fmt.Errorf("%w: see %s", errAsked, flag)
		}
	}
	return nil
}

// main is run, and the number it says.
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run draws what the arguments ask and keeps its raw, saying what it drew; zero
// when it did, one when it could not.
func run(arguments []string, said, wrong io.Writer) int {
	given, err := parse(arguments, wrong)
	if err != nil {
		fmt.Fprintln(wrong, err)
		return 1
	}
	started := time.Now()
	if err := made(given); err != nil {
		fmt.Fprintln(wrong, "moonrise:", err)
		return 1
	}
	fmt.Fprintf(said, "%s: %dx%d, %d rays a pixel, %v\n", given.out,
		given.width, given.height, given.samples*given.samples,
		time.Since(started).Round(time.Millisecond))
	return 0
}
