package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// laneRow is the row the cables and pipes of a lane are drawn at, by the
// index of the lane.
const laneRow = 3

// The two walls of a lane, by the side of its gap they stand on.
const (
	lowWall = iota
	highWall
	wallsOfLane
)

// yard is a roof used as a yard: a rail round the inside of its parapet
// and a line of washing across its middle, each where the roof is clear.
func (growing *accretion) yard(index int) {
	drawing := growing.drawing
	roof, top := growing.roofOf(index)
	bar, rail := drawing.fixed("rail-bar"), drawing.fixed("rail-height")
	steel := paint("rail", drawing.pick("cage-colour",
		drawing.options("cage-colour"), place{Bay: index}))
	for side := range sidesAround {
		edge := roof.edge(side, bar)
		if growing.settle(Volume{Kind: "rail", Plot: index,
			Low:  [3]float64{edge.X0, top, edge.Z0},
			High: [3]float64{edge.X1, top + rail, edge.Z1}}) {
			growing.parts = append(growing.parts, put("rail", Box{
				edge.width(), rail, edge.depth()}, steel,
				maths.V(edge.X0, top, edge.Z0)))
		}
	}
	growing.washing(index, roof.inset(bar), top, steel)
}

// washing is a line across the middle of a roof along its longer side,
// on a post at each end, with things hung on it at the culture's pitch,
// if it fits on the roof and the air there is clear.
func (growing *accretion) washing(index int, roof rect, top float64,
	steel *thing.Material) {
	drawing := growing.drawing
	at := place{Bay: index}
	length := drawing.number("line-length", at)
	high, bar := drawing.fixed("line-height"), drawing.fixed("rail-bar")
	if length > roof.width() {
		return
	}
	from := maths.V((roof.X0+roof.X1-length)/half, top,
		(roof.Z0+roof.Z1)/half)
	if !growing.settle(Volume{Kind: "line", Plot: index,
		Low:  [3]float64{from.X, top, from.Z - bar},
		High: [3]float64{from.X + length, top + high, from.Z + bar}}) {
		return
	}
	growing.parts = append(growing.parts,
		put("post", Box{bar, high, bar}, steel, from),
		put("post", Box{bar, high, bar}, steel,
			from.Add(maths.V(length-bar, 0, 0))),
		put("line", Box{length, bar, bar}, steel,
			from.Add(maths.V(0, high-bar, 0))))
	pitch := drawing.fixed("cloth-pitch")
	for cloth := range int(length / pitch) {
		here := place{Bay: index, Opening: cloth}
		drop := min(drawing.number("cloth-drop", here), high-bar)
		look := paint("cloth", drawing.pick("cloth-colour",
			drawing.options("cloth-colour"), here))
		growing.parts = append(growing.parts, put("cloth", Box{pitch / half,
			drop, bar}, look, from.Add(maths.V(float64(cloth)*pitch+
			pitch/(2*half), high-bar-drop, 0))))
	}
}

// lane is what runs through a lane that has a wall on both sides: as many
// cables across it as the accretion reaches, each under the lower of the
// two walls, and by the gate a pipe along it against its low wall.
func (growing *accretion) lane(index int, alley rect) {
	drawing := growing.drawing
	top, walled := growing.walled(alley)
	if !walled {
		return
	}
	at := place{Row: laneRow, Bay: index}
	floor := growing.out.Deck
	axis := alley.gapAxis()
	start, end := alley.edgeOf(1 - axis)
	thick := drawing.fixed("cable-thickness")
	black := paint("cable", drawing.pick("cable-colour",
		drawing.options("cable-colour"), place{}))
	for cable := range drawing.reach("cable-count", at, growing.grown) {
		here := place{Row: laneRow, Bay: index, Opening: cable}
		height := floor + drawing.number("cable-high", here)
		if height+thick > top {
			continue
		}
		along := start + drawing.number("cable-along", here)*(end-start-thick)
		growing.wire(alley, axis, height, along, black)
	}
	if drawing.gate("pipe-share", at, growing.grown, nil) {
		growing.pipe(at, alley, axis, floor, top)
	}
}

// walled is the top of the lower of a lane's two walls, the highest of
// the buildings that stand along each of its sides, and whether both of
// its sides have one.
func (growing *accretion) walled(alley rect) (float64, bool) {
	axis := alley.gapAxis()
	low, high := alley.edgeOf(axis)
	from, to := alley.edgeOf(1 - axis)
	var tops [wallsOfLane]float64
	for _, plot := range growing.out.Plots {
		near, far := plot.Lot.edgeOf(axis)
		start, end := plot.Lot.edgeOf(1 - axis)
		if min(end, to)-max(start, from) <= meeting {
			continue
		}
		if same(far, low) {
			tops[lowWall] = max(tops[lowWall], plot.Height)
		}
		if same(near, high) {
			tops[highWall] = max(tops[highWall], plot.Height)
		}
	}
	top := min(tops[lowWall], tops[highWall])
	return top, top > 0
}

// wire is one cable across a lane, from wall to wall, at a height and a
// place along it, if the air there is clear.
func (growing *accretion) wire(alley rect, axis int, height, along float64,
	black *thing.Material) {
	thick := growing.drawing.fixed("cable-thickness")
	size := Box{thick, thick, alley.depth()}
	corner := maths.V(along, height, alley.Z0)
	if axis == acrossX {
		size = Box{alley.width(), thick, thick}
		corner = maths.V(alley.X0, height, along)
	}
	if growing.settle(volumeOf("cable", -1, standingAt(maths.Vec{}), size,
		corner)) {
		growing.parts = append(growing.parts, put("cable", size, black,
			corner))
	}
}

// pipe is a pipe along the whole of a lane against its low wall, at the
// height the culture draws, if it is under the lower wall's top and the
// air there is clear.
func (growing *accretion) pipe(at place, alley rect, axis int, floor,
	top float64) {
	drawing := growing.drawing
	size := drawing.fixed("pipe-size")
	height := floor + drawing.number("pipe-high", at)
	if height+size > top {
		return
	}
	form := Box{alley.width(), size, size}
	if axis == acrossX {
		form = Box{size, size, alley.depth()}
	}
	corner := maths.V(alley.X0, height, alley.Z0)
	if !growing.settle(volumeOf("pipe", -1, standingAt(maths.Vec{}), form,
		corner)) {
		return
	}
	look := paint("pipe", drawing.pick("pipe-colour",
		drawing.options("pipe-colour"), at))
	growing.parts = append(growing.parts, put("pipe", form, look, corner))
}
