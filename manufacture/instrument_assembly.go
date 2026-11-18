package manufacture

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// An instrumentAssembly owns its parts and the first construction error.
// Every part is a closed mesh; the GPU uses its ordinary mesh route.
type instrumentAssembly struct {
	object  good
	fine    cut
	trouble error
}

// add retains a part in its own frame and material slot.
func (assembly *instrumentAssembly) add(shape form,
	space maths.Subspace, material slot,
) {
	assembly.object.parts = append(assembly.object.parts,
		part{shape: shape, space: space,
			fine: assembly.fine, of: material})
}

// box places the primitive from its minimum floor corner.
func (assembly *instrumentAssembly) box(width, height, depth float64,
	at maths.Vec, material slot,
) {
	assembly.boxAt(width, height, depth, at, maths.Local(), material)
}

// boxAt places a closed box from its minimum corner in a supplied frame.
func (assembly *instrumentAssembly) boxAt(width, height, depth float64,
	corner maths.Vec, pose maths.Subspace, material slot,
) {
	centre := corner.Add(maths.V(width*spaceHalf, spaceZero, depth*spaceHalf))
	local := maths.NewSubspace(centre, maths.Identity, spaceOne)
	assembly.add(boxShape(width, height, depth), local.Within(pose), material)
}

// rod aligns a checked tube between explicit endpoints.
func (assembly *instrumentAssembly) rod(from, to maths.Vec,
	radius float64, material slot,
) {
	assembly.rodAlong(from, to.Sub(from), radius, material)
}

// rodAlong lays a checked tube from a point along a run. tubes laid
// along the same run are the same to the last bit, and so share one
// mesh; endpoints subtracted need not be.
func (assembly *instrumentAssembly) rodAlong(from, run maths.Vec,
	radius float64, material slot,
) {
	if assembly.trouble != nil {
		return
	}
	length := measure.Metres(run.Len())
	girth := measure.Metres(radius)
	tube, err := newTube(length, girth, girth)
	if err != nil {
		assembly.trouble = err
		return
	}
	space, placementError := spaceAlong(from, run)
	if placementError != nil {
		assembly.trouble = placementError
		return
	}
	assembly.add(tube, space, material)
}

// finish returns no object when any component construction failed.
func (assembly *instrumentAssembly) finish(bounds maths.Box) (*good, error) {
	if assembly.trouble != nil {
		return nil, assembly.trouble
	}
	assembly.object.extent = &bounds
	return &assembly.object, nil
}
