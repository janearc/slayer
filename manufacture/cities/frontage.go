package cities

import (
	"fmt"
	"math"
	"strings"
)

// A street along water is a line of fronts. A canal or a quay has its
// wall, then a paved street a few metres wide, then the houses, their
// fronts to the water and their doors on the street, row after row along
// the bank, with a lane across between each row and the next.
//
// At a bend the line of fronts bends with the water, and a row that would
// turn the corner stops short of it, so that two rows meet at a cross
// street and not in a heap of roofs.
//
// A frontage lays what it is given along that line of fronts, which is the
// middle line of the channel moved sideways by an offset and mitred at its
// joints. A terrace is laid as runs end to end, the front of each on the
// line, with a gap between runs.
//
// A tree or a boat is laid at the spacing
// its data gives, one to each stretch of that length. A run or an item is
// laid only where it fits whole between the ends of one straight reach of
// the line, short of each end by half a gap and, where the line turns
// toward the items, by the part of a row's depth the turn takes.

// Frontage is the line of fronts a stand lays its items along.
type Frontage struct {
	// Line is the middle of the channel the fronts face, as points of
	// east and north in metres, from its first end to its last.
	Line [][2]float64
	// Offset is how far the line of fronts lies from Line, in metres,
	// to the side the items stand on.
	Offset float64
	// Right is whether the items stand on the right of someone going
	// along Line from its first point. They stand on the left otherwise.
	Right bool
	// Gap is the room between one run and the next, and half of it the
	// room kept from each end of the line, in metres.
	Gap float64
}

// FrontageAlong is the frontage of a line: the fronts Offset metres from
// it, on its right or its left, with Gap between runs.
func FrontageAlong(line [][2]float64, offset float64, right bool,
	gap float64) Frontage {
	return Frontage{Line: line, Offset: offset, Right: right, Gap: gap}
}

// frontage lays terraces, trees and boats along a channel's line.
//
// A site is where an item stands and the bearing its front looks along. A
// run's site is its back corner, as a terrace is drawn from, and its front
// looks across the street to the water. An item laid at a spacing stands
// at the middle of its stretch, and looks as far from the direction the
// line runs in as its data says, or is left to face as it is stood.
//
// The runs of the line extend along it in the way someone looks who has the
// water on their right hand: for items on the left of the line that is its
// own direction, and for items on the right it is the line taken from its
// last point.
type frontage struct{}

// frontReach is one straight stretch of the line of fronts.
type frontReach struct {
	// start is where the reach begins on the line of fronts, along is
	// the unit direction it runs in, and length how long it is.
	start, along point
	length       float64
	// opening and closing are the tangent of half the turn the line
	// makes where the reach begins and where it ends, where that turn
	// comes toward the items, and nought where it goes away from them
	// or the line ends.
	opening, closing float64
}

// fronted is a frontage being laid: the item, the room between runs, the
// sites so far and the number of the item to lay next.
type fronted struct {
	item   Item
	gap    float64
	most   int
	sites  []Site
	number int
	// reaches is how many straight reaches the line of fronts has.
	reaches int
	// spans are the distances along the reach being laid, from where the
	// reach begins, that each run or item starts and ends at, and told
	// is what each reach laid, for the line the habit says.
	spans []string
	told  []string
}

// leastMitre is the least 1 plus the cosine of the angle between two
// directions that the line of fronts can be mitred at; a hairpin turn
// is under it and has no corner.
const leastMitre = 1e-6

// Sites is the runs or items laid along the line of fronts, reach by
// reach from the start of the line the items face, as many as most says or
// as many as fit if most is nought.
func (frontage) Sites(ground Ground, item Item, _ Seed, most int) (Laid,
	error) {
	if ground.Frontage == nil {
		return Laid{}, fmt.Errorf("cities: a frontage needs the line of " +
			"a channel to stand along")
	}
	if item.Run == nil && !item.Spaced {
		return Laid{}, fmt.Errorf("cities: a frontage lays terraces end to "+
			"end or items at a spacing, and the data gives %s neither",
			item.Kind)
	}
	reaches, err := ground.Frontage.reaches()
	if err != nil {
		return Laid{}, err
	}
	laying := &fronted{item: item, gap: ground.Frontage.Gap, most: most,
		reaches: len(reaches)}
	for index, each := range reaches {
		if err := laying.along(index, each); err != nil {
			return Laid{}, err
		}
	}
	return Laid{Sites: laying.sites, Said: fmt.Sprintf("a frontage laid %d "+
		"along %d reaches, %s", len(laying.sites), laying.reaches,
		strings.Join(laying.told, "; "))}, nil
}

// along lays one reach, and notes where on it each run or item stands, in
// metres from where the reach begins on the line of fronts.
func (laying *fronted) along(index int, each frontReach) error {
	laying.spans = nil
	err := laying.reach(each)
	laying.told = append(laying.told, fmt.Sprintf("reach %d from %.1f e, "+
		"%.1f n by %.1f m: %s", index, each.start.east, each.start.north,
		each.length, strings.Join(laying.spans, " ")))
	return err
}

// reaches is the straight reaches of the line of fronts, in the order they
// are laid, or why the line has no fronts.
func (along Frontage) reaches() ([]frontReach, error) {
	if len(along.Line) < 2 || !(along.Gap >= 0) || !(along.Offset > 0) ||
		math.IsInf(along.Offset, 0) {
		return nil, fmt.Errorf("cities: a frontage needs a line of two "+
			"points or more, a gap that is not negative and an offset to "+
			"one side above nought, not %d points, gap %v, offset %v",
			len(along.Line), along.Gap, along.Offset)
	}
	line := along.points()
	directions, err := directionsOf(line)
	if err != nil {
		return nil, err
	}
	corners := make([]point, len(line))
	for index := range line {
		corners[index] = frontCorner(line, directions, index, along.Offset)
	}
	reaches := make([]frontReach, len(directions))
	for index, direction := range directions {
		reaches[index] = frontReach{start: corners[index], along: direction,
			length:  corners[index+1].minus(corners[index]).dot(direction),
			opening: turnToward(directions, index),
			closing: turnToward(directions, index+1)}
	}
	return reaches, nil
}

// points is the line as places, taken from its last point when the items
// stand on its right, so that the items are always on the left of it.
func (along Frontage) points() []point {
	line := make([]point, len(along.Line))
	for index, each := range along.Line {
		line[index] = point{each[0], each[1]}
	}
	if along.Right {
		for low, high := 0, len(line)-1; low < high; low, high = low+1,
			high-1 {
			line[low], line[high] = line[high], line[low]
		}
	}
	return line
}

// directionsOf is the unit direction of each stretch of a line, or why a
// stretch has none or the line turns back on itself.
func directionsOf(line []point) ([]point, error) {
	directions := make([]point, len(line)-1)
	for index := range directions {
		step := line[index+1].minus(line[index])
		length := math.Hypot(step.east, step.north)
		if !(length > 0) {
			return nil, fmt.Errorf("cities: the line of a frontage stays "+
				"at point %d for its next", index)
		}
		directions[index] = point{step.east / length, step.north / length}
		if index > 0 && 1+directions[index-1].dot(directions[index]) <
			leastMitre {
			return nil, fmt.Errorf("cities: the line of a frontage turns "+
				"back on itself at point %d, where it has no corner", index)
		}
	}
	return directions, nil
}

// frontCorner is where the line of fronts has its corner over a point of
// the line: the point moved to the left by the offset, and at a joint along
// the mitre, so that both its reaches lie the offset from the line.
func frontCorner(line, directions []point, index int,
	offset float64) point {
	if index == 0 {
		return line[0].plus(leftOf(directions[0]), offset)
	}
	if index == len(line)-1 {
		return line[index].plus(leftOf(directions[index-1]), offset)
	}
	before, after := leftOf(directions[index-1]), leftOf(directions[index])
	scale := offset / (1 + before.dot(after))
	return line[index].plus(point{before.east + after.east,
		before.north + after.north}, scale)
}

// turnToward is the tangent of half the turn the line makes at the
// beginning of the reach numbered from nought, where it turns toward the
// left, which is toward the items, and nought where it turns away, runs
// straight or has no joint there.
func turnToward(directions []point, index int) float64 {
	if index == 0 || index >= len(directions) {
		return 0
	}
	before, after := directions[index-1], directions[index]
	turn := before.east*after.north - before.north*after.east
	if turn <= 0 {
		return 0
	}
	return turn / (1 + before.dot(after))
}

// reach lays along one straight reach as many as fit, one after another,
// and goes on to the next reach when the next does not fit whole in this
// one.
func (laying *fronted) reach(each frontReach) error {
	earliest := 0.0
	for !laying.full() {
		size, err := laying.room(laying.number)
		if err != nil {
			return err
		}
		start := math.Max(earliest,
			laying.gap/2+each.opening*size.Depth)
		last := each.length - laying.gap/2 - each.closing*size.Depth
		if start+size.Length > last {
			return nil
		}
		laying.sites = append(laying.sites, laying.site(each, start, size))
		laying.spans = append(laying.spans, fmt.Sprintf("%.1f-%.1f", start,
			start+size.Length))
		laying.number++
		earliest = start + size.Length + laying.between()
	}
	return nil
}

// full is whether as many are laid as the stand asks for.
func (laying *fronted) full() bool {
	return laying.most > 0 && len(laying.sites) >= laying.most
}

// between is the room kept between one run and the next: the gap for a
// terrace, and none for items at a spacing, which have their room in it.
func (laying *fronted) between() float64 {
	if laying.item.Spaced {
		return 0
	}
	return laying.gap
}

// room is the room the item of a number takes along the line, and how deep
// it is: its spacing, or the run its terrace draws.
func (laying *fronted) room(number int) (Run, error) {
	if laying.item.Spaced {
		return Run{Length: laying.item.Spacing}, nil
	}
	size, err := laying.item.Run(number)
	if err == nil && !(size.Length > 0) {
		err = fmt.Errorf("cities: run %d of %s is %v m long and takes no "+
			"room", number, laying.item.Kind, size.Length)
	}
	return size, err
}

// site is the site of the run or item that starts a distance along a
// reach. A run's front stands on the line of fronts and its site is its
// back corner, a depth behind it; an item at a spacing stands at the middle
// of its stretch.
func (laying *fronted) site(each frontReach, start float64, size Run) Site {
	across := rightOf(each.along)
	if laying.item.Spaced {
		at := each.start.plus(each.along, start+size.Length/2)
		heading := bearingOf(each.along.east, each.along.north) +
			laying.item.Turn
		return Site{East: at.east, North: at.north, Faced: laying.item.Turned,
			Heading: math.Mod(heading+fullDegrees, fullDegrees)}
	}
	at := each.start.plus(each.along, start).plus(across, -size.Depth)
	return Site{East: at.east, North: at.north, Faced: true,
		Heading: bearingOf(across.east, across.north)}
}

// plus is a place moved along a direction by a distance.
func (at point) plus(direction point, distance float64) point {
	return point{at.east + direction.east*distance,
		at.north + direction.north*distance}
}

// minus is the way from one place to another.
func (at point) minus(other point) point {
	return point{at.east - other.east, at.north - other.north}
}

// dot is the sum of the products of two directions' parts.
func (at point) dot(other point) float64 {
	return at.east*other.east + at.north*other.north
}

// leftOf is a direction turned a quarter turn to the left, anticlockwise.
func leftOf(direction point) point {
	return point{-direction.north, direction.east}
}

// rightOf is a direction turned a quarter turn to the right, clockwise.
func rightOf(direction point) point {
	return point{direction.north, -direction.east}
}
