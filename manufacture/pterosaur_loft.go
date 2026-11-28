package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// A lofted tube is the one way this animal's solid parts are made: a
// spine of control rings, each with a centre, a half width, a half
// height and a squareness, joined by smooth interpolation into a closed
// surface. A skull, a neck, a toe and a claw are all lofts with
// different rings. The lofts carry no animal shape of their own.

// tubeKey is one control ring of a lofted tube: where the ring is, how
// wide and how tall, and how square its outline is.
type tubeKey struct {
	centre                maths.Vec
	halfWidth, halfHeight float64
	// squareness is the exponent of the outline: 2 is an ellipse and
	// larger values are boxier.
	squareness float64
}

// tubeSurface is a lofted tube as indexed triangles: rings of sides
// points, then the centre point of each end cap.
type tubeSurface struct {
	points       []maths.Vec
	faces        [][3]int
	rings, sides int
}

// The numbers the loft machinery uses.
const (
	// tubeMinimumRings keeps a very short tube from collapsing to a
	// flat strip.
	tubeMinimumRings = 4
	// tubeEllipse is the squareness of a round outline.
	tubeEllipse = 2.0
	// tubeParallelLimit is how closely a spine may run along the hint
	// direction before another hint is used.
	tubeParallelLimit = .95
	// tubeFloor is the smallest half width or half height a ring may
	// have, in metres, so a tip never collapses to a point and its
	// faces always have area.
	tubeFloor = .0015
)

// catmullScalar is the uniform Catmull-Rom value between the middle two
// of four samples, t from 0 to 1.
func catmullScalar(first, second, third, fourth, t float64) float64 {
	const half = .5
	square, cube := t*t, t*t*t
	return half * (2*second + (third-first)*t +
		(2*first-5*second+4*third-fourth)*square +
		(3*second-first-3*third+fourth)*cube)
}

// catmullVector is catmullScalar for each part of a vector.
func catmullVector(first, second, third, fourth maths.Vec,
	t float64) maths.Vec {
	return maths.V(
		catmullScalar(first.X, second.X, third.X, fourth.X, t),
		catmullScalar(first.Y, second.Y, third.Y, fourth.Y, t),
		catmullScalar(first.Z, second.Z, third.Z, fourth.Z, t))
}

// keyAt is the interpolated ring at position t along the keys, t from 0
// to one less than the number of keys.
func keyAt(keys []tubeKey, t float64) tubeKey {
	last := len(keys) - 1
	index := int(math.Min(math.Floor(t), float64(last-1)))
	local := t - float64(index)
	pick := func(offset int) tubeKey {
		return keys[int(clamp(float64(index+offset), 0, float64(last)))]
	}
	a, b, c, d := pick(-1), pick(0), pick(1), pick(2)
	scalar := func(get func(tubeKey) float64) float64 {
		return catmullScalar(get(a), get(b), get(c), get(d), local)
	}
	width := func(k tubeKey) float64 { return k.halfWidth }
	height := func(k tubeKey) float64 { return k.halfHeight }
	square := func(k tubeKey) float64 { return k.squareness }
	return tubeKey{
		centre: catmullVector(a.centre, b.centre, c.centre, d.centre,
			local),
		halfWidth:  math.Max(tubeFloor, scalar(width)),
		halfHeight: math.Max(tubeFloor, scalar(height)),
		squareness: scalar(square),
	}
}

// tubeDenseSteps is how many samples between two keys are used to measure
// the spine's length before the rings are placed evenly along it.
const tubeDenseSteps = 24

// tubeRings is the keys sampled into rings evenly spaced along the
// spine's length, about spacing apart, and the distance of each ring from
// the spine's start in metres.
func tubeRings(keys []tubeKey, spacing float64) ([]tubeKey, []float64) {
	dense := (len(keys) - 1) * tubeDenseSteps
	cumulative := make([]float64, dense+1)
	previous := keys[0].centre
	for i := 1; i <= dense; i++ {
		here := keyAt(keys, float64(i)/tubeDenseSteps).centre
		cumulative[i] = cumulative[i-1] + here.Sub(previous).Len()
		previous = here
	}
	length := cumulative[dense]
	count := int(math.Max(tubeMinimumRings, math.Ceil(length/spacing)))
	rings := make([]tubeKey, count+1)
	along := make([]float64, count+1)
	at := 0
	for i := range rings {
		along[i] = float64(i) / float64(count) * length
		for at < dense-1 && cumulative[at+1] < along[i] {
			at++
		}
		span := cumulative[at+1] - cumulative[at]
		local := 0.0
		if span > 0 {
			local = clamp((along[i]-cumulative[at])/span, 0, 1)
		}
		rings[i] = keyAt(keys, (float64(at)+local)/tubeDenseSteps)
	}
	return rings, along
}

// tubeAxes is the across and rise directions of every ring, carried
// along the spine without twist from the first ring's hint.
func tubeAxes(rings []tubeKey, hint maths.Vec) ([]maths.Vec, []maths.Vec) {
	across := make([]maths.Vec, len(rings))
	rise := make([]maths.Vec, len(rings))
	for i := range rings {
		before := int(math.Max(0, float64(i-1)))
		after := int(math.Min(float64(len(rings)-1), float64(i+1)))
		tangent := rings[after].centre.Sub(rings[before].centre).Unit()
		if i == 0 {
			across[i] = firstAcross(tangent, hint)
		} else {
			previous := across[i-1]
			along := previous.Dot(tangent)
			carried := previous.Sub(tangent.Scale(along))
			across[i] = carried.Unit()
		}
		rise[i] = across[i].Cross(tangent).Unit()
	}
	return across, rise
}

// firstAcross is a unit direction across the first ring: at right angles
// to the tangent and to the hint, or to the long axis when the tangent
// runs along the hint.
func firstAcross(tangent, hint maths.Vec) maths.Vec {
	if math.Abs(tangent.Dot(hint.Unit())) > tubeParallelLimit {
		hint = maths.V(1, 0, 0)
		if math.Abs(tangent.Dot(hint)) > tubeParallelLimit {
			hint = maths.V(0, 0, 1)
		}
	}
	return tangent.Cross(hint).Unit()
}

// signedPower is |x| to the power, with x's sign.
func signedPower(x, power float64) float64 {
	return math.Copysign(math.Pow(math.Abs(x), power), x)
}

// tubeLoft is the closed surface through the keys, about spacing apart
// along the spine and sides round. relief, if given, multiplies the
// outline at a ring (its distance along the spine in metres) and angle,
// so a part can carry ridges, scales and swellings as real surface form.
func tubeLoft(keys []tubeKey, spacing float64, sides int, hint maths.Vec,
	relief func(along, angle float64) float64) tubeSurface {
	rings, distances := tubeRings(keys, spacing)
	across, rise := tubeAxes(rings, hint)
	surface := tubeSurface{rings: len(rings), sides: sides}
	for i, ring := range rings {
		along := distances[i]
		for k := 0; k < sides; k++ {
			angle := 2 * math.Pi * float64(k) / float64(sides)
			scale := 1.0
			if relief != nil {
				scale = relief(along, angle)
			}
			power := 2 / ring.squareness
			x := signedPower(math.Cos(angle), power)
			y := signedPower(math.Sin(angle), power)
			offset := across[i].Scale(x * ring.halfWidth * scale).
				Add(rise[i].Scale(y * ring.halfHeight * scale))
			surface.points = append(surface.points,
				ring.centre.Add(offset))
		}
	}
	surface.points = append(surface.points, rings[0].centre,
		rings[len(rings)-1].centre)
	surface.faces = tubeFaces(len(rings), sides)
	return surface
}

// tubeFaces is the triangles of a tube of rings rings of sides points,
// with a fan closing each end on the two centre points after the rings.
func tubeFaces(rings, sides int) [][3]int {
	var faces [][3]int
	first, last := rings*sides, rings*sides+1
	for i := 0; i < rings-1; i++ {
		for k := 0; k < sides; k++ {
			next := (k + 1) % sides
			a, b := i*sides+k, i*sides+next
			c, d := (i+1)*sides+next, (i+1)*sides+k
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	for k := 0; k < sides; k++ {
		next := (k + 1) % sides
		end := (rings - 1) * sides
		faces = append(faces, [3]int{first, next, k},
			[3]int{last, end + k, end + next})
	}
	return faces
}

// mapped is the surface with every point moved by place; the faces stay.
func (surface tubeSurface) mapped(place func(maths.Vec) maths.Vec) tubeSurface {
	moved := surface
	moved.points = make([]maths.Vec, len(surface.points))
	for i, point := range surface.points {
		moved.points[i] = place(point)
	}
	return moved
}

// at is the point of ring, side.
func (surface tubeSurface) at(ring, side int) maths.Vec {
	return surface.points[ring*surface.sides+side%surface.sides]
}

// closedMesh is the surface as a smooth mesh, wound so faces turn
// counterclockwise seen from outside whichever way the loft ran.
func (surface tubeSurface) closedMesh() (*mesh.Mesh, error) {
	faces := surface.faces
	if solidVolume(surface.points, faces) < 0 {
		faces = flippedFaces(faces)
	}
	return meshOfTriangles(surface.points, nil, faces)
}

// solidVolume is the signed volume a closed set of faces encloses:
// positive when the faces turn counterclockwise seen from outside.
func solidVolume(points []maths.Vec, faces [][3]int) float64 {
	total := 0.0
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		total += a.Dot(b.Cross(c))
	}
	return total / 6
}

// flippedFaces is the faces each turned the other way.
func flippedFaces(faces [][3]int) [][3]int {
	turned := make([][3]int, len(faces))
	for i, face := range faces {
		turned[i] = [3]int{face[0], face[2], face[1]}
	}
	return turned
}

// meshOfTriangles is a smooth mesh whose vertex normals are the area
// weighted sum of the faces round each vertex. rest, if not nil, is the
// undeformed shape, one point per vertex, for paint that follows motion.
func meshOfTriangles(points, rest []maths.Vec,
	faces [][3]int) (*mesh.Mesh, error) {
	normals := faceNormals(points, faces)
	if rest == nil {
		return mesh.New(points, faces, normals, true)
	}
	return mesh.NewWithRestPoints(points, rest, faces, normals, true)
}

// faceNormals is one vertex normal per point, the sum of the area
// weighted normals of every face that uses the point.
func faceNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := make([]maths.Vec, len(points))
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		weighted := b.Sub(a).Cross(c.Sub(a))
		for _, corner := range face {
			normals[corner] = normals[corner].Add(weighted)
		}
	}
	return normals
}

// ellipsoidSurface is a closed ellipsoid as a loft along axis: halfLength
// along it and halfWidth and halfHeight across, rings rings and sides
// round. It makes joints, the eye, and the swellings of a muscle.
func ellipsoidSurface(centre, axis maths.Vec, halfLength, halfWidth,
	halfHeight float64, rings, sides int) tubeSurface {
	keys := make([]tubeKey, rings)
	direction := axis.Unit()
	for i := range keys {
		share := (float64(i) + .5) / float64(rings)
		angle := math.Pi * share
		offset := -halfLength * math.Cos(angle)
		keys[i] = tubeKey{
			centre:     centre.Add(direction.Scale(offset)),
			halfWidth:  halfWidth * math.Sin(angle),
			halfHeight: halfHeight * math.Sin(angle),
			squareness: tubeEllipse,
		}
	}
	spacing := 2 * halfLength / float64(rings)
	return tubeLoft(keys, spacing, sides, maths.V(0, 1, 0), nil)
}
