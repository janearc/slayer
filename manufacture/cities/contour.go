package cities

import (
	"fmt"
	"math"
)

// A town on a cliff is rows of houses that follow the hill. Each row runs
// along a contour, the line of ground at one height, with its front to
// the slope's fall or against it, and the next row stands above it with a
// lane between. The ground says where a row goes; the set picks only the
// lane's width and how many rows and runs there are.
//
// A row is made of runs, each a terrace drawn from the culture and
// standing straight. A run starts where the one before it ended and is
// turned to the contour where it stands, so a row bent through a few
// straight runs follows a hillside that bends.
//
// At a bend the fronts of two runs meet at a slight angle: the outer
// corner leaves a gap and the inner overlaps, by the depth of the row
// times the tangent of the turn.

// Climb is how rows of terraces climb a slope and how they stand on it.
type Climb struct {
	// Rows is how many rows climb the slope, one above the next.
	Rows int
	// Runs is how many straight runs of terrace make a row, end to end.
	Runs int
	// Lane is the width of the gap between one row and the one above it,
	// in metres: a lane or a stair.
	Lane float64
	// Uphill is whether the fronts of the rows, and so their doors, look
	// up the slope. Otherwise they look down it.
	Uphill bool
}

// point is a place on the map, or a direction on it, east and north.
type point struct{ east, north float64 }

// contour lays rows of terraces along the contours of a slope. The first
// row starts at the centre of the ground and each row after it stands
// above the last, so that a hillside reads as a town climbing it.
//
// It reads the ground's height, how the rows climb and the room each run
// takes. It lays no run that starts farther than the ground's far from
// its centre; the ground's bearings and its near are not used.
//
// A site is the back corner of a run where it starts, which is where a
// terrace is drawn from, and the bearing its front looks along. A run
// extends to the left of someone looking along its front, as a terrace
// does.
type contour struct{}

// contoured is a contour being laid: what it is laid on, the sites so
// far, and how finely the ground is read.
type contoured struct {
	ground Ground
	item   Item
	most   int
	span   float64
	least  float64
	sites  []Site
	rows   int
	left   int
}

// Sites is the runs of the rows, in the order they are laid, row by row
// from the lowest and run by run along each. most is the most runs, or
// nought for every run of every row that is within reach.
func (contour) Sites(ground Ground, item Item, _ Seed, most int) (Laid,
	error) {
	if ground.Height == nil || ground.Climb == nil || item.Run == nil ||
		ground.Climb.Rows < 1 || ground.Climb.Runs < 1 {
		return Laid{}, fmt.Errorf("cities: a contour needs the height of " +
			"the ground, rows and runs to lay, and the room each run takes")
	}
	laying, err := newContoured(ground, item, most)
	if err != nil {
		return Laid{}, err
	}
	if err := laying.climb(); err != nil {
		return Laid{}, err
	}
	return Laid{Sites: laying.sites, Said: fmt.Sprintf("a contour laid %d "+
		"runs in %d rows, and left %d past its reach", len(laying.sites),
		laying.rows, laying.left)}, nil
}

// climb lays the rows one above the next, from the centre of the ground,
// until there are as many as the climb says, as many runs as were asked
// for, or the next row would start beyond the ground's reach.
func (laying *contoured) climb() error {
	start := point{laying.ground.East, laying.ground.North}
	for row := 0; row < laying.ground.Climb.Rows && !laying.full(); row++ {
		next, err := laying.row(start, row)
		if err != nil {
			return err
		}
		laying.rows++
		start = next
		if laying.beyond(start) {
			break
		}
	}
	return nil
}

// newContoured is a contour ready to lay, with the two numbers it reads the
// ground by.
func newContoured(ground Ground, item Item, most int) (*contoured, error) {
	span, err := choice("contour-span")
	if err != nil {
		return nil, err
	}
	least, err := choice("least-fall")
	if err != nil {
		return nil, err
	}
	return &contoured{ground: ground, item: item, most: most, span: span,
		least: least}, nil
}

// full is whether as many runs are laid as the stand asks for.
func (laying *contoured) full() bool {
	return laying.most > 0 && len(laying.sites) >= laying.most
}

// beyond is whether a place is farther from the centre of the ground than
// the ground reaches.
func (laying *contoured) beyond(at point) bool {
	return math.Hypot(at.east-laying.ground.East,
		at.north-laying.ground.North) > laying.ground.Far
}

// downhill is the unit direction the ground falls fastest at a place, read
// from the heights a span either side of it, or why there is none.
func (laying *contoured) downhill(at point) (point, error) {
	height := laying.ground.Height
	alongEast := (height(at.east+laying.span, at.north) -
		height(at.east-laying.span, at.north)) / (2 * laying.span)
	alongNorth := (height(at.east, at.north+laying.span) -
		height(at.east, at.north-laying.span)) / (2 * laying.span)
	rise := math.Hypot(alongEast, alongNorth)
	if !(rise >= laying.least) {
		return point{}, fmt.Errorf("cities: the ground at %.1f e, %.1f n "+
			"rises %.3f over a metre, less than the %.3f a row needs to "+
			"follow its contour", at.east, at.north, rise, laying.least)
	}
	return point{-alongEast / rise, -alongNorth / rise}, nil
}

// front is the unit direction the front of a run looks along at a place:
// down the slope, or up it as the climb says.
func (laying *contoured) front(at point) (point, error) {
	down, err := laying.downhill(at)
	if laying.ground.Climb.Uphill {
		return point{-down.east, -down.north}, err
	}
	return down, err
}

// lengthwise is the unit direction a run extends in from its start, given
// the way its front looks: a quarter turn anticlockwise of the front.
func lengthwise(front point) point {
	return point{-front.north, front.east}
}

// run is one run started at a place, numbered from nought in the order of
// laying: its site, and where the next run starts. The run is turned to the
// contour where its middle stands, which is found from the contour at its
// start.
func (laying *contoured) run(start point, index int) (Site, point, error) {
	size, err := laying.item.Run(index)
	if err != nil {
		return Site{}, point{}, err
	}
	front, err := laying.front(start)
	if err != nil {
		return Site{}, point{}, err
	}
	reach := lengthwise(front)
	middle := point{start.east + reach.east*size.Length/2,
		start.north + reach.north*size.Length/2}
	if front, err = laying.front(middle); err != nil {
		return Site{}, point{}, err
	}
	reach = lengthwise(front)
	site := Site{East: start.east, North: start.north, Faced: true,
		Heading: bearingOf(front.east, front.north)}
	return site, point{start.east + reach.east*size.Length,
		start.north + reach.north*size.Length}, nil
}

// row lays the runs of one row end to end from a place, and gives where the
// row above it starts. A run that would start past the ground's reach ends
// the row.
func (laying *contoured) row(start point, row int) (point, error) {
	runs := laying.ground.Climb.Runs
	at := start
	for run := 0; run < runs && !laying.full(); run++ {
		if laying.beyond(at) {
			laying.left += runs - run
			break
		}
		site, end, err := laying.run(at, row*runs+run)
		if err != nil {
			return point{}, err
		}
		laying.sites = append(laying.sites, site)
		at = end
	}
	return laying.above(start, row)
}

// above is where the row above a row starts: uphill of its start by the
// lane and the depth that sets the lane between the two. A row that looks
// down the slope has the row above it against its back, and the lane lies
// between that back and the front of the row above; a row that looks up it
// has its own front to the lane.
func (laying *contoured) above(start point, row int) (point, error) {
	down, err := laying.downhill(start)
	if err != nil {
		return point{}, err
	}
	runs := laying.ground.Climb.Runs
	next := row + 1
	if laying.ground.Climb.Uphill {
		next = row
	}
	size, err := laying.item.Run(next * runs)
	if err != nil {
		return point{}, err
	}
	reach := size.Depth + laying.ground.Climb.Lane
	return point{start.east - down.east*reach,
		start.north - down.north*reach}, nil
}
