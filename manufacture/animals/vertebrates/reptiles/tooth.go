package reptiles

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// Tooth makes a capped pointed tooth; lean offsets its tip sideways.
func Tooth(root maths.Vec, radius, length float64, sides int, lean float64) (*shape.Mesh, error) {
	vertices := make([]maths.Vec, 0, sides+2)
	faces := make([][3]int, 0, 2*sides)
	for j := 0; j < sides; j++ {
		angle := 2 * math.Pi * float64(j) / float64(sides)
		vertices = append(vertices, root.Add(maths.V(radius*math.Cos(angle), 0, radius*math.Sin(angle))))
	}
	base := len(vertices)
	vertices = append(vertices, root)
	tip := len(vertices)
	vertices = append(vertices, root.Add(maths.V(lean, -length, 0)))
	for j := 0; j < sides; j++ {
		next := (j + 1) % sides
		faces = append(faces, [3]int{j, next, tip}, [3]int{base, next, j})
	}
	return shape.NewMesh(vertices, faces, nil, false)
}
