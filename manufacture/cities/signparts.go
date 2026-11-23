package cities

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// tubeSides is how many flat sides the section of a swept tube has: six
// is round enough at the size of a street.
const tubeSides = 6

// miterFloor is the least share of its radius that a tube keeps at a
// corner: where two steps meet at a sharper corner than the share allows,
// the tube is no thicker than that, so a spike does not run off.
const miterFloor = 0.4

// steepBar is how upright a bar is, as the share of its length that is
// height, when its sideways reference is the x axis and not the vertical.
const steepBar = 0.99

// meshForm is a form that is a set of triangles already made: a swept tube
// or any shape built by hand.
type meshForm struct{ triangles []Triangle }

// Mesh is a copy of the triangles, the same at every level.
func (form meshForm) Mesh(Detail) (*Mesh, error) {
	return &Mesh{Triangles: slices.Clone(form.triangles)}, nil
}

// beam is a bar from one point to another of a frame, as thick as the
// first side across it and as wide as the second. Its thick side runs
// perpendicular to the bar and the vertical, as far as the bar allows,
// and its wide side across both: a flat bar laid on a slope has its wide
// side along the horizon.
func beam(name string, from, to maths.Vec, thick, wide float64,
	material *thing.Material) placedElement {
	way := to.Sub(from)
	along := way.Scale(1 / way.Len())
	helper := maths.Up
	if math.Abs(along.Dot(maths.Up)) > steepBar {
		helper = maths.XAxis
	}
	across := along.Cross(helper).Unit()
	up := across.Cross(along)
	origin := from.Sub(up.Scale(thick / half)).Sub(across.Scale(wide / half))
	return placedElement{Element: solid{Name: name, Form: Box{way.Len(), thick,
		wide}, Material: material}, Place: maths.NewSubspace(origin,
		maths.Orientation{X: along, Y: up, Z: across}, 0)}
}

// strut is a square bar of a side from one point to another.
func strut(name string, from, to maths.Vec, side float64,
	material *thing.Material) placedElement {
	return beam(name, from, to, side, side, material)
}

// plateFrame is the frame of a plate that looks out along a direction, with
// its up as given: its z is the direction, its y the up, its origin the
// point given. Parts built in it stand out of the plate toward +z.
func plateFrame(origin, out, up maths.Vec) maths.Subspace {
	return maths.NewSubspace(origin, maths.Orientation{
		X: up.Cross(out), Y: up, Z: out}, 0)
}

// fixing is a plate of a width and a height with a bolt at each corner,
// in the frame of its own: its front at z nought, its middle at the
// origin, standing back from the front by the plate's thickness. A person
// at street level sees the bolts.
func (job signJob) fixing(wide, high float64) []placedElement {
	drawing := job.drawing
	thick, side := drawing.fixed("sign-plate-thick"),
		drawing.fixed("sign-bolt-side")
	inset, head := drawing.fixed("sign-bolt-inset"),
		drawing.fixed("sign-bolt-head")
	parts := []placedElement{put("sign plate", Box{wide, high, thick}, job.frame,
		maths.V(-wide/half, -high/half, -thick))}
	for _, across := range []float64{-1, 1} {
		for _, up := range []float64{-1, 1} {
			at := maths.V(across*(wide/half-inset), up*(high/half-inset), 0)
			parts = append(parts, put("sign bolt", Box{side, side, head},
				job.frame, at.Sub(maths.V(side/half, side/half, 0))))
		}
	}
	return parts
}

// plated is a plate with its bolts hung on a surface: at a point of the
// sign's frame, looking out along a direction, with its up as given.
func (job signJob) plated(at, out, up maths.Vec, wide,
	high float64) []placedElement {
	return within(job.fixing(wide, high), plateFrame(at, out, up))
}

// sections are the rings of a tube's section, one at each point of the
// path: tubeSides corners round the point, in the plane across the path
// and out of the page. The path is flat, in a plane of constant z.
func sections(path []maths.Vec, closed bool, radius float64) [][]maths.Vec {
	rings := make([][]maths.Vec, len(path))
	for index, at := range path {
		before, after := at, at
		switch {
		case closed:
			before = path[(index+len(path)-1)%len(path)]
			after = path[(index+1)%len(path)]
		case index > 0:
			before = path[index-1]
		}
		if !closed && index+1 < len(path) {
			after = path[index+1]
		}
		side := cornerOf(before, at, after, radius)
		ring := make([]maths.Vec, tubeSides)
		for step := range ring {
			angle := fullTurn * float64(step) / tubeSides
			ring[step] = at.Add(side.Scale(math.Cos(angle))).Add(
				maths.ZAxis.Scale(radius * math.Sin(angle)))
		}
		rings[index] = ring
	}
	return rings
}

// cornerOf is how far out of a path's point, in its plane, a tube of a
// radius reaches at the point, where the path comes from one point and goes
// to another: the radius along the way that bisects the two steps' normals,
// longer at a corner by as much as keeps the walls a radius thick, and no
// longer than the floor allows.
func cornerOf(before, at, after maths.Vec, radius float64) maths.Vec {
	in, out := at.Sub(before), after.Sub(at)
	if in.Len() == 0 {
		in = out
	}
	if out.Len() == 0 {
		out = in
	}
	in, out = in.Unit(), out.Unit()
	first, second := maths.V(in.Y, -in.X, 0), maths.V(out.Y, -out.X, 0)
	bisector := first.Add(second).Unit()
	return bisector.Scale(radius / math.Max(bisector.Dot(first), miterFloor))
}

// sweep is the triangles of a tube of a radius along a path, closed on
// itself or open, with its ends capped if it is open. The path is flat, in
// a plane of constant z, and the tube's section is a hexagon.
func sweep(path []maths.Vec, closed bool, radius float64) []Triangle {
	rings := sections(path, closed, radius)
	steps := len(path) - 1
	if closed {
		steps = len(path)
	}
	var skin []Triangle
	for index := range steps {
		next := (index + 1) % len(path)
		axis := path[index].Add(path[next]).Scale(1 / float64(half))
		for step := range tubeSides {
			round := (step + 1) % tubeSides
			quad := []maths.Vec{rings[index][step], rings[index][round],
				rings[next][round], rings[next][step]}
			away := middle(quad).Sub(axis)
			skin = append(skin,
				outward(Triangle{quad[0], quad[1], quad[2]}, away),
				outward(Triangle{quad[0], quad[2], quad[3]}, away))
		}
	}
	if !closed {
		last := len(path) - 1
		skin = append(skin, fill(rings[0], path[0].Sub(path[1]))...)
		skin = append(skin, fill(rings[last],
			path[last].Sub(path[last-1]))...)
	}
	return skin
}
