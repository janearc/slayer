package set

import (
	"fmt"
	"math"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// the bounds of a cover's rules
const (
	// leastBlend and mostBlend bound the width a range's edge is softened
	// over, in the range's own units: a hundredth, so no edge is a step,
	// and a window's widest across.
	leastBlend = 0.01          // picked: a hundredth of a unit
	mostBlend  = longestRelief // a window's widest across, as relief's

)

// Cover is how the ground is coloured by rule, from where it stands and
// which way it leans: a list of layers, the first the ground under all the
// rest and each later one laid over those before it where its height and
// its slope hold, softened across a width.
//
// level is the height of the sea on the terrain's own datum, in metres,
// which the heights of every rule are above. A terrain that says its sea
// has one owner of the level, and the cover takes it from there; a terrain
// that does not says it in the cover, and levelSaid is whether the cover
// did.
type Cover struct {
	level     float64
	levelSaid bool
	layers    []CoverLayer
}

// CoverLayer is one surface of a cover: a name, the share of red, green
// and blue light it gives back in linear light and where that came from,
// and the rules that say where it lies.
type CoverLayer struct {
	name        string
	reflectance [3]float64
	source      string
	height      reach
	slope       reach
}

// reach is a stretch of a number with soft ends: from, until, and the
// width each end is softened over, centred on it. an end that is not said
// is infinity, and said is whether the stretch was said at all.
type reach struct {
	from, until, blend float64
	said               bool
}

// Cover is the rule that colours the terrain, as the height of the sea in
// metres on the terrain's datum and the surfaces from the ground up, and
// whether the set says one. A terrain that says none has one colour.
func (ground Terrain) Cover() (level float64, layers []CoverLayer,
	said bool) {
	if ground.cover == nil {
		return 0, nil, false
	}
	return ground.cover.level, slices.Clone(ground.cover.layers), true
}

// Name is what the surface is called.
func (layer CoverLayer) Name() string { return layer.name }

// Reflectance is the share of red, green and blue light the surface gives
// back, in linear light.
func (layer CoverLayer) Reflectance() [3]float64 { return layer.reflectance }

// Source is where the surface's reflectance came from, in the set's words.
func (layer CoverLayer) Source() string { return layer.source }

// Height is the heights, in metres above the sea, the surface lies at: the
// lowest and the highest, each infinity where the rule has no such end,
// the width each end is softened over, and whether the set says a height.
func (layer CoverLayer) Height() (from, until, blend float64, said bool) {
	return layer.height.from, layer.height.until, layer.height.blend,
		layer.height.said
}

// Slope is the slopes, rise over run, the surface lies at, said as Height
// is.
func (layer CoverLayer) Slope() (from, until, blend float64, said bool) {
	return layer.slope.from, layer.slope.until, layer.slope.blend,
		layer.slope.said
}

// cover is a step that reads the terrain's cover: the sea's level and the
// layers.
func (reading reader) cover(ground *Terrain) step {
	return func(_, value *yaml.Node) error {
		made := &Cover{}
		ground.cover = made
		err := reading.keys(value, map[string]step{
			"level":  reading.coverSea(made),
			"layers": reading.coverLayers(made),
		}, nil)
		if err == nil {
			err = reading.all(value, "a cover", "layers")
		}
		return err
	}
}

// coverSea is a step that reads the level a cover says the sea stands at,
// in metres on the terrain's datum, and notes that it was said.
func (reading reader) coverSea(made *Cover) step {
	read := reading.number(&made.level, -deepestSea, highestSea)
	return func(name, value *yaml.Node) error {
		made.levelSaid = true
		return read(name, value)
	}
}

// coverLevel settles the sea's level of a terrain's cover once the whole
// terrain is read. Where the terrain says its sea, that is the level: a
// cover that says another is refused, with both numbers. Where it does
// not, the cover must say it.
func (reading reader) coverLevel(at *yaml.Node, ground *Terrain) error {
	made := ground.cover
	switch {
	case made == nil:
		return nil
	case ground.watered && made.levelSaid && made.level != ground.sea:
		return reading.refuse(at, "the cover says the sea stands at %g m "+
			"and the terrain's sea says %g m: the terrain's sea is the "+
			"level, so leave the cover's level out or make it the same",
			made.level, ground.sea)
	case ground.watered:
		made.level = ground.sea
	case !made.levelSaid:
		return reading.refuse(at, "a cover on a terrain that says no sea "+
			"says its level, the sea's height in metres on the datum")
	}
	return nil
}

// foundCover refuses an ocean whose middle stands at a height other than
// the level a terrain's cover is laid by, with both numbers: the sand's
// edge and the water's edge are one height.
func (reading reader) foundCover() error {
	ground, world := reading.set.terrain, reading.set.world
	if ground == nil || ground.cover == nil || world == nil ||
		world.ocean == nil {
		return nil
	}
	if up := world.ocean.middle[2]; up != ground.cover.level {
		return fmt.Errorf("%s: the ocean's middle stands at %g m and the "+
			"terrain's cover is laid by a sea at %g m: they are one "+
			"level, so make them the same", reading.file, up,
			ground.cover.level)
	}
	return nil
}

// coverLayers is a step that reads a cover's layers, the first under the
// rest.
func (reading reader) coverLayers(made *Cover) step {
	return func(_, list *yaml.Node) error {
		if list.Kind != yaml.SequenceNode || len(list.Content) == 0 {
			return reading.refuse(list, "layers is a list of surfaces, "+
				"the ground first")
		}
		for index, item := range list.Content {
			layer, err := reading.coverLayer(item, index == 0)
			if err != nil {
				return err
			}
			made.layers = append(made.layers, layer)
		}
		return nil
	}
}

// coverLayer reads one surface. The first, the ground under the others, is
// everywhere and says no height or slope; each later one says at least one
// of them, or it would hide all that is under it.
func (reading reader) coverLayer(item *yaml.Node, first bool) (CoverLayer,
	error) {
	made := CoverLayer{height: everywhere(), slope: everywhere()}
	err := reading.keys(item, map[string]step{
		"name":        reading.text(&made.name),
		"reflectance": reading.three(&made.reflectance, noLight, allLight),
		"source":      reading.text(&made.source),
		"height":      reading.reach(&made.height, -deepestSea, highestSea),
		"slope":       reading.reach(&made.slope, 0, steepest),
	}, nil)
	if err == nil {
		err = reading.all(item, "a surface", "name", "reflectance", "source")
	}
	if err != nil {
		return made, err
	}
	says := made.height.said || made.slope.said
	switch {
	case first && says:
		return made, reading.refuse(item, "the first surface is the "+
			"ground under the others and says no height or slope")
	case !first && !says:
		return made, reading.refuse(item, "a surface after the first "+
			"says a height or a slope, or it hides all under it")
	}
	return made, nil
}

// everywhere is the stretch a surface has where its set says none: it
// holds every number.
func everywhere() reach {
	return reach{from: math.Inf(-1), until: math.Inf(1)}
}

// reach is a step that reads a stretch of a number: its from, its until or
// both, and the blend its ends are softened over, which it always says.
func (reading reader) reach(into *reach, least, most float64) step {
	return func(name, value *yaml.Node) error {
		wrote := map[string]bool{}
		from := reading.number(&into.from, least, most)
		until := reading.number(&into.until, least, most)
		err := reading.keys(value, map[string]step{
			"from":  reading.noted(wrote, from),
			"until": reading.noted(wrote, until),
			"blend": reading.number(&into.blend, leastBlend, mostBlend),
		}, nil)
		if err == nil {
			err = reading.all(value, "a "+name.Value+" rule", "blend")
		}
		if err != nil {
			return err
		}
		if !wrote["from"] && !wrote["until"] {
			return reading.refuse(value, "a %s rule says from, until or "+
				"both", name.Value)
		}
		if !(into.from < into.until) {
			return reading.refuse(value, "a %s rule's from is not under "+
				"its until", name.Value)
		}
		into.said = true
		return nil
	}
}
