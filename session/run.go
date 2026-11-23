// Package session stages and draws a hillside of made things, or a set file,
// or a sequence of frames. Run is the whole of it; the hillside command and
// the render queue call it.
package session

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
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

// asked is what the command was asked to draw, and how
type asked struct {
	// preparedSet freezes imported geometry under the job's source identity.
	preparedSet *set.Set
	// oceanSpectrum is the continuous spectrum model estimate captured while
	// staging an authored finite ocean, absent when the set has no ocean.
	oceanSpectrum *OceanSpectrumEstimate
	// oceanSunTag is the exact scene-level sun/fill insertion recorded for an
	// explicitly enabled mean-plane ocean path; nil preserves all old lights.
	oceanSunTag *oceanSunTag
	// composite supplies an explicit primary-path pass instead of a full draw.
	composite *gpu.Frames
	live      *liveScene
	// sections is how the film draws in sections, and kept what a
	// sequence keeps of its last frame to draw the next from
	sections sectionsAsked
	kept     *keeper
	// metered is whether the exposure is the sky meter's, a first guess
	// that the picture meter sets right once the scene is staged
	metered bool
	// cached is what staging took from the disk caches or worked out
	cached                 []cacheNote
	seed                   uint64
	counts                 map[string]int
	apart, exposure        float64
	variety                int
	from, wide, tilt       float64
	width, height, samples int
	night, onGPU, above    bool
	nativeShadows          bool
	nativeMeshes           bool
	cachedLighting         bool
	// filter is where in a pixel its rays start, as the set's film
	// says; the grid, each at its own square's middle, where no set says
	filter thing.Filter
	// fastLighting is the lamp-store shortcut: boxed mist and smoke lit
	// from the gpu's lamp store, an approximation; a flag says it
	fastLighting bool
	// skyLooks is the sky-looks shortcut, beside fastLighting: the
	// sky's sixteen looks shared among a pixel's samples
	skyLooks bool
	// restir is the restir shortcut: direct light from a crowd of lamps by
	// reservoir resampling (doc/restir.md), an approximation
	restir bool
	// switches is the flags and shortcuts the command line or the job
	// said, by name, each on or off; the set's come after them
	switches map[string]bool
	// flagFastlight and flagVibes are the command line's own two flags
	flagFastlight, flagVibes bool
	// exactSky is the command line turning sky-looks off by name
	exactSky bool
	earth    bool
	// departures is a line for each key of the set's fantasy block
	departures []string
	sky        string
	at         time.Duration
	// sparksFor is how long a recipe fire throws sparks from its
	// ignition, those already thrown flying on: improv/moonlake's burst.
	// Nought is for as long as it burns. sparksSphere throws their spread
	// as a sphere, not a box
	sparksFor    time.Duration
	sparksSphere bool
	// shellInches fires a display shell of that many inches from each
	// recipe fire in place of its sparks, shellAt after the film's start:
	// fire support's mortar. Zero fires none
	shellInches float64
	shellAt     time.Duration
	// fireballRadius lets a fireball of that radius go from each recipe
	// fire, fireballAt after the film's start: fire support's thermal.
	// Zero lets none go
	fireballRadius float64
	fireballAt     time.Duration
	out, profile   string
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
	// stood is the table of everything the staging stood, kept for
	// Places; stoodKept is whether a staging stood things on a ground,
	// so that stood is its table, empty or not.
	stood     []standing
	stoodKept bool
	// camera is the set's camera a job names, which stands in place of
	// its first; nothing is the first
	camera string
	// cameraPath is a job-authored moving eye. A fixed explicit initial
	// medium cannot follow it safely, so set staging refuses the combination.
	cameraPath bool
	// initialMedia is an explicit set-camera start stack; air is the zero value.
	initialMedia []trace.InitialMedium
	// wire is the job's film asking for the edges of what the first
	// rays meet in place of their light: trace.Wire, on the cpu only
	wire bool
	// keeping is what a session's frame drawn on the cpu stops on and
	// keeps its rows in; nil, outside a session, draws all at once
	keeping *trace.Keeping
	// steps is where a session's staging is and what stops it; nil,
	// outside a session, stages to the end and says nothing
	steps *steps
	// lake is improv/moonlake's still water, its elevation in metres,
	// moonScale how many times its true size the moon's disc is drawn,
	// moonGlow how many times its true light the disc gives, and giants
	// how many times taller the rim's named trees are grown, closeScale
	// the close spruces' the same.
	//
	// rig is how many times their size the named stage rigs stand;
	// lakeGlow makes the lake a mask of itself, and closeGlow the close
	// spruces a mask of themselves
	lake, moonScale, moonGlow, giants, closeScale, rig float64
	lakeGlow, closeGlow                                bool
	eye                                                *maths.Subspace
	// frame and glass are the set's rig's film and lens, where its film
	// has a rig, with open how long the shutter is open and glimpses
	// how many instants are drawn while it is; nil is the pinhole
	frame *lens.Frame
	glass *lens.Thin
	// lensed is what the rig's lens is focused on and made of, so a film
	// can focus it again from each frame's eye; nil is no rig
	lensed   *focusing
	open     time.Duration
	glimpses int
	// optics is a job's own lens and shutter, as a session draws them
	// (session_exposure.go); the zero optics is a pinhole and a shutter
	// that is closed
	optics optics
	// floor is how high above the sea a set's lowest ground is, in metres:
	// a set's world counts heights up from it. zero for hillside's own
	// ground, which stands at the sea
	floor float64
	// overlay is a set's hud, drawn on its picture; nil is none
	overlay *overlay
	// said is which of the picture's size and rays the command line gave
	said sizing
	// terrain is what was asked to be done with the set's datasets
	terrain terrainAsk
	// print is the print a set drawn from its world half says, which
	// develops its picture; nil is hillside's own tone
	print *colour.Tone
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
	usualWide, usualTall, usualRays, usualApart = 320, 200, 2, 2.5
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

// sizing is which of -w, -h and -n the command line gave. with a set
// they are settings that stand in place of what its film says.
type sizing struct {
	width, height, samples bool
	// exposure, wide and renderer are which of them a job said, which
	// stand over the set's; the command line says none of them with a
	// set
	exposure, wide, renderer bool
	// fantasy holds the names of the scene flags the command line gave,
	// which stand over the set's fantasy block
	fantasy map[string]bool
}

// note keeps a flag that was given, where it is one of the three.
func (sizes *sizing) note(each *flag.Flag) {
	switch each.Name {
	case "w":
		sizes.width = true
	case "h":
		sizes.height = true
	case "n":
		sizes.samples = true
	}
}

// sized is what was asked with the film's size and rays, each as the
// command line said it where it did, and the film's where not.
func (given asked) sized(width, height, rays int) asked {
	if !given.said.width {
		given.width = width
	}
	if !given.said.height {
		given.height = height
	}
	if !given.said.samples {
		given.samples = rays
	}
	return given
}

var errAsked = errors.New("asked for a hillside that cannot be drawn")

// flagKinds freezes the legacy count flags. New recipes use the YAML door,
// through Puzzlemouth, without extending this older flag parser.
func flagKinds() []string {
	return []string{"bonfire", "boulder", "campfire", "can", "cobble",
		"fern", "home", "log", "maser", "norway", "pebble", "spotlight",
		"spruce"}
}

// parse reads the legacy flags. it refuses a count less
// than zero, a picture with no width, height or rays, things no distance apart,
// and no exposure, each by name. what it prints goes to say.
func parse(arguments []string, say io.Writer) (asked, error) {
	given := usualAsked()
	flags := flag.NewFlagSet("hillside", flag.ContinueOnError)
	flags.SetOutput(say)
	counts := map[string]*int{}
	for _, name := range flagKinds() {
		counts[name] = flags.Int(name, given.counts[name], "how many of "+name)
	}
	flags.Uint64Var(&given.seed, "seed", given.seed, "the seed of the hillside")
	flags.Float64Var(&given.apart, "apart", given.apart,
		"how far apart things stand, in metres")
	flags.Float64Var(&given.exposure, "exposure", given.exposure,
		"how bright a watt of a lamp is in the picture")
	flags.IntVar(&given.variety, "variety", given.variety,
		"how many different ones of each kind; nought, each its own")
	flags.IntVar(&given.width, "w", given.width, "width in pixels")
	flags.IntVar(&given.height, "h", given.height, "height in pixels")
	flags.IntVar(&given.samples, "n", given.samples,
		"rays per pixel along each side")
	flags.BoolVar(&given.night, "night", false,
		"by moonlight, so the lamps show")
	flags.BoolVar(&given.onGPU, "gpu", false, "draw it on the gpu")
	flags.BoolVar(&given.above, "above", false,
		"look down from high over it, so every place shows")
	flags.Float64Var(&given.from, "from", 0,
		"look straight down from this many metres over its middle")
	flags.Float64Var(&given.wide, "wide", given.wide,
		"the lens's width, in degrees")
	flags.Float64Var(&given.tilt, "tilt", given.tilt,
		"how far below level the eye at -from looks, in degrees")
	flags.BoolVar(&given.earth, "earth", false,
		"stand it on the earth, the sun at an endless distance")
	flags.StringVar(&given.sky, "sky", given.sky,
		"the sky: painted, or black as above the air")
	flags.DurationVar(&given.at, "at", 0, "the time to draw it at")
	flags.DurationVar(&given.sparksFor, "sparks-for", 0,
		"improv: recipe fires throw sparks only this long, those thrown flying on")
	flags.BoolVar(&given.sparksSphere, "sparks-sphere", false,
		"improv: recipe fires throw their sparks' spread as a sphere, not a box")
	flags.Float64Var(&given.shellInches, "shell", 0,
		"improv: each recipe fire fires a display shell this many inches "+
			"across, 2 to 12, in place of its sparks")
	flags.DurationVar(&given.shellAt, "shell-at", 0,
		"improv: when the shells are fired, after the film's start")
	flags.Float64Var(&given.fireballRadius, "fireball", 0,
		"improv: each recipe fire lets go a fireball this many metres in "+
			"radius, which rises into a mushroom")
	flags.DurationVar(&given.fireballAt, "fireball-at", 0,
		"improv: when the fireballs are let go, after the film's start")
	flags.BoolVar(&given.flagFastlight, "fastlight", false,
		"fast light, a flag: mist and smoke lit from a store of each "+
			"lamp's light, an approximation; gpu only")
	flags.BoolVar(&given.exactSky, "exact-sky", false,
		"beside -fastlight: every ray takes all the sky's looks, as "+
			"the exact tracer does")
	flags.BoolVar(&given.flagVibes, "vibes", false,
		"a flag for pictures to watch, not to measure: every flag on")
	flags.StringVar(&given.out, "out", given.out,
		"where the picture goes")
	flags.StringVar(&given.profile, "cpuprofile", "",
		"where a profile of the drawing goes, for go tool pprof")
	flags.StringVar(&given.keep, "raw", "",
		"where the picture's raw goes, its light as a pfm")
	flags.Float64Var(&given.lake, "lake", 0,
		"improv: still water up to this elevation, metres")
	flags.Float64Var(&given.moonScale, "moon", given.moonScale,
		"improv: the moon's disc this many times its true size")
	flags.Float64Var(&given.moonGlow, "moonglow", given.moonGlow,
		"improv: the moon's disc gives this many times its true light")
	flags.BoolVar(&given.lakeGlow, "lakeglow", false,
		"improv: the lake glows and nothing else, a mask of the water")
	flags.BoolVar(&given.closeGlow, "closeglow", false,
		"improv: the close spruces glow, a mask of them")
	flags.Float64Var(&given.closeScale, "close", given.closeScale,
		"improv: the close spruces grown this many times taller")
	flags.Float64Var(&given.giants, "giants", given.giants,
		"improv: the rim's named trees grown this many times taller")
	flags.Float64Var(&given.rig, "rig", given.rig,
		"improv: the stage rigs (things named rig ...) this many times their size, unlit")
	flags.StringVar(&given.placed, "places", "",
		"where a set's table of every thing it stood goes: what it is of, "+
			"its kind, east, north and height")
	flags.StringVar(&given.prove, "prove", "",
		"a raw these flags drew, to prove instead of drawing")
	flags.StringVar(&given.proof, "proof", "",
		"where -prove's sheet goes; beside the raw by default")
	flags.Var(&given.pixels, "pixel",
		"a pixel X,Y for -prove to prove first; again for more")
	flags.IntVar(&given.many, "many", given.many,
		"how many more pixels -prove picks by the raw's bytes")
	flags.IntVar(&given.alight, "alight", 0,
		"how many of the spruces are on fire")
	flags.StringVar(&given.set, "set", "",
		"a set file that says the world and the film instead")
	terrainFlags(flags, &given.terrain)
	err := flags.Parse(arguments)
	if err == nil {
		err = strayError(flags.Args())
	}
	if err != nil {
		return asked{}, err
	}
	flags.Visit(given.said.noteFantasy)
	given = given.flagged()
	if given.set != "" {
		flags.Visit(given.said.note)
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
// the counts are checked here, before anything is made: a count below
// nought cannot be drawn.
func (given asked) sound() error {
	if err := given.terrain.sound(given.set); err != nil {
		return err
	}
	for _, name := range manufacture.KindNames() {
		if count := given.counts[name]; count < 0 {
			return fmt.Errorf("%w: %d of %s", errAsked, count, name)
		}
	}
	checks := []struct {
		flag string
		good bool
	}{
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
		{"-alight", given.alight >= 0 &&
			given.alight <= given.counts["spruce"]},
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

// Run is the hillside command: it takes the command's arguments, says what it
// did to said and what went wrong to wrong, and gives back the exit number. The
// stamp says which build is running.
func Run(arguments []string, said, wrong io.Writer, stamp Stamp) int {
	build, built, who, sessions = stamp.build, stamp.built, stamp.who, stamp.sessions
	return run(arguments, said, wrong)
}

// run parses, makes the hillside, draws it and saves it, or proves a raw it
// drew, and says what it did and what went wrong.
//
// it gives back zero when a picture was written, a proof held or help was asked
// for, two for flags it could not take, and one for a hillside it could not
// make or a proof that failed.
func run(arguments []string, said, wrong io.Writer) int {
	if len(arguments) > 0 && arguments[0] == "sequence" {
		return sequenceRun(arguments, said, wrong)
	}
	whole := time.Now()
	given, err := parse(arguments, wrong)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 2
	}
	if given.terrain.asked() {
		if code, done := tendTerrain(given, said, wrong); done {
			return code
		}
	}
	if err := cachingTrees(os.LookupEnv); err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 1
	}
	var world *thing.World
	var made tally
	if given.set != "" {
		world, made, given, err = staged(given)
	} else {
		world, made, err = hillside(given)
	}
	if given.live != nil {
		defer given.live.closeBeds()
	}
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 1
	}
	built := time.Since(whole)
	fmt.Fprintln(said, made)
	for _, line := range given.departures {
		fmt.Fprintln(said, line)
	}
	if given.oceanSpectrum != nil {
		fmt.Fprintln(said, given.oceanSpectrum.line())
	}
	if err := fastRefusal(given); err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 1
	}
	if given.prove != "" {
		return proving(world, given, made.across, arguments, said, wrong)
	}
	start := time.Now()
	var drawn *frame.Frame
	var measured gpu.Profile
	err = profiled(given.profile, func() (err error) {
		drawn, measured, err = drawMeasured(world, given, made.across)
		return err
	})
	rendered := time.Since(start)
	output := time.Now()
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
	fmt.Fprintf(said, "timing: build %v, render %v, output %v, total %v\n",
		built.Round(time.Millisecond), rendered.Round(time.Millisecond),
		time.Since(output).Round(time.Millisecond),
		time.Since(whole).Round(time.Millisecond))
	reportGPUProfile(said, measured)
	reportTrees(said)
	return 0
}

// tally is what a hillside came to
type tally struct {
	sparks                                      int
	things, faces, kept, lamps, unseen, burning int
	across                                      float64
	// ground is what the run says of ground laid on a sphere, or nothing.
	ground string
	// sky is what the run says of a sky worked from an air, or nothing.
	sky string
	// sea is what the run says of a sea with no edge over the ground, or
	// nothing.
	sea string
}

// String is the tally in a line
func (made tally) String() string {
	said := fmt.Sprintf("%d things of %d faces", made.things, made.faces)
	if made.kept != made.faces {
		said += fmt.Sprintf(" (%d kept)", made.kept)
	}
	said += fmt.Sprintf(", lit by %d lamps", made.lamps)
	if made.sparks > 0 {
		said += fmt.Sprintf(", %d sparks", made.sparks)
	}
	if made.burning > 0 {
		said += fmt.Sprintf(", %d alight", made.burning)
	}
	if made.unseen > 0 {
		said += fmt.Sprintf("; %d more shine where no eye sees, and "+
			"light nothing in this picture", made.unseen)
	}
	if made.ground != "" {
		said += ". " + made.ground
	}
	if made.sky != "" {
		said += ". " + made.sky
	}
	if made.sea != "" {
		said += ". " + made.sea
	}
	return said
}

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

// stir delegates native construction identity to the shared placement primitive.
func stir(seed uint64, place int) uint64 { return placement.Seed(seed, place) }

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
// counting what it adds: made now, or the stock's.
func stand(world *thing.World, made *tally, given asked, have stock,
	name string, place int, at maths.Subspace) error {
	one, fresh, err := have.made(given, name, place)
	if err != nil {
		return err
	}
	if fresh {
		made.kept += one.stored
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

// firstStray is the place of the first argument that is no flag, defined as
// the first place of a list.
const firstStray = 0

// strayError is the refusal of arguments left over once the flags are read,
// or nil when there are none.
func strayError(stray []string) error {
	if len(stray) == nothing {
		return nil
	}
	return fmt.Errorf("%w: %q is not a flag, and it and all "+
		"after it would be ignored; a set file is given as -set %s",
		errAsked, stray[firstStray], stray[firstStray])
}

// lensFor is the lens the picture is seen through, as asked: a set's
// rig's, a job's own opened lens, or a pinhole as wide as asked
func lensFor(given asked) film.Sensor {
	if given.glass != nil {
		return *given.glass
	}
	if given.optics.opened != nil {
		return *given.optics.opened
	}
	return film.Pinhole{Wide: given.wide * maths.Degree}
}

// drawMeasured retains backend timing where the single-frame GPU path
// supplies it. Rig exposures still report their complete wall time in run.
func drawMeasured(world *thing.World, given asked,
	across float64) (*frame.Frame, gpu.Profile, error) {
	given.placing(world)
	if err := refuseGPUInitialMediaAperture(given); err != nil {
		return nil, gpu.Profile{}, err
	}
	if err := wireRefusal(given); err != nil {
		return nil, gpu.Profile{}, err
	}
	eye, glass := eyeFor(given, across), lensFor(given)
	if given.composite != nil {
		if len(given.initialMedia) > 0 {
			return nil, gpu.Profile{}, errors.New(
				"initial media: composite passes do not support an explicit camera stack")
		}
		return given.composite.Draw(world, wholeGeneration, eye, glass,
			given.at, given.width, given.height, given.samples)
	}
	if err := fastRefusal(given); err != nil {
		return nil, gpu.Profile{}, err
	}
	if given.glass != nil {
		if given.cachedLighting {
			return nil, gpu.Profile{}, errors.New(
				"volume-lighting: cached requires a pinhole film")
		}
		return exposed(world, given, eye)
	}
	if given.onGPU && given.optics.closing != nil {
		return drawOpen(world, given, eye)
	}
	if given.onGPU && given.restir {
		if len(given.initialMedia) > 0 {
			return nil, gpu.Profile{}, errors.New(
				"restir does not draw from an explicit camera stack")
		}
		// restir is drawn before the lamp store and the sky looks, which
		// it does not use: it draws a crowd of lamps with no media
		return gpu.NewRestir().Draw(world, eye, glass, given.at,
			given.width, given.height, given.samples)
	}
	if given.onGPU {
		if len(given.initialMedia) > 0 {
			picture, profile, err := gpu.SampledWithInitialMedia(
				framesMode(given), world, eye, glass, given.at,
				given.width, given.height, given.samples,
				given.initialMedia)
			return picture, profile, err
		}
		if given.skyLooks {
			return gpu.SampledFaster(world, eye, glass, given.at,
				given.width, given.height, given.samples)
		}
		if given.fastLighting {
			return gpu.SampledFast(world, eye, glass, given.at,
				given.width, given.height, given.samples)
		}
		if given.cachedLighting {
			return gpu.SampledCached(world, eye, glass, given.at, given.width, given.height, given.samples)
		}
		if given.nativeMeshes {
			return gpu.SampledAccelerated(world, eye, glass, given.at,
				given.width, given.height, given.samples)
		}
		if given.nativeShadows {
			return gpu.SampledNative(world, eye, glass, given.at,
				given.width, given.height, given.samples)
		}
		return gpu.SampledMeasured(world, eye, glass, given.at,
			given.width, given.height, given.samples)
	}
	picture, err := sampled(world, given, eye, glass, given.at, given.width,
		given.height, given.samples)
	return picture, gpu.Profile{}, err
}

// sampled preserves the legacy air path and uses the explicit trace API only
// when a set camera authored a nonempty start stack.
//
// A wire picture is
// trace.Wire's, from the same rays; the media a camera starts in do not
// move the first thing its rays meet. A session's frame is drawn a row at
// a time, so it stops when its context is done and keeps its rows where
// its ask says; the picture is the same to the bit.
//
// A rig's draw is one of
// the instants its shutter exposes, so it stops and keeps no rows: a row's
// key holds no instant.
func sampled(world *thing.World, given asked, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, rays int,
) (*frame.Frame, error) {
	if given.wire {
		return trace.Wire(world, eye, lens, at, width, height, rays)
	}
	if given.keeping != nil && given.glass != nil {
		return trace.SampledKept(world, eye, lens, at, width, height, rays,
			given.initialMedia, given.keeping.Unkept())
	}
	if given.keeping != nil {
		return trace.SampledKept(world, eye, lens, at, width, height, rays,
			given.initialMedia, *given.keeping)
	}
	if len(given.initialMedia) == 0 {
		return trace.Sampled(world, eye, lens, at, width, height, rays)
	}
	return trace.SampledWithInitialMedia(world, eye, lens, at, width, height,
		rays, given.initialMedia)
}

// save develops the picture, draws a set's hud on it, and writes it
// as a png, each pixel as one pixel
func save(drawn *frame.Frame, given asked, world *thing.World,
	across float64) error {
	if err := os.MkdirAll(filepath.Dir(given.out), 0o755); err != nil {
		return err
	}
	img := given.tone().Develop(drawn)
	if err := overdraw(img, given, world, across); err != nil {
		return err
	}
	const square = 1
	return file.PNG(given.out, img, square)
}
