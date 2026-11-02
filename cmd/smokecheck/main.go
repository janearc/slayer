// Command smokecheck renders slayer's smokes twice, as the terminal sees
// them and as a real picture, and checks that the terminal's is the real
// one, coarser.
//
//	smokecheck smoke [N] [-k K] [-out DIR] [-fresh]
//
// with no number it does every smoke. the pictures and their verdicts go in
// smokecheck's cache, and a smoke this binary has drawn before is served, not
// drawn again; see package cache. DIR, when given, gets a copy of them.
//
// -fresh draws every smoke again. K makes the real picture K times finer
// again; at 1, each terminal pixel is three real pixels across and four
// down, the shape of a sextant.
//
// smokecheck checks slayer's own smokes.
package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/learn"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	cache "github.com/janearc/slayer/render/smokecache"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/thing"
)

// the terminal smokecheck pretends to be: the smokes' own picture-only
// size, 179 by 50 cells, and a sextant 0.75 as wide as it is tall, which is
// three by four
const (
	textW, textH = 358, 150
	across, down = 3, 4
)

// main runs the one verb there is.
func main() {
	if len(os.Args) < 2 || os.Args[1] != "smoke" {
		fmt.Fprintln(os.Stderr,
			"usage: smokecheck smoke [N] [-k K] [-out DIR] [-fresh]")
		os.Exit(2)
	}
	honest, err := run(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "smokecheck smoke:", err)
		os.Exit(1)
	}
	if !honest {
		os.Exit(1)
	}
}

// run does the smokes asked for and says whether every one was honest.
func run(args []string) (bool, error) {
	which := smoke.All()
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			s, ok := smoke.Get(n)
			if !ok {
				return false,
					fmt.Errorf("slayer has no smoke %d", n)
			}
			which, args = []smoke.Smoke{s}, args[1:]
		}
	}
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	out := fs.String("out", "", "a folder to copy the pictures to as well")
	k := fs.Int("k", 2, "how much finer again the real picture is")
	fresh := fs.Bool("fresh", false, "draw every smoke again")
	if err := fs.Parse(args); err != nil {
		return false, err
	}
	if *k < 1 {
		return false, fmt.Errorf("-k is at least 1, not %d", *k)
	}
	// each smoke's scene is built once and handed on: to the refusal here
	// and then to the shelf, which draws from it on a miss, so what was
	// refused is what is drawn
	scenes := make([]scene, len(which))
	for i, s := range which {
		scenes[i].world, scenes[i].eye = s.Scene()
		if least := Fineness(scenes[i].world); *k < least {
			return false, fmt.Errorf("smoke %d, %s, has a "+
				"field, and a reference is never coarser "+
				"than what it judges: -k is at least %d, "+
				"not %d", s.Number, s.Name, least, *k)
		}
	}
	if *out != "" {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			return false, err
		}
	}
	sh := openShelf(*fresh)
	defer sh.close()
	if sh.off != "" {
		fmt.Println("cache off, so every smoke is drawn and none kept:",
			sh.off)
	} else if !cache.Trimmed() {
		fmt.Println("built without -trimpath, so what this binary " +
			"draws is shared with no other tree's")
	}
	honest := true
	for i, s := range which {
		got, err := sh.get(s, scenes[i], *k)
		if err != nil {
			return false, err
		}
		if *out != "" {
			err = copyOut(got, s, *out)
		}
		got.entry.Close()
		if err != nil {
			return false, err
		}
		r := got.report
		verdict := "honest"
		if !r.Honest() {
			verdict, honest = "not honest", false
		}
		how := "served"
		if got.drawn {
			how = "drawn"
		}
		if got.unkept != "" {
			how += ", not kept: " + got.unkept
		}
		fmt.Printf("smoke %d, %s: %s, %s. %d of %d terminal pixels "+
			"compared, worst %.4f at %d,%d, mean %.4f\n", s.Number,
			s.Name, verdict, how, r.Compared, r.Pixels, r.Worst,
			r.At.X, r.At.Y, r.Mean)
	}
	fmt.Printf("drew %d, served %d\n", sh.drew, sh.served)
	if sh.off == "" {
		fmt.Println("pictures and verdicts in",
			filepath.Join(sh.root, "v1"))
	}
	if *out != "" {
		fmt.Println("copies in", *out)
	}
	return honest, nil
}

// scene is a smoke's world and where its eye is, as Scene built them.
type scene struct {
	world *thing.World
	eye   maths.Subspace
}

// Fineness is the least k a smoke's world may be judged at: two for a
// world with a field, since only near a hole is there light thinner than a
// pixel, which one real pixel a sextant third misses and the terminal's
// rays catch; one for the rest.
func Fineness(w *thing.World) int {
	if len(w.Fields) > 0 {
		return 2
	}
	return 1
}

// copyOut puts a smoke's pictures and verdict in out as well, named for the
// smoke, from the entry that was judged.
func copyOut(got shelved, s smoke.Smoke, out string) error {
	name := fmt.Sprintf("smoke-%d", s.Number)
	for from, to := range map[string]string{
		cache.Text: name + "-text.png", cache.Real: name + "-real.png",
		cache.Verdict: name + "-verdict.json"} {
		b, err := got.entry.Bytes(from)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, to), b,
			0o644); err != nil {
			return err
		}
	}
	return nil
}

// Both is one smoke as the terminal shows it, one pixel a sextant, and as a
// real picture of square pixels, from the one scene it is given and the
// same camera: the world is not built again for the second picture.
func Both(s smoke.Smoke, sc scene, k int) (text, real image.Image) {
	sextant := film.Pinhole{Wide: s.Wide, Pixel: 0.75}
	frame := trace.Sampled(sc.world, sc.eye, sextant, 0, textW, textH,
		s.Samples)
	text = learn.Chopped(terminal.Chop(colour.Develop(frame, 1)))

	square := film.Pinhole{Wide: s.Wide}
	big := trace.Render(sc.world, sc.eye, square, 0, textW*across*k,
		textH*down*k)
	return text, colour.Develop(big, 1)
}
