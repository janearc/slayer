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
	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// the swarm in units of the star's radius: a haze to fourteen
const (
	star = 1.0
	haze = 14.0
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
	world, count, err := swarm(given)
	if err != nil {
		fail(err)
	}
	fmt.Printf("%d panels on %d orbits\n", count, given.orbits)
	start := time.Now()
	drawn, err := draw(world, given)
	if err != nil {
		fail(err)
	}
	if err := save(drawn, given.out); err != nil {
		fail(err)
	}
	fmt.Printf("%s: %dx%d, %d rays a pixel, %v\n", given.out, given.width,
		given.height, given.samples*given.samples,
		time.Since(start).Round(time.Second))
}

// fail says what went wrong and stops
func fail(err error) {
	fmt.Fprintln(os.Stderr, "swarm:", err)
	os.Exit(1)
}

// swarm is the world: the lantern, a star of radius one in its swarm,
// and the haze its light shows in
func swarm(given asked) (*thing.World, int, error) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(0.002, 0.002, 0.004), Seed: 7}
	count, err := lantern.Lantern{Radius: star, Orbits: given.orbits,
		Panels: given.panels, Kelvin: 4800, Green: 38, Wide: given.wide,
		Wear: given.wear, Turn: time.Minute,
		Skin:   lantern.BlackGlass(given.glass),
		Lining: lantern.Matte(given.glass),
		Shells: lantern.Shells{Spread: given.spread,
			Fastest: given.fastest, Narrowest: given.narrowest},
	}.Place(world)
	if err != nil {
		return nil, 0, err
	}
	world.AddVolume(hazeAround())
	return world, count, nil
}

// hazeAround is the haze the shafts show in: thin, ragged, scattering
// the lantern's light where no panel shades it
func hazeAround() thing.Volume {
	wisps := medium.Wispy{Of: medium.Puff{Radius: haze, Thick: 0.05},
		Seed: 3, Scale: 2.5, Amount: 0.45}
	dust := &thing.Material{Name: "haze",
		Colour: maths.RGB(0.9, 0.82, 0.72)}
	return thing.Volume{Medium: wisps, Material: dust}
}

// draw is the picture from a little above the swarm's plane, on the cpu
// or the gpu as asked
func draw(world *thing.World, given asked) (*frame.Frame, error) {
	from := maths.V(0, 4, 26).Unit().Scale(given.away)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	lens := film.Pinhole{Wide: 45 * maths.Degree}
	width, height := given.width, given.height
	if given.onGPU {
		return gpu.Sampled(world, eye, lens, given.at, width, height,
			given.samples)
	}
	return trace.Sampled(world, eye, lens, given.at, width, height,
		given.samples), nil
}

// save develops the picture and writes it as a png
func save(drawn *frame.Frame, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	developed := colour.Tone{Knee: 0.7, Top: 8}.Develop(drawn)
	return file.PNG(out, developed, 1)
}
