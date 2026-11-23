// Package maths provides shared geometry and linear-light arithmetic: points,
// directions, rays, boxes, orientations, subspaces, and RGB light. Coordinates
// are right-handed, with y up and negative z forward.
package maths

import "math"

// Degree is one degree, in the radians every angle here is measured in. It
// is defined: a half turn is pi radians and 180 degrees.
const Degree = math.Pi / halfTurn

// What the geometry counts in, each defined.
const (
	nought     = 0   // defined: no amount
	unity      = 1   // defined: one
	dimensions = 3   // defined: space has three axes
	halfTurn   = 180 // defined: degrees in half a turn
)

// nearlyAlong is how near to straight along up a view may look before up
// is taken as minus z instead: the dot product of the two, about 2.56
// degrees from along.
//
// it is a threshold, not a measurement. it stays a constant here because
// the module reads no file, so every importer pays for none.
//
//ninja:vow source picked by hand and not derived; no file is read here
const nearlyAlong = 0.999

// nearlyAlongX is how near to along x an up may be before Upright uses z
// to find the other two axes. It is the dot product of the two.
//
//ninja:vow source picked by hand and not derived; no file is read here
const nearlyAlongX = 0.9

// rest is what is left of a whole when share of it is gone.
func rest(share float64) float64 { return unity - share }

// Vec is a point or a direction in some space.
type Vec struct {
	//ninja:vow gate a value, whose fields are what it is
	X, Y, Z float64 //ninja:vow name x, y, z at every call site
}

// V is a Vec from its three parts.
func V(x, y, z float64) Vec { //ninja:vow name x, y, z at every call site
	return Vec{x, y, z}
}

// The three axes, and up.
var (
	//ninja:vow well a fixed direction; nothing in slayer writes it
	XAxis = Vec{unity, nought, nought}
	//ninja:vow well a fixed direction; nothing in slayer writes it
	YAxis = Vec{nought, unity, nought}
	//ninja:vow well a fixed direction; nothing in slayer writes it
	ZAxis = Vec{nought, nought, unity}
	//ninja:vow well a fixed direction; nothing in slayer writes it
	Up = YAxis
)

// Add is vec moved by other.
func (vec Vec) Add(other Vec) Vec {
	return Vec{vec.X + other.X, vec.Y + other.Y, vec.Z + other.Z}
}

// Sub is how far vec is from other, and which way.
func (vec Vec) Sub(other Vec) Vec {
	return Vec{vec.X - other.X, vec.Y - other.Y, vec.Z - other.Z}
}

// Scale is vec stretched by factor.
func (vec Vec) Scale(factor float64) Vec {
	return Vec{vec.X * factor, vec.Y * factor, vec.Z * factor}
}

// Dot is how much vec and other point the same way. For two unit directions it
// is 1 when they agree, 0 at right angles and -1 when they are opposite.
func (vec Vec) Dot(other Vec) float64 {
	return vec.X*other.X + vec.Y*other.Y + vec.Z*other.Z
}

// Len is how long the vector is. Any vector with a part past about 1e154
// is measured as infinitely long, because its squares pass what a float
// holds.
//
// Below about 1e-156 the squares lose their last digits, more as they get
// smaller. The length is about six millionths short at 1e-160 and 0.6 per cent
// short at 1e-161. Between about 2e-162 and 1.6e-162 it reads too long, by
// 11 per cent and then 39 per cent: both read 2.22e-162.
//
// From about 1.5e-162 down it is zero. These were measured and a test
// holds them.
//
// Len is small enough that the compiler writes it into every caller and
// joins the multiplies and adds there. Any body large enough to measure
// such vectors would not be inlined, and every length slayer draws with
// would move by a last bit.
func (vec Vec) Len() float64 { return math.Sqrt(vec.Dot(vec)) }

// unmeasured is whether Len measures the vector as zero.
// That is the zero vector and every vector too short for Len.
//
// It is kept out of line on purpose. Inlined, it moved some Upright
// answers by a last bit on arm64. How many depends on the inputs: about
// one in a hundred of the differential's draw, and six in a hundred drawn
// evenly from -1 to 1 on each axis.
//
// The directive matters where the compiler fuses multiply and add, as it
// does on arm64. On amd64 at GOAMD64=v1 nothing fuses and it changes no bit.
// At a later GOAMD64 the compiler may fuse, and that was not measured.
//
//go:noinline
func unmeasured(vec Vec) bool { return vec.Len() == nought }

// unreadable is whether Len cannot give the vector a length at all: it has
// a part that is no number or infinite, or a part past about 1e154, whose
// square passes what a float holds. Rotation, LookAtUp, Upright and View
// refuse such a vector.
//
// It is kept out of line as unmeasured is, so that the multiplies and adds
// of its callers are not joined with Len's.
//
//go:noinline
func unreadable(vec Vec) bool {
	length := vec.Len()
	return math.IsNaN(length) || math.IsInf(length, eitherSign)
}

// unreadableTurn is the orientation of no numbers: what Rotation, LookAtUp
// and Upright give for a vector Len cannot measure at all. The fault shows
// where it is used. It is not hidden as no turn.
func unreadableTurn() Orientation {
	none := Vec{math.NaN(), math.NaN(), math.NaN()}
	return Orientation{none, none, none}
}

// Neg is vec pointing the other way.
func (vec Vec) Neg() Vec { return Vec{-vec.X, -vec.Y, -vec.Z} }

// Near is whether vec is within the given distance of other.
func (vec Vec) Near(other Vec, within float64) bool {
	return vec.Sub(other).Len() <= within
}

// Cross is the direction at right angles to both, by the right-hand rule.
func (vec Vec) Cross(other Vec) Vec {
	return Vec{vec.Y*other.Z - vec.Z*other.Y, vec.Z*other.X - vec.X*other.Z,
		vec.X*other.Y - vec.Y*other.X}
}

// Unit is the same direction, one unit long. It is as near as Len
// measures it. Unit is kept as small as Len, for the reason Len is.
//
// The zero vector stays zero. If the vector is too short for Len to
// measure, it is given back as it is. If it is too long, it becomes zero.
// If it has an infinite part, that part of the result is no number and
// its finite parts are zero.
func (vec Vec) Unit() Vec {
	length := vec.Len()
	if length == nought {
		return vec
	}
	return vec.Scale(unity / length)
}

// Ray starts at At and goes along Dir, which is one unit long. Nothing
// here makes it so. If Dir is not one unit long, the ray measures its
// distances in Dir's lengths.
type Ray struct {
	//ninja:vow gate a value, whose fields are what it is
	At, Dir Vec
}

// NewRay is the ray that starts at at and goes along dir. It keeps dir as
// given; it is not made one unit long.
func NewRay(at, dir Vec) Ray { return Ray{At: at, Dir: dir} }

// Point is where you are along the ray after distance.
func (ray Ray) Point(distance float64) Vec {
	return ray.At.Add(ray.Dir.Scale(distance))
}

// Box is the space between two corners, lined up with the axes. It is the
// cheap first question: could this possibly be here? It is never the
// truth about a shape. Its corners may be infinite, for shapes with no
// edge.
//
// Min is the lesser corner on every axis. Made otherwise, a box is met
// only by rays that cross it slantwise.
type Box struct {
	//ninja:vow gate a value, whose fields are what it is
	Min, Max Vec
}

// NewBox is the box with the corners low and high, in that order. It keeps
// them as given; it does not put the lesser corner first.
func NewBox(low, high Vec) Box { return Box{Min: low, Max: high} }

// Everywhere is the box of a shape with no edge.
func Everywhere() Box {
	inf := math.Inf(unity)
	return Box{Vec{-inf, -inf, -inf}, Vec{inf, inf, inf}}
}

// Hit is where the ray enters and leaves the box, if it meets the box
// between near and far. Enter is never before near.
// Start inside with near not behind the box and it enters at near.
//
// Nothing is met if the ray, near or far is no number.
// Nothing is met if enter would be plus infinity.
// Nothing is met if leave would be minus infinity.
func (box Box) Hit(ray Ray, near, far float64) (enter, leave float64, ok bool) {
	enter, leave = near, far
	for _, axis := range [dimensions]struct{ at, dir, lo, hi float64 }{
		{ray.At.X, ray.Dir.X, box.Min.X, box.Max.X},
		{ray.At.Y, ray.Dir.Y, box.Min.Y, box.Max.Y},
		{ray.At.Z, ray.Dir.Z, box.Min.Z, box.Max.Z},
	} {
		if axis.dir == nought {
			// written so that NaN is outside, never inside
			if !(axis.at >= axis.lo && axis.at <= axis.hi) {
				return nought, nought, false
			}
			continue
		}
		t0, t1 := (axis.lo-axis.at)/axis.dir, (axis.hi-axis.at)/axis.dir
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		enter, leave = math.Max(enter, t0), math.Min(leave, t1)
		// written so that a NaN misses, since it never enters before
		// it leaves
		if !(enter <= leave) {
			return nought, nought, false
		}
	}
	// the near and far as given are checked here, for a ray along no axis
	if !(enter <= leave) || math.IsInf(enter, unity) ||
		math.IsInf(leave, -unity) {
		return nought, nought, false
	}
	return enter, leave, true
}

// Inverse is the reciprocal of each component of vec, for Box.Meets. A zero
// component gives an infinity, which Meets never reads: it asks the ray's
// own direction for zero first.
func (vec Vec) Inverse() Vec {
	return Vec{unity / vec.X, unity / vec.Y, unity / vec.Z}
}

// slack is the share by which a product by a reciprocal can differ from the
// quotient Hit takes: a few units in the last place, widened to nine.
//
//ninja:vow source nine units in the last place, picked; maths reads no file
const slack = 1e-15

// soundLow is the nearest to zero a distance may be for that share to hold:
// below it the share is not the error.
//
//ninja:vow source nearest zero a share holds at; maths reads no file
const soundLow = 1e-280

// soundHigh is the furthest from zero a distance may be for that share to
// hold: above it, near overflow, the share is not the error.
//
//ninja:vow source furthest from zero a share holds at; maths reads no file
const soundHigh = 1e280

// reciprocalRoom is how many times inside the sound range a direction's size
// must lie, so that its reciprocal is held to the last place.
//
//ninja:vow source room for a reciprocal to the last place; maths reads no file
const reciprocalRoom = 1e20

// Meets is whether Hit would say the ray meets the box between near and far,
// for a ray whose Dir's reciprocal is inverse. It is for a walk of many boxes
// by one ray, which takes the reciprocal once and divides by nothing, and it
// says nothing of where the ray enters or leaves.
//
// It multiplies where Hit divides, so each distance may be a unit or two in
// the last place from Hit's. It answers by that only when the answer is the
// same for every distance within slack of its own, which is nearly always,
// and otherwise it asks Hit: its answer is Hit's answer for every ray, box,
// near and far, NaN and infinity included.
func (box Box) Meets(ray Ray, inverse Vec, near, far float64) bool {
	// the widest interval the exact distances can fill, and the narrowest
	lowWide, lowNarrow, highWide, highNarrow := near, near, far, far
	for _, axis := range [...]struct{ at, dir, inv, lo, hi float64 }{
		{ray.At.X, ray.Dir.X, inverse.X, box.Min.X, box.Max.X},
		{ray.At.Y, ray.Dir.Y, inverse.Y, box.Min.Y, box.Max.Y},
		{ray.At.Z, ray.Dir.Z, inverse.Z, box.Min.Z, box.Max.Z},
	} {
		if axis.dir == nought {
			// as Hit asks it: NaN is outside, never inside
			if !(axis.at >= axis.lo && axis.at <= axis.hi) {
				return false
			}
			continue
		}
		// a box turned inside out, or a direction that is no number or so
		// small or large that its reciprocal is not to the last place, is
		// Hit's
		size := math.Abs(axis.dir)
		if !(axis.lo <= axis.hi) || !(size >= soundLow/reciprocalRoom &&
			size <= soundHigh/reciprocalRoom) {
			return box.hits(ray, near, far)
		}
		nearer, farther := axis.lo-axis.at, axis.hi-axis.at
		if axis.dir < nought {
			nearer, farther = farther, nearer
		}
		first, last := nearer*axis.inv, farther*axis.inv
		if !sound(first, nearer) || !sound(last, farther) {
			return box.hits(ray, near, far)
		}
		in, out := math.Abs(first)*slack, math.Abs(last)*slack
		lowWide, lowNarrow = max(lowWide, first-in), max(lowNarrow, first+in)
		highWide, highNarrow = min(highWide, last+out), min(highNarrow, last-out)
	}
	if lowWide > highWide {
		return false
	}
	if lowNarrow <= highNarrow && lowNarrow <= math.MaxFloat64 &&
		highNarrow >= -math.MaxFloat64 {
		return true
	}
	return box.hits(ray, near, far)
}

// sound is whether a distance got as a product from a numerator is one whose
// error is the share Meets allows: a number of a size where a share of it is
// the error, or zero from a zero.
func sound(distance, numerator float64) bool {
	size := math.Abs(distance)
	return size >= soundLow && size <= soundHigh ||
		distance == nought && numerator == nought
}

// hits is Hit's answer without its distances.
func (box Box) hits(ray Ray, near, far float64) bool {
	_, _, ok := box.Hit(ray, near, far)
	return ok
}

// Orientation is which way a subspace's axes point, as three arrows. They
// say where its own x, y and z go in its parent's space. They are one
// unit long and at right angles to each other.
//
// The zero Orientation is no turn at all. The empty Subspace is exactly
// its parent's space.
type Orientation struct {
	//ninja:vow gate a value, whose fields are what it is
	X, Y, Z Vec //ninja:vow name x, y, z at every call site
}

// Identity is no turn.
//
//ninja:vow well the turn that turns nothing; nothing in slayer writes it
var Identity = Orientation{XAxis, YAxis, ZAxis}

// axes is the orientation, with the zero one read as no turn.
func (turn Orientation) axes() Orientation {
	if turn == (Orientation{}) {
		return Identity
	}
	return turn
}

// Out takes a direction in the subspace out to its parent.
func (turn Orientation) Out(direction Vec) Vec {
	turn = turn.axes()
	return turn.X.Scale(direction.X).Add(turn.Y.Scale(direction.Y)).
		Add(turn.Z.Scale(direction.Z))
}

// In takes a direction in the parent into the subspace.
func (turn Orientation) In(direction Vec) Vec {
	turn = turn.axes()
	return Vec{direction.Dot(turn.X), direction.Dot(turn.Y),
		direction.Dot(turn.Z)}
}

// Then is this orientation followed by a child's inside it: the child's
// orientation as its grandparent sees it.
func (turn Orientation) Then(child Orientation) Orientation {
	inner := child.axes()
	return Orientation{turn.Out(inner.X), turn.Out(inner.Y),
		turn.Out(inner.Z)}
}

// Rotation is the world's own orientation turned by angle about axis. The
// turn is counterclockwise looking down the axis toward its start: the
// right-hand rule. Turning about no axis at all is no turn. So is an axis
// too short for Len to measure.
//
// If the axis or the angle is no number or is infinite, the orientation is
// of no numbers. So it is for an axis too long for Len to measure, a part
// past about 1e154. The fault then shows where it is used. It is not hidden
// as no turn.
func Rotation(axis Vec, angle float64) Orientation {
	if unmeasured(axis) {
		return Identity
	}
	if unreadable(axis) {
		return unreadableTurn()
	}
	about := turning{axis: axis.Unit(), sin: math.Sin(angle),
		cos: math.Cos(angle)}
	return Orientation{about.of(XAxis), about.of(YAxis), about.of(ZAxis)}
}

// turning is a turn about a unit axis, with the sine and cosine of its
// angle worked out once.
type turning struct {
	axis     Vec
	sin, cos float64
}

// of is direction turned. rodrigues: the part of direction along the axis
// stays where it is, and the rest turns in the plane at right angles to
// the axis.
func (about turning) of(direction Vec) Vec {
	far := about.axis.Dot(direction) * rest(about.cos)
	along := about.axis.Scale(far)
	turned := direction.Scale(about.cos).
		Add(about.axis.Cross(direction).Scale(about.sin))
	return turned.Add(along)
}

// About is Rotation by another name, for the callers that still use it.
// New code calls Rotation.
func About(axis Vec, angle float64) Orientation { return Rotation(axis, angle) }

// LookAt is the orientation of something at from, facing to. Its up is
// kept as near the world's up as it can be. Forward is minus z.
func LookAt(from, to Vec) Orientation { return LookAtUp(from, to, Up) }

// LookAtUp is LookAt with an up of its own. It is for someone standing on
// a ground that is not the world's, whose up is the ground's. Give it an
// up of nothing and it uses the world's. Looking at where you stand is no
// turn. Nothing here includes anything too short for Len to measure.
//
// Within about 2.56 degrees of looking straight along up, the top of the
// view is taken as minus z instead. If the view is also within that of
// along z, the top is taken as plus x.
//
// So a camera sweeping past straight up rolls there.
//
// If up, or the distance from to to from, is no number, is infinite, or is
// too long for Len to measure, a part past about 1e154, the orientation is
// of no numbers, as it is for Rotation.
func LookAtUp(from, to, up Vec) Orientation {
	if from == to || unmeasured(from.Sub(to)) {
		return Identity
	}
	if unreadable(from.Sub(to)) || unreadable(up) {
		return unreadableTurn()
	}
	if unmeasured(up) {
		up = Up
	}
	back := from.Sub(to).Unit()
	up = up.Unit()
	if math.Abs(back.Dot(up)) > nearlyAlong {
		// looking straight up or down: any level will do
		up = ZAxis.Neg()
		if math.Abs(back.Dot(up)) > nearlyAlong {
			up = XAxis
		}
	}
	x := up.Cross(back).Unit()
	return Orientation{x, back.Cross(x), back}
}

// Upright is the orientation of something standing on up. Its y is up.
// Its x and z are a pair at right angles to it, chosen the same way every
// time. Standing on no up at all is no turn, nor is standing on one too
// short for Len to measure. An up that is no number, is infinite, or is too
// long for Len to measure gives an orientation of no numbers.
func Upright(up Vec) Orientation {
	if unmeasured(up) {
		return Identity
	}
	if unreadable(up) {
		return unreadableTurn()
	}
	y := up.Unit()
	other := XAxis
	if math.Abs(y.Dot(other)) > nearlyAlongX {
		other = ZAxis
	}
	share := y.Scale(y.Dot(other))
	x := other.Sub(share).Unit()
	return Orientation{x, y, x.Cross(y)}
}

// Subspace is a space inside a parent space. It says where its zero sits
// and which way its axes point. It says how long one of its units is. All
// of it is in the parent's terms.
//
// The empty Subspace is exactly its parent's space. Scale is finite and
// not zero, and zero is read as one. Negative ones turn rays inside
// back to front.
type Subspace struct {
	//ninja:vow gate a value, whose fields are what it is
	Origin Vec
	//ninja:vow gate a value, whose fields are what it is
	Orientation Orientation
	//ninja:vow gate a value, whose fields are what it is
	Scale float64 // zero means one
}

// Local is the unchanged frame, with the documented zero-value semantics.
func Local() Subspace { return Subspace{} }

// NewSubspace describes a frame in its parent. Placement users validate the
// supplied values as they do for Subspace literals; zero scale means one.
func NewSubspace(origin Vec, orientation Orientation, scale float64) Subspace {
	return Subspace{Origin: origin, Orientation: orientation, Scale: scale}
}

// scale is the subspace's scale, with zero read as one.
func (space Subspace) scale() float64 {
	if space.Scale == nought {
		return unity
	}
	return space.Scale
}

// Out takes a point in the subspace out to its parent.
func (space Subspace) Out(point Vec) Vec {
	scaled := point.Scale(space.scale())
	return space.Origin.Add(space.Orientation.Out(scaled))
}

// In takes a point in the parent into the subspace.
func (space Subspace) In(point Vec) Vec {
	moved := point.Sub(space.Origin)
	return space.Orientation.In(moved).Scale(unity / space.scale())
}

// RayIn takes a ray in the parent into the subspace.
// Its direction stays one unit long, so distances along it shrink by the
// scale. Take t inside: it is t times Scale outside.
// Zero for Scale is read as one, as everywhere else.
func (space Subspace) RayIn(ray Ray) Ray {
	return Ray{At: space.In(ray.At), Dir: space.Orientation.In(ray.Dir)}
}

// Forward is which way the subspace faces, in its parent: its minus z.
func (space Subspace) Forward() Vec {
	return space.Orientation.Out(ZAxis.Neg())
}

// Within is this subspace as the grandparent sees it, given the parent it
// sits in. It is the absolute answer, one multiplication a level.
func (space Subspace) Within(parent Subspace) Subspace {
	return Subspace{
		Origin:      parent.Out(space.Origin),
		Orientation: parent.Orientation.axes().Then(space.Orientation),
		Scale:       parent.scale() * space.scale(),
	}
}
