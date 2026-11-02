package set

import (
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture"
	yaml "go.yaml.in/yaml/v3"
)

// Terrain is the ground a set stands on, as it was measured: a window of
// a published elevation model, named by its key in sources.yaml, read
// from a file on the reader's shelf, in the model's own metres.
type Terrain struct {
	from, file         string
	west, north        float64
	across, down, step float64
	smooth             float64
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
func (ground Terrain) Window() (west, north, across, down, step float64) {
	return ground.west, ground.north, ground.across, ground.down,
		ground.step
}

// Film is how the set's pictures are taken: so many pixels wide and
// high, so many rays along each side of a pixel, and the exposure, what
// a watt a steradian is in the picture's light.
//
// or, in place of the height and the exposure, a rig: a camera with a
// frame, a lens and a shutter.
//
// and a print, how the raw becomes a picture: the knee where its
// shoulder starts, as a share of white, and its top, the brightest light
// it holds, in whites, which lands on white.
type Film struct {
	width, height, rays float64
	exposure            float64
	wide                float64
	rig                 *Rig
	print               *[2]float64 // knee, top
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

// Exposure is the film's exposure.
func (made Film) Exposure() float64 { return made.exposure }

// Wide is how wide the lens sees, in degrees, or zero where the film does not
// say and the drawer's own lens is used.
func (made Film) Wide() float64 { return made.wide }

// A Moon is the full moon in the set's sky: how far round from north
// toward east, and how high above the horizon, in degrees, and the
// source its face's mosaic is, by its key in sources.yaml.
type Moon struct {
	azimuth, altitude float64
	face              string
}

// Where is the moon's azimuth and altitude, in degrees.
func (made Moon) Where() (azimuth, altitude float64) {
	return made.azimuth, made.altitude
}

// Face is the key of its face's mosaic, or nothing for a plain face.
func (made Moon) Face() string { return made.face }

// A Sun is the sun in the set's sky: how far round from north toward
// east, and how high above the horizon, in degrees. below the horizon
// it lights only the air: twilight.
type Sun struct {
	azimuth, altitude float64
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
	seed                 float64
	east, north          float64
	alight               bool
}

// Name is the thing's name, which cameras use to find it.
func (made Thing) Name() string { return made.name }

// Kind is the manufactory's kind it is made as.
func (made Thing) Kind() string { return made.kind }

// Seed is the seed it is made from.
func (made Thing) Seed() uint64 { return uint64(made.seed) }

// At is where it stands, an easting and a northing.
func (made Thing) At() (east, north float64) { return made.east, made.north }

// Alight is whether it burns.
func (made Thing) Alight() bool { return made.alight }

// StandsIn is what the thing stands in for, or nothing.
func (made Thing) StandsIn() string { return made.standsIn }

// A Stand is many of one kind scattered over the set's ground by a seed:
// how a hillside is covered.
type Stand struct {
	name, kind, standsIn string
	count, seed          float64
	alight               float64
	above, below         float64
	within               *wedge
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
	err := reading.keys(value, map[string]step{
		"from":   reading.text(&ground.from),
		"file":   reading.text(&ground.file),
		"west":   reading.number(&ground.west, 0, math.Inf(1)),
		"north":  reading.number(&ground.north, 0, math.Inf(1)),
		"across": reading.number(&ground.across, 1, 1e5),
		"down":   reading.number(&ground.down, 1, 1e5),
		"step":   reading.number(&ground.step, 0.01, 1e4),
		"smooth": reading.number(&ground.smooth, 0, 1e4),
	}, nil)
	reading.set.terrain = ground
	return err
}

// film reads how the set's pictures are taken. a film with a rig says
// all of it, the frame, the lens and the shutter, and no height or
// exposure, which the rig decides.
func (reading reader) film(_, value *yaml.Node) error {
	made := &reading.set.film
	rig := &Rig{}
	err := reading.keys(value, map[string]step{
		"print":    reading.printed(&made.print),
		"width":    reading.whole(&made.width, 1, 1e5),
		"height":   reading.whole(&made.height, 1, 1e5),
		"rays":     reading.whole(&made.rays, 1, 64),
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
	}, nil)
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
	err := reading.keys(value, map[string]step{
		"azimuth":  reading.number(&made.azimuth, 0, 360),
		"altitude": reading.number(&made.altitude, -90, 90),
	}, nil)
	reading.set.sun = made
	return err
}

// moon reads where the full moon stands in the set's sky.
func (reading reader) moon(_, value *yaml.Node) error {
	made := &Moon{}
	err := reading.keys(value, map[string]step{
		"azimuth":  reading.number(&made.azimuth, 0, 360),
		"altitude": reading.number(&made.altitude, -90, 90),
		"face":     reading.text(&made.face),
	}, nil)
	reading.set.moon = made
	return err
}

// things reads each made thing, by its name.
func (reading reader) things(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Thing{name: name.Value}
		err := reading.keys(block, map[string]step{
			"kind":          reading.only(manufacture.KindNames()...),
			"seed":          reading.whole(&made.seed, 0, mostSeed),
			"east":          reading.number(&made.east, 0, math.Inf(1)),
			"north":         reading.number(&made.north, 0, math.Inf(1)),
			"alight":        reading.yes(&made.alight),
			"stands_in_for": reading.text(&made.standsIn),
		}, nil)
		made.kind = valueOf(block, "kind")
		reading.set.things = append(reading.set.things, made)
		return err
	})
}

// stands reads each stand, by its name.
func (reading reader) stands(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Stand{name: name.Value, above: math.Inf(-1),
			below: math.Inf(1)}
		err := reading.keys(block, map[string]step{
			"kind":          reading.only(manufacture.KindNames()...),
			"count":         reading.whole(&made.count, 1, mostStand),
			"seed":          reading.whole(&made.seed, 0, mostSeed),
			"alight":        reading.whole(&made.alight, 0, mostStand),
			"stands_in_for": reading.text(&made.standsIn),
			"above":         reading.number(&made.above, -1e4, 1e4),
			"below":         reading.number(&made.below, -1e4, 1e4),
			"within":        reading.wedge(&made.within),
		}, nil)
		made.kind = valueOf(block, "kind")
		switch {
		case err != nil:
		case made.alight > made.count:
			err = reading.refuse(block, "stand %s has %v alight of %v",
				made.name, made.alight, made.count)
		case made.above >= made.below:
			err = reading.refuse(block, "stand %s grows above %v and "+
				"below %v, which is nowhere", made.name, made.above,
				made.below)
		}
		reading.set.stands = append(reading.set.stands, made)
		return err
	})
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
