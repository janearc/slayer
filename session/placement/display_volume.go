package placement

import (
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// VolumeBoundLimit bounds the metadata and drawing cost of volume outlines.
const VolumeBoundLimit = 16

// DisplayVolumeBound is the medium's local integration box at a staged frame.
// At places that box in the world. The box does not describe visible density.
type DisplayVolumeBound struct {
	Kind string
	Box  maths.Box
	At   maths.Subspace
}

// Valid rejects malformed or degenerate integration boxes.
func (bound DisplayVolumeBound) Valid() bool {
	if (bound.Kind != "Wisp" && bound.Kind != "Fizz") || !shape.ValidPlacement(bound.At) {
		return false
	}
	for _, span := range [3][2]float64{
		{bound.Box.Min.X, bound.Box.Max.X},
		{bound.Box.Min.Y, bound.Box.Max.Y},
		{bound.Box.Min.Z, bound.Box.Max.Z},
	} {
		if !displayFinite(span[0]) || !displayFinite(span[1]) || span[0] >= span[1] {
			return false
		}
	}
	return true
}

// WorkingVolumeBounds reads small, bounded medium types using the same frame
// time for their density field and motion. It samples no density.
func WorkingVolumeBounds(world *thing.World, at time.Duration) []DisplayVolumeBound {
	var bounds []DisplayVolumeBound
	for _, volume := range world.Volumes {
		if volume == nil || volume.Hidden() {
			continue
		}
		field := volume.Medium
		kind := volumeBoundKind(field)
		if kind == "" {
			continue
		}
		if timed, ok := field.(medium.Timed); ok {
			field = timed.Tick(at)
		}
		bound := DisplayVolumeBound{Kind: kind, Box: field.Bounds(), At: volume.At(at)}
		if bound.Valid() {
			bounds = append(bounds, bound)
			if len(bounds) == VolumeBoundLimit {
				break
			}
		}
	}
	return bounds
}

// WorkingGiantAtmospheres counts visible producer gas layers without
// evaluating density or mistaking the giant for a solid surface.
func WorkingGiantAtmospheres(world *thing.World) int {
	count := 0
	for _, volume := range world.Volumes {
		if volume == nil || volume.Hidden() {
			continue
		}
		if _, giant := volume.Medium.(medium.Giant); giant {
			count++
		}
	}
	return count
}

func volumeBoundKind(field medium.Medium) string {
	if cloud, ok := field.(medium.ClearCloud); ok {
		field = cloud.DensityField()
	}
	switch field := field.(type) {
	case medium.Fizz:
		return "Fizz"
	case medium.Wispy:
		if _, puff := field.Of.(medium.Puff); puff {
			return "Wisp"
		}
	}
	return ""
}

// VolumeEdge has normalized screen coordinates for one visible box edge.
type VolumeEdge [2][2]float64

// ProjectVolumeBound clips the box's actual edges to the working frustum.
// It draws no new edges along a clipping plane.
func (scene *DisplayScene) ProjectVolumeBound(bound DisplayVolumeBound) []VolumeEdge {
	corners := scene.eyeCorners(bound.At, bound.Box.Min, bound.Box.Max)
	half, aspect := scene.projectionShape()
	var edges []VolumeEdge
	for i, corner := range corners {
		for bit := 1; bit < len(corners); bit <<= 1 {
			if i&bit != 0 {
				continue
			}
			end := corners[i|bit]
			first, last, ok := visibleVolumeEdge(volumeFrustum(corner, half, aspect), volumeFrustum(end, half, aspect))
			if !ok {
				continue
			}
			step := end.Sub(corner)
			a, insideA := scene.projectVolumePoint(corner.Add(step.Scale(first)))
			b, insideB := scene.projectVolumePoint(corner.Add(step.Scale(last)))
			if insideA && insideB && a != b {
				edges = append(edges, VolumeEdge{a, b})
			}
		}
	}
	return edges
}

// volumeFrustum gives each point's signed room inside the six clip planes.
func volumeFrustum(point maths.Vec, half, aspect float64) [6]float64 {
	depth := -point.Z
	horizontal, vertical := depth*half, depth*half/aspect
	return [6]float64{depth - DisplayNear, DisplayFar - depth,
		horizontal + point.X, horizontal - point.X, vertical + point.Y, vertical - point.Y}
}

// visibleVolumeEdge intersects the edge's unit interval with each clip plane.
func visibleVolumeEdge(start, end [6]float64) (first, last float64, valid bool) {
	last = 1
	for i, a := range start {
		b := end[i]
		if !displayFinite(a) || !displayFinite(b) || (a < 0 && b < 0) {
			return 0, 0, false
		}
		if a < 0 || b < 0 {
			width := a - b
			if !displayFinite(width) {
				return 0, 0, false
			}
			crossing := a / width
			if a < 0 {
				first = max(first, crossing)
			} else {
				last = min(last, crossing)
			}
		}
	}
	return first, last, first < last
}

// projectVolumePoint keeps rounding at a clip boundary inside the viewport,
// while omitting intersections that lost accuracy in eye-space arithmetic.
func (scene *DisplayScene) projectVolumePoint(point maths.Vec) ([2]float64, bool) {
	const tolerance = 1e-8
	depth := -point.Z
	x, y := scene.project(point)
	if !(depth >= DisplayNear-tolerance && depth <= DisplayFar+tolerance &&
		x >= -tolerance && x <= 1+tolerance && y >= -tolerance && y <= 1+tolerance) {
		return [2]float64{}, false
	}
	return [2]float64{max(0, min(1, x)), max(0, min(1, y))}, true
}
