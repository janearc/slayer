package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The trunk's rings. Each has sealLoftSections points; the gap to the
// next is a share of the outline's mean radius there, so the muzzle and
// the tail are cut finer than the belly, between a least and a most.
// All picked: fine enough that no facet shows in a 1600-wide picture.
const (
	sealLoftSections  = 72
	sealLoftStepShare = .12
	sealLoftStepLeast = .003
	sealLoftStepMost  = .020
	// sealLoftPole is how far past the last ring each end's closing
	// point stands, as a share of that ring's mean radius.
	sealLoftPole = .5
)

// sealSurface is one closed surface at rest: its points and the
// triangles that join them, each wound so its normal points out.
type sealSurface struct {
	points []maths.Vec
	faces  [][3]int
}

// rings is the X of each ring of the trunk, from the tail's tip to the
// nose.
func (outline sealOutline) rings() []float64 {
	first, last := outline[0].x, outline[len(outline)-1].x
	stations := []float64{first}
	at := first
	for {
		step := sealLoftStepShare * outline.meanRadius(at)
		at += math.Min(math.Max(step, sealLoftStepLeast), sealLoftStepMost)
		if at >= last-sealLoftStepLeast {
			return append(stations, last)
		}
		stations = append(stations, at)
	}
}

// surface is the head, neck, trunk and tail as one surface lofted
// through the outline: a ring of points at each station, joined to the
// next, and closed by a point past each end.
func (outline sealOutline) surface() sealSurface {
	stations := outline.rings()
	surface := sealSurface{}
	for _, x := range stations {
		for column := range sealLoftSections {
			turn := 2 * math.Pi * float64(column) / sealLoftSections
			point, _ := outline.radial(x, turn)
			surface.points = append(surface.points, point)
		}
	}
	for row := 0; row+1 < len(stations); row++ {
		for column := range sealLoftSections {
			next := (column + 1) % sealLoftSections
			low, high := row*sealLoftSections, (row+1)*sealLoftSections
			surface.faces = append(surface.faces,
				[3]int{low + column, low + next, high + column},
				[3]int{low + next, high + next, high + column})
		}
	}
	surface.close(outline, 0, stations[0], -1)
	surface.close(outline, (len(stations)-1)*sealLoftSections,
		stations[len(stations)-1], 1)
	return surface
}

// close shuts the ring that starts at point first, at station x of the
// outline, with a fan to one point past it: way is one toward the nose
// and minus one toward the tail.
func (surface *sealSurface) close(outline sealOutline, first int,
	x, way float64,
) {
	pole := len(surface.points)
	surface.points = append(surface.points, maths.V(
		x+way*sealLoftPole*outline.meanRadius(x), outline.at(x).y, 0))
	for column := range sealLoftSections {
		next := (column + 1) % sealLoftSections
		face := [3]int{pole, first + column, first + next}
		if way < 0 {
			face = [3]int{pole, first + next, first + column}
		}
		surface.faces = append(surface.faces, face)
	}
}

// sealEdge is an edge of a surface as its two points' places, in the
// order a face runs it.
type sealEdge struct{ from, to int }

// open is how many of the surface's edges are not run exactly once each
// way: none for a closed surface whose faces all face the same side.
func (surface sealSurface) open() int {
	runs := map[sealEdge]int{}
	for _, face := range surface.faces {
		for corner := range face {
			runs[sealEdge{face[corner], face[(corner+1)%len(face)]}]++
		}
	}
	open := 0
	for edge, count := range runs {
		if count != 1 || runs[sealEdge{edge.to, edge.from}] != 1 {
			open++
		}
	}
	return open
}
