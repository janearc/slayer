package pigment

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// Patch describes a soft, oriented ellipsoid for a local blemish. Axes need
// only supply the in-plane directions. Radius gives the extent along the
// first axis, the second axis and the surface normal.
type Patch struct {
	Centre, Across, Along maths.Vec
	Radius                maths.Vec
}

type ellipse struct {
	centre, across, along, normal maths.Vec
	radius                        maths.Vec
}

// NewPatch makes a local mask. A shallow third radius confines pigment to
// one side of a thin surface.
func NewPatch(patch Patch) (Mask, error) {
	across := patch.Across.Unit()
	normal := across.Cross(patch.Along.Unit())
	radius := patch.Radius
	if !finite(patch.Centre) || !finite(across) || !(normal.Len() > 1e-8) ||
		!positive(radius.X) || !positive(radius.Y) || !positive(radius.Z) {
		return nil, fmt.Errorf("pigment: invalid patch axes or radii")
	}
	normal = normal.Unit()
	return ellipse{patch.Centre, across, normal.Cross(across), normal, radius}, nil
}

func (patch ellipse) Coverage(point maths.Vec, footprint float64) float64 {
	delta := point.Sub(patch.centre)
	first, second := delta.Dot(patch.across), delta.Dot(patch.along)
	depth := delta.Dot(patch.normal) / patch.radius.Z
	halfPixel := max(0, footprint/2)
	width, length := patch.radius.X+halfPixel, patch.radius.Y+halfPixel
	distance := maths.V(first/width, second/length, depth).Len()
	return (1 - maths.Smooth(.4, 1, distance)) *
		(patch.radius.X / width) * (patch.radius.Y / length)
}

// Stroke describes a soft line following rest-space points. Radius is the
// line's half-width. Feather is the fraction of that width used for its edge.
type Stroke struct {
	Points          []maths.Vec
	Radius, Feather float64
}

type strokeMask struct {
	points                 []maths.Vec
	radius, feather, taper float64
}

// NewStroke makes a connected line for a scar or an outline in a tattoo.
// Several strokes can form a design. Points are copied.
func NewStroke(stroke Stroke) (Mask, error) {
	return newStroke(stroke, 0)
}

// taper is the fraction of the half-width removed at each endpoint.
func newStroke(stroke Stroke, taper float64) (Mask, error) {
	if len(stroke.Points) < 2 || !positive(stroke.Radius) ||
		!share(stroke.Feather) || stroke.Feather == 0 {
		return nil, fmt.Errorf("pigment: a stroke needs points, radius and feather")
	}
	for _, point := range stroke.Points {
		if !finite(point) {
			return nil, fmt.Errorf("pigment: stroke points must be finite")
		}
	}
	for i, end := range stroke.Points[1:] {
		edge := end.Sub(stroke.Points[i])
		if !positive(edge.Dot(edge)) {
			return nil, fmt.Errorf("pigment: invalid stroke point %d", i)
		}
	}
	return strokeMask{slices.Clone(stroke.Points), stroke.Radius, stroke.Feather, taper}, nil
}

func (stroke strokeMask) Coverage(point maths.Vec, footprint float64) float64 {
	distance := math.Inf(1)
	progress := 0.0
	for i, end := range stroke.points[1:] {
		start := stroke.points[i]
		edge := end.Sub(start)
		along := min(1, max(0, point.Sub(start).Dot(edge)/edge.Dot(edge)))
		nearest := point.Sub(start.Add(edge.Scale(along)))
		squared := nearest.Dot(nearest)
		if stroke.taper > 0 && squared < distance {
			progress = float64(i) + along
		}
		distance = min(distance, squared)
	}
	radius := stroke.radius
	if stroke.taper > 0 {
		segments := float64(len(stroke.points) - 1)
		fromEnd := min(progress, segments-progress) / segments
		radius *= 1 - stroke.taper*(1-maths.Smooth(0, .23, fromEnd))
	}
	halfPixel := max(0, footprint/2)
	inside := max(0, radius*(1-stroke.feather)-halfPixel)
	outside := radius + halfPixel
	return (1 - maths.Smooth(inside, outside, math.Sqrt(distance))) *
		radius / outside
}
