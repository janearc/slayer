package gpu

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// tube stands a tube in the world as its mesh. A tube has no exact hit
// and the kernel has no march, so it is drawn as the smooth mesh that
// shape.Tube.Mesh makes at shape.TubeChordShare. The tube itself, a
// value of three numbers, is the key, so equal tubes are grown once
// and stood as many times as they are placed, however they are placed.
func (encoded *scene) tube(made shape.Tube, space maths.Subspace,
	material int, clear bool) error {
	var failure error
	grown := func() pendingMesh {
		skin, err := made.Mesh(shape.TubeChordShare())
		if err != nil {
			failure = err
			return pendingMesh{tree: grownOver(nil)}
		}
		return encoded.shapeTree(skin)
	}
	encoded.stand(made, grown, space, material, clear)
	if failure != nil {
		// not kept, so each placing of it is refused and none is
		// taken for a mesh that was grown
		delete(encoded.meshes, made)
		return unsupported{fmt.Sprintf("a tube that cannot be meshed: %v", failure)}
	}
	return nil
}
