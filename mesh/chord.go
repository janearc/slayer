package mesh

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// ChordQuery asks for the next outward crossing of a closed mesh from
// one of its surface hits. Toward points into the surface. Limit is a
// strict distance cutoff in mesh-local units.
type ChordQuery struct {
	From   Met
	Toward maths.Vec
	Limit  float64
}

// Chord is a straight segment across the mesh interior. Exit.Along()
// gives its length from Start, in mesh-local units.
type Chord struct {
	Start, End maths.Vec
	Exit       Met
}

// InteriorChord finds a directional interior segment of a closed mesh
// with nonintersecting faces. It rejects edge hits and crossings too close
// to distinguish from roundoff at the origin. Only this mesh is queried.
func (made *Mesh) InteriorChord(query ChordQuery) (Chord, bool) {
	if !made.closed || query.From.face < 0 || query.From.face >= len(made.faces) ||
		!finite(query.Toward) || !(query.Limit > 0) || !number(query.Limit) {
		return Chord{}, false
	}
	for _, weight := range query.From.weights {
		if !(weight > 8*roundoff) {
			return Chord{}, false
		}
	}
	direction := query.Toward.Unit()
	incidence := -made.across(query.From.face).Unit().Dot(direction)
	if !(incidence > 0) {
		return Chord{}, false
	}

	one, two, three := made.Corners(query.From.face)
	extent := max(chordMagnitude(one), chordMagnitude(two), chordMagnitude(three))
	ulp := math.Nextafter(extent, math.Inf(1)) - extent
	near := 16 * ulp / incidence
	if !(near < query.Limit) {
		return Chord{}, false
	}
	start, _ := made.VertexPointAt(query.From, made.points)
	ray := maths.Ray{At: start, Dir: direction}
	exit, found := made.Meet(ray, 0, query.Limit)
	if found && exit.face == query.From.face && exit.Along() <= near {
		exit, found = made.Meet(ray, exit.Along(), query.Limit)
	}
	if !found || exit.face == query.From.face || !(exit.Along() > near) ||
		!(made.across(exit.face).Unit().Dot(direction) > 0) {
		return Chord{}, false
	}
	for _, weight := range exit.weights {
		if !(weight > 8*roundoff) {
			return Chord{}, false
		}
	}
	return Chord{Start: start, End: ray.Point(exit.Along()), Exit: exit}, true
}

func chordMagnitude(point maths.Vec) float64 {
	return max(math.Abs(point.X), math.Abs(point.Y), math.Abs(point.Z))
}
