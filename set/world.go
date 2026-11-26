package set

import (
	"math"
	"slices"
	"strings"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/scene/thing"
	yaml "gopkg.in/yaml.v3"
)

// Terrain is the ground a set stands on, as it was measured: a window of
// a published elevation model, named by its key in sources.yaml, read
// from a file on the reader's shelf, in the model's own metres.
//
// A window is placed one of two ways. By west and north, in the model's own
// metres, which is how every model but a map of a whole sphere is read. Or,
// for a map of a sphere, by lat and lon: a place on the sphere, in degrees,
// with the window laid round it in local metres, and optionally a detail,
// which asks for ground finer than the map's, made from a seed.
type Terrain struct {
	from, file         string
	west, north        float64
	across, down, step float64
	smooth             float64
	lat, lon           float64
	seed               float64
	// reflectance is the share of red, green and blue light the ground
	// gives back, linear, where the set says one.
	reflectance [3]float64
	// slope and longest are the relief a detail asks below the map, and
	// relieved whether it asks one.
	slope, longest float64
	relieved       bool
	// placed is whether the set placed the window by lat and lon, and
	// detailed whether it asked for detail. A lat or a seed of nought is a
	// place and a seed, so they are said apart from the numbers. reflected
	// is whether the set said the ground's reflectance.
	placed, detailed, reflected bool
	// sea is the elevation of still water that fills the low ground, in
	// metres on the model's own datum, and watered whether the set says
	// one: nought is the sea.
	sea     float64
	watered bool
	// near is a finer window cut from the same model about a place of its
	// own inside this one, which the coarser window does not draw where
	// the near window lies.
	near *Terrain
	// cover is the rule that colours the ground by height above the sea
	// and slope, where the set says one.
	cover *Cover
	// made is the ground the set asks made finer than a window's survey
	// where its camera resolves finer, where it asks any.
	made *Made
}

// Near is the finer window the terrain holds inside it, and whether the
// set says one. It is placed by lat and lon like the terrain it is in,
// and says its own across, down and step; the model, the smoothing and
// the sea are the terrain's. A set's places are metres in the terrain's
// own window, so the near window is a place in it, not another frame.
func (ground Terrain) Near() (Terrain, bool) {
	if ground.near == nil {
		return Terrain{}, false
	}
	return *ground.near, true
}

// Sea is the elevation of the still water the ground is flooded to, in
// metres on the model's own datum, and whether the set says there is any.
func (ground Terrain) Sea() (elevation float64, said bool) {
	return ground.sea, ground.watered
}

// Relief is the relief the detail asks of the ground made below the map:
// its least steepness, rise over run, the longest edge in metres it
// reaches, and whether the set asked one.
func (ground Terrain) Relief() (slope, longest float64, asked bool) {
	return ground.slope, ground.longest, ground.relieved
}

// Reflectance is the share of red, green and blue light the ground gives
// back, in linear light, and whether the set said it. A set that does not
// say has the ground the stage gives every set.
func (ground Terrain) Reflectance() (red, green, blue float64, said bool) {
	shares := ground.reflectance
	return shares[0], shares[1], shares[2], ground.reflected
}

// the bounds of the place a window may be asked for on a sphere
const (
	southPole   = -90  // defined: degrees south at the south pole
	northPole   = 90   // defined: degrees north at the north pole
	farWest     = -360 // defined: a longitude a whole turn west of the first
	farEast     = 360  // defined: a longitude a whole turn east of the first
	leastDetail = 0    // defined: the smallest seed, which is a whole number
	noLight     = 0    // defined: a ground that gives back none of a light
	allLight    = 1    // defined: a ground that gives back all of it
	flattest    = 0    // defined: a relief of no steepness
	steepest    = 10   // picked: past a cliff, so a typo is refused
	// shortestRelief and longestRelief bound the longest edge a relief
	// reaches, in metres: a centimetre, titan's finest, and the widest
	// window a set may lay on a sphere's flat patch.
	shortestRelief = 0.01 // defined: titan's inchworm
	longestRelief  = 1e5  // defined: a window's widest across
)

// Place is the latitude and longitude, in degrees, the window is laid
// round, and whether the set said one. A window that is placed has its own
// local coordinates: see Window.
func (ground Terrain) Place() (latitude, longitude float64, placed bool) {
	return ground.lat, ground.lon, ground.placed
}

// Detail is the seed of the ground made below the map's own detail, and
// whether the set asked for any. A set that did not ask has the map alone.
func (ground Terrain) Detail() (seed uint64, asked bool) {
	return uint64(ground.seed), ground.detailed
}

// Smooth is how wide, in metres, a gaussian blurs the model's heights, so its
// cells do not show as steps: zero, unsaid, is the model as measured.
func (ground Terrain) Smooth() float64 { return ground.smooth }

// From is the source's key in sources.yaml, which pins the model.
func (ground Terrain) From() string { return ground.from }

// File is where the model's heights are, under the shelf.
func (ground Terrain) File() string { return ground.file }

// Window is the window's first point, easting and northing in the
// model's metres, how many metres it spans east and south, and how far
// apart its points are.
//
// A window placed by lat and lon has no model's metres. Its metres are
// local: east runs from nought at its west edge, north from nought at its
// south edge, so its first point, the north-west corner, is at nought east
// and as far north as the window is deep, and no coordinate is negative.
func (ground Terrain) Window() (west, north, across, down, step float64) {
	if _, _, placed := ground.Place(); placed {
		return 0, ground.down, ground.across, ground.down, ground.step
	}
	return ground.west, ground.north, ground.across, ground.down,
		ground.step
}

// Film is how the set's pictures are taken: so many pixels wide and
// high, so many rays along each side of a pixel, and the exposure, what
// a watt a steradian is in the picture's light.
//
// and a filter, where in a pixel its rays start and so how its edges
// are drawn: the tent unless the film says another.
//
// or, in place of the height and the exposure, a rig: a camera with a
// frame, a lens and a shutter.
//
// and a print, how the raw becomes a picture: the knee where its
// shoulder starts, as a share of white, and its top, the brightest light
// it holds, in whites, which lands on white.
type Film struct {
	width, height, rays float64
	filter              string
	fps                 float64
	exposure            float64
	wide                float64
	rig                 *Rig
	print               *[2]float64 // knee, top
	sections            sectioning
}

// sectioning is what a film says of drawing in sections: the width a
// frame must be over, the side of a section, and whether it said keep,
// and what.
type sectioning struct {
	over, size float64
	said, keep bool
	drift      float64
}

// Drift is how much the film lets a kept section drift: the share of a
// volume's light from the lamps that did not change that the lamps that
// did may come to, and a section with the volume in sight still be kept
// from the last frame. Zero unless the film says, and then a section
// with a volume in sight is kept only when nothing changed.
func (made Film) Drift() float64 { return made.sections.drift }

const (
	// sectionsOver is the width a frame must pass to be drawn in sections,
	// for a film that says none. Picked: a frame of 320 draws quickly
	// whole.
	sectionsOver = 320
	// sectionsSize is the side of a section in pixels, for a film that
	// says none. Picked: 64 is small beside a 2K frame's bodies.
	sectionsSize = 64
	// notSaid is the over or size of a film that says none. It is defined
	// as zero: the reader takes no less than one pixel for either.
	notSaid = 0
	// leastSide is the fewest pixels a section or a frame may be across.
	// It is defined as one pixel.
	leastSide = 1
	// mostSide is the most pixels a frame may be across, as the film's
	// width and height take it. Picked with them.
	mostSide = 1e5
	// leastDrift is a drift of none. It is defined as zero.
	leastDrift = 0
	// mostDrift is the whole of the light of the lamps that did not
	// change. It is defined as one.
	mostDrift = 1
)

// spoken is whether a film said a number of its sections, where it said
// at least one pixel.
func spoken(number float64) bool { return number != notSaid }

// Sections is how the film's frames are drawn in pieces: a frame wider
// than over pixels is drawn in sections size pixels square, and where
// keep holds a section that cannot have changed since the last frame is
// kept from it. A film that says nothing of sections has them at 320 and
// 64, kept; keep false draws every section of every frame, to check the
// kept ones against.
func (made Film) Sections() (over, size int, keep bool) {
	over, size, keep = sectionsOver, sectionsSize, true
	if spoken(made.sections.over) {
		over = int(made.sections.over)
	}
	if spoken(made.sections.size) {
		size = int(made.sections.size)
	}
	if made.sections.said {
		keep = made.sections.keep
	}
	return over, size, keep
}

// Print is the print's knee and top, and whether the film says a print.
func (made Film) Print() (knee, top float64, has bool) {
	if made.print == nil {
		return 0, 0, false
	}
	return made.print[0], made.print[1], true
}

// Rig is how a camera takes the set's picture: the frame of film in
// millimetres, the lens in front of it, and the shutter, open so many
// seconds, with so many glimpses rendered while it is.
type Rig struct {
	width, height     float64
	focal, stop       float64
	focus             string
	shutter, glimpses float64
}

// Rig is the film's rig, and whether it says one.
func (made Film) Rig() (Rig, bool) {
	if made.rig == nil {
		return Rig{}, false
	}
	return *made.rig, true
}

// Frame is the frame's width and height, in millimetres.
func (made Rig) Frame() (width, height float64) {
	return made.width, made.height
}

// Lens is the lens's focal length in millimetres, its f-number, and
// the name of the thing it is focused on.
func (made Rig) Lens() (focal, stop float64, focus string) {
	return made.focal, made.stop, made.focus
}

// Shutter is how long the shutter is open, in seconds, and how many
// glimpses are rendered while it is.
func (made Rig) Shutter() (seconds float64, glimpses int) {
	return made.shutter, int(made.glimpses)
}

// Size is the picture's width and height in pixels and its rays.
func (made Film) Size() (width, height, rays int) {
	return int(made.width), int(made.height), int(made.rays)
}

// filters is what a film's filter may say, and the filter each word is:
// the tent two pixels wide, the trapezoid a pixel and a half wide and
// the box one pixel wide, which a pixel's rays are spread under; and
// the grid, each ray at the middle of its own square of the pixel.
var filters = map[string]thing.Filter{
	"tent":      thing.TentFilter,
	"trapezoid": thing.TrapezoidFilter,
	"box":       thing.BoxFilter,
	"grid":      thing.GridFilter,
}

// Filter is where in a pixel the film's rays start, and so how its
// edges are drawn: the tent where the film says none.
func (made Film) Filter() thing.Filter {
	if made.filter == "" {
		return thing.TentFilter
	}
	return filters[made.filter]
}

// Exposure is the film's exposure.
func (made Film) Exposure() float64 { return made.exposure }

// Wide is how wide the lens sees, in degrees, or zero where the film does not
// say and the drawer's own lens is used.
func (made Film) Wide() float64 { return made.wide }

// A Moon is the full moon in the set's sky: how far round from north
// toward east, and how high above the horizon, in degrees, and the
// source its face's mosaic is, by its key in sources.yaml.
type Moon struct {
	disc              bool
	azimuth, altitude float64
	face              string
}

// Where is the moon's azimuth and altitude, in degrees.
func (made Moon) Where() (azimuth, altitude float64) {
	return made.azimuth, made.altitude
}

// Disc says whether to draw the lunar surface as scene geometry.
func (made Moon) Disc() bool { return made.disc }

// Face is the key of its face's mosaic, or nothing for a plain face.
func (made Moon) Face() string { return made.face }

// A Sun is the sun in the set's sky: how far round from north toward
// east, and how high above the horizon, in degrees. A typed sun below
// the horizon lights only the air: twilight, on a moon or planet set. A
// worked sun at or below the horizon is down (see Down).
type Sun struct {
	azimuth, altitude float64
	tint              *[sunChannels]float64
	// worked is whether the set's place, date and clock time put the sun
	// where it is, and not the set's azimuth and altitude
	worked bool
}

// Where is the sun's azimuth and altitude, in degrees.
func (made Sun) Where() (azimuth, altitude float64) {
	return made.azimuth, made.altitude
}

// A Thing is one made thing on the set, by its name: its kind, from the
// manufactory, its seed, where it stands as an easting and a northing,
// whether it is alight, and what it stands in for, when its kind is not
// what the scene means and says so.
type Thing struct {
	name, kind, standsIn string
	recipe, complexity   string
	seed                 float64
	east, north          float64
	height               float64
	position             *WorldPosition
	heading              *float64
	scale                *float64
	partSpin             *PartSpin
	alight               bool
	space                spaceThing
	cluster              clusterAsk
	superblock           superblockAsk
}

// Name is the thing's name, which cameras use to find it.
func (made Thing) Name() string { return made.name }

// Kind is the manufactory's kind it is made as.
func (made Thing) Kind() string { return made.kind }

// Seed is the seed it is made from.
func (made Thing) Seed() uint64 { return uint64(made.seed) }

// At is where it stands, an easting and a northing.
func (made Thing) At() (east, north float64) { return made.east, made.north }

// Height is the optional lift above terrain, in metres. Zero stands on it.
func (made Thing) Height() float64 { return made.height }

// Scale is a dimensionless uniform size and whether it was explicitly given.
// An omitted scale means one.
func (made Thing) Scale() (float64, bool) {
	if made.scale == nil {
		return 1, false
	}
	return *made.scale, true
}

// Position is the thing's explicit absolute east/north/up coordinate in
// metres, and whether the set gave one. It is independent of terrain height.
func (made Thing) Position() (WorldPosition, bool) {
	if made.position == nil {
		return WorldPosition{}, false
	}
	return *made.position, true
}

// Alight is whether it burns.
func (made Thing) Alight() bool { return made.alight }

// StandsIn is what the thing stands in for, or nothing.
func (made Thing) StandsIn() string { return made.standsIn }

// A Stand is many of one kind scattered over the set's ground by a seed:
// how a hillside is covered.
type Stand struct {
	name, kind, standsIn string
	recipe, complexity   string
	habit                string
	excluded             string
	clearance            float64
	weather              string
	fade                 bool
	count, seed          float64
	alight               float64
	above, below         float64
	leastWater           float64
	steepest             float64
	within               *wedge
	area                 *area
	climb                *cities.Climb
	frontage             *frontage
}

// an area is a rectangle of the terrain, in metres east and north of its
// south west corner, which a grid fills: the west and south edges are in
// it and the east and north are not.
type area struct {
	west, south, east, north float64
}

// Area is the rectangle of terrain the stand's grid fills, in metres east
// and north of the terrain's south west corner, and whether the set gives
// one.
func (made Stand) Area() (west, south, east, north float64, has bool) {
	if made.area == nil {
		return 0, 0, 0, 0, false
	}
	at := made.area
	return at.west, at.south, at.east, at.north, true
}

// a wedge is the ground a stand grows in, seen from a thing: between two
// bearings in degrees, clockwise from north and round past it if the
// second is the smaller, and between two distances in metres.
type wedge struct {
	of        string
	from, to  float64
	near, far float64
}

// Band is the heights the stand grows between, in the terrain's metres:
// above one and below the other. unsaid, it grows at every height.
func (made Stand) Band() (above, below float64) {
	return made.above, made.below
}

// LeastWater is the least depth of water, in metres, over the ground where
// a stand of a kind that floats may stand. unsaid, it is three metres.
func (made Stand) LeastWater() float64 {
	if made.leastWater == noLeastWater {
		return defaultLeastWater
	}
	return made.leastWater
}

// noLeastWater is what a stand that says no least water holds, and
// defaultLeastWater what it then means: the deepest draught of the boats
// is about two metres, and the ground is sampled at its step.
const noLeastWater, defaultLeastWater = 0, 3.0

// Steepest is the most the ground may rise in a metre of run where the
// stand grows, along the ground's steepest way. unsaid, it grows on
// ground of any steepness.
func (made Stand) Steepest() float64 { return made.steepest }

// Within is the wedge the stand grows in: the thing it is seen from, the
// bearings it runs between, clockwise from north, its nearest and
// farthest in metres, and whether the set gives one. unsaid, the stand
// grows over all the ground.
func (made Stand) Within() (of string, from, to, near, far float64,
	has bool) {
	if made.within == nil {
		return "", 0, 0, 0, 0, false
	}
	at := made.within
	return at.of, at.from, at.to, at.near, at.far, true
}

// Name is the stand's name.
func (made Stand) Name() string { return made.name }

// Kind is the manufactory's kind its things are.
func (made Stand) Kind() string { return made.kind }

// Count is how many it has, and the seed that places them.
func (made Stand) Count() (count int, seed uint64) {
	return int(made.count), uint64(made.seed)
}

// Alight is how many of them burn.
func (made Stand) Alight() int { return int(made.alight) }

// Habit is the name of the habit that lays the stand out, or nothing for a
// stand laid out evenly over its ground as a stand always was.
func (made Stand) Habit() string { return made.habit }

// StandsIn is what the stand's kind stands in for, or nothing.
func (made Stand) StandsIn() string { return made.standsIn }

// the most a stand may hold, and the most seed a seed may be: a float64
// says every whole number to 2^53 exactly
const (
	mostStand = 100000
	mostSeed  = 1 << 53
)

// terrain reads the set's terrain.
func (reading reader) terrain(_, value *yaml.Node) error {
	ground := &Terrain{}
	wrote := map[string]bool{}
	err := reading.keys(value, map[string]step{
		"from": reading.text(&ground.from),
		"file": reading.text(&ground.file),
		"west": reading.noted(wrote,
			reading.number(&ground.west, 0, math.Inf(1))),
		"north": reading.noted(wrote,
			reading.number(&ground.north, 0, math.Inf(1))),
		"lat": reading.noted(wrote,
			reading.number(&ground.lat, southPole, northPole)),
		"lon": reading.noted(wrote,
			reading.number(&ground.lon, farWest, farEast)),
		"detail": reading.noted(wrote,
			reading.block(map[string]step{"seed": reading.whole(
				&ground.seed, leastDetail, mostSeed),
				"relief": reading.relief(ground)})),
		"across": reading.number(&ground.across, 1, 1e5),
		"down":   reading.number(&ground.down, 1, 1e5),
		"step":   reading.number(&ground.step, 0.01, 1e4),
		"smooth": reading.number(&ground.smooth, 0, 1e4),
		"reflectance": reading.noted(wrote,
			reading.three(&ground.reflectance, noLight, allLight)),
		"sea": reading.noted(wrote, reading.quantity(&ground.sea, "m",
			-deepestSea, highestSea)),
		"near":  reading.nearer(ground),
		"cover": reading.cover(ground),
		"made":  reading.made(ground),
	}, nil)
	reading.set.terrain = ground
	if err != nil {
		return err
	}
	if ground.cover != nil && wrote["reflectance"] {
		return reading.refuse(value, "a terrain says its reflectance or "+
			"its cover, not both: the first surface of a cover is the "+
			"ground's reflectance")
	}
	ground.placed, ground.detailed = wrote["lat"] && wrote["lon"],
		wrote["detail"]
	ground.watered = wrote["sea"]
	ground.reflected = wrote["reflectance"]
	if err := reading.coverLevel(value, ground); err != nil {
		return err
	}
	if err := reading.terrainPlace(value, wrote); err != nil {
		return err
	}
	if err := reading.terrainNear(value, ground); err != nil {
		return err
	}
	return reading.terrainMade(value, ground)
}

// nearer is a step that reads the finer window a terrain holds: the place
// of its middle by lat and lon, its size and its step, all of them said.
func (reading reader) nearer(ground *Terrain) step {
	return func(_, value *yaml.Node) error {
		finer := &Terrain{placed: true}
		err := reading.keys(value, map[string]step{
			"lat":    reading.number(&finer.lat, southPole, northPole),
			"lon":    reading.number(&finer.lon, farWest, farEast),
			"across": reading.number(&finer.across, 1, 1e5),
			"down":   reading.number(&finer.down, 1, 1e5),
			"step":   reading.number(&finer.step, 0.01, 1e4),
		}, nil)
		if err == nil {
			err = reading.all(value, "a near window", "lat", "lon",
				"across", "down", "step")
		}
		ground.near = finer
		return err
	}
}

// terrainNear refuses a near window in a terrain that is not placed by lat
// and lon, or whose step is not finer than the terrain's own.
func (reading reader) terrainNear(at *yaml.Node, ground *Terrain) error {
	switch {
	case ground.near == nil:
		return nil
	case !ground.placed:
		return reading.refuse(at, "a near window is laid about a place "+
			"by lat and lon in a terrain placed so, which this is not")
	case ground.near.step >= ground.step:
		return reading.refuse(at, "a near window is finer than the "+
			"terrain it is in: its step %g m is not under %g m",
			ground.near.step, ground.step)
	}
	return nil
}

// relief is a step that reads the relief a detail asks below the map: its
// least steepness, rise over run, and the longest edge, in metres, it
// reaches up to, both said.
func (reading reader) relief(ground *Terrain) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"slope": reading.number(&ground.slope, flattest, steepest),
			"longest": reading.number(&ground.longest, shortestRelief,
				longestRelief),
		}, nil)
		if err == nil {
			err = reading.all(value, "a relief", "slope", "longest")
		}
		ground.relieved = err == nil
		return err
	}
}

// noted is a step that notes its key was written, in wrote, then reads it
// as read does.
func (reading reader) noted(wrote map[string]bool, read step) step {
	return func(name, value *yaml.Node) error {
		wrote[name.Value] = true
		return read(name, value)
	}
}

// terrainPlace refuses a terrain that is placed both ways, or half of the
// way by lat and lon, or asked for detail where it has no sphere to lay
// it on. wrote is the keys the terrain wrote.
func (reading reader) terrainPlace(at *yaml.Node, wrote map[string]bool) error {
	metres := wrote["west"] || wrote["north"]
	degrees := wrote["lat"] || wrote["lon"]
	switch {
	case metres && degrees:
		return reading.refuse(at, "terrain is placed by west and north, or "+
			"by lat and lon, not both: it says %s and %s",
			saidOf(wrote, "west", "north"), saidOf(wrote, "lat", "lon"))
	case degrees && !(wrote["lat"] && wrote["lon"]):
		return reading.refuse(at, "lat and lon go together: terrain says "+
			"%s", saidOf(wrote, "lat", "lon"))
	case wrote["detail"] && !degrees:
		return reading.refuse(at, "detail is for terrain placed by lat and "+
			"lon, which says neither")
	}
	return nil
}

// saidOf is the names, of those given, that were written, with and between.
func saidOf(wrote map[string]bool, names ...string) string {
	var said []string
	for _, name := range names {
		if wrote[name] {
			said = append(said, name)
		}
	}
	return strings.Join(said, " and ")
}

// film reads how the set's pictures are taken. a film with a rig says
// all of it, the frame, the lens and the shutter, and no height or
// exposure, which the rig decides.
func (reading reader) film(_, value *yaml.Node) error {
	made := &reading.set.film
	rig := &Rig{}
	steps := reading.with(map[string]step{
		"print":    reading.printed(&made.print),
		"sections": reading.sectioned(&made.sections),
		"width":    reading.whole(&made.width, 1, 1e5),
		"height":   reading.whole(&made.height, 1, 1e5),
		"rays":     reading.whole(&made.rays, 1, 64),
		"filter": reading.word(&made.filter, "tent", "trapezoid", "box",
			"grid"),
		"exposure": reading.number(&made.exposure, 1e-12, 1e12),
		"wide":     reading.number(&made.wide, 0.01, 179),
		"frame": reading.block(map[string]step{
			"width":  reading.number(&rig.width, 1, 1000),
			"height": reading.number(&rig.height, 1, 1000)}),
		"lens": reading.block(map[string]step{
			"focal": reading.number(&rig.focal, 1, 1e5),
			"stop":  reading.number(&rig.stop, 0.5, 1000),
			"focus": reading.text(&rig.focus)}),
		"shutter":  reading.number(&rig.shutter, 1e-6, 1e5),
		"glimpses": reading.whole(&rig.glimpses, 1, 1000),
	}, reading.spaceFilm())
	err := reading.keys(value, steps, nil)
	if err != nil || *rig == (Rig{}) {
		return err
	}
	if rig.width == 0 || rig.height == 0 || rig.focal == 0 ||
		rig.stop == 0 || rig.focus == "" || rig.shutter == 0 ||
		rig.glimpses == 0 {
		return reading.refuse(value, "a rig says its frame's "+
			"width and height, its lens's focal, stop and focus, "+
			"its shutter and its glimpses, all of them")
	}
	if made.height != 0 || made.exposure != 0 || made.wide != 0 {
		return reading.refuse(value, "a rig decides the height, "+
			"the exposure and how wide the picture sees, so the "+
			"film does not say them")
	}
	made.rig = rig
	return nil
}

const (
	// firstKey is the place of a mapping's first key in its content. It
	// is defined as the first place, zero.
	firstKey = 0
	// valueAfter is how far after a key its value stands. It is defined
	// as one: a mapping's content alternates keys and values.
	valueAfter = 1
	// pairWidth is how many places a key and its value take. It is
	// defined as two.
	pairWidth = 2
)

// sectionsStep reads a film's sections into a sectioning.
type sectionsStep struct {
	reading reader
	into    *sectioning
}

// sectioned is a step that reads how a film draws in sections.
func (reading reader) sectioned(into *sectioning) step {
	return sectionsStep{reading: reading, into: into}.read
}

// saysKeep is whether a mapping has the key keep.
func saysKeep(value *yaml.Node) bool {
	for at := firstKey; at+valueAfter < len(value.Content); at += pairWidth {
		if value.Content[at].Value == "keep" {
			return true
		}
	}
	return false
}

// read reads the keys over, size, keep and drift of a film's sections.
func (from sectionsStep) read(_, value *yaml.Node) error {
	into, reading := from.into, from.reading
	said := false
	err := reading.keys(value, map[string]step{
		"over":  reading.whole(&into.over, leastSide, mostSide),
		"size":  reading.whole(&into.size, leastSide, mostSide),
		"keep":  reading.yes(&said),
		"drift": reading.number(&into.drift, leastDrift, mostDrift),
	}, nil)
	if saysKeep(value) {
		into.said, into.keep = true, said
	}
	return err
}

// printed is a step that reads a print: its knee, its top above the
// knee, and where they came from.
func (reading reader) printed(into **[2]float64) step {
	return func(_, value *yaml.Node) error {
		var made [2]float64
		var source string
		err := reading.keys(value, map[string]step{
			"knee":   reading.number(&made[0], 0, brightest),
			"top":    reading.number(&made[1], 0, brightest),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(value, "a print", "knee", "top", "source")
		}
		if err == nil && !(made[1] > made[0]) {
			err = reading.refuse(value, "a top of %v at or under a knee "+
				"of %v: the curve would end before it bends", made[1],
				made[0])
		}
		*into = &made
		return err
	}
}

// sun reads where the sun stands in the set's sky.
func (reading reader) sun(_, value *yaml.Node) error {
	made := &Sun{}
	input := sunTintReader{reading: reading, sun: made}
	err := reading.keys(value, map[string]step{
		"tint":     input.read,
		"azimuth":  reading.number(&made.azimuth, 0, 360),
		"altitude": reading.number(&made.altitude, -90, 90),
	}, nil)
	if err == nil && says(value, "azimuth") != says(value, "altitude") {
		err = reading.refuse(value, "a sun says its azimuth and its "+
			"altitude, both or neither")
	}
	reading.set.sun = made
	return err
}

// moon reads where the full moon stands in the set's sky.
func (reading reader) moon(_, value *yaml.Node) error {
	made := &Moon{disc: true}
	err := reading.keys(value, map[string]step{
		"azimuth":  reading.number(&made.azimuth, 0, 360),
		"altitude": reading.number(&made.altitude, -90, 90),
		"face":     reading.text(&made.face),
		"disc":     reading.yes(&made.disc),
	}, nil)
	reading.set.moon = made
	return err
}

// mostSpaceCoordinate is the existing public thing schema cap, in metres.
const mostSpaceCoordinate = 1e9

// things reads each made thing, by its name.
func (reading reader) things(_, value *yaml.Node) error {
	least, most := float64(sceneScalarZero), math.Inf(scenePositiveInfinity)
	if reading.spaceCoordinates {
		least, most = -mostSpaceCoordinate, mostSpaceCoordinate
	}
	maximum := math.Sqrt(math.MaxFloat64)
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Thing{name: name.Value}
		steps := reading.with(map[string]step{
			"recipe":     reading.text(&made.recipe),
			"complexity": reading.placedComplexity(&made.complexity),
			"kind":       reading.kind,
			"seed": reading.whole(&made.seed,
				sceneScalarZero, mostSeed),
			"east":      reading.number(&made.east, least, most),
			"north":     reading.number(&made.north, least, most),
			"heading":   reading.heading(&made.heading),
			"part-spin": reading.partSpin(&made.partSpin),
			"height": reading.quantity(&made.height, "m",
				sceneScalarZero, maximum),
			"position":      reading.worldPosition(&made.position),
			"alight":        reading.yes(&made.alight),
			"stands_in_for": reading.text(&made.standsIn),
			"parts":         reading.clusterParts(&made.cluster.parts),
			"count": reading.whole(&made.cluster.count, 1,
				mostClusterParts),
			"band_spacing": reading.quantity(&made.cluster.bandSpacing,
				"m", leastBandSpacing, maximum),
			"parcel":  reading.parcel(&made.superblock),
			"ceiling": reading.whole(&made.superblock.ceiling, 1, mostCeiling),
			"habit":   reading.only(manufacture.SuperblockHabit),
			"accretion": reading.accretion(
				&made.superblock),
			"occupants": reading.whole(&made.superblock.occupants, 1,
				mostOccupants),
			"crowding": reading.number(&made.superblock.crowding,
				leastCrowding, mostCrowding),
		}, reading.spaceThing(&made))
		err := reading.keys(block, steps, nil)
		if err == nil && made.position != nil {
			for _, key := range []string{"east", "north", "height", "place", "orbit", "around"} {
				if says(block, key) {
					return reading.refuse(valueNode(block, key),
						"thing %s says position and %s; an absolute position stands alone",
						made.name, key)
				}
			}
		}
		made.kind = valueOf(block, "kind")
		made.superblock.habit = valueOf(block, "habit")
		made.superblock.drawn = made.superblock.drawn || !says(block,
			"accretion")
		if err == nil {
			err = reading.superblockKeys(block, made)
		}
		if err == nil && made.partSpin != nil && !reading.spaceCoordinates {
			err = reading.refuse(valueNode(block, "part-spin"), "part-spin is supported only with setting: space")
		}
		if err == nil && made.scale != nil {
			if !reading.second || !says(block, "east") || !says(block, "north") ||
				says(block, "place") || says(block, "orbit") || says(block, "around") || says(block, "attitude") || made.partSpin != nil {
				return reading.refuse(valueNode(block, "scale"), "scale requires an explicit east/north placement without part-spin or attitude")
			}
		}
		if err == nil && reading.second {
			err = reading.standing(block, made)
		}
		if err == nil {
			err = reading.owned(block, made)
		}
		reading.set.things = append(reading.set.things, made)
		return err
	})
}

// stands reads each stand, by its name.
func (reading reader) stands(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Stand{name: name.Value, above: math.Inf(-1),
			below: math.Inf(1), steepest: math.Inf(1)}
		err := reading.keys(block, map[string]step{
			"recipe":     reading.text(&made.recipe),
			"complexity": reading.placedComplexity(&made.complexity),
			"exclude": reading.block(map[string]step{
				"around":    reading.text(&made.excluded),
				"clearance": reading.quantity(&made.clearance, "m", 0, 1e5)}),
			"atmosphere": reading.block(map[string]step{
				"weather":          reading.text(&made.weather),
				"extent":           reading.only("canopy-and-ground"),
				"fade-at-boundary": reading.yes(&made.fade)}),
			"kind":          reading.only(manufacture.KindNames()...),
			"count":         reading.whole(&made.count, 1, mostStand),
			"seed":          reading.whole(&made.seed, sceneScalarZero, mostSeed),
			"alight":        reading.whole(&made.alight, 0, mostStand),
			"stands_in_for": reading.text(&made.standsIn),
			"above":         reading.number(&made.above, -1e4, 1e4),
			"below":         reading.number(&made.below, -1e4, 1e4),
			"least-water":   reading.quantity(&made.leastWater, "m", 0.01, 1e4),
			"steepest":      reading.number(&made.steepest, flattest, steepest),
			"within":        reading.wedge(&made.within),
			"area":          reading.rectangle(&made.area),
			"habit":         reading.only(cities.HabitNames()...),
			"climb":         reading.climb(&made.climb),
			"frontage":      reading.frontage(&made.frontage),
		}, nil)
		made.kind = valueOf(block, "kind")
		made.habit = valueOf(block, "habit")
		if err == nil {
			err = reading.standFits(block, made)
		}
		if err == nil {
			err = reading.standSlope(block, made)
		}
		reading.set.stands = append(reading.set.stands, made)
		return err
	})
}

// standFits refuses a stand whose parts cannot stand together: a habit with
// no ground to lay over, a climb or a frontage on a habit that takes none,
// more alight than it has, or heights that leave no ground.
func (reading reader) standFits(block *yaml.Node, made Stand) error {
	contoured := made.habit == cities.HabitContour
	fronted := made.habit == cities.HabitFrontage
	switch {
	case fronted != (made.frontage != nil) ||
		fronted && (made.within != nil || made.area != nil):
		return reading.refuse(block, "stand %s: the habit frontage takes a "+
			"frontage along a channel and no within region or area, and a "+
			"frontage is for no other habit", made.name)
	case made.habit != "" && !fronted && made.within == nil &&
		made.area == nil:
		return reading.refuse(block, "stand %s has the habit %s, which "+
			"lays out sites over a within region or an area, and says "+
			"neither", made.name, made.habit)
	case contoured != (made.climb != nil) ||
		contoured && made.within == nil:
		return reading.refuse(block, "stand %s: the habit contour takes "+
			"a climb and a within region, and a climb is for no other "+
			"habit", made.name)
	case made.alight > made.count:
		return reading.refuse(block, "stand %s has %v alight of %v",
			made.name, made.alight, made.count)
	case made.above >= made.below:
		return reading.refuse(block, "stand %s grows above %v and "+
			"below %v, which is nowhere", made.name, made.above,
			made.below)
	}
	return nil
}

// standSlope refuses a steepest on a stand laid by a habit: a habit lays
// its sites by its own rule and reads no slope, so the key would be lost.
func (reading reader) standSlope(block *yaml.Node, made Stand) error {
	if made.habit != "" && !math.IsInf(made.steepest, 1) {
		return reading.refuse(block, "stand %s has the habit %s and says "+
			"steepest; only a stand scattered over terrain reads the "+
			"ground's slope", made.name, made.habit)
	}
	return nil
}

// wedge reads the ground a stand grows in, every part said: the thing
// it is seen from, the bearings it runs between, and how near and far,
// the far beyond the near.
func (reading reader) wedge(into **wedge) step {
	return func(_, value *yaml.Node) error {
		got := wedge{from: math.NaN(), to: math.NaN(), near: math.NaN(),
			far: math.NaN()}
		err := reading.keys(value, map[string]step{
			"of":   reading.text(&got.of),
			"from": reading.number(&got.from, 0, 360),
			"to":   reading.number(&got.to, 0, 360),
			"near": reading.number(&got.near, 0, 1e6),
			"far":  reading.number(&got.far, 0, 1e6),
		}, nil)
		switch {
		case err != nil:
			return err
		case got.of == "" || math.IsNaN(got.from) || math.IsNaN(got.to) ||
			math.IsNaN(got.near) || math.IsNaN(got.far):
			return reading.refuse(value, "within says of, from, to, near "+
				"and far")
		case !(got.far > got.near):
			return reading.refuse(value, "within reaches from %v to %v m, "+
				"which is nowhere", got.near, got.far)
		}
		*into = &got
		return nil
	}
}

// rectangle reads the area a stand's grid fills, every edge said, the east
// beyond the west and the north beyond the south.
func (reading reader) rectangle(into **area) step {
	return func(_, value *yaml.Node) error {
		got := area{west: math.NaN(), south: math.NaN(), east: math.NaN(),
			north: math.NaN()}
		err := reading.keys(value, map[string]step{
			"west":  reading.number(&got.west, -1e7, 1e7),
			"south": reading.number(&got.south, -1e7, 1e7),
			"east":  reading.number(&got.east, -1e7, 1e7),
			"north": reading.number(&got.north, -1e7, 1e7),
		}, nil)
		switch {
		case err != nil:
			return err
		case math.IsNaN(got.west) || math.IsNaN(got.south) ||
			math.IsNaN(got.east) || math.IsNaN(got.north):
			return reading.refuse(value, "area says west, south, east and "+
				"north")
		case !(got.east > got.west) || !(got.north > got.south):
			return reading.refuse(value, "area has its east beyond its west "+
				"and its north beyond its south, not %v to %v and %v to %v",
				got.west, got.east, got.south, got.north)
		}
		*into = &got
		return nil
	}
}

// valueOf is a key's word in a block, or nothing.
func valueOf(block *yaml.Node, key string) string {
	for i := 0; i+1 < len(block.Content); i += 2 {
		if block.Content[i].Value == key {
			return block.Content[i+1].Value
		}
	}
	return ""
}

// whole is a step that reads a whole number, from least to most.
func (reading reader) whole(target *float64, least, most float64) step {
	read := reading.number(target, least, most)
	return func(name, value *yaml.Node) error {
		if err := read(name, value); err != nil {
			return err
		}
		if *target != math.Trunc(*target) {
			return reading.refuse(value, "%s takes a whole number, not %q",
				name.Value, value.Value)
		}
		return nil
	}
}

// yes is a step that reads true or false.
func (reading reader) yes(target *bool) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.ScalarNode ||
			!slices.Contains([]string{"true", "false"}, value.Value) {
			return reading.refuse(value, "%s takes true or false, not %q",
				name.Value, value.Value)
		}
		*target = value.Value == "true"
		return nil
	}
}
