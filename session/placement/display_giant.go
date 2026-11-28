package placement

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// GiantBoundaryLimit bounds the metadata and projection cost per view.
const GiantBoundaryLimit = 16

// GlobeOutlineLimit bounds the number of analytic globe overlays.
const GlobeOutlineLimit = 16

// DisplaySphereOutline is the centre and radius of a spherical outline.
type DisplaySphereOutline struct {
	Centre maths.Vec
	Radius float64
}

// DisplayGiantBoundary describes the zero-density outer boundary of a producer gas giant.
type DisplayGiantBoundary = DisplaySphereOutline

func (giant DisplaySphereOutline) Valid() bool {
	return displayFinite(giant.Centre.X) && displayFinite(giant.Centre.Y) &&
		displayFinite(giant.Centre.Z) && displayFinite(giant.Radius) && giant.Radius > 0
}

// WorkingGiantBoundaries takes actual positions and extents from the producer.
func WorkingGiantBoundaries(world *thing.World, at time.Duration) []DisplayGiantBoundary {
	var boundaries []DisplayGiantBoundary
	for _, volume := range world.Volumes {
		if volume == nil || volume.Hidden() {
			continue
		}
		giant, ok := volume.Medium.(medium.Giant)
		if !ok {
			continue
		}
		place := volume.At(at)
		if !shape.ValidPlacement(place) {
			continue
		}
		scale := place.Scale
		if scale == 0 {
			scale = 1
		}
		boundary := DisplayGiantBoundary{Centre: place.Origin, Radius: giant.Radius * scale}
		if boundary.Valid() {
			boundaries = append(boundaries, boundary)
			if len(boundaries) == GiantBoundaryLimit {
				break
			}
		}
	}
	return boundaries
}

// ProjectGiantBoundary clips the unfilled limb of a gas giant wholly in
// front of the eye to the image, without contributing depth or picks.
func (scene *DisplayScene) ProjectGiantBoundary(boundary DisplayGiantBoundary) []VolumeEdge {
	return scene.projectSphereOutline(boundary, 0)
}

// ProjectDistantSphere traces the pinhole's tangent cone to a wholly distant
// sphere and clips its unfilled outline to the image. No depth, colour, or
// density value enters the working geometry or pick buffers.
func (scene *DisplayScene) ProjectDistantSphere(boundary DisplaySphereOutline) []VolumeEdge {
	return scene.projectSphereOutline(boundary, DisplayFar)
}

func (scene *DisplayScene) projectSphereOutline(boundary DisplaySphereOutline, minFrontDepth float64) []VolumeEdge {
	if !boundary.Valid() || !shape.ValidPlacement(scene.Eye) {
		return nil
	}
	centre := scene.Eye.In(boundary.Centre)
	radius := boundary.Radius
	if scene.Eye.Scale != 0 {
		radius /= scene.Eye.Scale
	}
	frontDepth := -centre.Z - radius
	if !displayFinite(frontDepth) || frontDepth <= minFrontDepth {
		return nil
	}
	distance := math.Hypot(math.Hypot(centre.X, centre.Y), centre.Z)
	if !displayFinite(distance) || distance <= radius {
		return nil
	}
	axis := centre.Scale(1 / distance)
	pole := maths.YAxis
	if math.Abs(axis.Y) > .9 {
		pole = maths.XAxis
	}
	u := axis.Cross(pole).Unit()
	v := axis.Cross(u)
	sinAngle := radius / distance
	cosAngle := math.Sqrt((1 - sinAngle) * (1 + sinAngle))
	const segments = 96
	var edges []VolumeEdge
	startX, startY := scene.project(axis.Scale(cosAngle).Add(u.Scale(sinAngle)))
	start := [2]float64{startX, startY}
	for index := 1; index <= segments; index++ {
		angle := 2 * math.Pi * float64(index) / segments
		sin, cos := math.Sincos(angle)
		dir := axis.Scale(cosAngle).Add(u.Scale(cos * sinAngle)).Add(v.Scale(sin * sinAngle))
		x, y := scene.project(dir)
		end := [2]float64{x, y}
		if edge, ok := clipGiantEdge(start, end); ok {
			edges = append(edges, edge)
		}
		start = end
	}
	return edges
}

func clipGiantEdge(start, end [2]float64) (VolumeEdge, bool) {
	room := func(point [2]float64) [6]float64 {
		return [6]float64{point[0], 1 - point[0], point[1], 1 - point[1], 1, 1}
	}
	first, last, ok := visibleVolumeEdge(room(start), room(end))
	if !ok {
		return VolumeEdge{}, false
	}
	point := func(at float64) [2]float64 {
		return [2]float64{
			max(0, min(1, start[0]+at*(end[0]-start[0]))),
			max(0, min(1, start[1]+at*(end[1]-start[1]))),
		}
	}
	edge := VolumeEdge{point(first), point(last)}
	return edge, edge[0] != edge[1]
}
