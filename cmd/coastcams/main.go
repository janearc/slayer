// Command coastcams prints the coast cameras of a set's generated planet,
// ready to paste under the set's cameras: coast-low, over the shore
// looking down, and coast-high, over the sea looking at the shore. They
// are found from the planet's own ground (package scene/generate), so
// running it again after any terrain change puts them back on the coast.
//
// usage:
//
//	coastcams cameras SET [--from X,Y,Z] [--height METRES] [--body NAME]
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/set"
)

// build, built, who and sessions are stamped by game at link time.
var build, built, who, sessions string

// The defaults: the direction the walk starts from, in the planet's own
// frame, and the low camera's height in metres. By eye (2026-11-26): the
// Earth like sets' coast-low cameras stood about twenty kilometres up.
const (
	defaultFrom   = "1,0,0"
	defaultHeight = 20000
	// axes is how many numbers make a direction.
	axes = 3
	// metresPerKilometre turns a height into the kilometres a title says.
	metresPerKilometre = 1000
)

// main runs coastcams as the command line asks and exits with its status.
func main() {
	os.Exit(tool().Run(os.Args[1:], os.Stdout, os.Stderr))
}

// tool is the coastcams command: its one verb, cameras.
func tool() *pzm.Tool {
	made := pzm.New("coastcams", "print a set's generated planet's coast cameras",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.FullCPU)
	made.Verb("cameras", verb.NoShort, "print the coast cameras of a set",
		camerasPage, "Example_cameras", facts, cameras,
		pzm.List("from", verb.NoShort,
			"x,y,z, the direction the walk starts from"),
		pzm.Value("height", verb.NoShort,
			"metres the low camera stands over the shore"),
		pzm.Value("body", verb.NoShort, "the generated body, by name"))
	return made
}

// camerasPage is the cameras verb's page.
const camerasPage = `# cameras

cameras SET [--from X,Y,Z] [--height METRES] [--body NAME]

Reads SET, takes its generated body (the one named by --body, or the first),
and walks a great circle over its ground from the direction --from, in the
planet's own frame before its tilt, until land meets sea under a clear sky:
the planet's own clouds are asked over each shore and round it, and where no
shore along the circle is clear, the least clouded is taken. Prints two
cameras as YAML, to paste under the set's cameras: coast-low, --height metres
over the shore looking straight down, and coast-high, sixteen times as high
and as far back over the sea, looking at the same shore.

The ground and the clouds are the ones the set draws, so run it again after
any change to either. A planet with no sea, or none along the circle, prints
nothing and says so; try another --from. It writes nothing.
`

// cameras is the cameras verb: it reads the set, finds the coast and
// prints the two cameras.
func cameras(call pzm.Call) error {
	args := call.Args()
	if len(args) != 1 {
		return fmt.Errorf("cameras: name one SET file, not %d", len(args))
	}
	from, height, err := asked(call)
	if err != nil {
		return err
	}
	planet, middle, detail, err := planetOf(args[0], call.Value("body"))
	if err != nil {
		return err
	}
	found, err := generate.FindCoast(planet, detail, from, height)
	if err != nil {
		return fmt.Errorf("cameras: %w", err)
	}
	return printCameras(call.Out(), planet.Name, middle, height, found)
}

// asked is the direction and height the call asks for, or the defaults.
func asked(call pzm.Call) (maths.Vec, float64, error) {
	numbers := strings.Split(defaultFrom, ",")
	if call.Given("from") {
		numbers = call.List("from")
	}
	if len(numbers) != axes {
		return maths.Vec{}, 0, fmt.Errorf("cameras: --from %q is not x,y,z",
			strings.Join(numbers, ","))
	}
	var parts [axes]float64
	for index, text := range numbers {
		number, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return maths.Vec{}, 0, fmt.Errorf("cameras: --from: %w", err)
		}
		parts[index] = number
	}
	height, err := heightOf(call)
	return maths.V(parts[0], parts[1], parts[2]), height, err
}

// heightOf is the height the call asks for, in metres, or the default.
func heightOf(call pzm.Call) (float64, error) {
	if !call.Given("height") {
		return defaultHeight, nil
	}
	height, err := strconv.ParseFloat(call.Value("height"), 64)
	if err != nil {
		return 0, fmt.Errorf("cameras: --height: %w", err)
	}
	return height, nil
}

// planetOf reads the set and gives a generated body: the planet, where
// its middle stands, and the detail its ground is tiled at.
func planetOf(file, name string) (generate.Planet, maths.Vec, int, error) {
	source, err := os.ReadFile(file)
	if err != nil {
		return generate.Planet{}, maths.Vec{}, 0, err
	}
	read, err := set.Read(file, source)
	if err != nil {
		return generate.Planet{}, maths.Vec{}, 0, err
	}
	world, _ := read.World()
	for _, body := range world.Bodies() {
		planet, generated := body.Generated()
		if generated && (name == "" || body.Name() == name) {
			return planet, body.Middle(), body.Detail(), nil
		}
	}
	return generate.Planet{}, maths.Vec{}, 0, fmt.Errorf("cameras: %s has "+
		"no generated body called %q", file, name)
}

// printCameras writes the two cameras as YAML under a cameras key's
// indent, their places moved by the body's middle.
func printCameras(out io.Writer, name string, middle maths.Vec,
	height float64, found generate.Cameras) error {
	km := height / metresPerKilometre
	err := printCamera(out, "coast-low", fmt.Sprintf(
		"%s, %.0f km down at the coast", name, km), middle, found.Low)
	if err != nil {
		return err
	}
	return printCamera(out, "coast-high", fmt.Sprintf(
		"%s, high over the sea, looking at the coast", name), middle,
		found.High)
}

// printCamera writes one camera's YAML, its places in the set file's map
// metres.
func printCamera(out io.Writer, name, title string, middle maths.Vec,
	camera generate.Camera) error {
	at := toMap(camera.At.Add(middle))
	look := toMap(camera.LookAt.Add(middle))
	_, err := fmt.Fprintf(out, "    %s:\n      title: %s\n"+
		"      location: {at: [%.0f, %.0f, %.0f]}\n"+
		"      orientation: {look_at: [%.0f, %.0f, %.0f]}\n", name, title,
		at[0], at[1], at[2], look[0], look[1], look[2])
	return err
}

// toMap turns a place in the world's space, y up, into a set file's
// metres, which run as a map's do: x east, y north, z up. A set reads a
// file's x, y and z as the world's x, z and minus y, so this is that
// reading turned back. Printed as the world's, a camera stood a quarter
// turn about x from the coast it was found for.
func toMap(place maths.Vec) [axes]float64 {
	return [axes]float64{place.X, 0 - place.Z, place.Y}
}
