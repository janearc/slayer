package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

const humanAround = 288

type humanVertex struct {
	point, chart maths.Vec
	weights      jointWeights
	tissue       tissueMotion
	site         proportionSite
}

type humanMesh struct {
	vertices    []humanVertex
	faces       [][3]int
	lashMargins [2]lidMargin
	digitHair   []digitHairPatch
	earFaces    [2]int
}

func (model *humanMesh) add(vertex humanVertex) int {
	index := len(model.vertices)
	model.vertices = append(model.vertices, vertex)
	return index
}

func (model *humanMesh) point(point maths.Vec, weights jointWeights) int {
	return model.add(humanVertex{point: point, chart: point, weights: weights})
}

func (model *humanMesh) limbPoint(point maths.Vec, weights jointWeights, site proportionSite) int {
	return model.add(humanVertex{point: point, chart: point, weights: weights, site: site})
}

func (model *humanMesh) connect(lower, upper []int) {
	for i, first := range lower {
		next := (i + 1) % len(lower)
		model.quad(first, lower[next], upper[next], upper[i])
	}
}

func (model *humanMesh) quad(first, second, third, fourth int) {
	model.faces = append(model.faces, [3]int{first, second, third}, [3]int{first, third, fourth})
}

func (model *humanMesh) cap(rim []int, pole int) {
	for i, first := range rim {
		model.faces = append(model.faces, [3]int{first, rim[(i+1)%len(rim)], pole})
	}
}

func (model *humanMesh) centre(rim []int) maths.Vec {
	var centre maths.Vec
	for _, index := range rim {
		centre = centre.Add(model.vertices[index].point)
	}
	return centre.Scale(1 / float64(len(rim)))
}

// angles transports the angular positions of a boundary into a circular
// sweep. The far ring is uniformly sampled and indexed from its front.
type rimAngles struct {
	near  []float64
	front int
}

func (model *humanMesh) angles(rim []int, centre, across, forward maths.Vec,
	width, depth float64,
) rimAngles {
	result := rimAngles{near: make([]float64, len(rim))}
	for i, index := range rim {
		delta := model.vertices[index].point.Sub(centre)
		result.near[i] = math.Atan2(delta.Dot(across)/width, delta.Dot(forward)/depth)
		if math.Abs(result.near[i]) < math.Abs(result.near[result.front]) {
			result.front = i
		}
	}
	return result
}

func (angles rimAngles) angle(index int, round float64) float64 {
	uniform := 2 * math.Pi * float64(index-angles.front) / float64(len(angles.near))
	offset := math.Remainder(angles.near[index]-uniform, 2*math.Pi)
	return uniform + (1-round)*offset
}

func turnRim(rim []int, front int) []int {
	ordered := make([]int, len(rim))
	for i := range ordered {
		ordered[i] = rim[(i+front)%len(rim)]
	}
	return ordered
}

func mix(left, right float64, share float64) float64 {
	return left + (right-left)*share
}

func mixPoint(left, right maths.Vec, share float64) maths.Vec {
	return left.Add(right.Sub(left).Scale(share))
}
