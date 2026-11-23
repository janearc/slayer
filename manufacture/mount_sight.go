package manufacture

import (
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// A mount is a part a thing carries at a place on it, such as a gargoyle
// under a tower's ledge. When a picture is asked to pay only for what it
// can show, a mount is made as finely as its span in pixels allows: not at
// all under a pixel, where the wall it stands on stands for it; as a plain
// block of its box a little above that, so the silhouette is kept; and
// whole beyond.

// The pixel spans a mount is made by.
const (
	// mountLeastPixels is the span, in pixels, under which a mount is not
	// made: it cannot colour a pixel by itself.
	mountLeastPixels = 1.0
	// mountWholePixels is the span, in pixels, from which a mount is made
	// whole. Picked: a creature needs a few pixels across to read as one,
	// and under four a block of its box shows the same. Not measured.
	mountWholePixels = 4.0
)

// mountSampleSeed is the seed a kind of mount is made from, once, to
// measure its box. Picked: any seed will do, since a seed strays from a
// kind's size by no more than sizeSwing.
const mountSampleSeed = 1

// mountSight is what a mount is judged by: the eye, and where the thing
// that carries the mounts stands in the eye's world.
type mountSight struct {
	eye   strategies.Eye
	place maths.Subspace
}

// span is the box of a mount of a kind, in the mount's own frame, and how
// many pixels across the film it spans stood at its place on the thing
// that carries it.
func (sight mountSight) span(row kind, place maths.Subspace,
	from seed) (maths.Box, float64, error) {
	box, err := mountExtent(row, from)
	if err != nil {
		return maths.Box{}, 0, err
	}
	world := transformed(box, place.Within(sight.place))
	return box, sight.eye.Cover(world).Pixels, nil
}

// mountBoxes is the box of each gargoyle kind, in its own frame, by the
// name of the kind, measured once.
var mountBoxes sync.Map

// mountExtent is the box of a kind made as a mount, from a seed. A
// gargoyle's box is found by making one of its kind once and keeping the
// box, grown by the share its seed may stray from the kind's size, since
// making one for every mount is the cost this is here to spare. Any other
// kind says its extent as it does anywhere.
func mountExtent(row kind, from seed) (maths.Box, error) {
	if _, gargoyle := row.how.(gargoyleRow); !gargoyle {
		return extentOf(row, from)
	}
	if known, found := mountBoxes.Load(row.name); found {
		return known.(maths.Box), nil
	}
	box, err := greedyExtent(row.how, sow(mountSampleSeed))
	if err != nil {
		return maths.Box{}, err
	}
	by := box.Max.Sub(box.Min).Scale(sizeSwing)
	grown := maths.Box{Min: box.Min.Sub(by), Max: box.Max.Add(by)}
	kept, _ := mountBoxes.LoadOrStore(row.name, grown)
	return kept.(maths.Box), nil
}

// blockOfBox is a plain block of a box, in the stone of the cornice row,
// stood at a mount's place: what a mount is made as when it spans too few
// pixels to be worth its shape.
func (row gargoyleCorniceRow) blockOfBox(box maths.Box, place maths.Subspace,
	from seed) (good, error) {
	middle := box.Min.Add(box.Max).Scale(0.5)
	size := box.Max.Sub(box.Min)
	made, err := boxes([]block{{at: maths.V(middle.X, 0, middle.Z),
		low: box.Min.Y, wide: size.X, tall: size.Y, deep: size.Z}})
	if err != nil {
		return good{}, err
	}
	made = made.in(looks{body: row.stone.look("gargoyle "+row.stone.name,
		from.number, 1)})
	return made.placed(place)
}
