// Command hosted writes a quasar short with zappy hosting it: zappy's
// intro over the tune's first part, then the short's frames from its
// walk script, zappy small at the foot drifting and flickering in time
// with the tune, the picture lifted on the kicks, and the tune's own
// sound from its film.
//
// Until zappy's clips exist, the host is a grey outline and the intro a
// black card with zappy's line on it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"strings"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/improv/host"
	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/interfaces/movie"
)

// The numbers the command is made of.
const (
	ok     = 0 // defined
	failed = 1 // defined
	// the frame and its picture, the walk package's: defined there
	frameWide   = 1080
	frameTall   = 1920
	pictureTall = 1080
	// the host's box, under the panel's sentence, and its size: picked
	// by the quasar slice, 2026-11-12, clear of the sentence's four rows
	boxTop   = 1730
	boxFoot  = 1910
	boxSide  = 20
	hostSide = 160
	seed     = 618 // picked: any number, kept so frames repeat
)

// asked is the command line, read.
type asked struct {
	script, pictures, short, out string
	score, sheet, film, clips    string
	intro, kicks                 string
	introSeconds                 float64
}

// main runs the command on its arguments and exits with the code run gives.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run reads the command line and writes the hosted short.
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

// read is the command line.
func read(args []string, problems io.Writer) (asked, error) {
	flags := flag.NewFlagSet("hosted", flag.ContinueOnError)
	flags.SetOutput(problems)
	var given asked
	for _, each := range []struct {
		into  *string
		name  string
		usage string
	}{
		{&given.script, "script", "the short's walk script"},
		{&given.pictures, "pictures", "its pictures' folder"},
		{&given.short, "short", "the short's name"},
		{&given.out, "out", "the film"},
		{&given.score, "score", "the tune's score"},
		{&given.sheet, "sheet", "the tune's cut sheet"},
		{&given.film, "film", "the tune's own film, for its sound"},
		{&given.clips, "clips", "zappy's clips' folder"},
		{&given.intro, "intro", "zappy's opening line"},
		{&given.kicks, "kicks", "the parts that kick, by comma"},
	} {
		flags.StringVar(each.into, each.name, "", each.usage)
	}
	flags.Float64Var(&given.introSeconds, "intro-seconds", 0,
		"how long the intro is")
	if err := flags.Parse(args); err != nil {
		return asked{}, err
	}
	if given.script == "" || given.out == "" || given.score == "" ||
		given.sheet == "" || given.film == "" {
		return asked{}, errors.New("hosted: give -script, -out, " +
			"-score, -sheet and -film")
	}
	return given, nil
}

// write makes the show and writes it.
func write(given asked, out io.Writer) error {
	script, err := walk.ReadScript(given.script, given.pictures)
	if err != nil {
		return err
	}
	short, found := script.Short(given.short)
	if !found {
		return fmt.Errorf("hosted: no short is called %q", given.short)
	}
	tune, err := tuneOf(given)
	if err != nil {
		return err
	}
	clips, err := host.ReadClips(given.clips, image.Pt(hostSide, hostSide))
	if err != nil {
		return err
	}
	box, err := host.NewBox(image.Rect(boxSide, boxTop, frameWide-boxSide,
		boxFoot), image.Pt(hostSide, hostSide))
	if err != nil {
		return err
	}
	kept := walk.NewStills()
	show := host.Show{
		Frames: func(i int) (*image.RGBA, error) {
			return short.Frame(i, kept)
		},
		Count: short.Frames(), Tune: tune, Clips: clips, Box: box,
		Speaks: speaking(short), Lifts: set(given.kicks),
		Picture: image.Rect(0, 0, frameWide, pictureTall),
		Intro: host.Intro{Words: given.intro,
			Seconds: given.introSeconds},
		Seed: seed, Size: image.Pt(frameWide, frameTall)}
	frames, err := show.Write(context.Background(), given.out)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s: %d frames\n", given.out, frames)
	return nil
}

// tuneOf is the tune from its score, its sheet and its own film's sound.
func tuneOf(given asked) (host.Tune, error) {
	score, err := cheeptune.ReadScore(given.score)
	if err != nil {
		return host.Tune{}, err
	}
	sheet, err := cheeptune.ReadCuts(given.sheet, score)
	if err != nil {
		return host.Tune{}, err
	}
	sound, err := movie.ReadSound(given.film)
	if err != nil {
		return host.Tune{}, err
	}
	return host.NewTune(score, sheet, sound, names(given.kicks))
}

// speaking is when the host speaks: every shot with a sentence, whole.
func speaking(short walk.Short) []host.Span {
	var spans []host.Span
	from := 0.0
	for _, shot := range short.Shots() {
		if shot.Sentence() != "" {
			spans = append(spans, host.Span{From: from,
				Until: from + shot.Seconds()})
		}
		from += shot.Seconds()
	}
	return spans
}

// names is a comma list's names.
func names(list string) []string {
	var named []string
	for _, name := range strings.Split(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			named = append(named, name)
		}
	}
	return named
}

// set is a comma list as a set.
func set(list string) map[string]bool {
	made := map[string]bool{}
	for _, name := range names(list) {
		made[name] = true
	}
	return made
}
