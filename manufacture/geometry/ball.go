package geometry

import (
	"github.com/janearc/slayer/maths"
	"math"
)

// icosahedron is the twenty-sided ball every lump starts as: its
// twelve corners on the unit ball and its faces by corner, each wound
// outward.
func Icosahedron[F Triangle]() []F {
	golden := (1 + math.Sqrt(5)) / 2
	corners := []maths.Vec{
		maths.V(-1, golden, 0), maths.V(1, golden, 0),
		maths.V(-1, -golden, 0), maths.V(1, -golden, 0),
		maths.V(0, -1, golden), maths.V(0, 1, golden),
		maths.V(0, -1, -golden), maths.V(0, 1, -golden),
		maths.V(golden, 0, -1), maths.V(golden, 0, 1),
		maths.V(-golden, 0, -1), maths.V(-golden, 0, 1),
	}
	faces := [][3]int{{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10},
		{0, 10, 11}, {1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6},
		{7, 1, 8}, {3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8},
		{3, 8, 9}, {4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7},
		{9, 8, 1}}
	var ball []F
	for _, corner := range faces {
		ball = append(ball, F{corners[corner[0]].Unit(),
			corners[corner[1]].Unit(), corners[corner[2]].Unit()})
	}
	return ball
}

// roundTheMiddle is how many sides an icosahedron has round its
// middle, which each cutting doubles.
const roundTheMiddle = 5

// ball is the unit ball cut finely enough to have as many sides round its
// middle as the cut asks: every face of the icosahedron made four, their new
// corners pushed out to the ball, as often as it takes.
//
// a corner two faces share is worked out from the same two corners either way,
// so the ball closes exactly.
func Ball[F Triangle](around int) []F {
	made := Icosahedron[F]()
	for round := roundTheMiddle; round < around; round *= 2 {
		var finer []F
		for _, one := range made {
			ab := one[0].Add(one[1]).Unit()
			bc := one[1].Add(one[2]).Unit()
			ca := one[2].Add(one[0]).Unit()
			finer = append(finer, F{one[0], ab, ca},
				F{one[1], bc, ab}, F{one[2], ca, bc},
				F{ab, bc, ca})
		}
		made = finer
	}
	return made
}
