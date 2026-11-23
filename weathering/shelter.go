package weathering

import (
	"math"
	"strings"
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// flatEnough is the least that a triangle's normal must lie along up for
// the triangle to count as something that stands over a place: a floor,
// a roof, the underside of a sill. picked: a slope steeper than sixty
// degrees from the wall is a roof or a ledge, and not a wall.
const flatEnough = 0.5

// rimSlack is how far outside a triangle, in metres, a point may fall and
// still be under it, so that a point on the edge of a sill is under it.
const rimSlack = 1e-6

// largeFootprint is the most cells a triangle's footprint may cover and
// still be kept in them. A wider triangle, such as a ground of kilometres,
// would cost cells in proportion to its area, so it goes on a short list
// that every look tests directly. picked: a hundred metres across at
// two metres a cell; a building's faces are far under it, and a world has
// only a handful of faces over it, so the list stays short.
const largeFootprint = 2500

// overhead is a triangle that may stand over a place, in world space.
type overhead struct {
	corners [3]maths.Vec
	name    string
	// order is the triangle's place among every triangle the shelter
	// keeps, laid or standing, which settles a tie between two
	order int
}

// Shelter is every near-flat triangle of a world, kept by where it lies
// on the ground, so that what stands over a point is found by looking
// straight up. Each triangle is kept once, in the order it came.
//
// A cell holds the places in that list of the triangles over it, and
// triangles too wide for cells have their places on a list of their own,
// so a triangle over many cells costs each of them four bytes and not a
// copy. A place in the list is also the order a triangle came in, which
// settles a tie between two the same way whichever way each is kept. It is
// built once, from the world, and only read after.
//
// The world's surfaces are held, not laid, until the first look: only
// the CPU tracer looks, through Weathered.Facing, so a draw on the GPU
// never pays for the triangles. Looks may come from many goroutines at
// once; the first lays every held surface, and the rest wait for it.
//
// The cells are one flat list of places, each cell's run of it in the
// order the triangles came, found by the cell's slot: cells gives the
// slot, and a slot's run is entries from offsets[slot] to
// offsets[slot+1]. They are worked once, at the first look, after which
// nothing more may be added.
type Shelter struct {
	size       float64
	all        []overhead
	cells      map[[2]int]int32
	offsets    []int32
	entries    []int32
	large      []int32
	largeAbove int
	held       []surface
	laid       sync.Once
	looked     bool
	// next is the order the next triangle kept will have
	next int
	// the groups and shared meshes that stand upright, each laid once in
	// its own space, and how many places hold each; see instanced.go
	designs       map[shape.Shape]*design
	uses          map[shape.Shape]int
	standing      []standing
	standingCells map[[2]int][]int32
	standingLarge []int32
}

// surface is a thing of the world as Apply found it, held until the
// shelter is first looked in: its shape, the way out of its own space
// into the world, and its material's name. A group or a mesh that stands
// upright is held with its place, as one that may be stood and not laid.
type surface struct {
	shape  shape.Shape
	out    func(maths.Vec) maths.Vec
	name   string
	design bool
	place  maths.Subspace
}

// hold keeps a surface to be laid in the shelter at the first look. What
// is held is the surface as it is now, so a thing added to the world
// after, such as a board over an empty window, is not in the shelter.
func (shelter *Shelter) hold(of shape.Shape, out func(maths.Vec) maths.Vec,
	name string) {
	shelter.held = append(shelter.held, surface{shape: of, out: out,
		name: name})
}

// ready lays every held surface's triangles in the shelter, in the order
// they were held, and works the cells, the first time it is called, and
// lets the held go. Every later call, from any goroutine, waits for that
// one and returns.
func (shelter *Shelter) ready() {
	shelter.laid.Do(func() {
		for _, each := range shelter.held {
			if shelter.stands(each) {
				shelter.stand(each.shape, each.place, each.name)
				continue
			}
			triangles(each.shape, each.out, func(first, second,
				third maths.Vec) {
				shelter.Add(first, second, third, each.name)
			})
		}
		shelter.held, shelter.uses = nil, nil
		shelter.settle()
	})
}

// settle works the cells in two passes over the triangles kept in them:
// the first counts each cell's places and gives each cell its slot and
// its run, the second writes each triangle's place into the run of every
// cell it covers, in the order the triangles came.
func (shelter *Shelter) settle() {
	shelter.cells = map[[2]int]int32{}
	shelter.footprints(func(cell [2]int, _ int32) { shelter.cells[cell]++ })
	shelter.offsets = make([]int32, len(shelter.cells)+1)
	slot := int32(0)
	for cell, count := range shelter.cells {
		shelter.cells[cell] = slot
		shelter.offsets[slot+1] = count
		slot++
	}
	// each run's start is put one slot on, and moved along as it is
	// written, so that when every place is in, it is the run's end
	start := int32(0)
	for next := range shelter.offsets[1:] {
		count := shelter.offsets[next+1]
		shelter.offsets[next+1] = start
		start += count
	}
	shelter.entries = make([]int32, start)
	shelter.footprints(func(cell [2]int, at int32) {
		end := &shelter.offsets[shelter.cells[cell]+1]
		shelter.entries[*end] = at
		*end++
	})
	shelter.settleStanding()
	shelter.looked = true
}

// footprints calls visit with every cell each triangle not on the large
// list covers, and its place, triangle by triangle in the order they
// came, and for each across then along, as Add once filled the cells.
func (shelter *Shelter) footprints(visit func(cell [2]int, at int32)) {
	large := shelter.large
	for index, each := range shelter.all {
		at := int32(index)
		if len(large) > 0 && large[0] == at {
			large = large[1:]
			continue
		}
		low, high := shelter.footprint(each.corners)
		for x := low[0]; x <= high[0]; x++ {
			for z := low[1]; z <= high[1]; z++ {
				visit([2]int{x, z}, at)
			}
		}
	}
}

// footprint is the lowest and highest cells under a triangle's corners.
func (shelter *Shelter) footprint(corners [3]maths.Vec) ([2]int, [2]int) {
	first, second, third := corners[0], corners[1], corners[2]
	low := shelter.cell(min(first.X, second.X, third.X),
		min(first.Z, second.Z, third.Z))
	high := shelter.cell(max(first.X, second.X, third.X),
		max(first.Z, second.Z, third.Z))
	return low, high
}

// NewShelter is an empty shelter whose cells are size metres wide.
func NewShelter(size float64) *Shelter {
	return &Shelter{size: size, largeAbove: largeFootprint}
}

// cell is the cell a point's place on the ground is in.
func (shelter *Shelter) cell(x, z float64) [2]int {
	return [2]int{int(math.Floor(x / shelter.size)),
		int(math.Floor(z / shelter.size))}
}

// Add keeps a triangle, of the surface of a material, if it is flat
// enough to stand over anything, for every cell its footprint touches, or
// on the list of large triangles if that is more than largeFootprint
// cells, so that its cost does not grow with its area. The cells are
// worked at the first look, so a triangle added after one would never be
// found: that is a mistake of the caller's, and it panics.
func (shelter *Shelter) Add(first, second, third maths.Vec, name string) {
	if shelter.looked {
		panic("weathering: a triangle added to a shelter already looked in")
	}
	across := second.Sub(first).Cross(third.Sub(first))
	if math.Abs(across.Y) < flatEnough*across.Len() {
		return
	}
	corners := [3]maths.Vec{first, second, third}
	low, high := shelter.footprint(corners)
	at := int32(len(shelter.all))
	shelter.all = append(shelter.all, overhead{corners: corners, name: name,
		order: shelter.next})
	shelter.next++
	wide, deep := float64(high[0]-low[0]+1), float64(high[1]-low[1]+1)
	if wide*deep > float64(shelter.largeAbove) {
		shelter.large = append(shelter.large, at)
	}
}

// kept is the two lists of triangles that may be over a point, each by
// its place in the shelter's list: those in the point's cell, then the
// large ones. A triangle is in the cell of every place it covers, or on
// the large list, so none is missed. It is an array, and not a slice of
// lists, so that a look allocates nothing.
func (shelter *Shelter) kept(from maths.Vec) [2][]int32 {
	slot, has := shelter.cells[shelter.cell(from.X, from.Z)]
	if !has {
		return [2][]int32{nil, shelter.large}
	}
	run := shelter.entries[shelter.offsets[slot]:shelter.offsets[slot+1]]
	return [2][]int32{run, shelter.large}
}

// nearer is whether a triangle at a distance beats the best so far: it is
// nearer, or as near and came in later, as when each was kept in order.
func nearer(distance, best float64, found bool, order, latest int) bool {
	return distance < best || distance == best && (!found || order > latest)
}

// Above is the nearest thing straight over a place within reach metres:
// how far above, the name of its material, and whether there is one.
func (shelter *Shelter) Above(from maths.Vec, reach float64) (float64, string,
	bool) {
	if shelter == nil {
		return 0, "", false
	}
	shelter.ready()
	best, name, found, latest := reach, "", false, -1
	for _, list := range shelter.kept(from) {
		for _, at := range list {
			each := shelter.all[at]
			rise, under := each.over(from)
			if under && rise > 0 &&
				nearer(rise, best, found, each.order, latest) {
				best, name, found, latest = rise, each.name, true, each.order
			}
		}
	}
	for _, list := range shelter.standingNear(from) {
		for _, at := range list {
			one := &shelter.standing[at]
			local := one.place.In(from)
			for _, kept := range one.design.kept(local) {
				for _, index := range kept {
					each := one.design.all[index]
					rise, under := each.over(local)
					rise *= one.scale
					order := one.first + each.order
					if under && rise > 0 &&
						nearer(rise, best, found, order, latest) {
						best, name, found, latest = rise, one.name, true, order
					}
				}
			}
		}
	}
	return best, name, found
}

// Below is the nearest thing straight under a place within reach metres,
// a floor, a roof or a ledge a foot could stand on: how far below, and
// whether there is one.
func (shelter *Shelter) Below(from maths.Vec, reach float64) (float64,
	bool) {
	if shelter == nil {
		return 0, false
	}
	shelter.ready()
	best, found, latest := reach, false, -1
	for _, list := range shelter.kept(from) {
		for _, at := range list {
			each := shelter.all[at]
			rise, under := each.over(from)
			if under && rise <= 0 &&
				nearer(-rise, best, found, each.order, latest) {
				best, found, latest = -rise, true, each.order
			}
		}
	}
	for _, list := range shelter.standingNear(from) {
		for _, at := range list {
			one := &shelter.standing[at]
			local := one.place.In(from)
			for _, kept := range one.design.kept(local) {
				for _, index := range kept {
					each := one.design.all[index]
					rise, under := each.over(local)
					rise *= one.scale
					order := one.first + each.order
					if under && rise <= 0 &&
						nearer(-rise, best, found, order, latest) {
						best, found, latest = -rise, true, order
					}
				}
			}
		}
	}
	return best, found
}

// over is how far above a point the triangle is, and whether the point
// is under it at all.
func (each overhead) over(from maths.Vec) (float64, bool) {
	first, second, third := each.corners[0], each.corners[1], each.corners[2]
	area := edge(first, second, third)
	one := edge(second, third, from) / area
	two := edge(third, first, from) / area
	three := 1 - one - two
	if min(one, two, three) < -rimSlack {
		return 0, false
	}
	height := one*first.Y + two*second.Y + three*third.Y
	return height - from.Y, true
}

// edge is twice the signed area of the triangle a, b, c as seen from
// above, which is also how far c is to the side of the line from a to b.
func edge(first, second, third maths.Vec) float64 {
	return (second.X-first.X)*(third.Z-first.Z) -
		(second.Z-first.Z)*(third.X-first.X)
}

// named is whether a material's name holds any of the words; no words
// hold every name.
func named(name string, words []string) bool {
	if len(words) == 0 {
		return true
	}
	for _, word := range words {
		if strings.Contains(name, word) {
			return true
		}
	}
	return false
}
