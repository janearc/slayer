package cities

import (
	"fmt"
	"math"
	"slices"
)

// Plan is the shape of a tower's plan, the outline of a floor.
type Plan string

// The plans a tower may have. Each is a set of convex parts, because a
// prism's caps are fans: a star is two squares, one turned an eighth of a
// turn, and a Y is three wings.
const (
	// planRectangle is a rectangle.
	planRectangle Plan = "rectangle"
	// planRounded is a rectangle with its corners rounded.
	planRounded Plan = "rounded"
	// planPolygon is a regular polygon.
	planPolygon Plan = "polygon"
	// planStar is an eight-pointed star: two squares, one turned.
	planStar Plan = "star"
	// planY is three wings round a core, a hundred and twenty degrees
	// apart.
	planY Plan = "y"
	// planSail is a rectangle with one face bowed out, like a sail.
	planSail Plan = "sail"
	// planLobed is an eight-pointed star whose inner corners are filled
	// by arcs: two squares, one turned, and eight sectors of a circle.
	planLobed Plan = "lobed"
	// planEllipse is an ellipse, cut into flat sides.
	planEllipse Plan = "ellipse"
)

// PlanNames are the plans there are.
func PlanNames() []Plan {
	return []Plan{planRectangle, planRounded, planPolygon, planStar, planY,
		planSail, planLobed, planEllipse}
}

// lobeCore is how far in from a notch of a lobed plan, as a share of the
// notch's distance from the middle, the inner corner of its arc stands.
// It stands inside both squares, so the arc's straight edges are hidden.
const lobeCore = 0.5

// lobeArc is the angle in degrees an arc of a lobed plan sweeps: the
// whole of the angle outside the star at a notch, a hundred and thirty
// five degrees, between the two edges that meet there.
const lobeArc = 135

// notchShare is how far out along a square's edge, as a share of half
// its side, the other square's edge crosses it: the root of two, less one.
var notchShare = math.Sqrt2 - 1

// notchEvery is the angle in degrees from one notch of a star to the next.
const notchEvery = 45

// wingCount is how many wings a Y has.
const wingCount = 3

// starTurn is how far the second square of a star is turned, in degrees:
// an eighth of a turn, so that the eight points are evenly spaced.
const starTurn = 45

// footprintShape is a plan with its measures. Which measures it uses depends on
// the plan: a rectangle uses Width and Depth; a star's Width is the side
// of each of its squares; a polygon's Width is across its corners.
type footprintShape struct {
	Plan Plan
	// Width and Depth are the size across and along, in metres.
	Width, Depth float64
	// Round is a rounded plan's corner radius, and Bow how far a sail's
	// face stands out of its chord, in metres.
	Round, Bow float64
	// Lobe is the radius of the arcs of a lobed plan, in metres.
	Lobe float64
	// Sides is a polygon's or an ellipse's side count, and Steps the
	// flat pieces a rounded corner, a sail's bow or a lobe's arc is cut
	// into.
	Sides, Steps int
	// Wings are the lengths of a Y's wings from its middle, and
	// WingWidth how wide each is.
	Wings     [wingCount]float64
	WingWidth float64
}

// scaled is the shape with every measure of size multiplied.
func (shape footprintShape) scaled(by float64) footprintShape {
	shape.Width *= by
	shape.Depth *= by
	shape.Round *= by
	shape.Bow *= by
	shape.Lobe *= by
	shape.WingWidth *= by
	for index := range shape.Wings {
		shape.Wings[index] *= by
	}
	return shape
}

// parts are the convex parts of the plan, each a list of corners on the
// ground, centred on the origin, wound so that the outward side of each
// edge is the side a face looks out of. Every part of a shape and of a
// scaling of it has the same number of corners, corner for corner.
func (shape footprintShape) parts() ([][]outlineCorner, error) {
	if !(shape.Width > 0) || (!(shape.Depth > 0) &&
		shape.Plan != planPolygon && shape.Plan != planStar &&
		shape.Plan != planLobed && shape.Plan != planY) {
		return nil, fmt.Errorf("cities: a plan needs a positive width and, "+
			"unless it is round or a star, depth, not %v by %v",
			shape.Width, shape.Depth)
	}
	if shape.Plan == planLobed && (!(shape.Lobe > 0) || shape.Steps < 1) {
		return nil, fmt.Errorf("cities: a lobed plan needs a positive arc "+
			"radius and at least one piece to cut it in, not %v and %d",
			shape.Lobe, shape.Steps)
	}
	if shape.Plan == planEllipse && shape.Sides < leastSides {
		return nil, fmt.Errorf("cities: an ellipse needs at least %d sides, "+
			"not %d", leastSides, shape.Sides)
	}
	switch shape.Plan {
	case planRectangle:
		return [][]outlineCorner{boxCorners(shape.Width, shape.Depth)}, nil
	case planRounded:
		return [][]outlineCorner{shape.rounded()}, nil
	case planPolygon:
		return [][]outlineCorner{shape.polygon()}, nil
	case planStar:
		square := boxCorners(shape.Width, shape.Width)
		return [][]outlineCorner{square, turned(square, starTurn)}, nil
	case planY:
		return shape.wings(), nil
	case planSail:
		return [][]outlineCorner{shape.sail()}, nil
	case planLobed:
		return shape.lobes(), nil
	case planEllipse:
		return [][]outlineCorner{shape.ellipse()}, nil
	}
	return nil, fmt.Errorf("cities: no plan %q, only %v", shape.Plan,
		PlanNames())
}

// boxCorners is a rectangle's corners about the origin, wound so that
// each edge's outward side is on its left as an edge is walked from above
// with z toward the viewer.
func boxCorners(width, depth float64) []outlineCorner {
	return []outlineCorner{{-width / 2, -depth / 2}, {-width / 2, depth / 2},
		{width / 2, depth / 2}, {width / 2, -depth / 2}}
}

// rounded is the rectangle with each corner cut into Steps flat pieces
// of a quarter circle of the shape's radius.
func (shape footprintShape) rounded() []outlineCorner {
	halfWide, halfDeep := shape.Width/2, shape.Depth/2
	centres := []outlineCorner{{-halfWide + shape.Round, -halfDeep + shape.Round},
		{-halfWide + shape.Round, halfDeep - shape.Round},
		{halfWide - shape.Round, halfDeep - shape.Round},
		{halfWide - shape.Round, -halfDeep + shape.Round}}
	var corners []outlineCorner
	for index, centre := range centres {
		start := -math.Pi/2 - float64(index)*math.Pi/2
		for step := 0; step <= shape.Steps; step++ {
			angle := start - float64(step)*math.Pi/2/float64(shape.Steps)
			corners = append(corners, outlineCorner{
				centre.X + shape.Round*math.Cos(angle),
				centre.Z + shape.Round*math.Sin(angle)})
		}
	}
	return corners
}

// polygon is a regular polygon of the shape's sides, across its corners
// the shape's width.
func (shape footprintShape) polygon() []outlineCorner {
	corners := make([]outlineCorner, shape.Sides)
	for index := range corners {
		angle := math.Pi + fullTurn*float64(index)/float64(shape.Sides)
		corners[index] = outlineCorner{shape.Width / 2 * math.Sin(angle),
			shape.Width / 2 * math.Cos(angle)}
	}
	return corners
}

// wings are the three rectangles of a Y, each from a little behind the
// middle out to the wing's length, turned a third of a turn from the last.
func (shape footprintShape) wings() [][]outlineCorner {
	parts := make([][]outlineCorner, wingCount)
	for index, length := range shape.Wings {
		behind := shape.WingWidth / 2
		wing := []outlineCorner{{-behind, -shape.WingWidth / 2},
			{-behind, shape.WingWidth / 2},
			{length, shape.WingWidth / 2}, {length, -shape.WingWidth / 2}}
		parts[index] = turned(wing, float64(index)*360/wingCount)
	}
	return parts
}

// sail is a rectangle whose front face is bowed outward in Steps flat
// pieces, by the shape's bow at its middle.
func (shape footprintShape) sail() []outlineCorner {
	corners := []outlineCorner{{-shape.Width / 2, -shape.Depth / 2},
		{-shape.Width / 2, shape.Depth / 2}}
	for step := 1; step < shape.Steps; step++ {
		across := float64(step)/float64(shape.Steps)*2 - 1
		corners = append(corners, outlineCorner{across * shape.Width / 2,
			shape.Depth/2 + shape.Bow*(1-across*across)})
	}
	return append(corners, outlineCorner{shape.Width / 2, shape.Depth / 2},
		outlineCorner{shape.Width / 2, -shape.Depth / 2})
}

// ellipse is an ellipse of the shape's width and depth, cut into Sides
// flat sides, wound as a polygon is.
func (shape footprintShape) ellipse() []outlineCorner {
	corners := make([]outlineCorner, shape.Sides)
	for index := range corners {
		angle := math.Pi + fullTurn*float64(index)/float64(shape.Sides)
		corners[index] = outlineCorner{shape.Width / 2 * math.Sin(angle),
			shape.Depth / 2 * math.Cos(angle)}
	}
	return corners
}

// lobes are the parts of a lobed plan: the two squares of a star, and
// at each of its eight notches the sector of a circle of the shape's lobe
// radius that fills the notch, from one edge round to the other. A
// sector's inner corner stands inside both squares, so only its arc
// shows, and no part reaches past the star's tips.
func (shape footprintShape) lobes() [][]outlineCorner {
	square := boxCorners(shape.Width, shape.Width)
	parts := [][]outlineCorner{square, turned(square, starTurn)}
	half := shape.Width / 2
	notch := outlineCorner{half, half * notchShare}
	reach := math.Hypot(notch.X, notch.Z)
	first := math.Atan2(notch.Z, notch.X)
	for index := range 360 / notchEvery {
		outward := first + float64(index)*notchEvery*math.Pi/halfTurn
		centre := outlineCorner{reach * math.Cos(outward),
			reach * math.Sin(outward)}
		sector := []outlineCorner{{centre.X * lobeCore, centre.Z * lobeCore}}
		start := outward - lobeArc/2*math.Pi/halfTurn
		for step := 0; step <= shape.Steps; step++ {
			angle := start + float64(step)*lobeArc*math.Pi/halfTurn/
				float64(shape.Steps)
			sector = append(sector, outlineCorner{
				centre.X + shape.Lobe*math.Cos(angle),
				centre.Z + shape.Lobe*math.Sin(angle)})
		}
		if signedArea(sector) > 0 {
			slices.Reverse(sector)
		}
		parts = append(parts, sector)
	}
	return parts
}

// turned is the corners turned about the origin by an angle in degrees,
// counterclockwise looking down from above.
func turned(corners []outlineCorner, degrees float64) []outlineCorner {
	angle := degrees * math.Pi / halfTurn
	sin, cos := math.Sin(angle), math.Cos(angle)
	out := make([]outlineCorner, len(corners))
	for index, corner := range corners {
		out[index] = outlineCorner{corner.X*cos + corner.Z*sin,
			corner.Z*cos - corner.X*sin}
	}
	return out
}

// moved is the corners shifted along x and z.
func moved(corners []outlineCorner, along outlineCorner) []outlineCorner {
	out := make([]outlineCorner, len(corners))
	for index, corner := range corners {
		out[index] = outlineCorner{corner.X + along.X, corner.Z + along.Z}
	}
	return out
}

// signedArea is the shoelace sum of the corners: negative for the
// winding the plans are made in.
func signedArea(corners []outlineCorner) float64 {
	var sum float64
	for index, corner := range corners {
		next := corners[(index+1)%len(corners)]
		sum += corner.X*next.Z - next.X*corner.Z
	}
	return sum
}

// inset is a convex polygon drawn in by a distance on every side: each
// edge moved inward, and the corners found where the moved edges meet. It
// refuses a polygon with no room left.
func inset(corners []outlineCorner, by float64) ([]outlineCorner, error) {
	sign := math.Copysign(1, signedArea(corners))
	lines := make([][3]float64, len(corners))
	for index, corner := range corners {
		next := corners[(index+1)%len(corners)]
		run := math.Hypot(next.X-corner.X, next.Z-corner.Z)
		normalX, normalZ := sign*(next.Z-corner.Z)/run,
			-sign*(next.X-corner.X)/run
		lines[index] = [3]float64{normalX, normalZ,
			normalX*corner.X + normalZ*corner.Z - by}
	}
	out := make([]outlineCorner, len(corners))
	for index := range corners {
		before := lines[(index+len(corners)-1)%len(corners)]
		after := lines[index]
		det := before[0]*after[1] - before[1]*after[0]
		if math.Abs(det) < parallelLimit {
			return nil, fmt.Errorf("cities: cannot draw in an edge that "+
				"runs on from the one before it, at corner %d", index)
		}
		out[index] = outlineCorner{(before[2]*after[1] - before[1]*after[2]) / det,
			(before[0]*after[2] - before[2]*after[0]) / det}
	}
	if math.Signbit(signedArea(out)) != math.Signbit(signedArea(corners)) {
		return nil, fmt.Errorf("cities: no room to draw a plan in by %v",
			by)
	}
	return out, nil
}

// parallelLimit is how near zero a determinant is to be no determinant:
// two edges that run on from each other.
const parallelLimit = 1e-9

// spans are the stretches of an edge, as shares from nought to one along
// it, that no part of the others covers, given the others as polygons at
// the height the edge is seen at.
func spans(from, to outlineCorner, others [][]outlineCorner) [][2]float64 {
	free := [][2]float64{{0, 1}}
	for _, other := range others {
		low, high, covered := coverage(from, to, other)
		if !covered {
			continue
		}
		var left [][2]float64
		for _, piece := range free {
			if piece[0] < low {
				left = append(left, [2]float64{piece[0],
					math.Min(piece[1], low)})
			}
			if piece[1] > high {
				left = append(left, [2]float64{math.Max(piece[0], high),
					piece[1]})
			}
		}
		free = left
	}
	return slices.DeleteFunc(free, func(piece [2]float64) bool {
		return !(piece[1] > piece[0])
	})
}

// coverage is the stretch of the edge from one corner to another that
// lies inside a convex polygon, as shares along it, and whether any does.
func coverage(from, to outlineCorner, polygon []outlineCorner) (low, high float64, ok bool) {
	sign := math.Copysign(1, signedArea(polygon))
	low, high = 0, 1
	for index, corner := range polygon {
		next := polygon[(index+1)%len(polygon)]
		edgeX, edgeZ := next.X-corner.X, next.Z-corner.Z
		start := sign * (edgeX*(from.Z-corner.Z) - edgeZ*(from.X-corner.X))
		end := sign * (edgeX*(to.Z-corner.Z) - edgeZ*(to.X-corner.X))
		switch {
		case start <= 0 && end <= 0:
			return 0, 0, false
		case start < 0:
			low = math.Max(low, start/(start-end))
		case end < 0:
			high = math.Min(high, start/(start-end))
		}
	}
	return low, high, high > low
}
