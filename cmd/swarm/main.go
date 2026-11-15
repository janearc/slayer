// Command swarm draws a dyson swarm: a star wrapped in thousands of small
// mirrored panels, each on its own tilted orbit and facing in, inside a
// thin haze, so the star's light comes out between the panels in shafts,
// the way a pierced tin lantern throws its pattern.
//
//	swarm [-w 1600] [-h 900] [-n 2] [-orbits 60] [-panels 120]
//	      [-out out/swarm.png] [-gpu]
//
// The star's light is slayer's point emitter, a black body's spectrum
// from libtheme, since scene/thing's lamps are a sun, a spot and a flood.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/trace"
)

// asked is what the command was asked to draw, and how
type asked struct {
	width, height, samples int
	orbits, panels         int
	wide, wear             float64
	glass, spread, away    float64
	narrowest              float64
	fastest                time.Duration
	at                     time.Duration
	out                    string
	onGPU                  bool
}

// parse reads the flags
func parse() asked {
	var given asked
	flag.IntVar(&given.width, "w", 1600, "width in pixels")
	flag.IntVar(&given.height, "h", 900, "height in pixels")
	flag.IntVar(&given.samples, "n", 2, "rays per pixel along each side")
	flag.IntVar(&given.orbits, "orbits", 60,
		"how many tilted orbits the panels are on")
	flag.IntVar(&given.panels, "panels", 120,
		"how many panels to an orbit")
	flag.Float64Var(&given.wide, "wide", 0.4,
		"how wide each band of panels is, in radii of the star")
	flag.DurationVar(&given.at, "at", 0, "the time to draw it at")
	flag.Float64Var(&given.wear, "wear", 0,
		"how battered the panels are, nought to one")
	flag.Float64Var(&given.glass, "glass", 0, "panels of black glass "+
		"outside at this index, matte black inside; nought is warm "+
		"metal")
	flag.Float64Var(&given.away, "from", 26,
		"how far out the eye is, in radii of the star")
	flag.Float64Var(&given.spread, "spread", 0,
		"radii of the star between one ring and the next out")
	flag.Float64Var(&given.narrowest, "narrowest", 0,
		"the outermost ring's width, narrowing from -wide")
	flag.DurationVar(&given.fastest, "fastest", 0,
		"the outermost ring's turn, quickening from a minute")
	flag.StringVar(&given.out, "out", "out/swarm.png",
		"where the picture goes")
	flag.BoolVar(&given.onGPU, "gpu", false,
		"draw it on the gpu, with slayer's tracer")
	flag.Parse()
	return given
}

// main parses, builds the swarm, draws it and saves it
func main() {
	given := parse()
	scene, err := named.Get("swarm")
	if err != nil {
		fail(err)
	}
	built, err := scene.Build(given.with(), os.Stdout)
	if err != nil {
		fail(err)
	}
	fmt.Printf("%d panels on %d orbits\n", given.orbits*given.panels,
		given.orbits)
	start := time.Now()
	drawn, err := draw(built, scene.Defaults(), given)
	if err != nil {
		fail(err)
	}
	if err := save(drawn, scene.Defaults(), given.out); err != nil {
		fail(err)
	}
	fmt.Printf("%s: %dx%d, %d rays a pixel, %v\n", given.out, given.width,
		given.height, given.samples*given.samples,
		time.Since(start).Round(time.Second))
}

// with is the scene's parameters as the flags give them; -wide is the
// scene's band-width and -from its distance.
func (given asked) with() map[string]any {
	return map[string]any{"orbits": given.orbits,
		"panels": given.panels, "band-width": given.wide,
		"wear": given.wear, "glass": given.glass,
		"distance": given.away, "spread": given.spread,
		"narrowest": given.narrowest, "fastest": given.fastest}
}

// fail says what went wrong and stops
func fail(err error) {
	fmt.Fprintln(os.Stderr, "swarm:", err)
	os.Exit(1)
}

// draw is the picture from the scene's eye through its lens, on the cpu
// or the gpu as asked
func draw(built *named.Built, defaults named.Defaults,
	given asked) (*frame.Frame, error) {
	lens := film.Pinhole{Wide: defaults.Wide()}
	width, height := given.width, given.height
	if given.onGPU {
		return gpu.Sampled(built.World(), built.Eye(), lens, given.at,
			width, height, given.samples)
	}
	return trace.Sampled(built.World(), built.Eye(), lens, given.at,
		width, height, given.samples)
}

// save develops the picture with the scene's tone and writes it as a png
func save(drawn *frame.Frame, defaults named.Defaults, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return file.PNG(out, defaults.Tone().Develop(drawn), 1)
}
