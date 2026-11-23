package weathering

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// A mesh group is one design stood in many places: a stand of mangroves
// is a few trees, each stood a thousand times. Laid triangle by triangle
// in the world, the shelter would keep every copy of every leaf, and its
// cost would follow the count of trees and not the trees.
//
// So a group that stands upright is laid once, in its own space, as a
// design, and each place it stands is kept as a standing: the design, the
// place, and where its triangles come in the shelter's order. A look finds
// the standings over its cell and asks each design in its own space.
//
// A mesh that is not a group is shared the same way: a stand whose trees
// are made as hulls is one mesh for each design, stood at every tree. So
// a mesh held upright in more than one place is a design too. A mesh held
// once is laid in the world as before, since a design of its own would
// cost it more than its triangles do.
//
// Upright is what lets a design be asked in its own space: a look is
// straight up, and a place turned only about the vertical, moved and
// scaled the same every way, keeps straight up straight up. A group on
// its side is laid in the world as any other surface.

// uprightSlack is how far a place's up may lie from the world's up and
// still count as upright. picked: as Orientation keeps its axes, well
// above rounding and far below any turn a scene asks for.
const uprightSlack = 1e-9

// design is a group's triangles laid once in its own space, and the box
// they fill there.
type design struct {
	*Shelter
	low, high maths.Vec
}

// standing is a design standing in the world: where, how much larger
// than its own space, the material it is of, and the order of its first
// triangle among all the shelter keeps.
type standing struct {
	design *design
	place  maths.Subspace
	scale  float64
	name   string
	first  int
}

// holdPlaced holds a surface standing at a place: a group or a mesh that
// stands upright as one that may be stood as a design, and anything else
// to be laid.
func (shelter *Shelter) holdPlaced(of shape.Shape, place maths.Subspace,
	name string) {
	if !shared(of) || !upright(place) {
		shelter.hold(of, place.Out, name)
		return
	}
	if shelter.uses == nil {
		shelter.uses = map[shape.Shape]int{}
	}
	shelter.uses[of]++
	shelter.held = append(shelter.held, surface{shape: of, out: place.Out,
		name: name, design: true, place: place})
}

// shared is whether a surface is one that many places may share: a mesh
// or a group of them, each held by its pointer.
func shared(of shape.Shape) bool {
	switch of.(type) {
	case *mesh.Mesh, *shape.Mesh, *shape.MeshGroup:
		return true
	}
	return false
}

// stands is whether a surface held as a possible design is stood as one:
// a group always, as before, and a mesh held in more than one place.
func (shelter *Shelter) stands(held surface) bool {
	if !held.design {
		return false
	}
	_, group := held.shape.(*shape.MeshGroup)
	return group || shelter.uses[held.shape] > 1
}

// upright is whether a place keeps the world's up as its own: turned
// only about the vertical, and of a scale more than nothing.
func upright(place maths.Subspace) bool {
	up := place.Orientation.Out(maths.YAxis)
	scale := scaleOf(place)
	return math.Abs(up.X) <= uprightSlack && math.Abs(up.Y-1) <= uprightSlack &&
		math.Abs(up.Z) <= uprightSlack && scale > 0 && !math.IsInf(scale, 0)
}

// scaleOf is how many metres of the world a metre of a place's own
// space is; a scale of nothing is one, as a Subspace takes it.
func scaleOf(place maths.Subspace) float64 {
	if place.Scale == 0 {
		return 1
	}
	return place.Scale
}

// stand keeps a group or a mesh standing at an upright place, laying its
// design the first time it is stood. One with no triangle flat enough to
// stand over anything is not kept.
func (shelter *Shelter) stand(group shape.Shape, place maths.Subspace,
	name string) {
	made, known := shelter.designs[group]
	if !known {
		made = designOf(group, shelter.size, shelter.largeAbove)
		if shelter.designs == nil {
			shelter.designs = map[shape.Shape]*design{}
		}
		shelter.designs[group] = made
	}
	if made.next == 0 {
		return
	}
	shelter.standing = append(shelter.standing, standing{design: made,
		place: place, scale: scaleOf(place), name: name,
		first: shelter.next})
	shelter.next += made.next
}

// designOf is a group's or a mesh's design: its triangles in its own
// space, in the order it gives them, in cells as wide as the world's.
func designOf(group shape.Shape, size float64, largeAbove int) *design {
	made := &design{Shelter: NewShelter(size)}
	made.largeAbove = largeAbove
	triangles(group, func(point maths.Vec) maths.Vec { return point },
		func(first, second, third maths.Vec) {
			made.Add(first, second, third, "")
		})
	made.settle()
	if len(made.all) == 0 {
		return made
	}
	made.low, made.high = made.all[0].corners[0], made.all[0].corners[0]
	for _, each := range made.all {
		for _, corner := range each.corners {
			made.low = maths.V(min(made.low.X, corner.X),
				min(made.low.Y, corner.Y), min(made.low.Z, corner.Z))
			made.high = maths.V(max(made.high.X, corner.X),
				max(made.high.Y, corner.Y), max(made.high.Z, corner.Z))
		}
	}
	return made
}

// settleStanding puts each standing in every cell its design covers in
// the world, or on the list of large standings if that is more than the
// shelter's large footprint.
func (shelter *Shelter) settleStanding() {
	if len(shelter.standing) == 0 {
		return
	}
	shelter.standingCells = map[[2]int][]int32{}
	for index, one := range shelter.standing {
		at := int32(index)
		low, high := one.footprint(shelter)
		wide, deep := float64(high[0]-low[0]+1), float64(high[1]-low[1]+1)
		if wide*deep > float64(shelter.largeAbove) {
			shelter.standingLarge = append(shelter.standingLarge, at)
			continue
		}
		for x := low[0]; x <= high[0]; x++ {
			for z := low[1]; z <= high[1]; z++ {
				cell := [2]int{x, z}
				shelter.standingCells[cell] = append(
					shelter.standingCells[cell], at)
			}
		}
	}
}

// footprint is the lowest and highest cells of a shelter under a
// standing's design, by the corners of the design's box in the world.
func (one standing) footprint(shelter *Shelter) ([2]int, [2]int) {
	low, high := one.design.low, one.design.high
	west, north := math.Inf(1), math.Inf(1)
	east, south := math.Inf(-1), math.Inf(-1)
	for _, x := range [2]float64{low.X, high.X} {
		for _, y := range [2]float64{low.Y, high.Y} {
			for _, z := range [2]float64{low.Z, high.Z} {
				at := one.place.Out(maths.V(x, y, z))
				west, east = min(west, at.X), max(east, at.X)
				north, south = min(north, at.Z), max(south, at.Z)
			}
		}
	}
	return shelter.cell(west, north), shelter.cell(east, south)
}

// standingNear is the two lists of standings that may be over or under
// a point, each by its place in the shelter's standings: those in the
// point's cell, then the large ones.
func (shelter *Shelter) standingNear(from maths.Vec) [2][]int32 {
	return [2][]int32{shelter.standingCells[shelter.cell(from.X, from.Z)],
		shelter.standingLarge}
}
