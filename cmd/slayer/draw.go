package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/learn"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/compare"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// draw draws a set file the way the smokecheck draws a smoke: as the
// terminal shows it and as a real picture, from one world and one
// camera, and judges the terminal's against the real, with the
// comparison. nothing is cached: a set is drawn every time, into -out.
//
// -at is the world's time, for a set whose things move. -camera names
// the camera, the set's first when unsaid. -samples is rays a terminal
// pixel.
//
// beside the two pictures and the verdict it writes a third, the real
// picture developed through the set's print, which is how the set asks
// to be seen. the verdict is on the plain two, as a smoke's is.
func draw(args []string) (bool, error) {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return false, fmt.Errorf("which set? slayer set FILE [flags]")
	}
	path, args := args[0], args[1:]
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	out := fs.String("out", ".", "the folder the pictures go in")
	at := fs.Duration("at", 0, "the world's time to draw it at")
	k := fs.Int("k", 2, "how much finer again the real picture is")
	samples := fs.Int("samples", 1, "rays a terminal pixel")
	camera := fs.String("camera", "", "the camera to draw, by name")
	if err := fs.Parse(args); err != nil {
		return false, err
	}
	if *k < 1 {
		return false, fmt.Errorf("-k is at least 1, not %d", *k)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	read, err := set.Read(path, written)
	if err != nil {
		return false, err
	}
	said, has := read.World()
	if !has {
		return false, fmt.Errorf("%s says no world to draw", path)
	}
	world, err := stage.World(said)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if least := Fineness(world); *k < least {
		return false, fmt.Errorf("%s has a field, and a reference is "+
			"never coarser than what it judges: -k is at least %d, "+
			"not %d", path, least, *k)
	}
	if *camera == "" && len(read.Cameras()) > 0 {
		*camera = read.Cameras()[0]
	}
	shot, err := read.Shot(*camera, nowhere)
	if err != nil {
		return false, err
	}
	eye := shot.Eye()
	filmed, _ := read.Film()
	wide := stage.Lens(filmed).Wide

	sextant := film.Pinhole{Wide: wide, Pixel: 0.75}
	cells := trace.Sampled(world, eye, sextant, *at, textW, textH,
		*samples)
	text := learn.Chopped(terminal.Chop(colour.Develop(cells, 1)))
	square := film.Pinhole{Wide: wide}
	big := trace.Render(world, eye, square, *at, textW*across*(*k),
		textH*down*(*k))
	real := colour.Develop(big, 1)

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if *at != 0 {
		name += "-at-" + at.String()
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return false, err
	}
	look := stage.Print(filmed)
	for _, p := range []struct {
		suffix string
		aspect float64
		write  func(string, float64) error
	}{
		{"-text.png", 0.75, func(f string, a float64) error { return file.PNG(f, text, a) }},
		{"-real.png", 1, func(f string, a float64) error { return file.PNG(f, real, a) }},
		{"-look.png", 1, func(f string, a float64) error { return file.PNG(f, look.Develop(big), a) }},
	} {
		if err := p.write(filepath.Join(*out, name+p.suffix), p.aspect); err != nil {
			return false, err
		}
	}
	r := compare.Cells(real, text, across*(*k), down*(*k))
	if err := writeVerdict(filepath.Join(*out, name+"-verdict.json"),
		r); err != nil {
		return false, err
	}
	fmt.Printf("set %s at %s: %s. %d of %d terminal pixels "+
		"compared, worst %.4f at %d,%d, mean %.4f\n", name, at, judged(r),
		r.Compared, r.Pixels, r.Worst, r.At.X, r.At.Y, r.Mean)
	return r.Honest(), nil
}

// judged is a comparison's verdict, in a word or two.
func judged(r compare.Report) string {
	if !r.Honest() {
		return "not honest"
	}
	return "honest"
}

// nowhere answers a camera that stands off an asset: a set drawn here
// places none of its assets, so such a camera is refused by name.
func nowhere(string) (maths.Vec, bool) { return maths.Vec{}, false }
