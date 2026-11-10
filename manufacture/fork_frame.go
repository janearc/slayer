package manufacture

import "github.com/janearc/slayer/maths"

// forkFrame is one closed U-shaped extrusion, without internal box caps.
type forkFrame struct {
	width, height, arm, depth float64
}

type forkPanel struct{ left, right, low, high float64 }

// skin splits both caps and the boundary at every panel junction.
func (frame forkFrame) skin(cut) surface {
	left, right := -frame.width*spaceHalf, frame.width*spaceHalf
	insideLeft, insideRight := left+frame.arm, right-frame.arm
	panels := []forkPanel{
		{left, insideLeft, spaceZero, frame.arm},
		{insideLeft, insideRight, spaceZero, frame.arm},
		{insideRight, right, spaceZero, frame.arm},
		{left, insideLeft, frame.arm, frame.height},
		{insideRight, right, frame.arm, frame.height}}
	var skin surface
	for _, panel := range panels {
		skin = frame.cap(skin, panel)
	}
	outline := []maths.Vec{
		maths.V(left, spaceZero, spaceZero),
		maths.V(insideLeft, spaceZero, spaceZero),
		maths.V(insideRight, spaceZero, spaceZero),
		maths.V(right, spaceZero, spaceZero),
		maths.V(right, frame.arm, spaceZero),
		maths.V(right, frame.height, spaceZero),
		maths.V(insideRight, frame.height, spaceZero),
		maths.V(insideRight, frame.arm, spaceZero),
		maths.V(insideLeft, frame.arm, spaceZero),
		maths.V(insideLeft, frame.height, spaceZero),
		maths.V(left, frame.height, spaceZero),
		maths.V(left, frame.arm, spaceZero)}
	front := maths.ZAxis.Scale(-frame.depth * spaceHalf)
	back := front.Neg()
	for index, here := range outline {
		next := outline[(index+spaceOne)%len(outline)]
		skin = quad(skin, here.Add(front), next.Add(front),
			next.Add(back), here.Add(back))
	}
	return skin
}

// cap winds front and back in opposite directions.
func (frame forkFrame) cap(skin surface, panel forkPanel) surface {
	front, back := -frame.depth*spaceHalf, frame.depth*spaceHalf
	skin = quad(skin, maths.V(panel.left, panel.low, front),
		maths.V(panel.left, panel.high, front),
		maths.V(panel.right, panel.high, front),
		maths.V(panel.right, panel.low, front))
	return quad(skin, maths.V(panel.left, panel.low, back),
		maths.V(panel.right, panel.low, back),
		maths.V(panel.right, panel.high, back),
		maths.V(panel.left, panel.high, back))
}
