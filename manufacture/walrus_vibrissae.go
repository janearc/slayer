package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// The bristles of the muzzle pads. The sources, as read:
//   - "There can be 400 to 700 vibrissae in 13 to 15 rows" (Kastelein,
//     Stevens and Mosterd 1990, Aquatic Mammals, on Wikipedia's Walrus page)
//   - "some 480-odd vibrissae roughly the texture of uncooked spaghetti"
//     (California Wild, California Academy of Sciences, fall 2002)
//   - "Each whisker averages 8 centimetres ... and about 3 millimetres
//     thick" (Guinness World Records, most whiskers on a pinniped)
//   - "large, fleshy, forward-facing mystacial pads" sprouting "short,
//     stiff, whitish" bristles (OBIS-SEAMAP species profile)
//
// So: fourteen rows of seventeen a side, 476 in all, each about 8 cm and
// 3 mm at its root, stiff and straight. The rows lie on the pad in the
// skull's frame, staggered and a little uneven; their spacing is chosen to
// fill the pad, which shows between them.
const (
	// walrusFollicleRows is the number of rows down the pad.
	walrusFollicleRows = 14
	// walrusFollicleColumns is the number of follicles in a row.
	walrusFollicleColumns = 17
	// walrusFollicleTop is the top row's drop below the skull's axis:
	// above it, so less than nought. Chosen, with the two steps, so the
	// rows cover the face of the pads from under the nostrils to the lip.
	walrusFollicleTop = -.06
	// walrusFollicleRowStep is the distance between rows.
	walrusFollicleRowStep = .0165
	// walrusFollicleColumnStep is the distance between follicles in a row.
	walrusFollicleColumnStep = .0088
	// walrusFollicleInner is how far from the middle the first column sits.
	walrusFollicleInner = .012
	// walrusFollicleJitter is how far a follicle strays from the lattice,
	// as a share of a step. Chosen.
	walrusFollicleJitter = .25
	// walrusStrandSegments is the straight pieces in each bristle and
	// walrusStrandSides the flat faces round it.
	walrusStrandSegments = 6
	walrusStrandSides    = 6
	// walrusStrandRadius is a bristle's radius at its root: 3 mm thick.
	walrusStrandRadius = .0015
	// walrusStrandLength is a bristle's average length: 8 cm.
	walrusStrandLength = .08
)

// walrusFollicle is the drop below the skull's axis and the signed offset
// to the side of the follicle at a row and column on one side of the pad.
// Odd rows are shifted half a step, and every follicle strays a little.
func walrusFollicle(row, column int, side float64) (drop, across float64) {
	sign := int(side)
	shift := .5*float64(row%2) +
		walrusFollicleJitter*(2*walrusHash(row, column, sign, 11)-1)
	lift := walrusFollicleJitter * (2*walrusHash(row, column, sign, 12) - 1)
	across = walrusFollicleInner + (float64(column)+shift)*walrusFollicleColumnStep
	drop = walrusFollicleTop + (float64(row)+lift)*walrusFollicleRowStep
	return drop, side * across
}

// walrusFollicleDistance is how far, across the pad, rest point p lies from
// the nearest follicle, measured in the skull's frame.
func walrusFollicleDistance(p maths.Vec) float64 {
	_, across, drop := walrusFrame.local(p)
	side := 1.
	if across < 0 {
		side = -1
	}
	row := int(math.Round((drop - walrusFollicleTop) / walrusFollicleRowStep))
	column := int(math.Round((math.Abs(across) - walrusFollicleInner) /
		walrusFollicleColumnStep))
	nearest := math.Inf(1)
	for r := max(0, row-1); r <= min(walrusFollicleRows-1, row+1); r++ {
		for c := max(0, column-1); c <= min(walrusFollicleColumns-1, column+1); c++ {
			d, a := walrusFollicle(r, c, side)
			nearest = math.Min(nearest, math.Hypot(drop-d, across-a))
		}
	}
	return nearest
}

// walrusPadSurface is how far forward along the skull's axis, as a share
// of the skull's length, the hide lies at that offset to the side and drop
// below the axis, looking from behind the pad, or false if no hide lies on
// that line.
func walrusPadSurface(field func(maths.Vec) float64, across,
	drop float64) (float64, bool) {
	const (
		inside, outside = .70, 1.45
		steps           = 26
	)
	at := func(t float64) maths.Vec { return walrusFrame.at(t, across, drop) }
	if field(at(inside)) >= 0 || field(at(outside)) < 0 {
		return 0, false
	}
	low, high := inside, outside
	for i := 0; i < steps; i++ {
		middle := (low + high) / 2
		if field(at(middle)) < 0 {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / 2, true
}

// walrusAddStrand adds one closed six-sided bristle to the mesh being
// built: from a little below the hide, thick at its root and tapering to a
// blunt tip, stiff, so nearly straight.
func walrusAddStrand(vs []maths.Vec, fs [][3]int, root, grow maths.Vec, length float64,
	skin func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	const (
		buried       = .004
		droop        = .06
		tip          = .45
		sides, steps = walrusStrandSides, walrusStrandSegments
	)
	up := maths.V(0, 1, 0)
	across := up.Sub(grow.Scale(up.Dot(grow))).Unit()
	around := grow.Cross(across)
	first := len(vs)
	start := root.Sub(grow.Scale(buried))
	for i := 0; i <= steps; i++ {
		t := float64(i) / steps
		centre := start.Add(grow.Scale(length * t)).Add(maths.V(0, -droop*length*t*t, 0))
		radius := walrusStrandRadius * (1 - (1-tip)*t)
		for j := 0; j < sides; j++ {
			angle := 2 * math.Pi * float64(j) / sides
			offset := across.Scale(radius * math.Cos(angle)).Add(around.Scale(radius * math.Sin(angle)))
			vs = append(vs, skin(centre.Add(offset)))
		}
	}
	for i := 0; i < steps; i++ {
		for j := 0; j < sides; j++ {
			a := first + i*sides + j
			b := first + i*sides + (j+1)%sides
			c := first + (i+1)*sides + (j+1)%sides
			d := first + (i+1)*sides + j
			fs = append(fs, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	rootMiddle := len(vs)
	vs = append(vs, skin(start))
	tipMiddle := len(vs)
	vs = append(vs, skin(start.Add(grow.Scale(length)).Add(maths.V(0, -droop*length, 0))))
	for j := 0; j < sides; j++ {
		k := (j + 1) % sides
		fs = append(fs, [3]int{rootMiddle, first + k, first + j},
			[3]int{tipMiddle, first + steps*sides + j, first + steps*sides + k})
	}
	return vs, fs
}

// walrusVibrissae is every bristle of one muzzle pad as one mesh: a stiff
// strand rooted in each follicle, leaving the pad along its own normal
// and tipped steeply down, so the rows hang over one another as a
// moustache does, splayed a little outward toward the pad's edge and a
// little at random. Lengths spread round 8 cm. The strands move with the head by skin. A follicle with no
// hide under it is refused, not left out.
func walrusVibrissae(side float64, field func(maths.Vec) float64,
	skin func(maths.Vec) maths.Vec) (*shape.Mesh, error) {
	const (
		forward, downward = 0., 1.6
		outward, edge     = .20, .15
		wander, spread    = .04, .3
	)
	var vs []maths.Vec
	var fs [][3]int
	sign := int(side)
	for row := 0; row < walrusFollicleRows; row++ {
		for column := 0; column < walrusFollicleColumns; column++ {
			drop, across := walrusFollicle(row, column, side)
			t, ok := walrusPadSurface(field, across, drop)
			if !ok {
				return nil, fmt.Errorf("%w: the follicle in row %d, column "+
					"%d on side %g", errWalrusLandmark, row, column, side)
			}
			root := walrusFrame.at(t, across, drop)
			stray := maths.V(walrusHash(row, column, sign, 13)-.5,
				walrusHash(row, column, sign, 14)-.5,
				walrusHash(row, column, sign, 15)-.5).Scale(2 * wander)
			grow := walrusGradient(field, root).
				Add(walrusFrame.axis.Scale(forward)).
				Add(walrusFrame.down.Scale(downward)).
				Add(maths.V(0, 0, side*outward*math.Abs(across)/edge)).
				Add(stray).Unit()
			length := walrusStrandLength *
				(1 - spread/2 + spread*walrusHash(row, column, sign, 5))
			vs, fs = walrusAddStrand(vs, fs, root, grow, length, skin)
		}
	}
	return shape.NewMesh(vs, fs, nil, false)
}
