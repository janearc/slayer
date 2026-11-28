package placement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// compactWorldOrigin anchors a distant, compact aggregate independently of
// camera motion. If the top-level placement has no useful origin, try the
// first visible surface's native float64 bounds midpoint.
func compactWorldOrigin(parts []*thing.Thing) maths.Vec {
	for _, part := range parts {
		if part.Hidden() {
			continue
		}
		at := part.At(0)
		if part.Shape == nil || !shape.ValidPlacement(at) {
			break
		}
		if worldCoordinateMagnitude(at.Origin) > DisplayFar && worldFitsOrigin(parts, at.Origin) {
			return at.Origin
		}
		bounds := part.Shape.Bounds()
		centre := at.Out(bounds.Min.Scale(.5).Add(bounds.Max.Scale(.5)))
		if worldCoordinateMagnitude(centre) > DisplayFar && worldFitsOrigin(parts, centre) {
			return centre
		}
		break
	}
	return maths.Vec{}
}

// worldFitsOrigin holds the aggregate within DisplayFar along each local
// axis. A dispersed world retains its original frame and ordered mesh.
func worldFitsOrigin(parts []*thing.Thing, origin maths.Vec) bool {
	for _, part := range parts {
		if part.Hidden() {
			continue
		}
		at := part.At(0)
		if part.Shape == nil || !shape.ValidPlacement(at) {
			return false
		}
		at.Origin = at.Origin.Sub(origin)
		bounds := part.Shape.Bounds()
		for _, x := range [...]float64{bounds.Min.X, bounds.Max.X} {
			for _, y := range [...]float64{bounds.Min.Y, bounds.Max.Y} {
				for _, z := range [...]float64{bounds.Min.Z, bounds.Max.Z} {
					if !(worldCoordinateMagnitude(at.Out(maths.V(x, y, z))) <= DisplayFar) {
						return false
					}
				}
			}
		}
	}
	return true
}

func worldCoordinateMagnitude(point maths.Vec) float64 {
	return max(math.Abs(point.X), math.Abs(point.Y), math.Abs(point.Z))
}

// worldMeshKey includes the CPU hit metadata as well as the GPU geometry.
// Changing only a material name must still replace the cached hit result.
func worldMeshKey(skin *DisplayMesh, components displayComponents) (string, error) {
	type hitKey struct {
		Shape, Material string
		At              maths.Subspace
	}
	hits := make([]hitKey, 0, len(components))
	for _, part := range components {
		hits = append(hits, hitKey{Shape: fmt.Sprintf("%T", part.hitter), Material: part.material, At: part.at})
	}
	raw, err := json.Marshal(struct {
		Mesh   *DisplayMesh
		Limits MaterialLimits
		Hits   []hitKey
	}{skin, skin.materialLimits, hits})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}
