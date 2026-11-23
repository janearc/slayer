package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// How a bone is drawn: a closed spindle from its head to its tail, round
// in section, with a waisted shaft and swollen ends where the joints
// are. The counts and shares are chosen, enough to read the shape.
const (
	drawAround = 10 // points round each ring
	drawWaist  = .2 // share of the thickness the shaft loses at its middle
	drawRound  = 8  // power that rounds the ends; higher squares them off
)

// drawRings are the shares along a bone at which its rings lie, close
// together at the ends, where the spindle closes.
var drawRings = []float64{.015, .06, .16, .33, .5, .67, .84, .94, .985}

// spindleRadius is the half thickness of the drawn bone at share t of
// the way from its head to its tail.
func spindleRadius(bone Bone, t float64) float64 {
	thick := bone.thickHead + t*(bone.thickTail-bone.thickHead)
	end := 1 - math.Pow(2*t-1, drawRound)
	return thick * (1 - drawWaist*math.Sin(math.Pi*t)) * math.Sqrt(end)
}

// across is two unit directions at right angles to way and each other.
func across(way maths.Vec) (maths.Vec, maths.Vec) {
	other := maths.V(0, 0, 1)
	if math.Abs(way.Z) > .9 {
		other = maths.V(1, 0, 0)
	}
	first := way.Cross(other).Unit()
	return first, way.Cross(first).Unit()
}

// Mesh is every bone, carried by frames (one for each bone, as Frames
// gives), as a closed spindle of its own: the points and the triangles
// joining them, wound outward.
func (skeleton *Skeleton) Mesh(frames []Frame) ([]maths.Vec, [][3]int) {
	points := []maths.Vec{}
	faces := [][3]int{}
	for i, bone := range skeleton.bones {
		start := len(points)
		for _, p := range spindlePoints(bone) {
			points = append(points, frames[i].Apply(p))
		}
		faces = append(faces, spindleFaces(start)...)
	}
	return points, faces
}

// spindlePoints are one bone's points at rest: the head's pole, each
// ring in turn, then the tail's pole.
func spindlePoints(bone Bone) []maths.Vec {
	way := bone.tail.Sub(bone.head)
	first, second := across(way.Unit())
	points := []maths.Vec{bone.head}
	for _, t := range drawRings {
		middle := bone.head.Add(way.Scale(t))
		radius := spindleRadius(bone, t)
		for k := 0; k < drawAround; k++ {
			angle := 2 * math.Pi * float64(k) / drawAround
			firstPart := first.Scale(radius * math.Cos(angle))
			secondPart := second.Scale(radius * math.Sin(angle))
			offset := firstPart.Add(secondPart)
			points = append(points, middle.Add(offset))
		}
	}
	return append(points, bone.tail)
}

// spindleCorner is the place of point k of ring r of the spindle whose
// points begin at start, after its head's pole.
func spindleCorner(start, r, k int) int {
	return start + 1 + r*drawAround + k%drawAround
}

// spindleFaces are the triangles of one spindle whose points begin at
// start: a fan at each pole and two triangles for each quad between.
func spindleFaces(start int) [][3]int {
	rings := len(drawRings)
	tail := start + 1 + rings*drawAround
	faces := [][3]int{}
	for k := 0; k < drawAround; k++ {
		faces = append(faces, [3]int{start, spindleCorner(start, 0, k+1),
			spindleCorner(start, 0, k)})
		faces = append(faces, [3]int{tail, spindleCorner(start, rings-1, k),
			spindleCorner(start, rings-1, k+1)})
		for r := 0; r+1 < rings; r++ {
			faces = append(faces, spindleQuad(start, r, k)...)
		}
	}
	return faces
}

// spindleQuad is the two triangles between rings r and r+1 at point k.
func spindleQuad(start, r, k int) [][3]int {
	a, b := spindleCorner(start, r, k), spindleCorner(start, r, k+1)
	c, d := spindleCorner(start, r+1, k), spindleCorner(start, r+1, k+1)
	return [][3]int{{a, b, c}, {b, d, c}}
}
