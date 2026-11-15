// Command orbital draws a pair of culture orbitals round a magnetar: two bands
// side by side on one axis, spinning opposite ways so their turning cancels, a
// gap between them where lights run along the walls and spill out onto the
// hulls.
//
// Each is the size a culture orbital is for one g and a day of 24 hours, 1.84
// million km in radius, and a thread at that size: 32,000 km wide, with walls a
// thousand high. The eye is inside, above the gap, looking along the floors as
// they curve up into an arch.
//
//	orbital [-w 1600] [-h 900] [-n 2] [-gpu] [-out out/orbital.png]
//	orbital -cells COLSxROWS [-frames 0] [-fps 24] [-n 2] [-far] [-keys]
//
// With -cells it writes no file. It writes the scene to standard output as
// frames of ANSI text, sextant cells in 24-bit colour, one frame after
// another as the scene turns. -frames 0 runs until the reader goes away.
//
// It asks the terminal nothing and sets no mode, so a window can show it.
// What it says of itself goes to standard error.
//
// With -keys as well it does ask: it takes the terminal over, draws a
// frame, and draws the next at each key. q, escape and ctrl-c stop it, and
// after the last frame any key does.
//
// SIGTERM, SIGINT and SIGHUP give the terminal back and exit 128 plus the
// signal. The frames are the same cells, at the same times, as the
// stream's.
//
// One unit is a thousand kilometres. The scene itself is scene/named's
// orbital; this command parses its flags and asks for it.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/trace"
)

// main draws the orbital scene from the flags and writes the picture.
func main() {
	w := flag.Int("w", 1600, "width in pixels")
	h := flag.Int("h", 900, "height in pixels")
	n := flag.Int("n", 2, "rays per pixel along each side")
	out := flag.String("out", "out/orbital.png", "where the picture goes")
	far := flag.Bool("far", false, "from far outside, 25 degrees above "+
		"their plane, the hoops standing upright, lit by the lantern")
	away := flag.Float64("from", 3600,
		"with -far, how far out the eye is, in thousands of km")
	frames := flag.Int("frames", 1, "how many frames; more than one "+
		"writes out-NNNN.png")
	rate := flag.Int("fps", 24, "frames a second")
	first := flag.Int("first", 0, "the frame to start at, to resume a film")
	still := flag.Bool("still", false, "hold the lantern's rings still")
	thicker := flag.Float64("gas", 1, "how many times thicker the nebula")
	neon := flag.Float64("neon", 0, "the share of the nebula that is "+
		"neon, nought to one")
	argon := flag.Float64("argon", 0, "the share of the nebula that is "+
		"argon, ionized; what neon and argon leave is hydrogen")
	glow := flag.Float64("glow", 0.2, "how bright the nebula's glow is, "+
		"as a luminance")
	keep := flag.Bool("raw", false, "keep the raw beside each png, as a "+
		".pfm: its light before it was developed")
	stops := flag.Float64("exposure", 0, "stops of exposure to develop "+
		"at; each one doubles the light")
	eyeAt := flag.String("eye", "", "where the eye is, as x,y,z in "+
		"thousands of km from the lantern; empty keeps the view's own")
	lookAt := flag.String("look", "0,0,0", "with -eye, what it looks at")
	upward := flag.String("up", "1,0,0", "with -eye, which way is up")
	wide := flag.Float64("wide", 90, "how wide the pinhole sees, "+
		"in degrees across")
	day := flag.Duration("day", 24*time.Hour, "how long one band takes "+
		"to turn; the pair turn opposite ways; nought holds them still")
	toCells := flag.String("cells", "", "COLSxROWS: write the scene to "+
		"standard output as frames of terminal cells, and no file; "+
		"-frames 0 has no end")
	onGPU := flag.Bool("gpu", false, "draw the pictures on the gpu, "+
		"with slayer's tracer; -cells draws on the cpu")
	keys := flag.Bool("keys", false, "with -cells, draw a frame "+
		"and wait for a key: q, escape or ctrl-c stop, any other "+
		"key draws the next; after the last frame any key ends it; "+
		"needs a terminal; -frames 0 has no end")
	flag.Parse()

	asked, err := askCells(cellsFlags{size: *toCells, set: givenFlags(),
		first: *first, frames: *frames, rate: *rate, rays: *n,
		printed: printing{stops: *stops, raw: *keep,
			wide: *wide * maths.Degree},
		keys: *keys, gpu: *onGPU, in: os.Stdin, out: os.Stdout})
	if err != nil {
		fail(err)
	}
	scene, err := named.Get("orbital")
	if err != nil {
		fail(err)
	}
	built, err := scene.Build(map[string]any{"far": *far,
		"distance": *away, "still": *still, "gas": *thicker,
		"neon": *neon, "argon": *argon, "glow": *glow, "day": *day},
		asked.chat())
	if err != nil {
		fail(err)
	}
	eye := built.Eye()
	if *eyeAt != "" {
		if eye, err = placed(*eyeAt, *lookAt, *upward); err != nil {
			fail(err)
		}
	}
	if asked.on {
		asked.follow(built)
		if err := asked.run(os.Stdout, built.World(), eye); err != nil {
			fail(err)
		}
		return
	}
	for shot := *first; shot < *first+*frames; shot++ {
		at := startsAt +
			time.Duration(shot)*time.Second/time.Duration(*rate)
		path := *out
		if *frames > 1 || *first > 0 {
			path = strings.TrimSuffix(*out, ".png") +
				fmt.Sprintf("-%04d.png", shot)
		}
		printed := printing{stops: *stops, raw: *keep,
			wide: *wide * maths.Degree, gpu: *onGPU}
		if err := draw(built, eye, at, *w, *h, *n, path,
			printed); err != nil {
			fail(err)
		}
	}
}

// placed is an eye at one point looking at another, its up as given,
// each written x,y,z in thousands of km
func placed(at, toward, up string) (maths.Subspace, error) {
	points := make([]maths.Vec, 0, 3)
	for _, written := range []string{at, toward, up} {
		point, err := point(written)
		if err != nil {
			return maths.Subspace{}, err
		}
		points = append(points, point)
	}
	return maths.Subspace{Origin: points[0], Orientation: maths.LookAtUp(
		points[0], points[1], points[2])}, nil
}

// point is a point written x,y,z
func point(written string) (maths.Vec, error) {
	parts := strings.Split(written, ",")
	if len(parts) != 3 {
		return maths.Vec{}, fmt.Errorf("%q is not x,y,z", written)
	}
	var axes [3]float64
	for i, part := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return maths.Vec{}, fmt.Errorf("%q is not x,y,z",
				written)
		}
		axes[i] = number
	}
	return maths.V(axes[0], axes[1], axes[2]), nil
}

// fail says what went wrong and stops
func fail(err error) {
	fmt.Fprintln(os.Stderr, "orbital:", err)
	os.Exit(1)
}

// printing is how a frame is printed: how many stops of exposure it is
// developed at, whether its raw is kept beside it, how wide the lens
// sees, and whether the gpu draws it
type printing struct {
	stops float64
	raw   bool
	wide  float64
	gpu   bool
}

// draw is one frame at a time, written as a png, and its raw too when
// printed says to keep it. The scene is told the frame's time first, so
// the lantern's beams keep it.
func draw(built *named.Built, eye maths.Subspace, at time.Duration,
	width, height, side int, path string, printed printing) error {
	start := time.Now()
	built.Set(at)
	drawn, err := sampled(built, eye, at, width, height, side, printed)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if printed.raw {
		kept := strings.TrimSuffix(path, ".png") + ".pfm"
		if err := raw.Write(kept, drawn); err != nil {
			return err
		}
	}
	developed := developer(printed).Develop(drawn)
	if err := file.PNG(path, developed, 1); err != nil {
		return err
	}
	fmt.Printf("%s: %dx%d, %d rays a pixel, %v\n", path, width, height,
		side*side, time.Since(start).Round(time.Second))
	return nil
}

// sampled is the frame drawn on the cpu, or on the gpu when printed says.
func sampled(built *named.Built, eye maths.Subspace, at time.Duration,
	width, height, side int, printed printing) (*frame.Frame, error) {
	lens := lensFor(printed, square)
	if printed.gpu {
		return gpu.Sampled(built.World(), eye, lens, at, width, height,
			side)
	}
	return trace.Sampled(built.World(), eye, lens, at, width, height,
		side)
}
