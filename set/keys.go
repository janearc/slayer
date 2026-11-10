package set

import (
	"fmt"
	"maps"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// Set is a set file read: what the scene wants, every value checked,
// and the keys it holds that nothing renders yet.
type Set struct {
	fronts               []FireFront
	beds                 []FuelBed
	name                 string
	latitude             float64 // degrees north
	longitude            float64 // degrees east
	clock                int     // minutes after local midnight
	cameras              []camera
	bounds               [3]float64 // metres a side; none means the bodies' scale
	assets               []asset
	unrendered           []string
	light                string
	terrain              *Terrain
	film                 Film
	things               []Thing
	stands               []Stand
	ecosystems           []Ecosystem
	texts                []Text
	moon                 *Moon
	sun                  *Sun
	planet               *Planet
	hud                  *Hud
	world                *World
	recipes              map[string]Recipe
	weather              map[string]Mist
	laserPointings       map[string]laserDirection
	lasers               []Laser
	atmosphere           string
	atmosphereAlbedo     float64
	atmosphereLayers     []AtmosphereLayer
	renderer, complexity string
	volumeLighting       string
	seed                 float64
	hasSeed              bool
	windSpeed, windFrom  float64
}

// Light is how the set is lit: ambient, the moon, or the sun.
func (made Set) Light() string { return made.light }

// Moonlit is the full moon in the set's sky, and whether it says where.
func (made Set) Moonlit() (Moon, bool) {
	if made.moon == nil {
		return Moon{}, false
	}
	return *made.moon, true
}

// Sunlit is the sun in the set's sky, and whether it says where.
func (made Set) Sunlit() (Sun, bool) {
	if made.sun == nil {
		return Sun{}, false
	}
	return *made.sun, true
}

// Ground is the set's terrain, and whether it has one.
func (made Set) Ground() (Terrain, bool) {
	if made.terrain == nil {
		return Terrain{}, false
	}
	return *made.terrain, true
}

// Film is how the set's pictures are taken, and whether it says.
func (made Set) Film() (Film, bool) { return made.film, made.film.width > 0 }

// Things is every made thing on the set, in the order written.
func (made Set) Things() []Thing { return slices.Clone(made.things) }

// Stands is every stand on the set, in the order written.
func (made Set) Stands() []Stand { return slices.Clone(made.stands) }

// camera is one camera the set file asks for: where it stands, off
// which asset, and what it looks at, tipped up how far; or the place it
// stands at and the place it looks at, in the set file's metres.
type camera struct {
	name, title, eye string
	pointTo          string
	alpha            float64 // degrees up from level
	from             string
	standoff         [3]float64 // metres from the asset it stands off
	at, lookAt       *[3]float64
}

// asset is one thing on the set: its kind, as its key says, and its
// name, which cameras use to find it.
type asset struct {
	kind, name string
}

// step reads one key's value into the set.
type step func(name, value *yaml.Node) error

// reader is a set file being read into a set.
type reader struct {
	file string
	set  *Set
}

// Read is the set a set file asks for. It refuses a file that is not YAML, a
// key nobody knows, a value its key does not take, and a camera that points at
// or stands off from an asset the set does not have. Keys it knows and cannot
// render yet are kept as Unrendered, and the read goes on.
func Read(file string, source []byte) (Set, error) {
	top, err := parse(file, source)
	if err != nil {
		return Set{}, err
	}
	if err = checkedEcosystem(file, source, top); err != nil {
		return Set{}, err
	}
	made := Set{complexity: "quick", recipes: map[string]Recipe{}, weather: map[string]Mist{}}
	reading := reader{file: file, set: &made}
	err = reading.keys(top, map[string]step{"scene": reading.scene,
		"$schema": reading.only(EcosystemSchema)}, nil)
	if err == nil && made.volumeLighting == "cached" && made.renderer != "gpu-native" {
		return Set{}, fmt.Errorf("%s: cached volume lighting requires gpu-native", file)
	}
	if err == nil {
		err = reading.resolvedRecipes()
	}
	if err == nil {
		err = reading.found()
	}
	if err == nil {
		err = reading.foundEcosystems()
	}
	if err == nil {
		err = reading.foundFronts()
	}
	if err == nil {
		err = reading.foundBeds()
	}
	if err == nil {
		err = reading.foundLasers()
	}
	if err == nil {
		err = reading.foundAtmosphere()
	}
	if err == nil {
		err = reading.pinned()
	}
	return made, err
}

// Unrendered is every key the set file holds that nothing renders yet,
// each with where it was written.
func (made Set) Unrendered() []string {
	return slices.Clone(made.unrendered)
}

// keys reads a block of keys: each one with a step is read by it, each
// one known for later is kept as unrendered, and any other is refused.
func (reading reader) keys(block *yaml.Node, steps map[string]step,
	later []string) error {
	if block.Kind != yaml.MappingNode {
		return reading.refuse(block, "this takes keys under it")
	}
	for index := 0; index+1 < len(block.Content); index += 2 {
		name, value := block.Content[index], block.Content[index+1]
		switch read, found := steps[name.Value]; {
		case found:
			if err := read(name, value); err != nil {
				return err
			}
		case slices.Contains(later, name.Value):
			reading.set.unrendered = append(reading.set.unrendered,
				fmt.Sprintf("%s:%d:%d %s", reading.file, name.Line,
					name.Column, name.Value))
		default:
			return reading.refuse(name, "%s is not a key here; this "+
				"block takes %s", name.Value, known(steps, later))
		}
	}
	return nil
}

// known is every key a block takes, for a refusal to list.
func known(steps map[string]step, later []string) []string {
	names := append(slices.Collect(maps.Keys(steps)), later...)
	slices.Sort(names)
	return names
}

// refuse is a refusal at the place a node was written.
func (reading reader) refuse(at *yaml.Node, said string,
	values ...any) error {
	return fmt.Errorf("%s:%d:%d: %s", reading.file, at.Line, at.Column,
		fmt.Sprintf(said, values...))
}

// named is whether the set has an asset, a thing, a text or a planet of
// a name.
func (made *Set) named(name string) bool {
	if made.planet != nil && made.planet.name == name {
		return true
	}
	return slices.ContainsFunc(made.assets,
		func(each asset) bool { return each.name == name }) ||
		slices.ContainsFunc(made.things,
			func(each Thing) bool { return each.name == name }) ||
		slices.ContainsFunc(made.texts,
			func(each Text) bool { return each.name == name })
}

// found refuses a camera that names an asset the set does not have, a
// lens focused on one, a stand that grows within sight of one, and a
// box of a material the world does not hold.
func (reading reader) found() error {
	if world := reading.set.world; world != nil {
		for _, each := range world.boxes {
			if !slices.ContainsFunc(world.materials,
				func(held Material) bool {
					return held.name == each.material
				}) {
				return fmt.Errorf("%s: box %s is made of %q, and the "+
					"world holds no material by that name",
					reading.file, each.name, each.material)
			}
		}
		if err := reading.carried(world); err != nil {
			return err
		}
	}
	for _, each := range reading.set.stands {
		if of, _, _, _, _, has := each.Within(); has &&
			!reading.set.named(of) {
			return fmt.Errorf("%s: stand %s grows within sight of %q, and "+
				"the set has no asset by that name", reading.file,
				each.name, of)
		}
	}
	if rig := reading.set.film.rig; rig != nil &&
		!reading.set.named(rig.focus) {
		return fmt.Errorf("%s: the lens is focused on %q, and the set "+
			"has no asset by that name", reading.file, rig.focus)
	}
	for _, each := range reading.set.cameras {
		for _, wanted := range []string{each.pointTo, each.from} {
			if wanted != "" && !reading.set.named(wanted) {
				return fmt.Errorf("%s: camera %s wants %q, and the set "+
					"has no asset by that name", reading.file, each.name,
					wanted)
			}
		}
	}
	return nil
}
