package gpu

import "github.com/janearc/slayer/maths"

var instanceRows = planRows(instanceStruct,
	"into_x", "into_y", "into_z", "out_x", "out_y", "out_z", "info")

// packedInstance keeps the original transform arithmetic and named ABI fields,
// while passing fixed stack arrays instead of allocating maps and slices.
func (encoded *scene) packedInstance(placed standing) {
	space := placed.space
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	origin := space.Out(maths.Vec{})
	var intoRows, outRows [3][4]float64
	for index, axis := range [3]maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
		row := space.Orientation.Out(axis).Unit().Scale(1 / scale)
		intoRows[index] = vectorRow(row, -row.Dot(origin))
		way := space.Orientation.Out(axis).Unit()
		outRows[index] = vectorRow(way, 0)
	}
	info := [4]float64{float64(placed.root), float64(placed.material),
		float64(flag(placed.clear)), float64(placed.group)}
	instanceRows.write(&encoded.instances,
		intoRows[0], intoRows[1], intoRows[2], outRows[0], outRows[1], outRows[2], info)
}
