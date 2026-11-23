package rings

import (
	"math"

	kerr "github.com/janearc/celestial/hole"
)

// Point is a place on a ray's path in the plane of the eye and the disc's
// axis, in masses: along the disc's plane toward the eye's side, and up
// the axis.
type Point struct {
	Along, Up float64
}

// Crossing is where a ray crosses the disc's plane: its radius in
// masses, and whether it is on the far side of the hole from the eye.
type Crossing struct {
	Radius float64
	Behind bool
}

// Path is one ray followed back from the eye: every point it stepped
// to, every crossing of the disc's plane in order, whether it came from
// the hole, and the angle it went round the hole's centre.
type Path struct {
	points    []Point
	crossings []Crossing
	captured  bool
	swept     float64
}

// Points is the ray's path, from the eye, point by point.
func (path Path) Points() []Point {
	return append([]Point(nil), path.points...)
}

// Crossings is every crossing of the disc's plane, in the order the ray
// made them going back from the eye.
func (path Path) Crossings() []Crossing {
	return append([]Crossing(nil), path.crossings...)
}

// Lands is whether the ray's crossing of the disc's plane of an order
// lands on a disc of no thickness from inner to outer masses: order
// zero is its first crossing, the direct image; order one its second,
// after going round half a turn more; and so on. The disc is see
// through, as the tracer's is, so a ray brings back every order it
// lands.
func (path Path) Lands(order int, disc Disc) bool {
	if order < none || order >= len(path.crossings) {
		return false
	}
	return disc.holds(path.crossings[order].Radius)
}

// walk is a march in progress: the lens, the ray where it is, the path
// so far, the angle round the centre where it is, and whether it ended.
type walk struct {
	lens  Lens
	ray   kerr.Ray
	made  Path
	angle float64
	done  bool
}

// newWalk is a march set off from the eye along a ray.
func newWalk(lens Lens, ray kerr.Ray) *walk {
	at := pointOf(ray.Space())
	return &walk{lens: lens, ray: ray, made: Path{points: []Point{at}},
		angle: math.Atan2(at.Up, at.Along)}
}

// pointOf is a place in the hole's space as a point of the plane.
func pointOf(place kerr.Vec3) Point {
	return Point{Along: place.AlongX(), Up: place.AlongZ()}
}

// step is one step of the march: the ray carried on by PaceIn's step,
// its point and any crossing kept, and the march ended when the ray
// came from the hole or is past the eye and going out.
func (walk *walk) step() {
	hole := walk.lens.hole
	radius := hole.Radius(walk.ray.Space())
	if hole.Hovering(walk.ray) || radius < core*stillMass {
		walk.made.captured, walk.done = true, true
		return
	}
	next := hole.Step(walk.ray, hole.PaceIn(walk.ray))
	from, to := pointOf(walk.ray.Space()), pointOf(next.Space())
	walk.cross(from, to)
	walk.turnTo(to)
	walk.made.points = append(walk.made.points, to)
	walk.ray = next
	onward := hole.Radius(next.Space())
	if onward > radius && onward > farOut*walk.lens.distance {
		walk.done = true
	}
}

// cross keeps the crossing of the disc's plane between two points, if
// the step made one: where the straight line between them meets it.
func (walk *walk) cross(from, to Point) {
	if (from.Up > none) == (to.Up > none) {
		return
	}
	share := from.Up / (from.Up - to.Up)
	along := from.Along + share*(to.Along-from.Along)
	walk.made.crossings = append(walk.made.crossings,
		Crossing{Radius: math.Abs(along), Behind: along < none})
}

// turnTo adds the angle the step went round the centre to the sweep:
// the change of the point's angle, taken the short way round.
func (walk *walk) turnTo(to Point) {
	angle := math.Atan2(to.Up, to.Along)
	change := math.Remainder(angle-walk.angle, turn)
	walk.made.swept += change
	walk.angle = angle
}

// path is the march's path as it stands.
func (walk *walk) path() Path {
	if !walk.done {
		// out of steps: stuck, which the tracer takes as captured
		walk.made.captured = true
	}
	return walk.made
}
