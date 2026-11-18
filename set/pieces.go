package set

import (
	"math"
	"slices"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/swatches"
	yaml "gopkg.in/yaml.v3"
)

// World is the set's world said piece by piece, as the tracer holds it:
// what things are made of, the boxes and spheres made of them, the
// lamps, spots, floods and suns that light them, the sky, the bodies,
// the wisps of gas, the fizzes, and the motions that carry things.
//
// its places are in the world's unit, a metre unless it says, as a
// map's run (x east, y north, z up), and come back in the world's space.
//
// every piece but a body and a motion says where its values came from,
// as its source: eye, for judged by eye, or the measurement they were
// read from.
type World struct {
	unit          float64
	order         []string
	materials     []Material
	ocean         *WaveSurface
	boxes         []Box
	spheres       []Sphere
	lamps         []Lamp
	beams         []Beam
	spots         []Spotlight
	floods        []Flood
	sky           *Sky
	bodies        []Body
	suns          []Sunlight
	wisps         []Wisp
	fizzes        []Fizz
	vaporCavities []VaporCavitation
	motions       []Motion
	imports       []Import
	griffith      *GriffithBasin
}

// GriffithBasin is the authored Los Angeles approximation from the Heard
// Griffith stage, placed as one reusable procedural landscape in a set.
type GriffithBasin struct {
	at             *WorldPosition
	heading        float64
	scale          float64
	relief         float64
	cityForward    float64
	ridgeForward   float64
	cityDetail     float64
	terrainSamples int
	lightScale     float64
	lightFootprint float64
}

// Griffith returns the optional procedural Griffith/LA ridge and city.
func (made World) Griffith() (GriffithBasin, bool) {
	if made.griffith == nil {
		return GriffithBasin{}, false
	}
	return *made.griffith, true
}

// At is the landscape's absolute map position in metres.
func (made GriffithBasin) At() WorldPosition {
	if made.at == nil {
		return WorldPosition{}
	}
	return *made.at
}

// Heading is clockwise from north in degrees.
func (made GriffithBasin) Heading() float64 { return made.heading }

// Scale is the authored landscape scale; an omitted scale keeps native 1:1 metres.
func (made GriffithBasin) Scale() float64 {
	if made.scale == 0 {
		return 1
	}
	return made.scale
}

// ReliefScale is the authored vertical scale of the ridge; zero means native.
func (made GriffithBasin) ReliefScale() float64 {
	if made.relief == 0 {
		return 1
	}
	return made.relief
}

// CityForward is the local distance in metres that moves city blocks toward
// the seaward side; negative values bring the authored skyline closer.
func (made GriffithBasin) CityForward() float64 { return made.cityForward }

// RidgeForward moves the authored Griffith ridge toward the inland side in
// local metres, allowing the basin city to sit on the seaward side.
func (made GriffithBasin) RidgeForward() float64 { return made.ridgeForward }

// CityDetail is the deterministic fraction of repeated neighbourhood lots.
func (made GriffithBasin) CityDetail() float64 {
	if made.cityDetail == 0 {
		return 1
	}
	return made.cityDetail
}

// TerrainSamples is the square ridge mesh edge size, including its vertices.
func (made GriffithBasin) TerrainSamples() int {
	if made.terrainSamples == 0 {
		return 513
	}
	return made.terrainSamples
}

// LightScale is the emissive cue gain for city windows and street lamps.
func (made GriffithBasin) LightScale() float64 {
	if made.lightScale == 0 {
		return 1
	}
	return made.lightScale
}

// LightFootprint scales and spaces window/light geometry for far-view
// resolution; one leaves the authored size unchanged.
func (made GriffithBasin) LightFootprint() float64 {
	if made.lightFootprint == 0 {
		return 1
	}
	return made.lightFootprint
}

// Unit is how many metres one of the world's units is.
func (made World) Unit() float64 { return made.unit }

// Order is the world's keys in the order the set file writes them, so
// what builds the world can put its pieces in as they were written.
func (made World) Order() []string { return slices.Clone(made.order) }

// Spheres is every sphere, in the order written.
func (made World) Spheres() []Sphere { return slices.Clone(made.spheres) }

// Spots is every spot, in the order written.
func (made World) Spots() []Spotlight { return slices.Clone(made.spots) }

// Wisps is every wisp, in the order written.
func (made World) Wisps() []Wisp { return slices.Clone(made.wisps) }

// Fizzes is every fizz, in the order written.
func (made World) Fizzes() []Fizz { return slices.Clone(made.fizzes) }

// VaporCavities are the bounded authored propeller-vapor density fields.
func (made World) VaporCavities() []VaporCavitation {
	return slices.Clone(made.vaporCavities)
}

// Motions is every motion, in the order written.
func (made World) Motions() []Motion { return slices.Clone(made.motions) }

// World is the set's world, and whether it says one.
func (made Set) World() (World, bool) {
	if made.world == nil {
		return World{}, false
	}
	return *made.world, true
}

// Materials is every material, in the order written.
func (made World) Materials() []Material { return slices.Clone(made.materials) }

// OceanSurface is one finite, explicitly sampled procedural wave sheet.
func (made World) OceanSurface() (WaveSurface, bool) {
	if made.ocean == nil {
		return WaveSurface{}, false
	}
	return *made.ocean, true
}

// Boxes is every box, in the order written.
func (made World) Boxes() []Box { return slices.Clone(made.boxes) }

// Lamps is every lamp, in the order written.
func (made World) Lamps() []Lamp { return slices.Clone(made.lamps) }

// Floods is every flood, in the order written.
func (made World) Floods() []Flood { return slices.Clone(made.floods) }

// Sky is the world's sky, and whether it says one.
func (made World) Sky() (Sky, bool) {
	if made.sky == nil {
		return Sky{}, false
	}
	return *made.sky, true
}

// Bodies is every body, in the order written.
func (made World) Bodies() []Body { return slices.Clone(made.bodies) }

// Suns is every sun, in the order written.
func (made World) Suns() []Sunlight { return slices.Clone(made.suns) }

// A Colour is linear light in srgb's primaries, each part a share not
// below nought, as srgb_linear: [red, green, blue] writes it; or a
// swatch of slayer's by its name, as swatch: rust does.
type Colour struct {
	red, green, blue float64
	swatch           string
}

// Linear is the colour's red, green and blue, in linear light, where it
// is said so.
func (made Colour) Linear() (red, green, blue float64) {
	return made.red, made.green, made.blue
}

// Swatch is the name of the swatch the colour is, and whether it is one.
func (made Colour) Swatch() (string, bool) {
	return made.swatch, made.swatch != ""
}

// A Material describes a set-authored surface: its colour, glow, or clear
// dielectric interface. Colour and glow may be combined with one another
// and with the clear interface.
type Material struct {
	name   string
	source string
	colour *Colour
	glow   *[2]float64 // kelvin, luminance
	glows  *Colour
	clear  *Clear
}

// Clear describes a renderer clear dielectric surface. Through is the share
// passed, from above zero through one. Index is the refractive index, in the
// numeric-safe range 1e-9 through 1e9. Gloss scales Fresnel reflection, from
// zero through one. Rank selects the medium where clear surfaces overlap.
// Optional RGB extinction-per-metre describes unscattered attenuation in
// this named medium; it does not describe scattering or direct-light transport.
type Clear struct {
	Through float64
	Index   float64
	Gloss   float64
	Rank    int
	// mediumName is the material key, the stable ID shared by its
	// boundaries and an explicitly authored camera start stack.
	mediumName string
	// extinction is optional RGB inverse metres. Nil preserves the
	// historical clear-interface-only behavior.
	extinction    *[3]float64
	participating *clearScattering
}

type clearScattering struct {
	coefficient [3]float64
	phaseG      float64
	steps       int
	maxDistance float64
}

// OpticalMedium is the optional named unscattered-extinction descriptor
// carried by this clear interface. The material's key supplies its stable ID.
func (made Clear) OpticalMedium() (name string, extinction [3]float64,
	has bool) {
	if made.extinction == nil {
		return "", [3]float64{}, false
	}
	return made.mediumName, *made.extinction, true
}

// ParticipatingMedium returns absorption, scattering and the explicit bounded
// source-integration settings when this clear material opts into scattering.
func (made Clear) ParticipatingMedium() (absorption, scattering [3]float64,
	phaseG float64, steps int, maxDistanceMetres float64, has bool) {
	if made.participating == nil || made.extinction == nil {
		return [3]float64{}, [3]float64{}, 0, 0, 0, false
	}
	scatter := made.participating.coefficient
	var absorb [3]float64
	for channel := range absorb {
		absorb[channel] = made.extinction[channel] - scatter[channel]
	}
	return absorb, scatter, made.participating.phaseG,
		made.participating.steps, made.participating.maxDistance, true
}

// Clear is the material's clear dielectric interface, if it has one.
func (made Material) Clear() (Clear, bool) {
	if made.clear == nil {
		return Clear{}, false
	}
	return *made.clear, true
}

// Glowing is the colour the material glows with, and whether its glow
// is a colour rather than a black body's.
func (made Material) Glowing() (Colour, bool) {
	if made.glows == nil {
		return Colour{}, false
	}
	return *made.glows, true
}

// Name is the material's name, which a box uses to wear it.
func (made Material) Name() string { return made.name }

// Colour is the material's colour, and whether it has one.
func (made Material) Colour() (Colour, bool) {
	if made.colour == nil {
		return Colour{}, false
	}
	return *made.colour, true
}

// Glow is the temperature, in kelvin, of the black body whose light the
// material glows with, its luminance, and whether it glows.
func (made Material) Glow() (temperature, luminance float64, has bool) {
	if made.glow == nil {
		return 0, 0, false
	}
	return made.glow[0], made.glow[1], true
}

// A Box is a box of a material, its middle at a place, so wide, deep and
// tall, in metres.
type Box struct {
	name, material string
	middle, size   [3]float64
}

// Name is the box's name.
func (made Box) Name() string { return made.name }

// Material is the name of the material it wears.
func (made Box) Material() string { return made.material }

// Middle is where its middle stands, in the world's space.
func (made Box) Middle() maths.Vec { return fromMap(made.middle) }

// Size is how wide, tall and deep it is, along the world's x, y and z.
func (made Box) Size() maths.Vec {
	return maths.V(made.size[0], made.size[2], made.size[1])
}

// A Lamp is a bulb: its place, the temperature of the black body it
// shines as, its luminance at a metre, and its radius, inside which it
// does not light.
type Lamp struct {
	name                          string
	at                            [3]float64
	temperature, strength, radius float64
}

// Name is the lamp's name.
func (made Lamp) Name() string { return made.name }

// At is where it stands, in the world's space.
func (made Lamp) At() maths.Vec { return fromMap(made.at) }

// Shine is its temperature in kelvin and its luminance at a metre.
func (made Lamp) Shine() (temperature, strength float64) {
	return made.temperature, made.strength
}

// Radius is the bulb's radius in metres.
func (made Lamp) Radius() float64 { return made.radius }

// A Flood is light that reaches everything alike, from everywhere: a
// colour, so strong.
type Flood struct {
	name     string
	colour   Colour
	strength float64
}

// Name is the flood's name.
func (made Flood) Name() string { return made.name }

// Colour is the flood's colour.
func (made Flood) Colour() Colour { return made.colour }

// Strength is how many times its colour the flood's light is.
func (made Flood) Strength() float64 { return made.strength }

// A Sky is the light from far off in every way: at the horizon, at the
// zenith, and from the ground below; or the stars, scattered by a seed.
type Sky struct {
	horizon, zenith, ground Colour
	stars                   *Stars
}

// Light is the sky's horizon, zenith and ground.
func (made Sky) Light() (horizon, zenith, ground Colour) {
	return made.horizon, made.zenith, made.ground
}

// Starry is the sky's stars, and whether it is stars.
func (made Sky) Starry() (Stars, bool) {
	if made.stars == nil {
		return Stars{}, false
	}
	return *made.stars, true
}

// Stars are a sky of points scattered by a seed: the share of the sky
// that holds one, how big they are, and how bright.
type Stars struct {
	seed                float64
	share, size, bright float64
}

// Seed is the seed that scatters them.
func (made Stars) Seed() uint64 { return uint64(made.seed) }

// Look is the share of the sky that holds a star, the stars' size and
// their brightness.
func (made Stars) Look() (share, size, bright float64) {
	return made.share, made.size, made.bright
}

// A Body is a body from a template, its middle at a place, its moons'
// ground tiled so finely; and, where the set says, drawn at a radius of
// its own, its axis tilted, and its aurora shown or not. what it does
// not say is the template's.
type Body struct {
	name, template string
	middle         [3]float64
	detail         float64
	radius, tilt   *float64
	aurora         *bool
}

// Name is the body's name.
func (made Body) Name() string { return made.name }

// Template is the template it is made from, by its name.
func (made Body) Template() string { return made.template }

// Middle is where its middle stands, in the world's space.
func (made Body) Middle() maths.Vec { return fromMap(made.middle) }

// Radius is the radius it is drawn at, in the world's units, and whether
// the set says one; unsaid, it is its template's own.
func (made Body) Radius() (float64, bool) { return said(made.radius) }

// Tilt is how far its axis is tilted, in degrees, and whether the set
// says; unsaid, it is its template's.
func (made Body) Tilt() (float64, bool) { return said(made.tilt) }

// Aurora is whether its aurora shows, and whether the set says; unsaid,
// it is as its template has it.
func (made Body) Aurora() (shown, has bool) {
	if made.aurora == nil {
		return false, false
	}
	return *made.aurora, true
}

// Detail is how finely its moons' ground is tiled.
func (made Body) Detail() int { return int(made.detail) }

// said is a number the set may say, and whether it does.
func said(number *float64) (float64, bool) {
	if number == nil {
		return 0, false
	}
	return *number, true
}

// A Sunlight is a sun so far off that its light comes one way: the way to
// it, and its light.
type Sunlight struct {
	name   string
	toward [3]float64
	colour Colour
}

// Name is the sun's name.
func (made Sunlight) Name() string { return made.name }

// Toward is the way to the sun, in the world's space, as long as written.
func (made Sunlight) Toward() maths.Vec { return fromMap(made.toward) }

// Colour is the sun's light.
func (made Sunlight) Colour() Colour { return made.colour }

// the farthest a place or a size may be, in metres: past saturn's orbit
const farthest = 1e13

// the brightest a light may be, the hottest a black body, and the most
// a share of light may be
const (
	brightest = 1e12
	hottest   = 1e6
)

// world reads the set's world, piece by piece, keeping the order its
// keys are written in.
func (reading reader) world(_, value *yaml.Node) error {
	made := &World{unit: 1}
	reading.set.world = made
	steps := map[string]step{
		"unit": reading.number(&made.unit, math.SmallestNonzeroFloat64,
			farthest),
		"ocean":          reading.waveSurface,
		"materials":      reading.materials,
		"boxes":          reading.boxes,
		"spheres":        reading.spheres,
		"lamps":          reading.lamps,
		"beams":          reading.beams,
		"spots":          reading.spotlights,
		"floods":         reading.floods,
		"sky":            reading.backdrop,
		"bodies":         reading.bodies,
		"suns":           reading.suns,
		"wisps":          reading.wisps,
		"fizzes":         reading.fizzes,
		"vapor-cavities": reading.vaporCavities,
		"motions":        reading.motions,
		"imports":        reading.imports,
		"griffith-basin": reading.griffithBasin(&made.griffith),
	}
	for key, read := range steps {
		steps[key] = ordered(made, key, read)
	}
	return reading.keys(value, steps, nil)
}

// griffithBasin returns a step that reads a basin block into the target.
func (reading reader) griffithBasin(into **GriffithBasin) step {
	return func(_, value *yaml.Node) error {
		made := &GriffithBasin{}
		err := reading.keys(value, map[string]step{
			"at":                   reading.worldPosition(&made.at),
			"heading":              reading.quantity(&made.heading, "deg", 0, 360),
			"scale":                reading.number(&made.scale, 0.01, 10),
			"relief-scale":         reading.number(&made.relief, 0.01, 2),
			"city-forward":         reading.quantity(&made.cityForward, "m", -2000, 2000),
			"ridge-forward":        reading.quantity(&made.ridgeForward, "m", 0, 10000),
			"city-detail":          reading.number(&made.cityDetail, 0.1, 1),
			"terrain-samples":      reading.integer(&made.terrainSamples, 129, 513),
			"city-light-scale":     reading.number(&made.lightScale, 1e-9, 20),
			"city-light-footprint": reading.number(&made.lightFootprint, 0.5, 6),
		}, nil)
		if err == nil {
			err = reading.all(value, "a Griffith basin placement", "at", "heading")
		}
		if err == nil {
			*into = made
		}
		return err
	}
}

// ordered is a step that notes its key in the world's order, then reads.
func ordered(made *World, key string, read step) step {
	return func(name, value *yaml.Node) error {
		made.order = append(made.order, key)
		return read(name, value)
	}
}

// materials reads each material, by its name.
func (reading reader) materials(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Material{name: name.Value}
		var source string
		colour := Colour{}
		clear := Clear{}
		sourceStep := reading.source(&source)
		if says(valueNode(block, "clear"), "extinction-per-metre") {
			sourceStep = reading.provenance(&source)
		}
		err := reading.keys(block, map[string]step{
			"colour": reading.colour(&colour),
			"glow":   reading.glowed(&made),
			"clear":  reading.clear(&clear),
			"source": sourceStep,
		}, nil)
		if err == nil {
			err = reading.all(block, "a material", "source")
		}
		if err == nil && clear.extinction != nil &&
			(source == "" || source == "eye") {
			err = reading.refuse(block,
				"material %s has optical extinction but no explicit provenance; name measured data or label synthetic values", name.Value)
		}
		if err == nil && !says(block, "colour") && !says(block, "glow") &&
			!says(block, "clear") {
			err = reading.refuse(block, "material %s has no colour and "+
				"no glow or clear interface", made.name)
		}
		if says(block, "colour") {
			made.colour = &colour
		}
		if says(block, "clear") {
			if clear.extinction != nil {
				clear.mediumName = name.Value
			}
			made.clear = &clear
		}
		if reading.importProvenance {
			made.source = source
		}
		reading.set.world.materials = append(reading.set.world.materials,
			made)
		return err
	})
}

// provenance reads a nonempty scalar source for authored optical data.
// Unlike source's historical measured-source grammar, this does not depend
// on YAML quoting; the stored text must still say what the values represent.
func (reading reader) provenance(target *string) step {
	return func(name, value *yaml.Node) error {
		text := strings.TrimSpace(value.Value)
		if value.Kind != yaml.ScalarNode || value.Tag != "!!str" ||
			text == "" || text == "eye" {
			return reading.refuse(value,
				"%s takes a nonempty string provenance other than eye", name.Value)
		}
		*target = value.Value
		return nil
	}
}

// clear reads the renderer's transmissive dielectric surface fields.
func (reading reader) clear(target *Clear) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"through": reading.number(&target.Through,
				math.SmallestNonzeroFloat64, 1),
			"index": reading.number(&target.Index, 1e-9, 1e9),
			"gloss": reading.number(&target.Gloss, 0, 1),
			"rank":  reading.integer(&target.Rank, -2147483648, 2147483647),
			"extinction-per-metre": reading.optionalTriple(
				&target.extinction, 0, brightest),
			"scatter": reading.clearScatter(&target.participating),
		}, nil)
		if err == nil {
			err = reading.all(value, "a clear interface", "through",
				"index", "gloss")
		}
		if err == nil && target.participating != nil {
			if target.extinction == nil {
				err = reading.refuse(valueNode(value, "scatter"),
					"clear scattering requires extinction-per-metre")
			} else {
				err = reading.all(valueNode(value, "scatter"),
					"a bounded clear-medium scattering profile",
					"coefficient-per-metre", "phase-g", "steps", "max-distance")
			}
			if err == nil {
				for channel, coefficient := range target.participating.coefficient {
					if coefficient > target.extinction[channel] {
						err = reading.refuse(valueNode(value, "scatter"),
							"clear scattering coefficient %g exceeds extinction %g in channel %d",
							coefficient, target.extinction[channel], channel)
						break
					}
				}
			}
		}
		return err
	}
}

// clearScatter returns a step that reads a clear-scattering block into
// the target.
func (reading reader) clearScatter(target **clearScattering) step {
	return func(_, value *yaml.Node) error {
		made := &clearScattering{}
		err := reading.keys(value, map[string]step{
			"coefficient-per-metre": reading.three(&made.coefficient, 0, brightest),
			"phase-g": reading.number(&made.phaseG,
				-math.Nextafter(1, 0), math.Nextafter(1, 0)),
			"steps": reading.integer(&made.steps, 1, 64),
			"max-distance": reading.quantity(&made.maxDistance, "m",
				math.SmallestNonzeroFloat64, farthest),
		}, nil)
		if err == nil {
			*target = made
		}
		return err
	}
}

// optionalTriple reads a three-channel array when the key is present.
func (reading reader) optionalTriple(target **[3]float64, least,
	most float64) step {
	return func(name, value *yaml.Node) error {
		var parts [3]float64
		if err := reading.three(&parts, least, most)(name, value); err != nil {
			return err
		}
		*target = &parts
		return nil
	}
}

// glowed is a step that reads a material's glow: a black body's
// temperature and luminance, or a colour.
func (reading reader) glowed(made *Material) step {
	return func(name, value *yaml.Node) error {
		if says(value, "srgb_linear") || says(value, "swatch") {
			glows := Colour{}
			err := reading.colour(&glows)(name, value)
			made.glows = &glows
			return err
		}
		var black [2]float64
		err := reading.keys(value, map[string]step{
			"temperature": reading.number(&black[0], 1, hottest),
			"luminance":   reading.number(&black[1], 0, brightest),
		}, nil)
		if err == nil {
			err = reading.all(value, "a glow", "temperature", "luminance")
		}
		made.glow = &black
		return err
	}
}

// boxes reads each box, by its name.
func (reading reader) boxes(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Box{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"material": reading.text(&made.material),
			"middle":   reading.three(&made.middle, -farthest, farthest),
			"size": reading.three(&made.size,
				math.SmallestNonzeroFloat64, farthest),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a box", "material", "middle", "size",
				"source")
		}
		reading.set.world.boxes = append(reading.set.world.boxes, made)
		return err
	})
}

// lamps reads each lamp, by its name.
func (reading reader) lamps(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Lamp{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"at":          reading.three(&made.at, -farthest, farthest),
			"temperature": reading.number(&made.temperature, 1, hottest),
			"strength":    reading.number(&made.strength, 0, brightest),
			"radius":      reading.number(&made.radius, 0, farthest),
			"source":      reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a lamp", "at", "temperature",
				"strength", "radius", "source")
		}
		reading.set.world.lamps = append(reading.set.world.lamps, made)
		return err
	})
}

// floods reads each flood, by its name.
func (reading reader) floods(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Flood{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"colour":   reading.colour(&made.colour),
			"strength": reading.number(&made.strength, 0, brightest),
			"source":   reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a flood", "colour", "strength",
				"source")
		}
		reading.set.world.floods = append(reading.set.world.floods, made)
		return err
	})
}

// backdrop reads the world's sky: a gradient, or stars.
func (reading reader) backdrop(_, value *yaml.Node) error {
	made := &Sky{}
	var source string
	var stars Stars
	err := reading.keys(value, map[string]step{
		"horizon": reading.colour(&made.horizon),
		"zenith":  reading.colour(&made.zenith),
		"ground":  reading.colour(&made.ground),
		"stars": reading.block(map[string]step{
			"seed":   reading.whole(&stars.seed, 0, mostSeed),
			"share":  reading.number(&stars.share, 0, 1),
			"size":   reading.number(&stars.size, 0, brightest),
			"bright": reading.number(&stars.bright, 0, brightest)}),
		"source": reading.source(&source),
	}, nil)
	starry := says(value, "stars")
	switch {
	case err != nil:
	case starry && (says(value, "horizon") || says(value, "zenith") ||
		says(value, "ground")):
		err = reading.refuse(value, "a sky is a gradient or stars, not "+
			"both")
	case starry:
		err = reading.all(value, "a sky of stars", "stars", "source")
		if err == nil {
			err = reading.all(valueNode(value, "stars"), "stars", "seed",
				"share", "size", "bright")
		}
		made.stars = &stars
	default:
		err = reading.all(value, "a sky", "horizon", "zenith", "ground",
			"source")
	}
	reading.set.world.sky = made
	return err
}

// bodies reads each body, by its name. what a body does not say of its
// radius, tilt and aurora is its template's.
func (reading reader) bodies(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Body{name: name.Value}
		var radius, tilt float64
		var aurora bool
		err := reading.keys(block, map[string]step{
			"template": reading.word(&made.template, template.Names()...),
			"middle":   reading.three(&made.middle, -farthest, farthest),
			"detail":   reading.whole(&made.detail, 1, math.MaxInt32),
			"radius": reading.number(&radius, math.SmallestNonzeroFloat64,
				farthest),
			"tilt":   reading.number(&tilt, -180, 180),
			"aurora": reading.yes(&aurora),
		}, nil)
		if err == nil {
			err = reading.all(block, "a body", "template", "middle",
				"detail")
		}
		if says(block, "radius") {
			made.radius = &radius
		}
		if says(block, "tilt") {
			made.tilt = &tilt
		}
		if says(block, "aurora") {
			made.aurora = &aurora
		}
		reading.set.world.bodies = append(reading.set.world.bodies, made)
		return err
	})
}

// suns reads each sun, by its name.
func (reading reader) suns(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Sunlight{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"toward": reading.three(&made.toward, -farthest, farthest),
			"colour": reading.colour(&made.colour),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a sun", "toward", "colour", "source")
		}
		if err == nil && made.toward == [3]float64{} {
			err = reading.refuse(block, "sun %s is toward nought, nought "+
				"and nought, which is no way", made.name)
		}
		reading.set.world.suns = append(reading.set.world.suns, made)
		return err
	})
}

// source is a step that reads where a piece's values came from: eye,
// for judged by eye, or a measurement, quoted, which says what it was.
// a word that is neither, guess or a typo, says nothing.
func (reading reader) source(target *string) step {
	return func(name, value *yaml.Node) error {
		quoted := value.Style&(yaml.DoubleQuotedStyle|
			yaml.SingleQuotedStyle) != 0
		if value.Kind != yaml.ScalarNode || value.Value == "" ||
			!quoted && value.Value != "eye" {
			return reading.refuse(value, "%s takes eye or a quoted "+
				"measurement, not %q", name.Value, value.Value)
		}
		*target = value.Value
		return nil
	}
}

// colour is a step that reads a colour: srgb_linear and its three parts,
// or a swatch by its name, one of the two.
func (reading reader) colour(target *Colour) step {
	return func(_, value *yaml.Node) error {
		var parts [3]float64
		var named string
		err := reading.keys(value, map[string]step{
			"srgb_linear": reading.three(&parts, 0, brightest),
			"swatch":      reading.word(&named, swatches.Names()...),
		}, nil)
		if err == nil &&
			says(value, "srgb_linear") == says(value, "swatch") {
			err = reading.refuse(value, "a colour says srgb_linear or "+
				"swatch, one of the two")
		}
		*target = Colour{parts[0], parts[1], parts[2], named}
		return err
	}
}

// three is a step that reads three numbers, each from least to most, into
// target: a place, a size or a way, as [x, y, z].
func (reading reader) three(target *[3]float64, least,
	most float64) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode || len(value.Content) != 3 {
			return reading.refuse(value, "%s takes three numbers, as "+
				"[x, y, z]", name.Value)
		}
		for index, each := range value.Content {
			if err := reading.number(&target[index], least, most)(name,
				each); err != nil {
				return err
			}
		}
		return nil
	}
}

// all refuses a block that leaves out any of the keys a piece must say.
func (reading reader) all(block *yaml.Node, what string,
	keys ...string) error {
	for _, key := range keys {
		if !says(block, key) {
			return reading.refuse(block, "%s says %v, all of them, and "+
				"this has no %s", what, keys, key)
		}
	}
	return nil
}

// says is whether a block holds a key; no block holds none.
func says(block *yaml.Node, key string) bool {
	return valueNode(block, key) != nil
}

// valueNode is a key's value in a block, or nil, as it is for no block.
func valueNode(block *yaml.Node, key string) *yaml.Node {
	if block == nil {
		return nil
	}
	for i := 0; i+1 < len(block.Content); i += 2 {
		if block.Content[i].Value == key {
			return block.Content[i+1]
		}
	}
	return nil
}
