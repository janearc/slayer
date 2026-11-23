package set

import (
	"fmt"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// mostLinePoints is the most points a channel's line may have. It is the
// reader's own limit, there so that a mistyped list is refused rather than
// built into a mesh of thousands of faces.
const mostLinePoints = 64

// A Channel is the bed of still water cut into flat ground and walled: a
// line across the ground, a width between the walls, the depth of the bed
// under the top of the ground, and the ground on each side for a reach.
//
// Along each side stands a wall of a thickness, capped by a coping that
// juts over the water, and each end is closed by a wall of its own.
//
// The channel holds no water: the water is the world's ocean, laid at the
// height the set chooses between the bed and the top. It is the shape a
// canal, a ditch, a quay or a moat has, whatever the set calls it.
type Channel struct {
	name                         string
	line                         [][2]float64
	width, depth, top, reach     float64
	ground, bed                  string
	wall, coping                 string
	wallThickness                float64
	copingHeight, copingOverhang float64
}

// Name is the channel's name.
func (made Channel) Name() string { return made.name }

// Line is the channel's middle, as points of east and north in metres,
// from its first end to its last.
func (made Channel) Line() [][2]float64 { return slices.Clone(made.line) }

// Width is the clear width between the two walls, in metres.
func (made Channel) Width() float64 { return made.width }

// Depth is how far the bed lies under the top of the ground, in metres.
func (made Channel) Depth() float64 { return made.depth }

// Top is the height of the ground, and of the coping's upper face, in
// metres.
func (made Channel) Top() float64 { return made.top }

// Reach is how far the ground runs from each wall, in metres.
func (made Channel) Reach() float64 { return made.reach }

// Ground is the name of the material of the ground beside the channel.
func (made Channel) Ground() string { return made.ground }

// Bed is the name of the material of the channel's bed.
func (made Channel) Bed() string { return made.bed }

// Wall is the name of the material of the walls.
func (made Channel) Wall() string { return made.wall }

// WallThickness is how thick each wall is, across its length, in metres.
func (made Channel) WallThickness() float64 { return made.wallThickness }

// Coping is the name of the material of the coping on the walls.
func (made Channel) Coping() string { return made.coping }

// CopingHeight is how thick the coping is, in metres.
func (made Channel) CopingHeight() float64 { return made.copingHeight }

// CopingOverhang is how far the coping juts past the wall over the water,
// in metres.
func (made Channel) CopingOverhang() float64 { return made.copingOverhang }

// Channels is every channel, in the order written.
func (made World) Channels() []Channel { return slices.Clone(made.channels) }

// channels reads each channel, by its name.
func (reading reader) channels(_, value *yaml.Node) error {
	return reading.each(value, reading.channel)
}

// channel reads one channel and holds it, where it is whole and fits.
func (reading reader) channel(name, block *yaml.Node) error {
	made := Channel{name: name.Value}
	var source string
	err := reading.keys(block, reading.channelSteps(&made, &source), nil)
	if err == nil {
		err = reading.all(block, "a channel", "line", "width", "depth",
			"top", "reach", "ground", "bed", "wall", "coping", "source")
	}
	if err == nil {
		err = reading.all(valueNode(block, "wall"), "a channel's wall",
			"material", "thickness")
	}
	if err == nil {
		err = reading.all(valueNode(block, "coping"), "a channel's coping",
			"material", "height", "overhang")
	}
	if err == nil {
		err = reading.fitted(block, made)
	}
	reading.set.world.channels = append(reading.set.world.channels, made)
	return err
}

// channelSteps are the keys a channel takes, each reading into made.
func (reading reader) channelSteps(made *Channel,
	source *string) map[string]step {
	wall := map[string]step{
		"material": reading.text(&made.wall),
		"thickness": reading.quantity(&made.wallThickness, "m",
			leastLength, farthest),
	}
	coping := map[string]step{
		"material": reading.text(&made.coping),
		"height": reading.quantity(&made.copingHeight, "m",
			leastLength, farthest),
		"overhang": reading.quantity(&made.copingOverhang, "m", 0, farthest),
	}
	return map[string]step{
		"line":   lineReading{reading, &made.line}.read,
		"width":  reading.quantity(&made.width, "m", leastLength, farthest),
		"depth":  reading.quantity(&made.depth, "m", leastLength, farthest),
		"top":    reading.quantity(&made.top, "m", -farthest, farthest),
		"reach":  reading.quantity(&made.reach, "m", leastLength, farthest),
		"ground": reading.text(&made.ground),
		"bed":    reading.text(&made.bed),
		"wall":   reading.block(wall),
		"coping": reading.block(coping),
		"source": reading.source(source),
	}
}

// leastLength is the least a width, depth, thickness or reach may be, in
// metres: a millimetre, under which a face has no area worth the name.
const leastLength = 0.001

// A lineReading reads a channel's line into its target.
type lineReading struct {
	reading reader
	target  *[][2]float64
}

// read is the step that reads a list of [east, north] pairs, from two to
// mostLinePoints of them.
func (lined lineReading) read(name, value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode || len(value.Content) < 2 ||
		len(value.Content) > mostLinePoints {
		return lined.reading.refuse(value, "%s takes from two to %d "+
			"points, as [[east, north], ...]", name.Value, mostLinePoints)
	}
	points := make([][2]float64, len(value.Content))
	for index, each := range value.Content {
		if each.Kind != yaml.SequenceNode || len(each.Content) != 2 {
			return lined.reading.refuse(each, "%s takes each point as two "+
				"numbers, [east, north]", name.Value)
		}
		for axis, number := range each.Content {
			reads := lined.reading.number(&points[index][axis], -farthest,
				farthest)
			if err := reads(name, number); err != nil {
				return err
			}
		}
	}
	*lined.target = points
	return nil
}

// fitted refuses a channel whose parts cannot stand together: a line that
// stays at one point, a coping as thick as the channel is deep, or a
// coping that juts past the middle of the water.
func (reading reader) fitted(block *yaml.Node, made Channel) error {
	for index := 1; index < len(made.line); index++ {
		if made.line[index] == made.line[index-1] {
			return reading.refuse(valueNode(block, "line"),
				"the line stays at point %d for its next, and a channel "+
					"has no length there", index)
		}
	}
	if made.copingHeight >= made.depth {
		return reading.refuse(valueNode(block, "coping"),
			"the coping is %g m thick and the channel %g m deep, so the "+
				"wall has no height", made.copingHeight, made.depth)
	}
	if made.copingOverhang >= made.width/2 {
		return reading.refuse(valueNode(block, "coping"),
			"the coping juts %g m over water %g m wide, past its middle",
			made.copingOverhang, made.width)
	}
	return nil
}

// foundChannels refuses a channel that wears a material the world does
// not hold.
func (reading reader) foundChannels(world *World) error {
	for _, each := range world.channels {
		for _, wanted := range []string{each.ground, each.bed, each.wall,
			each.coping} {
			if !world.holds(wanted) {
				return fmt.Errorf("%s: channel %s is made of %q, and the "+
					"world holds no material by that name", reading.file,
					each.name, wanted)
			}
		}
	}
	return nil
}

// holds is whether the world has a material of a name.
func (made *World) holds(name string) bool {
	for _, each := range made.materials {
		if each.name == name {
			return true
		}
	}
	return false
}
