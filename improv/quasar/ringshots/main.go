// Command ringshots draws the photon ring short's pictures from smoke 36,
// which is not changed: its own world, fresh, seen from its own eye, with
// the bending taken off for one shot and the camera aimed and narrowed
// for the close-in.
//
// Shots, by -shot:
//
//	bent      smoke 36's world and eye, at this size and width
//	straight  the same with the hole's field taken off: nothing bends
//	aimed     the same as bent, the camera turned up by -aim radians
//	diagram   the top-down picture of a ray from each image's band
//	measure   no picture: where each band lies in a picture of this
//	          width, and the close-in that shows the third image
//
// -reach sets the hole's reach, in masses. Smoke 36's is forty and its
// eye is at fifty, so the tracer draws light straight from the eye to
// forty masses before it bends it; that puts the shadow's edge about six
// pixels low in the close-in, against the march from the eye. A reach of
// sixty holds the eye, and the shipped shots use it. It costs no time.
//
// The same reach is where outgoing light is let go straight: the tracer's
// own twelve masses stepped the disc's far rim and creased its face.
//
// Each writes the linear light as PFM beside a PNG developed with the
// smoke's own tone. The tracer draws on the processor: the GPU refuses a
// world with a field.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// The numbers the command is made of.
const (
	photonRing = 36 // defined: the smoke that draws the photon ring
	ok         = 0  // defined: a command that did its work
	failed     = 1  // defined: one that did not
	degree     = math.Pi / 180
)

// The numbers the command picked: the quasar slice, 2026-11-12.
const (
	// pickedWidth and pickedHeight are a square under a portrait
	// short's sentences: its width, and as tall.
	pickedWidth  = 1080
	pickedHeight = 1080
	// pickedRays is how many rays across and down a pixel, smoke 36's own.
	pickedRays = 9
)

// shot is what one picture is: its world, its eye, its lens and size.
type shot struct {
	world         *thing.World
	eye           maths.Subspace
	wide          float64
	width, height int
	rays          int
}

// asked is the command line, read.
type asked struct {
	shot, out     string
	width, height int
	rays          int
	wide, aim     float64
	reach         float64
}

// shots is each shot by name, made from smoke 36 and what was asked.
var shots = map[string]func(smoke.Smoke, asked) shot{
	"bent":     bent,
	"straight": straight,
	"aimed":    aimed,
}

// main runs the command on its arguments and exits with the code run gives.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run reads the command line, draws the shot and writes it, and says
// how long the drawing took.
func run(args []string, out, problems io.Writer) int {
	given, err := read(args, problems)
	if err != nil {
		fmt.Fprintln(problems, err)
		return failed
	}
	ring, found := smoke.Get(photonRing)
	if !found {
		fmt.Fprintln(problems, "ringshots: smoke 36 is gone")
		return failed
	}
	if done, err := drawn(given, ring, out); done {
		if err != nil {
			fmt.Fprintln(problems, err)
			return failed
		}
		return ok
	}
	made, known := shots[given.shot]
	if !known {
		fmt.Fprintf(problems, "ringshots: no shot is called %q: "+
			"ask for bent, straight, aimed, diagram or measure\n", given.shot)
		return failed
	}
	start := time.Now()
	if err := draw(made(ring, given), ring, given.out); err != nil {
		fmt.Fprintln(problems, err)
		return failed
	}
	fmt.Fprintf(out, "%s: %dx%d at %d rays in %v\n", given.shot,
		given.width, given.height, given.rays, time.Since(start))
	return ok
}

// drawn does the shots that trace nothing, and says whether the shot
// was one of them.
func drawn(given asked, ring smoke.Smoke, out io.Writer) (bool, error) {
	wide := ring.Wide
	if given.wide > 0 {
		wide = given.wide * degree
	}
	switch given.shot {
	case "measure":
		return true, report(out, given, wide)
	case "diagram":
		return true, drawDiagram(given)
	}
	return false, nil
}

// read is the command line: the shot, where it goes, its size, rays and
// width, and the close-in's aim.
func read(args []string, problems io.Writer) (asked, error) {
	flags := flag.NewFlagSet("ringshots", flag.ContinueOnError)
	flags.SetOutput(problems)
	var given asked
	flags.StringVar(&given.shot, "shot", "bent", "bent, straight or aimed")
	flags.StringVar(&given.out, "out", "", "where the PNG goes; the PFM "+
		"beside it")
	flags.IntVar(&given.width, "width", pickedWidth, "pixels across")
	flags.IntVar(&given.height, "height", pickedHeight, "pixels down")
	flags.IntVar(&given.rays, "rays", pickedRays, "rays across a pixel")
	flags.Float64Var(&given.wide, "wide", 0, "degrees across; nought "+
		"is the smoke's own")
	flags.Float64Var(&given.reach, "reach", 0, "the hole's reach in "+
		"masses, past the eye to march light from it; nought is the "+
		"smoke's own")
	flags.Float64Var(&given.aim, "aim", 0, "radians the camera turns "+
		"up from the hole's centre, for aimed")
	if err := flags.Parse(args); err != nil {
		return asked{}, err
	}
	if given.out == "" {
		return asked{}, errors.New("ringshots: say where the picture " +
			"goes with -out")
	}
	return given, nil
}

// bent is smoke 36 as it is, at the asked size and width.
func bent(ring smoke.Smoke, given asked) shot {
	world, eye := ring.Scene()
	wide := ring.Wide
	if given.wide > 0 {
		wide = given.wide * degree
	}
	if given.reach > 0 {
		// smoke 36's hole is mass one, so a reach in masses is in
		// the world's units
		for _, field := range world.Fields {
			field.Reach = given.reach
			field.LetGo = given.reach
		}
	}
	return shot{world: world, eye: eye, wide: wide, width: given.width,
		height: given.height, rays: given.rays}
}

// straight is smoke 36 with the hole's field taken off: the disc and the
// hole's black ball as they are, and no ray bent.
func straight(ring smoke.Smoke, given asked) shot {
	made := bent(ring, given)
	made.world.Fields = nil
	return made
}

// aimed is smoke 36 with the camera turned up from the hole's centre by
// the aim, about the camera's own across.
func aimed(ring smoke.Smoke, given asked) shot {
	made := bent(ring, given)
	from := made.eye.Origin
	toward := from.Scale(-1)
	lift := maths.Up.Scale(toward.Len() * math.Tan(given.aim))
	made.eye.Orientation = maths.LookAt(from, from.Add(toward).Add(lift))
	return made
}

// draw traces a shot and writes its light and its developed picture.
func draw(made shot, ring smoke.Smoke, out string) error {
	lens, err := film.PinholeOf(made.wide)
	if err != nil {
		return fmt.Errorf("ringshots: %w", err)
	}
	light, err := trace.Sampled(made.world, made.eye, lens, 0, made.width,
		made.height, made.rays)
	if err != nil {
		return fmt.Errorf("ringshots: %w", err)
	}
	if err := raw.Write(out+".pfm", light); err != nil {
		return fmt.Errorf("ringshots: %w", err)
	}
	file, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("ringshots: %w", err)
	}
	defer file.Close()
	if err := png.Encode(file, ring.Tone.Develop(light)); err != nil {
		return fmt.Errorf("ringshots: %w", err)
	}
	return file.Close()
}
