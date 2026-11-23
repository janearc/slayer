package stage

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// piece is a solid of a channel and the name of the material it wears.
type piece struct {
	shape    *mesh.Mesh
	material string
}

// channels builds each channel of the world, solid by solid: the bed, the
// walls with their copings, the ground on either side and the walls at
// the ends. No water is built; the world's ocean is the water.
func channels(world *thing.World, said set.World,
	worn map[string]*thing.Material, shift maths.Vec) error {
	at := maths.Local()
	at.Origin = moved(maths.V(0, 0, 0), shift)
	still := thing.Still(at)
	for _, each := range said.Channels() {
		pieces, err := cut(each)
		if err != nil {
			return fmt.Errorf("channel %s: %w", each.Name(), err)
		}
		for _, made := range pieces {
			world.Add(thing.Thing{Shape: made.shape,
				Material: worn[made.material], Motion: still})
		}
	}
	return nil
}

// cut is every solid of a channel: those along its line, then those that
// close its ends.
func cut(each set.Channel) ([]piece, error) {
	line := each.Line()
	longer, err := stretched(line, each.WallThickness())
	if err != nil {
		return nil, err
	}
	var pieces []piece
	for _, across := range lengthways(each) {
		way := line
		if across.stretched {
			way = longer
		}
		shape, err := sweep(way, across.section)
		if err != nil {
			return nil, err
		}
		pieces = append(pieces, piece{shape, across.material})
	}
	closing, err := closed(each)
	if err != nil {
		return nil, err
	}
	return append(pieces, closing...), nil
}

// slab is a section of a channel, the material it is made of, and whether
// it runs the length of the end walls too, past each end of the line.
type slab struct {
	material  string
	section   section
	stretched bool
}

// stretched is a line made longer at each end by a distance, along the way
// of its first and last segments.
func stretched(line []plan, distance float64) ([]plan, error) {
	first, last, err := endways(line)
	if err != nil {
		return nil, err
	}
	longer := slices.Clone(line)
	longer[0] = backed(line[0], first, distance)
	longer[len(line)-1] = backed(line[len(line)-1], last, -distance)
	return longer, nil
}

// endways are the unit ways of the first and the last segments of a line.
func endways(line []plan) (first, last plan, err error) {
	first, err = direction(line[0], line[1])
	if err != nil {
		return plan{}, plan{}, err
	}
	last, err = direction(line[len(line)-2], line[len(line)-1])
	return first, last, err
}

// lengthways are the sections of a channel along its line, each side of
// the water: the bed under it, a wall, its coping, and the ground beyond.
// Distances are from the line, the left of its way above nought. The bed
// and the walls stand on a floor one wall's thickness under the bed. The
// ground runs on past the line's ends as far as the end walls are thick.
func lengthways(each set.Channel) []slab {
	half := each.Width() / 2
	outer := half + each.WallThickness()
	reach := outer + each.Reach()
	foot := each.Top() - each.Depth()
	floor := foot - each.WallThickness()
	under := each.Top() - each.CopingHeight()
	jut := half - each.CopingOverhang()
	return []slab{
		{each.Bed(), section{-half, half, floor, foot}, false},
		{each.Wall(), section{half, outer, floor, under}, false},
		{each.Wall(), section{-outer, -half, floor, under}, false},
		{each.Coping(), section{jut, outer, under, each.Top()}, false},
		{each.Coping(), section{-outer, -jut, under, each.Top()}, false},
		{each.Ground(), section{outer, reach, floor, each.Top()}, true},
		{each.Ground(), section{-reach, -outer, floor, each.Top()}, true},
	}
}

// closed are the solids that close the two ends of a channel: across its
// walls, one wall thick, a wall of the wall's material and over it a
// coping.
func closed(each set.Channel) ([]piece, error) {
	line := each.Line()
	last := len(line) - 1
	first, finish, err := endways(line)
	if err != nil {
		return nil, err
	}
	thick := each.WallThickness()
	stubs := [][]plan{
		{backed(line[0], first, thick), line[0]},
		{line[last], backed(line[last], finish, -thick)},
	}
	var pieces []piece
	for _, stub := range stubs {
		made, err := stubbed(each, stub)
		if err != nil {
			return nil, err
		}
		pieces = append(pieces, made...)
	}
	return pieces, nil
}

// backed is a point moved back along a way by a distance, or forward where
// the distance is below nought.
func backed(point, way plan, distance float64) plan {
	return plan{point[0] - way[0]*distance, point[1] - way[1]*distance}
}

// stubbed is the wall and the coping across a short line that crosses the
// channel's end: as wide as the channel with its walls.
func stubbed(each set.Channel, stub []plan) ([]piece, error) {
	outer := each.Width()/2 + each.WallThickness()
	foot := each.Top() - each.Depth() - each.WallThickness()
	under := each.Top() - each.CopingHeight()
	wall, err := sweep(stub, section{-outer, outer, foot, under})
	if err != nil {
		return nil, err
	}
	coping, err := sweep(stub, section{-outer, outer, under, each.Top()})
	if err != nil {
		return nil, err
	}
	return []piece{{wall, each.Wall()}, {coping, each.Coping()}}, nil
}
