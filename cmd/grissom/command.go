package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// The defaults of the flags that have one, as grissom has always had them.
const (
	// defaultDepth is the mesh depth the sky's layers are laid at.
	defaultDepth = 13
	// defaultEarthDepth is the mesh depth the earth's hexes are at.
	defaultEarthDepth = 7
	// defaultWideDepth is the mesh depth the whole-sky maps are laid at.
	defaultWideDepth = 8
	// defaultFov is the sky view's width to start at, in degrees.
	defaultFov = 1.0
	// defaultDistance is how far out the earth view starts, in earth
	// radii. From here the globe is cut off top and bottom in a shot.
	defaultDistance = 3.2
	// defaultColourways is the directory of the hud's colourways.
	defaultColourways = "colourways"
	// defaultColourway is the hud colourway to start in.
	defaultColourway = "twilight"
	// allViews is the shot view that draws all three views.
	allViews = -1
	// surface is the earth's radius, which is where the camera would be
	// on the ground: a distance has to be more than this.
	surface = 1.0
	// filmDistance is how far out a film starts, in earth radii: far
	// enough that the whole globe is in the frame.
	filmDistance = 4.2
	// defaultFrames is a film's length: 372 frames, which at 24 frames a
	// second is fifteen and a half seconds.
	defaultFrames = 372
	// defaultTurns is how many whole turns of longitude a film goes round.
	defaultTurns = 2
	// defaultTilt is the latitude, in degrees, a film starts at north and
	// ends at south, so that both poles are seen.
	defaultTilt = 35
	// rightAngle is the latitude of a pole, in degrees: a tilt must be
	// less, or the camera would be over the pole and have no up.
	rightAngle = 90
	// fewestFrames is the least a film is: a first frame and a last.
	fewestFrames = 2
	// decimal is the base the whole-number flags are read in.
	decimal = 10
	// bitsOfFloat is the size the number flags are read at.
	bitsOfFloat = 64
)

// shotNames are the views' names as a shot's files say them, by view.
var shotNames = []string{"sky", "earth", "standing"}

// options is everything the command line says, read and checked.
type options struct {
	data, earth, sky, colourways, colourway string
	look, shot                              string
	depth, earthDepth, wideDepth            int
	view, ground, chroma, earthImage        int
	fov, distance                           float64
	earthOnly, noHud                        bool
	// weather is the netCDF file of the weather layer, the variable to
	// read from it, and the file's time step a still shows.
	weather, weatherVar string
	weatherTime         int
	// into is the directory a film's frames are written to, which is how
	// a run is known to be a film; frames, turns and tilt are its length,
	// whole turns of longitude and the latitude it swings between.
	into        string
	frames      int
	turns, tilt float64
}

// launcher is what a verb hands its options to: the viewer, or in a test
// something that only records them.
type launcher func(options options) error

// tool declares grissom's verbs and flags through puzzlemouth, and hands
// each run's options to launch.
func tool(launch launcher) *pzm.Tool {
	tool := pzm.New("grissom", "look at the sky and the earth on Metal",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	tool.Flags(sharedFlags()...)
	opens := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("window", verb.NoShort, "open the viewer's window",
		windowPage, "Example_window", opens, windowDo(launch))
	draws := verb.NewFacts(verb.WritesTo("pngs named from --shot"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("draw", verb.NoShort, "draw offscreen views to png files",
		drawPage, "Example_draw", draws, drawDo(launch),
		pzm.Value("shot", verb.NoShort,
			"output prefix; each view adds its name and .png"),
		pzm.Value("view", verb.NoShort,
			"view to draw: -1 all, 0 sky, 1 earth, 2 standing"))
	films := verb.NewFacts(verb.WritesTo("pngs named frame-NNNN in --into"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("film", verb.NoShort, "draw the earth turning to png frames",
		filmPage, "Example_film", films, filmDo(launch),
		pzm.Value("into", verb.NoShort, "the directory the frames go in"),
		pzm.Value("frames", verb.NoShort, "frames in the film; default 372"),
		pzm.Value("turns", verb.NoShort,
			"whole turns of longitude over the film; default 2"),
		pzm.Value("tilt", verb.NoShort,
			"the latitude to swing from and to, in degrees; default 35"))
	return tool
}

// filmPage is the film verb's page.
const filmPage = `# film

Draw the earth turning, offscreen, one png a frame into --into, as
frame-0001.png onward, all in one process. Over the film the place under
the camera goes round --turns whole turns of longitude, westward, as the
earth is seen to turn from outside, starting at --look's longitude, while
its latitude runs evenly from north --tilt to south --tilt, so both poles
and every longitude are seen. The camera is --distance out, 4.2 radii by default,
the whole globe. No hud and no dot are drawn.

With --weather the film runs once through the steps that hold data,
evenly. Between two steps a triangle's cover is a straight crossfade
between fields apart in time, and not a statement of where the cloud went.
--weather-time does not apply: a film runs through all of the steps.
Metal is needed. --dry-run says which files and writes none.
`

// filmDo is the film verb: the options, which files it would write, then
// the launcher unless it was asked only to say.
func filmDo(launch launcher) pzm.Do {
	return func(call pzm.Call) error {
		read, err := readOptions(call, filming)
		if err != nil {
			return err
		}
		if call.DryRun() {
			_, err = fmt.Fprintf(call.Out(), "would write %s through %s\n",
				filepath.Join(read.into, frameName(1)),
				filepath.Join(read.into, frameName(read.frames)))
			return err
		}
		return launch(read)
	}
}

// windowPage is the window verb's page.
const windowPage = `# window

Open the viewer on the sky, or on the earth with --earth-only, which
loads no sky. Drag to look, scroll to zoom, q quits. The keys are on the
hud, which --no-hud takes away.
`

// drawPage is the draw verb's page.
const drawPage = `# draw

Draw the views offscreen through the window's pipeline and write each as
a png: the --shot prefix, then -sky, -earth or -standing. --dry-run says
which files and writes none. Metal is needed. With --earth-only the
default view is the earth.
`

// sharedFlags are the flags both verbs take, each as it has always been
// named but with two dashes.
func sharedFlags() []pzm.Flag {
	return []pzm.Flag{
		pzm.Value("data", verb.NoShort, "the sgra data directory"),
		pzm.Value("earth", verb.NoShort,
			"the directory holding celestial-data's earth files"),
		pzm.Value("sky", verb.NoShort,
			"the directory holding celestial-data's whole-sky maps"),
		pzm.Value("depth", verb.NoShort,
			"mesh depth of the sky's layers; default 13"),
		pzm.Value("earth-depth", verb.NoShort,
			"mesh depth of the earth's hexes; default 7"),
		pzm.Value("wide-depth", verb.NoShort,
			"mesh depth of the whole-sky maps; default 8"),
		pzm.Value("fov", verb.NoShort,
			"the sky view's width to start at, in degrees; default 1"),
		pzm.Value("look", verb.NoShort,
			"lon,lat the earth view and the observer start over"),
		pzm.Value("distance", verb.NoShort,
			"the earth view's distance in radii; default 3.2"),
		pzm.Value("colourways", verb.NoShort,
			"the directory of the hud's colourways; default colourways"),
		pzm.Value("colourway", verb.NoShort,
			"the hud colourway to start in; default twilight"),
		pzm.Value("ground", verb.NoShort,
			"the background to start at, as b steps"),
		pzm.Value("chroma", verb.NoShort,
			"the colourfulness to start at, as c steps"),
		pzm.Value("earth-image", verb.NoShort,
			"the earth to start on, as m steps"),
		pzm.Value("weather", verb.NoShort,
			"a netCDF file whose variable is laid over the earth as cloud"),
		pzm.Value("weather-var", verb.NoShort,
			"the variable to read from --weather; default tcc"),
		pzm.Value("weather-time", verb.NoShort,
			"the file's time step to show; default the first with data"),
		pzm.Switch("earth-only", verb.NoShort,
			"load no sky and show the earth; --data and --sky unused"),
		pzm.Switch("no-hud", verb.NoShort,
			"draw neither the text nor the crosshair"),
	}
}

// windowDo is the window verb: the options, then the launcher.
func windowDo(launch launcher) pzm.Do {
	return func(call pzm.Call) error {
		read, err := readOptions(call, windowing)
		if err != nil {
			return err
		}
		return launch(read)
	}
}

// drawDo is the draw verb: the options, which files it would write, then
// the launcher unless it was asked only to say.
func drawDo(launch launcher) pzm.Do {
	return func(call pzm.Call) error {
		read, err := readOptions(call, drawing)
		if err != nil {
			return err
		}
		if call.DryRun() {
			return sayShots(call, read)
		}
		return launch(read)
	}
}

// sayShots says the files a draw would write.
func sayShots(call pzm.Call, read options) error {
	for _, each := range shotViews(read) {
		_, err := fmt.Fprintf(call.Out(), "would write %s-%s.png\n",
			read.shot, shotNames[each])
		if err != nil {
			return err
		}
	}
	return nil
}

// shotViews are the views a shot draws, in order.
func shotViews(read options) []int {
	if read.view == allViews {
		return []int{skyView, earthView, placeView}
	}
	return []int{read.view}
}

// verbKind is which verb a call is: only a draw has --shot and --view, and
// only a film has --into, --frames, --turns and --tilt.
type verbKind int

// The kinds of verb.
const (
	windowing verbKind = iota
	drawing
	filming
)

// readOptions reads and checks what a call was given. A film starts
// farther out than the others, and never has a hud.
func readOptions(call pzm.Call, kind verbKind) (options, error) {
	read := options{
		data: call.Value("data"), earth: call.Value("earth"),
		sky: call.Value("sky"), look: call.Value("look"),
		colourways: text(call, "colourways", defaultColourways),
		colourway:  text(call, "colourway", defaultColourway),
		earthOnly:  call.On("earth-only"), noHud: call.On("no-hud"),
		weather:    call.Value("weather"),
		weatherVar: text(call, "weather-var", defaultWeatherVariable),
	}
	far := defaultDistance
	if kind == filming {
		far = filmDistance
	}
	err := readNumbers(call, &read, far)
	if kind == drawing && err == nil {
		err = readShot(call, &read)
	}
	if kind == filming && err == nil {
		err = readFilm(call, &read)
	}
	if err == nil {
		err = check(read)
	}
	return read, err
}

// readNumbers reads the flags that are numbers, the distance starting at
// far when it was not given.
func readNumbers(call pzm.Call, read *options, far float64) error {
	return errors.Join(
		whole(call, "depth", defaultDepth, &read.depth),
		whole(call, "earth-depth", defaultEarthDepth, &read.earthDepth),
		whole(call, "wide-depth", defaultWideDepth, &read.wideDepth),
		whole(call, "ground", 0, &read.ground),
		whole(call, "chroma", 0, &read.chroma),
		whole(call, "earth-image", 0, &read.earthImage),
		whole(call, "weather-time", firstWithData, &read.weatherTime),
		number(call, "fov", defaultFov, &read.fov),
		number(call, "distance", far, &read.distance))
}

// readFilm reads the flags only a film has. A film draws no hud.
func readFilm(call pzm.Call, read *options) error {
	if !call.Given("into") || call.Value("into") == "" {
		return errors.New("film needs --into: the directory to write " +
			"the frames in")
	}
	read.into, read.noHud = call.Value("into"), true
	return errors.Join(
		whole(call, "frames", defaultFrames, &read.frames),
		number(call, "turns", defaultTurns, &read.turns),
		number(call, "tilt", defaultTilt, &read.tilt))
}

// checkFilm refuses a film that cannot be drawn, saying what was wrong.
func checkFilm(read options) error {
	switch {
	case read.into == "":
		return nil
	case read.frames < fewestFrames:
		return fmt.Errorf("--frames is %d; a film needs at least %d, a "+
			"first and a last", read.frames, fewestFrames)
	case read.turns <= 0:
		return fmt.Errorf("--turns is %g; it must be more than 0",
			read.turns)
	case read.tilt < 0 || read.tilt >= rightAngle:
		return fmt.Errorf("--tilt is %g; it must be from 0 up to less "+
			"than %d degrees, for at a pole the camera has no up",
			read.tilt, rightAngle)
	case read.weatherTime != firstWithData:
		return errors.New("--weather-time shows one step; a film runs " +
			"through all the steps that hold data")
	}
	return nil
}

// text is a flag's value, or its default when it was not given.
func text(call pzm.Call, name, none string) string {
	if call.Given(name) {
		return call.Value(name)
	}
	return none
}

// whole reads a whole-number flag into a place, or its default when the
// flag was not given.
func whole(call pzm.Call, name string, none int, into *int) error {
	*into = none
	if !call.Given(name) {
		return nil
	}
	got, err := strconv.ParseInt(call.Value(name), decimal, strconv.IntSize)
	if err != nil {
		return fmt.Errorf("--%s wants a whole number, got %q", name,
			call.Value(name))
	}
	*into = int(got)
	return nil
}

// number reads a number flag into a place, or its default when it was not
// given.
func number(call pzm.Call, name string, none float64, into *float64) error {
	*into = none
	if !call.Given(name) {
		return nil
	}
	got, err := strconv.ParseFloat(call.Value(name), bitsOfFloat)
	if err != nil {
		return fmt.Errorf("--%s wants a number, got %q", name,
			call.Value(name))
	}
	*into = got
	return nil
}

// readShot reads the flags only a draw has. The view is the sky unless
// it was given, and the earth under --earth-only.
func readShot(call pzm.Call, read *options) error {
	if !call.Given("shot") || call.Value("shot") == "" {
		return errors.New("draw needs --shot")
	}
	read.shot = call.Value("shot")
	none := skyView
	if read.earthOnly {
		none = earthView
	}
	return whole(call, "view", none, &read.view)
}

// checkWeather refuses a weather flag that has no file to apply to, or a
// time step that is not a step: the file's steps count from 0.
func checkWeather(read options) error {
	given := read.weatherTime != firstWithData
	switch {
	case read.weather == "" && given:
		return errors.New("--weather-time needs --weather: the netCDF " +
			"file to read the step from")
	case read.weather == "" && read.weatherVar != defaultWeatherVariable:
		return errors.New("--weather-var needs --weather: the netCDF " +
			"file to read the variable from")
	case read.weatherTime < firstWithData:
		return fmt.Errorf("--weather-time is %d; the file's time steps "+
			"count from 0", read.weatherTime)
	}
	return nil
}

// check refuses what the options cannot do, saying what is missing.
func check(read options) error {
	if err := errors.Join(checkWeather(read), checkFilm(read)); err != nil {
		return err
	}
	switch {
	case read.earthOnly && read.earth == "":
		return errors.New("--earth-only needs --earth: the directory " +
			"holding celestial-data's earth files")
	case read.distance <= surface:
		return fmt.Errorf("--distance is %g; it must be more than 1, "+
			"the earth's radius", read.distance)
	case read.depth < 0 || read.earthDepth < 0 || read.wideDepth < 0:
		return errors.New("a depth cannot be less than 0")
	case read.shot != "" && read.earthOnly && read.view != earthView &&
		read.view != placeView:
		return fmt.Errorf("--earth-only has no sky: --view is %d, and "+
			"it must be 1 or 2", read.view)
	case read.shot != "" && (read.view < allViews || read.view > placeView):
		return fmt.Errorf("--view is %d; it must be -1, 0, 1 or 2",
			read.view)
	}
	return nil
}
