// Command hillside draws a hillside of made things: as many spruces, ferns,
// stones, logs, cans, campfires, spotlights and masers as are asked for, each
// from the manufactory, each from its own seed or one of a variety, stood on a
// ground at seeded places.
//
//	hillside [-seed 1] [-spruce 5] [-fern 24] [-boulder 8] ...
//	         [-variety 0] [-apart 2.5] [-night] [-above] [-from 0]
//	         [-wide 55] [-tilt 90] [-earth] [-sky painted|black]
//	         [-gpu] [-at 0s]
//	         [-w 1280] [-h 800] [-n 2] [-out out/hillside.png]
//	         [-raw out/hillside.pfm] [-cpuprofile path]
//	hillside [the same flags] -prove out/hillside.pfm [-proof dir]
//	         [-pixel X,Y ...] [-many 64]
//	hillside -help
//
// -raw keeps the picture's light as well as its print. -prove draws
// nothing: with the flags that drew a raw, it makes 64 of its pixels
// again and proves each is the raw's bit for bit, with a sheet of their
// work in -proof, by default beside the raw (see slayer's fermat).
//
// -pixel names a pixel to prove first, and may be given again; -many is
// how many more the raw's bytes pick. the picks find a fault that runs
// through a picture, and a single pixel changed only where it is named.
//
// -h is the picture's height, as in slayer's other commands, so help
// is -help.
//
// there is a flag for every kind the manufactory has; a new kind there is a new
// flag here. the same flags and the same seed draw the same picture.
//
// things are flattened into meshes to be drawn, at about 1 KB a face
// (ninja-gaiden's measurement, 2026-10-06), so the command adds up the faces it
// keeps and refuses a hillside past -faces of them. a spruce is about thirteen
// thousand.
//
// -variety makes that many of each kind and stands each wherever it is
// picked, so a forest keeps a few trees' faces however many it draws.
// with none, every thing is its own.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/forest"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// asked is what the command was asked to draw, and how
type asked struct {
	seed                   uint64
	counts                 map[string]int
	apart, exposure        float64
	faces, variety         int
	from, wide, tilt       float64
	width, height, samples int
	night, onGPU, above    bool
	earth                  bool
	sky                    string
	at                     time.Duration
	out, profile           string
	// keep is where the raw goes; prove is a raw to prove, and proof
	// where its sheet goes
	keep, prove, proof string
	// pixels are the ones -pixel names, and many how many more to pick
	pixels named
	many   int
	// alight is how many of the spruces are on fire
	alight int
	// set is a set file that says the world and the film instead, and
	// eye where its first camera stands, once it is read; placed is
	// where the table of everything it stood goes
	set, placed string
	eye         *maths.Subspace
	// frame and glass are the set's rig's film and lens, where its film
	// has a rig, with open how long the shutter is open and glimpses
	// how many instants are drawn while it is; nil is the pinhole
	frame    *lens.Frame
	glass    *lens.Thin
	open     time.Duration
	glimpses int
	// floor is how high above the sea a set's lowest ground is, in metres:
	// a set's world counts heights up from it. zero for hillside's own
	// ground, which stands at the sea
	floor float64
	// overlay is a set's hud, drawn on its picture; nil is none
	overlay *overlay
}

// usual is how many of a kind a hillside has when nobody says: a
// stand of spruce, ferns under it, stones about, a log or two,
// somebody's can, and a fire. a kind not here is none.
var usual = map[string]int{"spruce": 5, "fern": 24, "boulder": 8,
	"cobble": 20, "log": 3, "can": 1, "campfire": 1, "spotlight": 1}

// the picture's own numbers. every one is made up to look right, by
// looking, unless its line says where it is from. none is the
// manufactory's.
const (
	// brightness is the exposure: how much maths.Light a watt a
	// steradian is. it is the picture's choice, not the lights', and
	// this one was chosen by looking: at it a campfire lights the
	// ground a few metres round it by night and is lost by day.
	brightness = 2000.0
	// most faces a hillside may have: about a gigabyte and a half, at
	// the 1 KB a face ninja-gaiden measured
	mostFaces = 1_500_000
	// how much light the sun gives the ground by day, and the moon by
	// night, and how much of either comes back from the other side
	day, moonlit, fill = 1.5, 0.03, 0.25
	// how far off the sun is, in metres: far enough that its light
	// falls alike across the hillside
	sunAway = 4000.0
	// earthRadius is the earth's mean radius in metres, the iugg's
	// 6371 km; under is how far under the ground the globe's top lies
	earthRadius, under = 6371000.0, 0.05
	// how far a thing may stand off its place on the grid, as a share
	// of the gap: enough to hide the grid, not enough to stand a
	// boulder on the fire
	strays = 0.5
	// the usual picture: so wide and so tall in pixels, so many rays
	// along each side of a pixel, things so many metres apart
	usualWide, usualTall, usualRays, usualApart = 1280, 800, 2, 2.5
	// the ground is flat, so a few cells a side do; it is at least so
	// many metres across and so many times the hillside, so that its
	// edge is past where the eye can tell it from the horizon
	groundCells, leastGround, groundTimes = 8, 160.0, 6.0
	// the eye stands off the hillside's front corner: so much of the
	// hillside's width to the right, up and back, and so many metres
	// more up and back, looking at a point so high over the middle,
	// through a lens so many degrees wide
	eyeRight, eyeUp, eyeBack = 0.35, 0.12, 0.8
	eyeHigher, eyeFurther    = 2.2, 4.0
	looksAt, lensWide        = 1.4, 55.0
	// from above: higher over the hillside than it is wide, and further
	// back, chosen so every corner falls in the usual picture (its test
	// holds that)
	aboveUp, aboveBack = 1.6, 1.4
)

// where the sun stands and where its light comes back from, and the
// colours of the sky by day and by night, of the ground past the
// hillside, and of soil
var (
	sunWay     = maths.V(-0.5, 0.8, 0.45)
	fillWay    = maths.V(0.6, 0.5, -0.3)
	dayHorizon = maths.RGB(0.7, 0.8, 0.95)
	dayZenith  = maths.RGB(0.25, 0.45, 0.9)
	farGround  = maths.RGB(0.34, 0.29, 0.18)
	duskLow    = maths.RGB(0.02, 0.03, 0.06)
	duskHigh   = maths.RGB(0.005, 0.008, 0.02)
	soil       = maths.RGB(0.2, 0.17, 0.1)
	// how the light becomes a picture: slayer's swarm command's own
	// film, a knee and a top
	developing = colour.Tone{Knee: 0.7, Top: 8}
)

var errAsked = errors.New("asked for a hillside that cannot be drawn")

// parse reads the flags: one for every kind there is. it refuses a count less
// than zero, a picture with no width, height or rays, things no distance apart,
// and no exposure, each by name. what it prints goes to say.
func parse(arguments []string, say io.Writer) (asked, error) {
	given := asked{counts: map[string]int{}}
	flags := flag.NewFlagSet("hillside", flag.ContinueOnError)
	flags.SetOutput(say)
	counts := map[string]*int{}
	for _, name := range manufacture.KindNames() {
		counts[name] = flags.Int(name, usual[name], "how many of "+name)
	}
	flags.Uint64Var(&given.seed, "seed", 1, "the seed of the hillside")
	flags.Float64Var(&given.apart, "apart", usualApart,
		"how far apart things stand, in metres")
	flags.Float64Var(&given.exposure, "exposure", brightness,
		"how bright a watt of a lamp is in the picture")
	flags.IntVar(&given.faces, "faces", mostFaces,
		"the most faces the hillside may keep")
	flags.IntVar(&given.variety, "variety", 0,
		"how many different ones of each kind; nought, each its own")
	flags.IntVar(&given.width, "w", usualWide, "width in pixels")
	flags.IntVar(&given.height, "h", usualTall, "height in pixels")
	flags.IntVar(&given.samples, "n", usualRays,
		"rays per pixel along each side")
	flags.BoolVar(&given.night, "night", false,
		"by moonlight, so the lamps show")
	flags.BoolVar(&given.onGPU, "gpu", false, "draw it on the gpu")
	flags.BoolVar(&given.above, "above", false,
		"look down from high over it, so every place shows")
	flags.Float64Var(&given.from, "from", 0,
		"look straight down from this many metres over its middle")
	flags.Float64Var(&given.wide, "wide", lensWide,
		"the lens's width, in degrees")
	flags.Float64Var(&given.tilt, "tilt", 90,
		"how far below level the eye at -from looks, in degrees")
	flags.BoolVar(&given.earth, "earth", false,
		"stand it on the earth, the sun at an endless distance")
	flags.StringVar(&given.sky, "sky", "painted",
		"the sky: painted, or black as above the air")
	flags.DurationVar(&given.at, "at", 0, "the time to draw it at")
	flags.StringVar(&given.out, "out", "out/hillside.png",
		"where the picture goes")
	flags.StringVar(&given.profile, "cpuprofile", "",
		"where a profile of the drawing goes, for go tool pprof")
	flags.StringVar(&given.keep, "raw", "",
		"where the picture's raw goes, its light as a pfm")
	flags.StringVar(&given.placed, "places", "",
		"where a set's table of every thing it stood goes: what it is of, "+
			"its kind, east, north and height")
	flags.StringVar(&given.prove, "prove", "",
		"a raw these flags drew, to prove instead of drawing")
	flags.StringVar(&given.proof, "proof", "",
		"where -prove's sheet goes; beside the raw by default")
	flags.Var(&given.pixels, "pixel",
		"a pixel X,Y for -prove to prove first; again for more")
	flags.IntVar(&given.many, "many", fermat.Many,
		"how many more pixels -prove picks by the raw's bytes")
	flags.IntVar(&given.alight, "alight", 0,
		"how many of the spruces are on fire")
	flags.StringVar(&given.set, "set", "",
		"a set file that says the world and the film instead")
	if err := flags.Parse(arguments); err != nil {
		return asked{}, err
	}
	if clash := clashes(flags, given); clash != "" {
		return asked{}, fmt.Errorf("%w: -set says the world and the "+
			"film, and -%s says them too", errAsked, clash)
	}
	for name, count := range counts {
		given.counts[name] = *count
	}
	return given, given.sound()
}

// sound is what is wrong with what was asked, by name, or nothing.
// the counts are added up here, before anything is made: each thing
// is a face at the very least, so more things than faces allowed can
// never be drawn, however long a list of them would take to write.
func (given asked) sound() error {
	things := 0
	for _, name := range manufacture.KindNames() {
		count := given.counts[name]
		if count < 0 || count > given.faces {
			return fmt.Errorf("%w: %d of %s, with -faces of %d",
				errAsked, count, name, given.faces)
		}
		things += count
	}
	checks := []struct {
		flag string
		good bool
	}{
		{"-faces", given.faces >= 1 && things <= given.faces},
		{"-w", given.width >= 1},
		{"-h", given.height >= 1},
		{"-n", given.samples >= 1},
		{"-variety", given.variety >= 0},
		{"-from", given.from >= 0 && !math.IsInf(given.from, 0)},
		{"-wide", given.wide > 0 && given.wide < 180},
		{"-tilt", given.tilt >= -90 && given.tilt <= 90},
		{"-sky", given.sky == "painted" || given.sky == "black"},
		{"-apart", given.apart > 0 && !math.IsInf(given.apart, 0)},
		{"-exposure", given.exposure > 0 &&
			!math.IsInf(given.exposure, 0)},
		// fermat proves the cpu's renders, which are exact
		{"-prove", given.prove == "" || !given.onGPU},
		{"-proof", given.proof == "" || given.prove != ""},
		{"-pixel", given.pixels.inside(given.width, given.height) &&
			(given.pixels == nil || given.prove != "")},
		// a flame's body is the cpu's until the gpu's kernel has one
		{"-alight", given.alight >= 0 &&
			given.alight <= given.counts["spruce"] &&
			(given.alight == 0 || !given.onGPU)},
		{"-many", given.many >= 0 &&
			(given.many == fermat.Many || given.prove != "") &&
			(given.many > 0 || given.pixels != nil)},
	}
	for _, check := range checks {
		if !check.good {
			return fmt.Errorf("%w: see %s", errAsked, check.flag)
		}
	}
	return nil
}

// profiled is work, with a cpu profile of it written to a path when
// there is one.
func profiled(path string, work func() error) error {
	if path == "" {
		return work()
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(out); err != nil {
		out.Close()
		return err
	}
	err = work()
	pprof.StopCPUProfile()
	return errors.Join(err, out.Close())
}

// main is run, and the number it says
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses, makes the hillside, draws it and saves it, or proves a raw it
// drew, and says what it did and what went wrong.
//
// it gives back zero when a picture was written, a proof held or help was asked
// for, two for flags it could not take, and one for a hillside it could not
// make or a proof that failed.
func run(arguments []string, said, wrong io.Writer) int {
	given, err := parse(arguments, wrong)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 2
	}
	var world *thing.World
	var made tally
	if given.set != "" {
		world, made, given, err = staged(given)
	} else {
		world, made, err = hillside(given)
	}
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 1
	}
	fmt.Fprintln(said, made)
	if given.prove != "" {
		return proving(world, given, made.across, arguments, said, wrong)
	}
	start := time.Now()
	var drawn *frame.Frame
	err = profiled(given.profile, func() (err error) {
		drawn, err = draw(world, given, made.across)
		return err
	})
	if err == nil {
		err = save(drawn, given, world, made.across)
	}
	if err == nil && given.keep != "" {
		err = keep(drawn, given.keep)
	}
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 1
	}
	fmt.Fprintf(said, "%s: %dx%d, %d rays a pixel, %v\n", given.out,
		given.width, given.height, given.samples*given.samples,
		time.Since(start).Round(time.Millisecond))
	return 0
}

// tally is what a hillside came to
type tally struct {
	things, faces, kept, lamps, unseen, burning int
	across                                      float64
}

// String is the tally in a line
func (made tally) String() string {
	said := fmt.Sprintf("%d things of %d faces", made.things, made.faces)
	if made.kept != made.faces {
		said += fmt.Sprintf(" (%d kept)", made.kept)
	}
	said += fmt.Sprintf(", lit by %d lamps", made.lamps)
	if made.burning > 0 {
		said += fmt.Sprintf(", %d alight", made.burning)
	}
	if made.unseen > 0 {
		said += fmt.Sprintf("; %d more shine where no eye sees, and "+
			"light nothing in this picture", made.unseen)
	}
	return said
}

var errTooMany = errors.New("too many faces to flatten")

// order is the kinds asked for, each as many times as asked, shuffled
// by the seed so the spruces do not all stand in one corner. the
// shuffle is fisher and yates': from the last place down, each place
// swaps with one at or below it.
func order(given asked) []string {
	var names []string
	for _, name := range manufacture.KindNames() {
		for range given.counts[name] {
			names = append(names, name)
		}
	}
	for last := len(names) - 1; last > 0; last-- {
		pick := int(noise.Hash(given.seed, int64(last)) *
			float64(last+1))
		names[last], names[pick] = names[pick], names[last]
	}
	return names
}

// splitmix64's own numbers: the odd step it counts by, and the two it
// multiplies by as it stirs.
const (
	stepBy    = 0x9e3779b97f4a7c15
	stirOnce  = 0xbf58476d1ce4e5b9
	stirTwice = 0x94d049bb133111eb
)

// stir is a thing's own seed: the hillside's and the place it stands at,
// stirred together by splitmix64's finaliser, so no two things on a hillside
// share one and no two hillsides share theirs.
//
// (the first version added the two before multiplying, so seed two was seed one
// moved along a place. hippo's review found it.)
func stir(seed uint64, place int) uint64 {
	mixed := seed ^ (uint64(place+1) * stepBy)
	mixed ^= mixed >> 30
	mixed *= stirOnce
	mixed ^= mixed >> 27
	mixed *= stirTwice
	return mixed ^ mixed>>31
}

// hillside is the world: ground, sky, sun or moon, and a made thing
// at every place, with whatever lights the things carry. the places
// are asked of forest before a thing is made, so a hillside of more
// places than forest has is refused by forest, in forest's words.
func hillside(given asked) (*thing.World, tally, error) {
	things := 0
	for _, count := range given.counts {
		things += count
	}
	side := max(1, int(math.Ceil(math.Sqrt(float64(things)))))
	places, err := forest.Placements(given.seed, side, given.apart,
		strays)
	if err != nil {
		return nil, tally{}, fmt.Errorf("%d things: %w", things, err)
	}
	made := tally{across: float64(side) * given.apart}
	world, err := bare(given, made.across)
	if err != nil {
		return nil, tally{}, err
	}
	made.lamps = len(world.Lamps)
	middle := given.apart * float64(side-1) / 2
	have := stock{}
	for index, name := range order(given) {
		placed := places[index]
		corner := maths.V(middle, 0, middle)
		at := maths.Subspace{Origin: placed.At().Sub(corner),
			Orientation: maths.About(maths.YAxis, placed.Turn()),
			Scale:       placed.Size()}
		err := stand(world, &made, given, have, name, index, at)
		if err != nil {
			return nil, tally{}, fmt.Errorf("%s at place %d: %w",
				name, index, err)
		}
	}
	return world, made, nil
}

// skyFor is the sky as asked: by day, by night, or black.
func skyFor(given asked) thing.Sky {
	switch {
	case given.sky == "black":
		return nil
	case given.night:
		return sky.Gradient{Horizon: duskLow, Zenith: duskHigh}
	}
	return sky.Gradient{Horizon: dayHorizon, Zenith: dayZenith,
		Ground: farGround}
}

// bare is the hillside with nothing on it: ground, sky, and the sun
// or the moon with a little of its light come back from the far side
func bare(given asked, across float64) (*thing.World, error) {
	world := thing.NewWorld()
	level := day
	world.Sky = skyFor(given)
	if given.night {
		level = moonlit
	}
	if err := lamps(world, given, level); err != nil {
		return nil, err
	}
	if err := onEarth(world, given); err != nil {
		return nil, err
	}
	size := math.Max(leastGround, groundTimes*across)
	ground, err := forest.Ground(groundCells, size)
	if err != nil {
		return nil, err
	}
	world.Add(thing.Thing{Shape: ground,
		Material: &thing.Material{Name: "soil", Colour: soil},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(-size/2, 0, -size/2)})})
	return world, nil
}

// stand stands the thing a place picks in the world, with its lights,
// counting what it adds: made now, or the stock's. a thing that would
// take the hillside past the faces it may keep is refused before it is
// added.
func stand(world *thing.World, made *tally, given asked, have stock,
	name string, place int, at maths.Subspace) error {
	one, fresh, err := have.made(given, name, place)
	if err != nil {
		return err
	}
	if fresh && made.kept+one.faces > given.faces {
		keeping := made.kept + one.faces
		return fmt.Errorf("%w: it would keep %d faces, past the "+
			"-faces of %d", errTooMany, keeping, given.faces)
	}
	if fresh {
		made.kept += one.faces
	}
	made.faces += one.faces
	made.things++
	for _, each := range stood(one.things, at) {
		world.Add(each)
	}
	if name == "spruce" && made.burning < given.alight {
		made.burning++
		lit, err := alight(world, one, at, stir(given.seed, place), given)
		if err != nil {
			return err
		}
		made.lamps += lit
	}
	lit, err := one.lamps(at)
	if err != nil {
		return err
	}
	for _, lamp := range lit {
		if lamp.Strength == (maths.Light{}) {
			made.unseen++
			continue
		}
		world.Lamp(lamp)
		made.lamps++
	}
	return nil
}

// lamps are the sun and its fill: lamps 4 km off over a flat hillside,
// or on the earth distant lights landing as much light on the ground.
func lamps(world *thing.World, given asked, level float64) error {
	far := level * sunAway * sunAway
	for _, light := range []struct {
		way   maths.Vec
		share float64
	}{{sunWay, 1}, {fillWay, fill}} {
		if !given.earth {
			world.Lamp(emit.Lit{At: light.way.Unit().Scale(sunAway),
				Strength: maths.Grey(light.share * far)})
			continue
		}
		sun, err := emit.NewDistant(light.way,
			maths.Grey(light.share*level))
		if err != nil {
			return err
		}
		world.Lamp(sun)
	}
	return nil
}

// onEarth stands the hillside on a globe of the earth, its top a hair
// below the ground so the two never share a surface. the globe is the
// soil's colour, picked by looking as the soil was.
func onEarth(world *thing.World, given asked) error {
	if !given.earth {
		return nil
	}
	ball, err := globe.New(earthRadius)
	if err != nil {
		return err
	}
	centre := maths.V(0, -earthRadius-under, 0)
	world.Add(thing.Thing{Shape: ball,
		Material: &thing.Material{Name: "earth", Colour: soil},
		Motion:   thing.Still(maths.Subspace{Origin: centre})})
	return nil
}

// seen is a light with nothing less than zero in it. a deep red glow reduced to
// a screen's three colours comes out a little under zero in blue, and a lamp of
// less than no blue would take blue away from whatever else lit the same
// ground.
func seen(light maths.Light) maths.Light {
	return maths.RGB(math.Max(0, light.R), math.Max(0, light.G),
		math.Max(0, light.B))
}

// eyeAt is where the picture is seen from, for a hillside so far
// across: above head height off its front corner, further back the
// wider it is, looking at a point over its middle.
func eyeAt(across float64) maths.Subspace {
	from := maths.V(eyeRight*across, eyeUp*across+eyeHigher,
		eyeBack*across+eyeFurther)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, looksAt, 0))}
}

// aboveAt is where the picture is seen from to show every place of a
// hillside so far across: high over its front corner, further back the
// wider it is, looking down at its middle on the ground.
func aboveAt(across float64) maths.Subspace {
	from := maths.V(eyeRight*across, aboveUp*across+eyeHigher,
		aboveBack*across+eyeFurther)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// eyeFor is where the picture is seen from, as asked: a set file's
// camera, straight down from a height, from above, or from the usual
// corner.
func eyeFor(given asked, across float64) maths.Subspace {
	switch {
	case given.eye != nil:
		return *given.eye
	case given.from > 0:
		from := maths.V(0, given.from, 0)
		down := given.tilt * maths.Degree
		way := maths.V(0, -math.Sin(down), -math.Cos(down))
		return maths.Subspace{Origin: from,
			Orientation: maths.LookAt(from, from.Add(way))}
	case given.above:
		return aboveAt(across)
	}
	return eyeAt(across)
}

// lensFor is the lens the picture is seen through, as asked: a set's
// rig's, or a pinhole as wide as asked
func lensFor(given asked) film.Sensor {
	if given.glass != nil {
		return *given.glass
	}
	return film.Pinhole{Wide: given.wide * maths.Degree}
}

// draw is the picture, on the cpu or the gpu as asked, or exposed
// through a set's rig
func draw(world *thing.World, given asked,
	across float64) (*frame.Frame, error) {
	eye, glass := eyeFor(given, across), lensFor(given)
	if given.glass != nil {
		return exposed(world, given, eye)
	}
	if given.onGPU {
		return gpu.Sampled(world, eye, glass, given.at, given.width,
			given.height, given.samples)
	}
	return trace.Sampled(world, eye, glass, given.at, given.width,
		given.height, given.samples)
}

// save develops the picture, draws a set's hud on it, and writes it
// as a png, each pixel as one pixel
func save(drawn *frame.Frame, given asked, world *thing.World,
	across float64) error {
	if err := os.MkdirAll(filepath.Dir(given.out), 0o755); err != nil {
		return err
	}
	img := developing.Develop(drawn)
	if err := overdraw(img, given, world, across); err != nil {
		return err
	}
	const square = 1
	return file.PNG(given.out, img, square)
}
