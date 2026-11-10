package set

import (
	"math"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// gases is every gas a planet's air may hold, by the name a set gives
// it: those the planet package has a measured index for.
var gases = []string{"neon", "krypton", "xenon", "methane"}

// later is gases a planet's air will hold once their light is built,
// each refused with the reason.
var later = map[string]string{
	"sodium": "sodium's d lines and its glow are not built yet",
	"radon":  "radon's index and its decay are not built yet",
}

// A Planet is a made-up planet in the set's middle: its name, its
// radius in metres, its gravity in metres a second squared, the albedo
// of the deck under its air, and its distance from its star in au.
//
// it has columns of air by name, and zones from pole to pole, each
// naming the column over it.
type Planet struct {
	name                  string
	radius, gravity, deck float64
	fromStar              float64
	columns               map[string][]Layer
	zones                 []Zone
	pole                  [2]float64
	rings                 []Ring
	sky                   *[3]float64
	storms                *Storms
	ringlets              *[3]float64
	xrays                 *[2]float64
	weather               *Weather
	heat                  *[2]float64
	cooking               *cooking
	airglow               *[3]float64
}

// Ringlets is the radial structure in every ring: its seed, its strength
// and how many metres across its coarsest ringlets are, and whether the
// set says.
func (made Planet) Ringlets() (seed uint64, strength, width float64,
	has bool) {
	if made.ringlets == nil {
		return 0, 0, 0, false
	}
	return uint64(made.ringlets[0]), made.ringlets[1], made.ringlets[2],
		true
}

// jupiter is jupiter's x-ray power in watts, the unit a planet's x-rays
// are given in: about one gigawatt, inside what was measured, its disc's
// 0.21 to 0.76 gw (dunn and others 2020) and its aurora's 0.4 to 1 gw
// (bhardwaj and others' 2007 review). picked within that.
const jupiter = 1e9

// Xrays is the planet's x-ray power in watts and the band about its
// equator they leave from, in degrees either side, and whether the set
// says.
func (made Planet) Xrays() (watts, band float64, has bool) {
	if made.xrays == nil {
		return 0, 0, false
	}
	return made.xrays[0] * jupiter, made.xrays[1], true
}

// Storms are how the planet's bands are torn and mixed: a seed, a turbulence
// from zero to one, how many eddies fit around it, how far its jets stretch
// them, the blend's width at a band's edge in degrees, how many times the flow
// folds back through itself, and its spots.
type Storms struct {
	seed, turbulence, eddies, stretch, edge float64
	folds                                   float64
	spots                                   []Spot
	jets                                    *[2]float64
	swarm                                   *[5]float64
	cap, clearing                           string
	churn                                   string
	churning                                float64
}

// Churn is the column the storms stir up at a band's edge, and how strongly it
// shows, zero to one; no column when the set says none.
func (made Storms) Churn() (column string, strength float64) {
	return made.churn, made.churning
}

// A Spot is a vortex: its latitude and longitude, longitude zero toward the
// camera, its radius in degrees, and its turn in half turns.
type Spot struct{ latitude, longitude, radius, turn float64 }

// Stormy is the planet's storms, and whether it has any.
func (made Planet) Stormy() (Storms, bool) {
	if made.storms == nil {
		return Storms{}, false
	}
	return *made.storms, true
}

// Flow is the storms' seed, turbulence, eddies, stretch, edge and folds; folds
// zero where the set says none.
func (made Storms) Flow() (seed uint64, turbulence, eddies, stretch,
	edge float64, folds int) {
	return uint64(made.seed), made.turbulence, made.eddies, made.stretch,
		made.edge, int(made.folds)
}

// Jets is how many degrees of longitude the storms' jets carry the air,
// how many degrees of latitude apart they turn about, and whether the
// set says.
func (made Storms) Jets() (shear, every float64, has bool) {
	if made.jets == nil {
		return 0, 0, false
	}
	return made.jets[0], made.jets[1], true
}

// Swarm is the storms' swarm of spots: its seed, how many, their radii
// from smallest to largest in degrees, the slope of how many more small
// ones there are, and whether the set says.
func (made Storms) Swarm() (seed uint64, count int, smallest, largest,
	slope float64, has bool) {
	if made.swarm == nil {
		return 0, 0, 0, 0, 0, false
	}
	swarm := made.swarm
	return uint64(swarm[0]), int(swarm[1]), swarm[2], swarm[3], swarm[4],
		true
}

// Marks is the columns the storms' vortices show, a cap and a clearing;
// none when the set says none.
func (made Storms) Marks() (cap, clearing string) {
	return made.cap, made.clearing
}

// Spots is the storms' spots.
func (made Storms) Spots() []Spot { return slices.Clone(made.spots) }

// Where is the spot's latitude, longitude, radius and turn.
func (made Spot) Where() (latitude, longitude, radius, turn float64) {
	return made.latitude, made.longitude, made.radius, made.turn
}

// InSky is where the planet stands in a ground set's sky: its azimuth
// and altitude in degrees and its distance in metres from the set's
// middle, and whether the set says. a planet with no sky is in space,
// at the set's middle.
func (made Planet) InSky() (azimuth, altitude, distance float64,
	has bool) {
	if made.sky == nil {
		return 0, 0, 0, false
	}
	return made.sky[0], made.sky[1], made.sky[2], true
}

// A Ring is a ring about the planet's equator: its inner and outer
// radius in metres, its optical depth through, its particles' albedo and
// lean, and what it is made of, by share of its mass.
type Ring struct {
	inner, outer, depth, albedo, lean float64
	blue                              float64
	madeOf                            map[string]float64
	grains                            *Grains
	moonlets                          *[6]float64
}

// Moonlets is the ring's moonlets: their seed, how many, their radii
// from smallest to largest in metres, the slope of their power law, the
// radius from which one clears its whole orbit, and whether the set
// gives it any.
func (made Ring) Moonlets() (seed uint64, count int, smallest, largest,
	slope, opens float64, has bool) {
	if made.moonlets == nil {
		return 0, 0, 0, 0, 0, 0, false
	}
	return uint64(made.moonlets[0]), int(made.moonlets[1]),
		made.moonlets[2], made.moonlets[3], made.moonlets[4],
		made.moonlets[5], true
}

// moons reads a ring's moonlets, every part said.
func (reading reader) moons(into **[6]float64) step {
	return func(_, value *yaml.Node) error {
		got := [6]float64{math.NaN(), math.NaN(), math.NaN(), math.NaN(),
			math.NaN(), math.NaN()}
		err := reading.keys(value, map[string]step{
			"seed":     reading.whole(&got[0], 0, mostSeed),
			"count":    reading.whole(&got[1], 0, 1e4),
			"smallest": reading.number(&got[2], 1, 1e7),
			"largest":  reading.number(&got[3], 1, 1e7),
			"slope":    reading.number(&got[4], 1.01, 10),
			"opens":    reading.number(&got[5], 1, 1e9),
		}, nil)
		if err != nil {
			return err
		}
		for _, part := range got {
			if math.IsNaN(part) {
				return reading.refuse(value, "moonlets say their seed, "+
					"count, smallest, largest, slope and opens")
			}
		}
		if got[3] < got[2] {
			return reading.refuse(value, "moonlets' largest is below "+
				"their smallest")
		}
		*into = &got
		return nil
	}
}

// Grains are how a ring's matter clumps: a pattern, a seed, its clumps from
// smallest to largest in metres, the slope of how many more small ones there
// are, the shear, how many times longer along the orbit than across a clump is
// drawn, and its clumping from zero to one.
type Grains struct {
	pattern                        string
	seed, smallest, largest, slope float64
	shear, clumping                float64
}

// grainPatterns is every pattern grains may take, by name; the planet
// package says what each looks like.
var grainPatterns = []string{"cells", "ridges", "smooth", "speckle"}

// Grained is the ring's grains, and whether the set gives it any.
func (made Ring) Grained() (Grains, bool) {
	if made.grains == nil {
		return Grains{}, false
	}
	return *made.grains, true
}

// Clumps is the grains' pattern, seed, sizes, slope, shear and clumping.
func (made Grains) Clumps() (pattern string, seed uint64, smallest,
	largest, slope, shear, clumping float64) {
	return made.pattern, uint64(made.seed), made.smallest, made.largest,
		made.slope, made.shear, made.clumping
}

// grained reads a ring's grains, every part said.
func (reading reader) grained(into **Grains) step {
	return func(_, value *yaml.Node) error {
		made := Grains{seed: math.NaN(), smallest: math.NaN(),
			largest: math.NaN(), slope: math.NaN(), shear: math.NaN(),
			clumping: math.NaN()}
		err := reading.keys(value, map[string]step{
			"pattern":  reading.word(&made.pattern, grainPatterns...),
			"seed":     reading.whole(&made.seed, 0, mostSeed),
			"smallest": reading.number(&made.smallest, 1e-3, 1e9),
			"largest":  reading.number(&made.largest, 1e-3, 1e9),
			"slope":    reading.number(&made.slope, 0.1, 10),
			"shear":    reading.number(&made.shear, 1, 1e4),
			"clumping": reading.number(&made.clumping, 0, 1),
		}, nil)
		if err != nil {
			return err
		}
		for _, part := range []float64{made.seed, made.smallest,
			made.largest, made.slope, made.shear, made.clumping} {
			if math.IsNaN(part) || made.pattern == "" {
				return reading.refuse(value, "grains say their pattern, "+
					"seed, smallest, largest, slope, shear and clumping")
			}
		}
		if made.largest < made.smallest {
			return reading.refuse(value, "grains' largest is below their "+
				"smallest")
		}
		*into = &made
		return nil
	}
}

// Blue is the ring's particles' albedo at 400 nm, and whether the set
// tints it; untinted, it is the albedo, which holds at 700 nm.
func (made Ring) Blue() (float64, bool) {
	return made.blue, !math.IsNaN(made.blue)
}

// ringStuff is what a ring may be made of, by the name a set gives it.
var ringStuff = []string{"ice", "zns-ag", "csi-tl"}

// Pole is which way the planet's pole points in the set's frame, as an
// azimuth and an altitude in degrees: straight up unless the set says.
func (made Planet) Pole() (azimuth, altitude float64) {
	return made.pole[0], made.pole[1]
}

// Rings is the planet's rings, inner to outer.
func (made Planet) Rings() []Ring { return slices.Clone(made.rings) }

// Reach is the ring's inner and outer radius, in metres.
func (made Ring) Reach() (inner, outer float64) {
	return made.inner, made.outer
}

// Optics is the ring's depth through, its albedo and its lean.
func (made Ring) Optics() (depth, albedo, lean float64) {
	return made.depth, made.albedo, made.lean
}

// MadeOf is what the ring is made of, by share of its mass.
func (made Ring) MadeOf() map[string]float64 {
	out := make(map[string]float64, len(made.madeOf))
	for name, share := range made.madeOf {
		out[name] = share
	}
	return out
}

// A Layer is the air between two pressures in bar, its gases by share
// of the molecules, and its haze: its depth, its albedo and its lean.
type Layer struct {
	top, bottom float64
	gases       map[string]float64
	haze        [3]float64
	blue        float64
}

// HazeBlue is the haze's albedo at 400 nm, and whether the set tints
// it; untinted, it is the albedo, which holds at 700 nm.
func (made Layer) HazeBlue() (float64, bool) {
	return made.blue, !math.IsNaN(made.blue)
}

// A Zone is the planet between two latitudes in degrees, and the name
// of the column over it.
type Zone struct {
	south, north float64
	column       string
}

// Name is the planet's name, which cameras use to find it.
func (made Planet) Name() string { return made.name }

// Size is the planet's radius, gravity and deck albedo.
func (made Planet) Size() (radius, gravity, deck float64) {
	return made.radius, made.gravity, made.deck
}

// FromStar is how far the planet is from its star, astronomical units.
func (made Planet) FromStar() float64 { return made.fromStar }

// Zones is the planet's zones, south to north.
func (made Planet) Zones() []Zone { return slices.Clone(made.zones) }

// Column is a column of air by its name, top down.
func (made Planet) Column(name string) []Layer {
	return slices.Clone(made.columns[name])
}

// Latitudes is the zone's southern and northern edge, and its column.
func (made Zone) Latitudes() (south, north float64, column string) {
	return made.south, made.north, made.column
}

// Pressures is the layer's top and bottom, in bar.
func (made Layer) Pressures() (top, bottom float64) {
	return made.top, made.bottom
}

// Gases is the layer's gases and their shares.
func (made Layer) Gases() map[string]float64 {
	out := make(map[string]float64, len(made.gases))
	for name, share := range made.gases {
		out[name] = share
	}
	return out
}

// Haze is the layer's haze: its depth, albedo and lean.
func (made Layer) Haze() (depth, albedo, lean float64) {
	return made.haze[0], made.haze[1], made.haze[2]
}

// Planet is the set's planet, and whether it has one.
func (made Set) Planet() (Planet, bool) {
	if made.planet == nil {
		return Planet{}, false
	}
	return *made.planet, true
}

// planet reads the set's planet. it must say made_up: true, since slayer
// has no real planet's face to draw; and its zones must cover it from
// pole to pole, each naming a column it has.
func (reading reader) planet(_, value *yaml.Node) error {
	made := &Planet{columns: map[string][]Layer{}, pole: [2]float64{0, 90}}
	var madeUp, said bool
	if err := reading.keys(value, reading.planetKeys(made, &madeUp,
		&said), nil); err != nil {
		return err
	}
	if !said || !madeUp {
		return reading.refuse(value, "a planet says made_up: true; "+
			"slayer has no real planet to draw")
	}
	if made.name == "" || made.radius == 0 || made.gravity == 0 ||
		made.fromStar == 0 || len(made.zones) == 0 {
		return reading.refuse(value, "a planet says its name, radius, "+
			"gravity, from_star and zones")
	}
	if made.storms != nil && made.storms.churn != "" {
		if _, has := made.columns[made.storms.churn]; !has {
			return reading.refuse(value, "the storms churn column %q, "+
				"which the planet has not", made.storms.churn)
		}
	}
	reading.set.planet = made
	return reading.covered(value, made)
}

// planetKeys is every key a planet takes, and the step that reads each.
func (reading reader) planetKeys(made *Planet, madeUp,
	said *bool) map[string]step {
	return map[string]step{
		"name":      reading.text(&made.name),
		"made_up":   reading.said(madeUp, said),
		"radius":    reading.number(&made.radius, 1, 1e9),
		"gravity":   reading.number(&made.gravity, 0.01, 1e4),
		"deck":      reading.number(&made.deck, 0, 1),
		"from_star": reading.number(&made.fromStar, 1e-3, 1e5),
		"columns":   reading.columns(made),
		"zones":     reading.zones(made),
		"pole": reading.block(map[string]step{
			"azimuth":  reading.number(&made.pole[0], 0, 360),
			"altitude": reading.number(&made.pole[1], -90, 90)}),
		"rings": reading.rings(made),
		"sky":   reading.sky(made),
		"storms": func(_, value *yaml.Node) error {
			storms, err := reading.storms(value)
			made.storms = storms
			return err
		},
		"ringlets": reading.trio(&made.ringlets, "seed", "strength",
			"width", [3][2]float64{{0, mostSeed}, {0, 10}, {1, 1e12}}),
		"xrays":   reading.rays(made),
		"weather": reading.weathered(&made.weather),
		"heat":    reading.heated(&made.heat),
		"cooking": reading.cooked(&made.cooking),
		"airglow": reading.glowing(&made.airglow),
	}
}

// trio reads a block of three numbers, each named and each between its
// own edges, every one said.
func (reading reader) trio(into **[3]float64, first, second, third string,
	edges [3][2]float64) step {
	return func(_, value *yaml.Node) error {
		got := [3]float64{math.NaN(), math.NaN(), math.NaN()}
		err := reading.keys(value, map[string]step{
			first:  reading.number(&got[0], edges[0][0], edges[0][1]),
			second: reading.number(&got[1], edges[1][0], edges[1][1]),
			third:  reading.number(&got[2], edges[2][0], edges[2][1]),
		}, nil)
		if err != nil {
			return err
		}
		for _, part := range got {
			if math.IsNaN(part) {
				return reading.refuse(value, "this says its %s, %s and %s",
					first, second, third)
			}
		}
		*into = &got
		return nil
	}
}

// rays reads the planet's x-rays: how many times jupiter's, of jupiter
// and nothing else, and the band about its equator they leave from.
func (reading reader) rays(made *Planet) step {
	return func(_, value *yaml.Node) error {
		got := [2]float64{math.NaN(), math.NaN()}
		err := reading.keys(value, map[string]step{
			"times": reading.number(&got[0], 0, 1e20),
			"of":    reading.only("jupiter"),
			"band":  reading.number(&got[1], 0.1, 45),
		}, nil)
		if err != nil {
			return err
		}
		if math.IsNaN(got[0]) || math.IsNaN(got[1]) ||
			valueOf(value, "of") != "jupiter" {
			return reading.refuse(value, "x-rays say how many times of "+
				"jupiter's, and their band")
		}
		made.xrays = &got
		return nil
	}
}

// said is a step that reads true or false, and notes that it was said.
func (reading reader) said(target, said *bool) step {
	read := reading.yes(target)
	return func(name, value *yaml.Node) error {
		*said = true
		return read(name, value)
	}
}

// covered refuses zones that leave a gap, overlap, stop short of a pole
// or name a column the planet does not have.
func (reading reader) covered(at *yaml.Node, made *Planet) error {
	south := -90.0
	for _, zone := range made.zones {
		if zone.south != south || !(zone.north > zone.south) {
			return reading.refuse(at, "the zones break at %v degrees",
				south)
		}
		if _, has := made.columns[zone.column]; !has {
			return reading.refuse(at, "a zone names column %q, which "+
				"the planet has not", zone.column)
		}
		south = zone.north
	}
	if south != 90 {
		return reading.refuse(at, "the zones stop at %v degrees", south)
	}
	return nil
}

// columns reads each column of air, by its name.
func (reading reader) columns(made *Planet) step {
	return func(_, value *yaml.Node) error {
		return reading.each(value, func(name, list *yaml.Node) error {
			layers, err := reading.layers(list)
			made.columns[name.Value] = layers
			return err
		})
	}
}

// layers reads a column's layers, top down, each starting where or
// below where the one above it ends.
func (reading reader) layers(list *yaml.Node) ([]Layer, error) {
	if list.Kind != yaml.SequenceNode || len(list.Content) == 0 {
		return nil, reading.refuse(list, "a column is a list of layers")
	}
	var layers []Layer
	for _, item := range list.Content {
		layer, err := reading.layer(item)
		if err != nil {
			return nil, err
		}
		if last := len(layers) - 1; last >= 0 &&
			layer.top < layers[last].bottom {
			return nil, reading.refuse(item, "this layer starts above "+
				"the bottom of the one before it")
		}
		layers = append(layers, layer)
	}
	return layers, nil
}

// layer reads one layer: its pressures, its gases, whose shares sum to
// one, and its haze.
func (reading reader) layer(item *yaml.Node) (Layer, error) {
	made := Layer{gases: map[string]float64{}, blue: math.NaN()}
	err := reading.keys(item, map[string]step{
		"top":    reading.number(&made.top, 0, 1e6),
		"bottom": reading.number(&made.bottom, 0, 1e6),
		"gases":  reading.mixed(made.gases),
		"haze": reading.block(map[string]step{
			"depth":  reading.number(&made.haze[0], 0, 1e6),
			"albedo": reading.number(&made.haze[1], 0, 1),
			"lean":   reading.number(&made.haze[2], -0.99, 0.99),
			"blue":   reading.number(&made.blue, 0, 1)}),
	}, nil)
	if err != nil {
		return made, err
	}
	if !(made.bottom > made.top) {
		return made, reading.refuse(item, "a layer's bottom is below "+
			"its top")
	}
	sum := 0.0
	for _, share := range made.gases {
		sum += share
	}
	if math.Abs(sum-1) > 1e-9 {
		return made, reading.refuse(item, "a layer's gases sum to %v, "+
			"not one", sum)
	}
	return made, nil
}

// mixed reads a layer's gases and their shares, refusing a gas not
// built yet with its reason.
func (reading reader) mixed(into map[string]float64) step {
	return func(_, value *yaml.Node) error {
		return reading.each(value, func(name, share *yaml.Node) error {
			if why, waits := later[name.Value]; waits {
				return reading.refuse(name, "%s: %s", name.Value, why)
			}
			if !slices.Contains(gases, name.Value) {
				return reading.refuse(name, "%s is not a gas slayer "+
					"knows; it knows %v", name.Value, gases)
			}
			var amount float64
			read := reading.number(&amount, 1e-9, 1)
			if err := read(name, share); err != nil {
				return err
			}
			into[name.Value] = amount
			return nil
		})
	}
}

// zones reads the planet's zones, south to north.
func (reading reader) zones(made *Planet) step {
	return func(_, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode {
			return reading.refuse(value, "zones is a list")
		}
		for _, item := range value.Content {
			zone := Zone{}
			err := reading.keys(item, map[string]step{
				"south":  reading.number(&zone.south, -90, 90),
				"north":  reading.number(&zone.north, -90, 90),
				"column": reading.text(&zone.column),
			}, nil)
			if err != nil {
				return err
			}
			made.zones = append(made.zones, zone)
		}
		return nil
	}
}

// rings reads the planet's rings, inner to outer, each clear of the one
// before it, each made of what slayer knows a ring may be made of.
func (reading reader) rings(made *Planet) step {
	return func(_, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode {
			return reading.refuse(value, "rings is a list")
		}
		edge := 0.0
		for _, item := range value.Content {
			ring, err := reading.ring(item)
			if err != nil {
				return err
			}
			if ring.inner < edge {
				return reading.refuse(item, "this ring starts inside "+
					"the one before it")
			}
			edge = ring.outer
			made.rings = append(made.rings, ring)
		}
		return nil
	}
}

// ring reads one ring: its reach, its optics and what it is made of,
// whose shares sum to one.
func (reading reader) ring(item *yaml.Node) (Ring, error) {
	made := Ring{madeOf: map[string]float64{}, blue: math.NaN()}
	err := reading.keys(item, map[string]step{
		"inner":    reading.number(&made.inner, 1, 1e12),
		"outer":    reading.number(&made.outer, 1, 1e12),
		"depth":    reading.number(&made.depth, 0, 1e3),
		"albedo":   reading.number(&made.albedo, 0, 1),
		"lean":     reading.number(&made.lean, -0.99, 0.99),
		"made_of":  reading.stuff(made.madeOf),
		"blue":     reading.number(&made.blue, 0, 1),
		"grains":   reading.grained(&made.grains),
		"moonlets": reading.moons(&made.moonlets),
	}, nil)
	if err != nil {
		return made, err
	}
	if !(made.outer > made.inner) {
		return made, reading.refuse(item, "a ring's outer edge is "+
			"inside its inner")
	}
	sum := 0.0
	for _, share := range made.madeOf {
		sum += share
	}
	if math.Abs(sum-1) > 1e-9 {
		return made, reading.refuse(item, "a ring's make-up sums to %v, "+
			"not one", sum)
	}
	return made, nil
}

// stuff reads what a ring is made of and the share of each.
func (reading reader) stuff(into map[string]float64) step {
	return func(_, value *yaml.Node) error {
		return reading.each(value, func(name, share *yaml.Node) error {
			if !slices.Contains(ringStuff, name.Value) {
				return reading.refuse(name, "%s is not something slayer "+
					"knows a ring is made of; it knows %v", name.Value,
					ringStuff)
			}
			var amount float64
			if err := reading.number(&amount, 1e-9, 1)(name,
				share); err != nil {
				return err
			}
			into[name.Value] = amount
			return nil
		})
	}
}

// sky reads where the planet stands in a ground set's sky, every part
// said: its azimuth, its altitude above the horizon, and its distance.
func (reading reader) sky(made *Planet) step {
	return func(_, value *yaml.Node) error {
		where := [3]float64{math.NaN(), math.NaN(), math.NaN()}
		err := reading.keys(value, map[string]step{
			"azimuth":  reading.number(&where[0], 0, 360),
			"altitude": reading.number(&where[1], 0, 90),
			"distance": reading.number(&where[2], 1, 1e15),
		}, nil)
		if err != nil {
			return err
		}
		for _, part := range where {
			if math.IsNaN(part) {
				return reading.refuse(value, "a planet's sky says its "+
					"azimuth, altitude and distance")
			}
		}
		made.sky = &where
		return nil
	}
}

// storms reads how the planet's bands are torn and mixed, every part of
// the flow said, and its spots.
func (reading reader) storms(value *yaml.Node) (*Storms, error) {
	made := &Storms{seed: math.NaN(), turbulence: math.NaN(),
		eddies: math.NaN(), stretch: math.NaN(), edge: math.NaN()}
	err := reading.keys(value, map[string]step{
		"seed":       reading.whole(&made.seed, 0, mostSeed),
		"turbulence": reading.number(&made.turbulence, 0, 1),
		"eddies":     reading.number(&made.eddies, 0.1, 1e4),
		"stretch":    reading.number(&made.stretch, 1, 1e3),
		"edge":       reading.number(&made.edge, 0, 90),
		"folds":      reading.whole(&made.folds, 0, 8),
		"jets":       reading.jets(made),
		"swarm":      reading.swarm(made),
		"marks": reading.block(map[string]step{
			"cap":      reading.text(&made.cap),
			"clearing": reading.text(&made.clearing)}),
		"spots": reading.spots(made),
		"churn": reading.block(map[string]step{
			"column":   reading.text(&made.churn),
			"strength": reading.number(&made.churning, 0, 1)}),
	}, nil)
	if err != nil {
		return nil, err
	}
	for _, part := range []float64{made.seed, made.turbulence,
		made.eddies, made.stretch, made.edge} {
		if math.IsNaN(part) {
			return nil, reading.refuse(value, "storms say their seed, "+
				"turbulence, eddies, stretch and edge")
		}
	}
	return made, nil
}

// jets reads the storms' jets, both parts said.
func (reading reader) jets(made *Storms) step {
	return func(_, value *yaml.Node) error {
		jets := [2]float64{math.NaN(), math.NaN()}
		err := reading.keys(value, map[string]step{
			"shear": reading.number(&jets[0], 0, 360),
			"every": reading.number(&jets[1], 0.1, 180),
		}, nil)
		if err != nil {
			return err
		}
		if math.IsNaN(jets[0]) || math.IsNaN(jets[1]) {
			return reading.refuse(value, "jets say their shear and every")
		}
		made.jets = &jets
		return nil
	}
}

// swarm reads the storms' swarm of spots, every part said.
func (reading reader) swarm(made *Storms) step {
	return func(_, value *yaml.Node) error {
		swarm := [5]float64{math.NaN(), math.NaN(), math.NaN(), math.NaN(),
			math.NaN()}
		err := reading.keys(value, map[string]step{
			"seed":     reading.whole(&swarm[0], 0, mostSeed),
			"count":    reading.whole(&swarm[1], 0, 2000),
			"smallest": reading.number(&swarm[2], 0.01, 30),
			"largest":  reading.number(&swarm[3], 0.01, 30),
			"slope":    reading.number(&swarm[4], 1.01, 10),
		}, nil)
		if err != nil {
			return err
		}
		for _, part := range swarm {
			if math.IsNaN(part) {
				return reading.refuse(value, "a swarm says its seed, count, "+
					"smallest, largest and slope")
			}
		}
		made.swarm = &swarm
		return nil
	}
}

// spots reads the storms' spots, each saying where it is, how wide and
// how far it turns.
func (reading reader) spots(made *Storms) step {
	return func(_, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode {
			return reading.refuse(value, "spots is a list")
		}
		for _, item := range value.Content {
			spot := Spot{}
			err := reading.keys(item, map[string]step{
				"latitude":  reading.number(&spot.latitude, -90, 90),
				"longitude": reading.number(&spot.longitude, -180, 180),
				"radius":    reading.number(&spot.radius, 0.1, 90),
				"turn":      reading.number(&spot.turn, -10, 10),
			}, nil)
			if err != nil {
				return err
			}
			made.spots = append(made.spots, spot)
		}
		return nil
	}
}
