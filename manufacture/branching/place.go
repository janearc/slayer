package branching

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Size is a part's length and the radius at its foot.
type Size struct{ Length, Girth measure.Length }

// Place is a part's own space within its carrier, and its size there.
// Arrangements that do not size parts leave Size empty and scale Space.
type Place struct {
	Space maths.Subspace
	Size  Size
}
