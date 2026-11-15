package planet

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/janearc/slayer/maths"
)

// A Chart is a face painted once onto a grid of latitude and longitude and
// read between its points, in place of working the air out again for
// every ray that meets the planet.
//
// It paints a tile of the grid the first time a ray lands in it, so only
// the part of the planet a picture shows is ever painted. Where the grid
// cannot stand for the face it asks the face itself: at the limb, where a
// point of the grid is out of the eye's sight, and for a pixel so wide
// that the face would draw its fronts more coarsely.
type Chart struct {
	face         *Face
	radius       float64
	across, rows int
	run, rise    float64
	wide         int
	tiles        []chartTile
	painted      atomic.Int64
}

// a chartTile is a square of the chart's points, painted once: three
// numbers a point, the first not a number where the eye cannot see it.
type chartTile struct {
	once  sync.Once
	light []float32
}

// chartSide is defined as how many points a tile is across: the chart's
// unit of painting is a tile of this side, so a picture's edge paints at
// most a tile's width of points it does not show.
const chartSide = 32

// chartLight is defined as how many numbers a point holds, counted: red,
// green and blue.
const chartLight = 3

// NewChart is a chart of a face on a body so many metres in radius, its
// points cell radians apart at the equator, and whether there is one:
// only a face whose air is read from a flow depends on nothing but the
// point, the eye and the star once its pixel is fine.
func NewChart(face *Face, radius, cell float64) (*Chart, bool) {
	if face.flow == nil || !positive(radius) || !positive(cell) {
		return nil, false
	}
	across := int(math.Ceil(circle / cell))
	rows := int(math.Ceil(math.Pi / cell))
	wide := (across + chartSide - whole) / chartSide
	tall := (rows + chartSide) / chartSide
	return &Chart{face: face, radius: radius, across: across, rows: rows,
		run: circle / float64(across), rise: math.Pi / float64(rows),
		wide: wide, tiles: make([]chartTile, wide*tall)}, true
}

// At is the light a point of the body sends the eye, read from the chart.
func (chart *Chart) At(point maths.Vec) maths.Light {
	return chart.Within(point, nothing)
}

// Within is the light a point of the body sends the eye for a pixel so
// many metres wide: read between the chart's four points about it, or the
// face's own where the chart cannot stand for it.
func (chart *Chart) Within(point maths.Vec, footprint float64) maths.Light {
	face := chart.face
	if footprint/point.Len() > frontStep*maths.Degree {
		return face.Within(point, footprint)
	}
	up := point.Unit()
	sine := up.Dot(face.ways.pole)
	below := math.Min(whole, sine)
	high := math.Asin(math.Max(-whole, below))
	round := math.Atan2(up.Dot(face.ways.east), up.Dot(face.ways.front))
	if round < nothing {
		round += circle
	}
	down, along := (high+rightAngle)/chart.rise, round/chart.run
	row := min(int(down), chart.rows-whole)
	i := min(int(along), chart.across-whole)
	north, east := down-float64(row), along-float64(i)
	next := (i + whole) % chart.across
	south, ok := chart.between(i, next, row, east)
	above, also := chart.between(i, next, row+whole, east)
	if !ok || !also {
		return face.Within(point, footprint)
	}
	return lightBetween(south, above, north)
}

// between is the chart's light between two points of a row, so far east
// of the first, and whether the eye sees both.
func (chart *Chart) between(i, next, row int, east float64) (maths.Light,
	bool) {
	west, ok := chart.point(i, row)
	other, also := chart.point(next, row)
	return lightBetween(west, other, east), ok && also
}

// tileSlot is the place of a point among a tile's numbers, by the point's
// column and row in the tile.
func tileSlot(column, row int) int {
	return (row%chartSide*chartSide + column%chartSide) * chartLight
}

// point is the chart's light at a point of its grid, and whether the eye
// sees the point; its tile is painted if no ray has landed in it yet.
func (chart *Chart) point(i, row int) (maths.Light, bool) {
	tile := &chart.tiles[(row/chartSide)*chart.wide+i/chartSide]
	tile.once.Do(tilePainting{chart, tile, i / chartSide, row / chartSide}.run)
	at := tileSlot(i, row)
	red := tile.light[at]
	if red != red {
		return maths.Grey(nothing), false
	}
	green, blue := tile.light[at+whole], tile.light[at+double]
	return maths.RGB(float64(red), float64(green), float64(blue)), true
}

// tilePainting is a tile and its place among the tiles, ready to paint.
type tilePainting struct {
	chart       *Chart
	tile        *chartTile
	column, row int
}

// run paints the tile by its place; it is what a tile's once runs.
func (job tilePainting) run() { job.chart.paint(job.tile, job.column, job.row) }

// paint paints a tile, by its place among the tiles: each point of it by
// the face, as a pixel of no width, and not a number where the eye does
// not see the point.
func (chart *Chart) paint(tile *chartTile, column, row int) {
	face := chart.face
	light := make([]float32, chartSide*chartSide*chartLight)
	for at := range chartSide * chartSide {
		i, latitude := column*chartSide+at%chartSide, row*chartSide+at/chartSide
		high := (float64(latitude)*chart.rise - rightAngle) / maths.Degree
		round := float64(i) * chart.run / maths.Degree
		point := face.ways.toward(high, round).Scale(chart.radius)
		made := face.Within(point, nothing)
		toward := face.eye.Sub(point).Unit()
		if !positive(point.Unit().Dot(toward)) {
			made.R = math.NaN()
		}
		light[at*chartLight] = float32(made.R)
		light[at*chartLight+whole] = float32(made.G)
		light[at*chartLight+double] = float32(made.B)
	}
	tile.light = light
	chart.painted.Add(whole)
}

// lightBetween is the light so far from one light toward another.
func lightBetween(from, toward maths.Light, share float64) maths.Light {
	return maths.RGB(from.R+share*(toward.R-from.R),
		from.G+share*(toward.G-from.G), from.B+share*(toward.B-from.B))
}

// tilesPainted is how many of the chart's tiles have been painted, and how
// many it has.
func (chart *Chart) tilesPainted() (painted, all int) {
	return int(chart.painted.Load()), len(chart.tiles)
}
