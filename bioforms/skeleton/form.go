package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// A bone's form is its shape at rest, finer than the spindle Mesh draws:
// a shaft with wide ends, a flat blade, a bone curved along a cubic, or a
// vertebra with its processes. A maker gives each bone the form it has and
// draws them all with Draw. A form also says where its middle line runs,
// which is what the skin is weighed against.

// sweepSides is the number of points round every ring of a swept tube.
const sweepSides = 10

// sweepTiny is a length below which a direction is taken to be none.
const sweepTiny = 1e-9

// BoneForm is a bone's shape at rest. Build adds it to a mesh as closed
// tubes, every point carried by place to where the pose puts it; Middle is
// its middle line, end to end, as points.
type BoneForm interface {
	Build(points []maths.Vec, faces [][3]int,
		place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int)
	Middle() []maths.Vec
}

// Draw is every bone carried by frames, each in its form where forms has
// one and as Mesh's spindle where it has none, as closed tubes wound
// outward.
func (skeleton *Skeleton) Draw(frames []Frame,
	forms map[string]BoneForm) ([]maths.Vec, [][3]int) {
	points := []maths.Vec{}
	faces := [][3]int{}
	for i, bone := range skeleton.bones {
		if form, ok := forms[bone.name]; ok {
			points, faces = form.Build(points, faces, frames[i].Apply)
			continue
		}
		start := len(points)
		for _, p := range spindlePoints(bone) {
			points = append(points, frames[i].Apply(p))
		}
		faces = append(faces, spindleFaces(start)...)
	}
	return points, faces
}

// Sweep adds a closed tube round path to a mesh: rings+1 rings of
// sweepSides points, each an ellipse whose half sizes half gives, the
// first across the path square to up, then a flat cap at either end. The
// tube is wound so that its faces look outward.
func Sweep(points []maths.Vec, faces [][3]int, rings int,
	path func(float64) maths.Vec, half func(float64) (float64, float64),
	up maths.Vec, place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	first, firstFace := len(points), len(faces)
	for ring := 0; ring <= rings; ring++ {
		t := float64(ring) / float64(rings)
		side, normal := SweepFrame(path, t, up)
		wide, deep := half(t)
		for j := 0; j < sweepSides; j++ {
			angle := 2 * math.Pi * float64(j) / sweepSides
			sidePart := side.Scale(wide * math.Cos(angle))
			normalPart := normal.Scale(deep * math.Sin(angle))
			offset := sidePart.Add(normalPart)
			points = append(points, path(t).Add(offset))
		}
	}
	points = append(points, path(0), path(1))
	faces = sweepFaces(faces, first, rings)
	for i := first; i < len(points); i++ {
		points[i] = place(points[i])
	}
	return points, outward(points, faces, firstFace)
}

// SweepFrame is the two directions across path at t: side, square to the
// path and to up, and normal, square to both. Where the path runs along
// up, side is taken square to the x axis instead.
func SweepFrame(path func(float64) maths.Vec, t float64,
	up maths.Vec) (side, normal maths.Vec) {
	const step = 1e-3
	ahead := path(math.Min(1, t+step))
	behind := path(math.Max(0, t-step))
	along := ahead.Sub(behind).Unit()
	side = along.Cross(up)
	if side.Len() < sweepTiny {
		side = along.Cross(maths.V(1, 0, 0))
	}
	side = side.Unit()
	return side, side.Cross(along).Unit()
}

// sweepFaces adds the faces of a tube of rings+1 rings starting at point
// first, and the two caps whose middles follow the last ring.
func sweepFaces(faces [][3]int, first, rings int) [][3]int {
	for i := 0; i < rings; i++ {
		for j := 0; j < sweepSides; j++ {
			a := first + i*sweepSides + j
			b := first + i*sweepSides + (j+1)%sweepSides
			faces = append(faces, [3]int{a, b, b + sweepSides},
				[3]int{a, b + sweepSides, a + sweepSides})
		}
	}
	start := first + (rings+1)*sweepSides
	last := first + rings*sweepSides
	for j := 0; j < sweepSides; j++ {
		k := (j + 1) % sweepSides
		faces = append(faces, [3]int{start, first + k, first + j},
			[3]int{start + 1, last + j, last + k})
	}
	return faces
}

// outward turns the faces from firstFace on inside out if, taken
// together, they enclose a negative volume.
func outward(points []maths.Vec, faces [][3]int, firstFace int) [][3]int {
	volume := 0.
	for _, face := range faces[firstFace:] {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		volume += a.Dot(b.Cross(c))
	}
	if volume >= 0 {
		return faces
	}
	for i := firstFace; i < len(faces); i++ {
		faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
	}
	return faces
}

// segmentPath is the straight path from a to b.
func segmentPath(a, b maths.Vec) func(float64) maths.Vec {
	return func(t float64) maths.Vec { return a.Add(b.Sub(a).Scale(t)) }
}

// fixedHalf is half sizes that do not change along a path.
func fixedHalf(wide, deep float64) func(float64) (float64, float64) {
	return func(float64) (float64, float64) { return wide, deep }
}
