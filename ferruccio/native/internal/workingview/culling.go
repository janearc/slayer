package workingview

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

// frustum uses conservative bounding spheres in the camera's local frame.
type frustum struct {
	eye                  maths.Subspace
	horizontal, vertical float64
}

type sphere struct {
	centre maths.Vec
	radius float64
}

// viewFrustum matches the working projection's field of view and aspect.
func viewFrustum(scene *placement.DisplayScene, width, height int) frustum {
	wide := scene.Lens.Wide
	if !(wide > 0 && wide < math.Pi) {
		wide = 60 * maths.Degree
	}
	pixel := scene.Lens.Pixel
	if !(pixel > 0) {
		pixel = 1
	}
	aspect := float64(width) / float64(height)
	if scene.Width > 0 && scene.Height > 0 {
		aspect = float64(scene.Width) * pixel / float64(scene.Height)
	}
	horizontal := math.Tan(wide / 2)
	return frustum{eye: scene.Eye, horizontal: horizontal, vertical: horizontal / aspect}
}

// meshSphere encloses a mesh's local bounds.
func meshSphere(mesh *placement.DisplayMesh) (sphere, bool) {
	low, high, ok := mesh.Bounds()
	if !ok {
		return sphere{}, false
	}
	return sphere{centre: low.Add(high).Scale(.5), radius: high.Sub(low).Len() * .5}, true
}

// visible rejects only spheres wholly outside the camera frustum.
func (view frustum) visible(bounds sphere, at maths.Subspace) bool {
	centre := view.eye.In(at.Out(bounds.centre))
	scale := at.Scale
	if scale == 0 {
		scale = 1
	}
	eyeScale := view.eye.Scale
	if eyeScale == 0 {
		eyeScale = 1
	}
	radius := bounds.radius * scale / eyeScale
	depth := -centre.Z
	const near, far = .05, 100000.0
	if depth+radius < near || depth-radius > far {
		return false
	}
	return math.Abs(centre.X) <= depth*view.horizontal+radius*math.Hypot(1, view.horizontal) &&
		math.Abs(centre.Y) <= depth*view.vertical+radius*math.Hypot(1, view.vertical)
}

// batchVisible rejects an entire batch only when its bounds miss the view.
func (view frustum) batchVisible(batch placement.StagedBatch) bool {
	if !batch.HasBounds {
		return true
	}
	return view.boxVisible(batch.Bounds)
}

// boxVisible conservatively tests a world-space box against the view.
func (view frustum) boxVisible(box maths.Box) bool {
	low, high := box.Min, box.Max
	return view.visible(sphere{centre: low.Add(high).Scale(.5), radius: high.Sub(low).Len() * .5}, maths.Local())
}
