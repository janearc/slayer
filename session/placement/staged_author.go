package placement

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// AuthorAt returns the nearest hit's authored identity, when one is known.
// An unknown surface in front blocks an authored surface behind it. The ray
// is in the mesh's coordinate frame and need not have a unit direction.
func (mesh *DisplayMesh) AuthorAt(ray maths.Ray) (thing.AuthorRef, bool) {
	if mesh == nil {
		return thing.AuthorRef{}, false
	}
	for _, value := range []float64{ray.At.X, ray.At.Y, ray.At.Z, ray.Dir.X, ray.Dir.Y, ray.Dir.Z} {
		if !displayFinite(value) {
			return thing.AuthorRef{}, false
		}
	}
	length := ray.Dir.Len()
	if length == 0 || math.IsInf(length, 0) {
		return thing.AuthorRef{}, false
	}
	ray.Dir = ray.Dir.Scale(1 / length)
	parts, ok := mesh.hitter.(displayComponents)
	if !ok {
		return thing.AuthorRef{}, false
	}
	_, index, hit := parts.hitComponent(ray, 0, math.Inf(1))
	if !hit || !parts[index].authored {
		return thing.AuthorRef{}, false
	}
	return parts[index].author, true
}
