package manufacture

import "github.com/janearc/slayer/maths"

// retailQuarter is a quarter: the middle of a quad is its four corners
// divided by four.
const retailQuarter = 0.25

// retailSlab is a closed plank that runs along x from left to right and
// slopes in z and y: its underside runs from the height low at the place
// back to the height high at the place front, and it is thick as given,
// measured straight up. An awning, an arm that holds it and a board that
// leans are slabs. The corners go round the plank's profile, and each
// side and each end is a quad wound to face out.
func retailSlab(left, right, back, low, front, high, thick float64) surface {
	var ends [retailRangeNumbers][4]maths.Vec
	for index, across := range [retailRangeNumbers]float64{left, right} {
		ends[index] = [4]maths.Vec{maths.V(across, low, back),
			maths.V(across, high, front),
			maths.V(across, high+thick, front),
			maths.V(across, low+thick, back)}
	}
	middle := maths.V((left+right)*spaceHalf, (low+high+thick)*spaceHalf,
		(back+front)*spaceHalf)
	skin := surface{}
	for side := range ends[0] {
		next := (side + 1) % len(ends[0])
		first, second := ends[0][side], ends[0][next]
		third, fourth := ends[1][next], ends[1][side]
		centre := first.Add(second).Add(third).Add(fourth).Scale(retailQuarter)
		skin = append(skin, cityQuad(first, second, third, fourth,
			centre.Sub(middle))...)
	}
	skin = append(skin, cityQuad(ends[0][0], ends[0][1], ends[0][2],
		ends[0][3], maths.XAxis.Neg())...)
	return append(skin, cityQuad(ends[1][0], ends[1][1], ends[1][2],
		ends[1][3], maths.XAxis)...)
}
