package primates

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

type nasalStation struct {
	y, width, ridge, wing float64
}

// From the subnasale, cross the columella and lobule into the dorsum and
// nasal root. Each row carries the ridge and the coupled alar sidewalls.
var nasalStations = [...]nasalStation{
	{1.517, .013, 0, 0},
	{1.522, .014, .0030, .0022},
	{1.526, .018, .0090, .0070},
	{1.532, .021, .0185, .0076},
	{1.537, .022, .0231, .0040},
	{1.543, .021, .0208, .0011},
	{1.550, .020, .0162, 0},
	{1.560, .022, .0105, 0},
	{1.570, .025, .0054, 0},
	{1.582, .029, .0020, 0},
	{1.594, .031, 0, 0},
}

func noseRelief(x, y float64) float64 {
	if y <= nasalStations[0].y || y >= nasalStations[len(nasalStations)-1].y || x >= .031 {
		return 0
	}
	station := nasalSection(y)
	dorsum := station.ridge * (1 - maths.Smooth(0, station.width, x))
	alarWall := station.wing * (1 - maths.Smooth(0, .0091, math.Abs(x-.0121)))
	return dorsum + alarWall
}

func nasalSection(y float64) nasalStation {
	i := 0
	for y > nasalStations[i+1].y {
		i++
	}
	left, right := nasalStations[i], nasalStations[i+1]
	first := nasalStations[max(0, i-1)]
	last := nasalStations[min(len(nasalStations)-1, i+2)]
	length := right.y - left.y
	share := (y - left.y) / length
	value := func(before, base, end, after float64) float64 {
		var from, to float64
		if i > 0 {
			from = length * (end - before) / (right.y - first.y)
		}
		if i+2 < len(nasalStations) {
			to = length * (after - base) / (last.y - left.y)
		}
		return hermite(base, end, from, to, share)
	}
	return nasalStation{y: y,
		width: value(first.width, left.width, right.width, last.width),
		ridge: value(first.ridge, left.ridge, right.ridge, last.ridge),
		wing:  math.Max(0, value(first.wing, left.wing, right.wing, last.wing)),
	}
}

func nasalSockets(stations []float64) [2]socket {
	bottom := sort.SearchFloat64s(stations, 1.5218)
	top := sort.SearchFloat64s(stations, 1.5315)
	return [2]socket{
		{3, 13, bottom, top, humanAround, 1},
		{humanAround - 13, humanAround - 3, bottom, top, humanAround, -1},
	}
}

func nasalFloor(x, y float64) maths.Vec {
	ring := headSection(y)
	across := x / ring.w
	cosine := math.Sqrt(1 - across*across)
	return maths.V(x, y, faceDepth(ring, math.Abs(x), cosine))
}

func (side humanSide) naris(model *humanMesh, rim []int) {
	const rows = 8
	weight := fixedWeight("head")
	centre := nasalFloor(side.sign*.0096, 1.5265)
	edges := make([]maths.Vec, len(rim))
	for i, index := range rim {
		delta := model.vertices[index].chart.Sub(centre)
		angle := math.Atan2(delta.Y/.0048, delta.X/.0074)
		sine, cosine := math.Sincos(angle)
		across := .00415 * cosine
		height := .00155*sine - .18*side.sign*across
		edges[i] = nasalFloor(centre.X+across, centre.Y+height)
	}
	previous := rim
	for row := 1; row <= rows; row++ {
		share := float64(row) / rows
		next := make([]int, len(rim))
		for i, index := range rim {
			point := mixPoint(model.vertices[index].chart, edges[i], share)
			next[i] = model.headVertex(nasalFloor(point.X, point.Y), weight)
		}
		model.connect(previous, next)
		previous = next
	}
	model.nasalVestibule(previous, centre, edges)
}

// The lip turns into a short blind vestibule angled upward and backward.
// Closing each recess preserves a single watertight exterior skin mesh.
func (model *humanMesh) nasalVestibule(rim []int, centre maths.Vec, edges []maths.Vec) {
	weight := fixedWeight("head")
	inward := maths.V(0, .0060, -.0041)
	previous := rim
	for _, layer := range [...]struct{ radius, depth float64 }{
		{.92, .10}, {.87, .30}, {.68, .58}, {.42, .78},
	} {
		next := make([]int, len(rim))
		for i, edge := range edges {
			point := mixPoint(centre, edge, layer.radius).Add(inward.Scale(layer.depth))
			next[i] = model.headVertex(point, weight)
		}
		model.connect(previous, next)
		previous = next
	}
	model.cap(previous, model.headVertex(centre.Add(inward), weight))
}
