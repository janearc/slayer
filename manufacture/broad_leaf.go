package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// broadLeaf is a thin closed blade with a picked outline and raised midrib.
// Its physical dimensions never scale with the tree or detail selection.
type broadLeaf struct {
	length, width, thickness, camber float64
	outline                          string
	lobes                            int
	depth                            float64
}

type leafStation struct{ left, middle, right maths.Vec }

// station samples across the midrib, keeping root and tip exactly pointed.
func (leaf broadLeaf) station(index, sections int, front bool) leafStation {
	share := float64(index) / float64(sections)
	width := leaf.width * spaceHalf * math.Sin(math.Pi*share) * leaf.outlineWidth(share)
	if index == spaceZero || index == sections {
		width = spaceZero
	}
	middle := leaf.camber * math.Sin(math.Pi*share)
	if index == spaceZero || index == sections {
		middle = spaceZero
	}
	offset := leaf.thickness * spaceHalf
	if !front {
		offset = -offset
	}
	height := leaf.length * share
	return leafStation{left: maths.V(-width, height, offset),
		middle: maths.V(spaceZero, height, middle+offset),
		right:  maths.V(width, height, offset)}
}

// outlineWidth distinguishes oak lobes, narrow blades and deltoid foliage.
func (leaf broadLeaf) outlineWidth(share float64) float64 {
	switch leaf.outline {
	case "lobed":
		ripple := math.Cos(fullTurn * float64(leaf.lobes) * share)
		return spaceOne - leaf.depth + leaf.depth*ripple
	case "lance":
		return math.Sin(math.Pi * share)
	case "deltoid":
		shoulder := plantValue("deltoid-shoulder")
		if share < shoulder {
			return share / shoulder
		}
		return (spaceOne - share) / (spaceOne - shoulder)
	default:
		return spaceOne
	}
}

// skin uses strips along the midrib so deep lobes need no concave fan.
func (leaf broadLeaf) skin(fine cut) surface {
	var skin surface
	for index := range fine.around {
		top := leaf.station(index, fine.around, true)
		nextTop := leaf.station(index+spaceOne, fine.around, true)
		back := leaf.station(index, fine.around, false)
		nextBack := leaf.station(index+spaceOne, fine.around, false)
		skin = leafBand(skin, top, nextTop, index, fine.around, true)
		skin = leafBand(skin, back, nextBack, index, fine.around, false)
		skin = quad(skin, top.left, nextTop.left, nextBack.left, back.left)
		skin = quad(skin, top.right, back.right, nextBack.right, nextTop.right)
	}
	return skin
}

// leafBand triangulates pointed ends without zero-area root or tip faces.
func leafBand(skin surface, here, next leafStation,
	index, sections int, front bool,
) surface {
	if index == spaceZero {
		skin = leafFace(skin, here.middle, next.middle, next.left, front)
		return leafFace(skin, here.middle, next.right, next.middle, front)
	}
	if index+spaceOne == sections {
		skin = leafFace(skin, here.left, here.middle, next.middle, front)
		return leafFace(skin, here.middle, here.right, next.middle, front)
	}
	skin = leafFace(skin, here.left, here.middle, next.middle, front)
	skin = leafFace(skin, here.left, next.middle, next.left, front)
	skin = leafFace(skin, here.middle, here.right, next.right, front)
	return leafFace(skin, here.middle, next.right, next.middle, front)
}

// leafFace reverses the back surface while preserving the same shared edges.
func leafFace(skin surface, a, b, c maths.Vec, front bool) surface {
	if front {
		return append(skin, face{a, b, c})
	}
	return append(skin, face{a, c, b})
}

// bounds encloses every outline and both surface details.
func (leaf broadLeaf) bounds() maths.Box {
	low := maths.V(-leaf.width*spaceHalf, spaceZero, -leaf.thickness*spaceHalf)
	high := maths.V(leaf.width*spaceHalf, leaf.length,
		leaf.camber+leaf.thickness*spaceHalf)
	return maths.SegmentBox(low, high, spaceZero)
}
