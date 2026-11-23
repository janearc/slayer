package skyglow

import (
	"cmp"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// third is a third, the mean of a triangle's three corners.
const third = 1.0 / 3.0

// classes is how many directions a bin keeps its panels' light in: the
// six ways along the axes, each by the axis its panel faces most nearly,
// and one more for light that goes every way alike.
const classes = 7

// everyWay is the class of a point source, whose light goes every way.
const everyWay = classes - 1

// Source is something that shines, as the air sees it: a place, the way
// it faces and how much light it sends along that way.
type Source struct {
	// At is where it shines from.
	At maths.Vec
	// Normal is the unit way a flat emitter faces; nought for one that
	// shines alike every way.
	Normal maths.Vec
	// Power is the intensity along Normal, per channel: for a flat
	// emitter its radiance times its area, so that along a way off its
	// normal it has that times the cosine; for a point, its intensity.
	Power maths.Light
	// Traced is set for a lamp the tracer already scatters itself, with
	// shadows, as a first order: its light is not added to the glow's
	// first order, only carried to the orders past it.
	Traced bool
}

// toward is the intensity the source sends along a unit way out of it.
func (source Source) toward(out maths.Vec) maths.Light {
	if source.Normal == (maths.Vec{}) {
		return source.Power
	}
	cosine := source.Normal.Dot(out)
	if cosine <= 0 {
		return maths.Light{}
	}
	return source.Power.Scale(cosine)
}

// binKey names a bin: its cell in the grid of bins, its direction class
// and whether the tracer scatters its light itself.
type binKey struct {
	x, y, z, class int
	traced         bool
}

// bin is the sum of what shines in one cell toward one class: its power,
// the weight its place and its way are averaged with, and the weighted
// sums of them.
type bin struct {
	power  maths.Light
	weight float64
	at     maths.Vec
	normal maths.Vec
}

// Cluster gathers emitters into bins, so that thousands of lit panels
// become the few hundreds of sources a solve can afford. A bin is one
// cell of a grid of cells a given size, and one of seven classes of
// direction, and its panels shine from their mean place, weighted by
// their power, in their mean way.
type Cluster struct {
	cell float64
	bins map[binKey]*bin
}

// NewCluster is a cluster with bins of a cell metres across, which MUST
// be more than nothing.
func NewCluster(cell float64) *Cluster {
	return &Cluster{cell: cell, bins: map[binKey]*bin{}}
}

// Panel adds the triangle a, b, c, turning counterclockwise seen from the
// front, and shining a radiance from that front, alike in every way
// along it by Lambert's law.
func (cluster *Cluster) Panel(a, b, c maths.Vec, radiance maths.Light) {
	cross := b.Sub(a).Cross(c.Sub(a))
	area := cross.Len() / 2
	if area == 0 {
		return
	}
	middle := a.Add(b).Add(c).Scale(third)
	normal := cross.Unit()
	cluster.add(middle, normal, radiance.Scale(area), false)
}

// Point adds an emitter that shines alike every way: an intensity, and
// whether the tracer scatters it itself.
func (cluster *Cluster) Point(at maths.Vec, intensity maths.Light,
	traced bool) {
	cluster.add(at, maths.Vec{}, intensity, traced)
}

// add puts power into the bin of a place and a way.
func (cluster *Cluster) add(at, normal maths.Vec, power maths.Light,
	traced bool) {
	weight := power.Luminance()
	if !(weight > 0) || math.IsInf(weight, 0) {
		return
	}
	key := binKey{
		x:      int(math.Floor(at.X / cluster.cell)),
		y:      int(math.Floor(at.Y / cluster.cell)),
		z:      int(math.Floor(at.Z / cluster.cell)),
		class:  classOf(normal),
		traced: traced,
	}
	held := cluster.bins[key]
	if held == nil {
		held = &bin{}
		cluster.bins[key] = held
	}
	held.power = held.power.Add(power)
	held.weight += weight
	held.at = held.at.Add(at.Scale(weight))
	held.normal = held.normal.Add(normal.Scale(weight))
}

// classOf is the direction class of a normal: the axis it leans on most
// and the sign of its lean, or every way for no normal at all.
func classOf(normal maths.Vec) int {
	if normal == (maths.Vec{}) {
		return everyWay
	}
	lean := [3]float64{normal.X, normal.Y, normal.Z}
	axis := 0
	for index, each := range lean {
		if math.Abs(each) > math.Abs(lean[axis]) {
			axis = index
		}
	}
	if lean[axis] < 0 {
		return 2*axis + 1
	}
	return 2 * axis
}

// Sources are the bins as sources, in a fixed order, so the same city
// always makes the same list.
func (cluster *Cluster) Sources() []Source {
	keys := make([]binKey, 0, len(cluster.bins))
	for key := range cluster.bins {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, compareKeys)
	made := make([]Source, 0, len(keys))
	for _, key := range keys {
		held := cluster.bins[key]
		source := Source{At: held.at.Scale(1 / held.weight),
			Power: held.power, Traced: key.traced}
		if key.class != everyWay {
			source.Normal = held.normal.Unit()
		}
		made = append(made, source)
	}
	return made
}

// compareKeys orders bins by cell, then class, then whether traced.
func compareKeys(a, b binKey) int {
	for _, order := range []int{cmp.Compare(a.x, b.x), cmp.Compare(a.y, b.y),
		cmp.Compare(a.z, b.z), cmp.Compare(a.class, b.class)} {
		if order != 0 {
			return order
		}
	}
	if a.traced != b.traced {
		if a.traced {
			return 1
		}
		return -1
	}
	return 0
}
