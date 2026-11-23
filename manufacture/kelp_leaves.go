package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// leaves adds a blade and its gas bladder at each node along a stipe's
// path. The nodes come at the file's spacing, drawn afresh for each, so
// they are irregular as the pages say; from a node's place a blade
// stands off the stipe, turned a golden angle from the last. A node on
// the canopy hangs its blade down, so none stands out of the water.
func (plant kelpPlant) leaves(into *kelpMembers, stipe int,
	path []maths.Vec, canopy int) {
	node := 0
	carry := plant.gap(stipe, node)
	for index := 0; index+1 < len(path); index++ {
		run := path[index+1].Sub(path[index])
		length := run.Len()
		way := run.Scale(1 / length)
		along := carry
		for along <= length {
			key := stipe*kelpNodeStride + node
			plant.node(into, key, path[index].Add(way.Scale(along)), way,
				index >= canopy)
			node++
			along += plant.gap(stipe, node)
		}
		carry = along - length
	}
}

// gap is the distance from one node of a stipe to the next.
func (plant kelpPlant) gap(stipe, node int) float64 {
	return kelpRead.pick("node-spacing", plant.from,
		stipe*kelpNodeStride+node)
}

// node adds the blade and the gas bladder at one node: its place, the
// way the stipe runs there, and whether it is on the canopy.
func (plant kelpPlant) node(into *kelpMembers, key int, point, way maths.Vec,
	floating bool) {
	book := kelpRead
	side := kelpSide(way, kelpGolden*float64(key)+
		fullTurn*plant.from.draw(growing, key, 5))
	if floating && side.Y > 0 {
		side = side.Neg()
	}
	frame := plant.bladeFrame(key, way, side, floating)
	scale := book.pick("blade-length", plant.from, key) /
		book.middle("blade-length")
	variant := int(plant.from.draw(growing, key, 1) *
		float64(len(plant.kit.blades)))
	into.blades = append(into.blades, shape.MeshMember{
		Mesh:  plant.kit.blades[variant],
		Space: maths.NewSubspace(point, frame, scale)})
	size := book.pick("float-length", plant.from, key)
	into.bladders = append(into.bladders, shape.MeshMember{
		Mesh: plant.kit.bladder,
		Space: maths.NewSubspace(point.Sub(way.Scale(size*kelpHalf)),
			maths.Upright(way), size)})
}

// bladeFrame is the way a blade lies: its length streaming downstream,
// toward plus x, lifted by its bladder and a little along its stipe and
// out to its side; on the canopy it floats level or sinks, never above the
// surface. It is turned about its own length by a draw of its own, so
// some show their broad face to a viewer and some their edge. In still
// water it streams nowhere and stands out from its stipe all round.
func (plant kelpPlant) bladeFrame(key int, way, side maths.Vec,
	floating bool) maths.Orientation {
	book := kelpRead
	lift := book.pick("blade-lift", plant.from, key)
	stream := book.constantOf("blade-stream") * plant.lean()
	axis := maths.XAxis.Scale(stream).
		Add(maths.Up.Scale(book.constantOf("blade-buoy"))).
		Add(side.Scale(math.Sin(lift))).
		Add(way.Scale(math.Cos(lift) * kelpAlong))
	if floating {
		axis.Y = -math.Abs(axis.Y)
	}
	axis = axis.Unit()
	width := way.Cross(axis)
	if width.Len() < kelpSmall {
		width = side
	}
	width = width.Unit()
	normal := width.Cross(axis)
	twist := math.Pi * plant.from.draw(growing, key, 6)
	width = width.Scale(math.Cos(twist)).Add(normal.Scale(math.Sin(twist)))
	return maths.Orientation{X: width, Y: axis, Z: width.Cross(axis)}
}

// kelpSide is a direction square to a way, turned by an angle about it.
func kelpSide(way maths.Vec, angle float64) maths.Vec {
	first := maths.Up.Cross(way)
	if first.Len() < kelpSmall {
		first = maths.XAxis.Cross(way)
	}
	first = first.Unit()
	second := way.Cross(first)
	return first.Scale(math.Cos(angle)).Add(second.Scale(math.Sin(angle)))
}
