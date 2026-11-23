package cities

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/maths"
)

// shell is a closed convex solid of two rings of corners, one above the
// other, joined corner to corner: a piece of a hollow cone or cylinder,
// where Prism, which faces its walls away from the middle of its bottom,
// would face the walls of a piece that leans far from its own middle
// the wrong way. Every face of a shell faces away from the middle of all
// its corners.
type shell struct{ Bottom, Top []maths.Vec }

// Mesh makes the shell's walls and caps, wound outward. It refuses rings
// that have not as many corners as each other, or fewer than a ring needs.
func (shell shell) Mesh(Detail) (*Mesh, error) {
	if len(shell.Bottom) != len(shell.Top) || len(shell.Bottom) < leastSides {
		return nil, fmt.Errorf("cities: a shell joins corner to corner, "+
			"with at least %d, and its bottom has %d and its top %d",
			leastSides, len(shell.Bottom), len(shell.Top))
	}
	inside := middle(slices.Concat(shell.Bottom, shell.Top))
	mesh := &Mesh{Triangles: sides(shell.Bottom, shell.Top, inside)}
	mesh.Triangles = append(mesh.Triangles,
		fill(shell.Bottom, middle(shell.Bottom).Sub(inside))...)
	mesh.Triangles = append(mesh.Triangles,
		fill(shell.Top, middle(shell.Top).Sub(inside))...)
	return mesh, nil
}
