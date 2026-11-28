package placement

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The minimum exposes nearby ground at eye level. The maximum bounds
// the approximation for distant views. Both are half-widths in metres.
const (
	minPlaneReach      = 25
	maxPlaneReach      = 2000
	planeReachByHeight = 4
)

// planeAroundEye draws the renderer's local y=0 surface around the
// projection of the staged eye, with sizing in world units.
func planeAroundEye(place maths.Subspace, eye maths.Vec, colour maths.Light, material *thing.Material) (StagedBatch, bool, error) {
	if material != nil && (material.Through > 0 || material.Radiant != nil) {
		// The working view is opaque. Drawing a boundary here would hide
		// surfaces which the renderer can see through it in some bands.
		return StagedBatch{}, false, nil
	}
	normal := place.Orientation.Out(maths.Up).Unit()
	height := eye.Sub(place.Origin).Dot(normal)
	centre := eye.Sub(normal.Scale(height))
	reach := min(maxPlaneReach, max(minPlaneReach, math.Abs(height)*planeReachByHeight))
	if !planeCoordinatesFit(centre, reach) || !planeCoordinatesFit(eye, reach) {
		return StagedBatch{}, false, nil
	}
	mesh, err := flatProxyMesh(-reach, reach, -reach, reach, colour)
	if err != nil {
		return StagedBatch{}, false, err
	}
	mesh.Notice = "Infinite planes shown as finite working-view patches around the staged camera, with flat colour and read-only geometry."
	at := maths.NewSubspace(centre, place.Orientation, 1)
	return StagedBatch{Mesh: mesh, Places: []maths.Subspace{at}}, true, nil
}

// Native working-view transforms use float32. A patch needs room for
// movement on both sides of the centre in that coordinate system.
func planeCoordinatesFit(point maths.Vec, reach float64) bool {
	for _, axis := range [...]float64{point.X, point.Y, point.Z} {
		base := float32(axis)
		if !displayFinite(float64(base)) || !(float32(axis-reach) < base && base < float32(axis+reach)) {
			return false
		}
	}
	return true
}
