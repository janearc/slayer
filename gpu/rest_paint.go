package gpu

import (
	"fmt"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// restPaintRefusal refuses a posed mesh whose material is painted: the
// processor reads such paint at the point's rest place (trace's
// paintPoint), and the GPU path has only the posed place to give it, so
// the markings would land where the skin now is, not where they belong.
// A posed mesh with no paint, or a painted mesh that is not posed, draws.
func restPaintRefusal(made *mesh.Mesh, material *thing.Material) error {
	if material == nil || material.Texture == nil || !made.Posed() {
		return nil
	}
	return unsupported{fmt.Sprintf("a posed mesh painted %q at its rest "+
		"points, which is CPU-only: draw it on the processor",
		material.Name)}
}
