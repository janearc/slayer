package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// MountKind is how a sign hangs on a building: as big words on a frame
// at a roof's edge, as a screen flat on a wall, as a blade, letters
// one under the next on a bracket out from a wall, or as an awning's
// valance over the foot of a wall.
type MountKind string

// The ways a sign hangs.
const (
	// MountRoof is a frame of big words standing on a roof's edge.
	MountRoof MountKind = "roof"
	// MountScreen is a panel carrying lines of letters, flat on a wall.
	MountScreen MountKind = "screen"
	// MountBlade is letters stacked upright, one under the next, on a
	// board hung from a bracket that stands out of a wall.
	MountBlade MountKind = "blade"
	// MountAwning is a sloping canopy of cloth over the foot of a wall,
	// with a valance hung from its front edge that carries words.
	MountAwning MountKind = "awning"
)

// signMount is a place where a building says that a sign may hang: a
// point, which way the sign faces, and the room it has. All of it is in
// the building's frame, in metres.
//
// For a roof sign the point is on the roof at the middle of the stretch
// of edge it may take, set back from the edge; for a screen it is the
// middle of the screen's foot on the wall; for a blade sign it is where
// the top of the bracket meets the wall.
type signMount struct {
	Kind MountKind
	// Point is where the sign is anchored to the building.
	Point maths.Vec
	// Out is the horizontal unit vector away from the building at the
	// point, the way a roof sign or a screen faces and a blade sign's
	// bracket runs. Along is the horizontal unit vector along the wall
	// to the right of a viewer looking at it.
	Out, Along maths.Vec
	// Wide is the room along the wall or the roof edge. High is the room
	// up from the point for a roof sign or a screen, and down from it
	// for a blade sign. Reach is how far the sign may stand out of the
	// building: the depth of a roof sign or a screen, the bracket of a
	// blade sign.
	Wide, High, Reach float64
}

// Mounting is a building that offers places for signs. The tower and a
// design of masses do; a sign has no say in where it is hung.
type Mounting interface {
	// SignMounts is every place the building offers, each with the room
	// it has. It offers more than are used: a street draws which.
	SignMounts() ([]signMount, error)
}

// space is the frame a sign is drawn in, with its origin at the mount's
// point. A roof sign or a screen has its x along the wall, its y up and
// its z out of the building, so its letters read left to right to a
// viewer facing it. A blade sign has its x out of the building along its
// bracket, its y up and its z across, which its two faces face.
func (mount signMount) space() maths.Subspace {
	if mount.Kind == MountBlade {
		return maths.NewSubspace(mount.Point, maths.Orientation{
			X: mount.Out, Y: maths.Up, Z: mount.Out.Cross(maths.Up)}, 0)
	}
	return maths.NewSubspace(mount.Point, maths.Orientation{
		X: mount.Along, Y: maths.Up, Z: mount.Out}, 0)
}

// wallEdge is an edge of a building's plan with what it offers: the
// middle of its foot, the horizontal unit vector away from the building
// there, how long it is and how high, and the stretches of it, as
// distances from its middle along it, that no other part of the plan
// covers. No stretches given means all of it is clear.
type wallEdge struct {
	mid, out       maths.Vec
	length, height float64
	spans          [][2]float64
}

// along is the unit vector along the edge, to the right of a viewer
// looking at it from outside.
func (wall wallEdge) along() maths.Vec { return maths.Up.Cross(wall.out) }

// clear is whether the stretch of the edge from one distance to another
// along it, from its middle, is a stretch nothing covers.
func (wall wallEdge) clear(low, high float64) bool {
	if len(wall.spans) == 0 {
		return true
	}
	for _, span := range wall.spans {
		if span[0] <= low && high <= span[1] {
			return true
		}
	}
	return false
}

// edgesOf are the edges of a plan given by its corners, in order, at one
// height, each facing away from the centre of the plan, with the height
// the wall has above it.
func edgesOf(corners []maths.Vec, centre maths.Vec,
	height float64) []wallEdge {
	var edges []wallEdge
	for index, start := range corners {
		end := corners[(index+1)%len(corners)]
		mid := start.Add(end).Scale(1.0 / half)
		run := end.Sub(start)
		out := maths.Up.Cross(run).Unit()
		if out.Dot(mid.Sub(centre)) < 0 {
			out = out.Neg()
		}
		edges = append(edges, wallEdge{mid: mid, out: out,
			length: run.Len(), height: height})
	}
	return edges
}

// roofMounts is the place a roof sign may stand along a roof's edge, if
// the edge is long enough: the middle of the edge, set back, with the
// edge's length less a margin at each end for room.
func (drawing *drawer) roofMounts(edge wallEdge) []signMount {
	wide := edge.length - 2*drawing.fixed("sign-edge-margin")
	if wide < drawing.fixed("sign-least-wide") {
		return nil
	}
	back := drawing.fixed("sign-roof-set-back")
	return []signMount{{Kind: MountRoof,
		Point: edge.mid.Sub(edge.out.Scale(back)), Out: edge.out,
		Along: edge.along(), Wide: wide,
		High: drawing.fixed("sign-roof-high"), Reach: back}}
}

// wallMounts are the places a screen and a blade sign may hang on a wall,
// if it is upright, long enough and high enough: the screen across the
// middle of it, the blade sign at its start, so that they clear each
// other.
func (drawing *drawer) wallMounts(wall wallEdge) []signMount {
	least := drawing.fixed("sign-least-high")
	if wall.length < drawing.fixed("sign-least-wide") || wall.height < least {
		return nil
	}
	var mounts []signMount
	rise, along := drawing.fixed("sign-screen-rise"), wall.along()
	high := math.Min(drawing.fixed("sign-screen-high"), wall.height-rise)
	wide := wall.length - 2*drawing.fixed("sign-screen-margin")
	if high >= least && wide >= drawing.fixed("sign-least-wide") &&
		wall.clear(-wide/half, wide/half) {
		mounts = append(mounts, signMount{Kind: MountScreen,
			Point: wall.mid.Add(maths.Up.Scale(rise)), Out: wall.out,
			Along: along, Wide: wide, High: high,
			Reach: drawing.fixed("sign-screen-reach")})
	}
	mounts = append(mounts, drawing.bladeMount(wall))
	return append(mounts, drawing.awningMounts(wall)...)
}

// bladeMount is the place a blade sign may hang at the start of a wall,
// its bracket's top at the most the wall gives it above the lowest it
// may hang.
func (drawing *drawer) bladeMount(wall wallEdge) signMount {
	inset, rise := drawing.fixed("sign-blade-inset"),
		drawing.fixed("sign-blade-rise")
	high := math.Min(drawing.fixed("sign-blade-high"), wall.height-rise)
	start := wall.mid.Sub(wall.along().Scale(wall.length/half - inset))
	return signMount{Kind: MountBlade,
		Point: start.Add(maths.Up.Scale(rise + high)), Out: wall.out,
		Along: wall.along(), High: high,
		Reach: drawing.fixed("sign-blade-reach")}
}

// upright is whether a face of a tower stands up straight enough to hang
// a sign flat on it: the share of its length that is height.
func (drawing *drawer) upright(frame faceFrame) bool {
	return frame.up.Y >= drawing.fixed("sign-vertical")
}

// SignMounts is the places a tower offers: along the top edge of each
// shaft's roof, a roof sign; and on each upright face of the lowest
// segment of each shaft, a screen and a blade sign. A face that another
// part of the plan covers offers only what it shows. The tower is not
// drawn to say them, and nothing of it changes.
func (tower *Tower) SignMounts() ([]signMount, error) {
	drawing, err := newDrawer(signCulture, 0)
	if err != nil {
		return nil, err
	}
	var mounts []signMount
	for index, shaft := range tower.Shafts {
		found, err := tower.shaftMounts(drawing, index, shaft)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, found...)
	}
	return mounts, drawing.trouble
}

// shaftMounts is the places one shaft offers.
func (tower *Tower) shaftMounts(drawing *drawer, index int,
	shaft shaft) ([]signMount, error) {
	bottom, top, err := shaft.layer(0)
	if err != nil {
		return nil, err
	}
	_, ends, err := shaft.layer(len(shaft.Segments) - 1)
	if err != nil {
		return nil, err
	}
	var mounts []signMount
	for _, part := range ends {
		corners, roof := roofCorners(part, tower.height(index)),
			centroid(part)
		for _, edge := range edgesOf(corners,
			maths.V(roof.X, tower.height(index), roof.Z), 0) {
			mounts = append(mounts, drawing.roofMounts(edge)...)
		}
	}
	height := float64(shaft.Segments[0].Storeys) * tower.storeyHeight()
	for _, frame := range frames(bottom, top, tower.baseHeight(), height) {
		if !drawing.upright(frame) {
			continue
		}
		mounts = append(mounts, drawing.wallMounts(wallEdge{
			mid: frame.origin, out: level(frame.out), length: frame.length,
			height: height, spans: frame.spans})...)
	}
	return mounts, nil
}

// roofCorners is the corners of a plan as points at a height.
func roofCorners(plan []outlineCorner, height float64) []maths.Vec {
	points := make([]maths.Vec, len(plan))
	for index, corner := range plan {
		points[index] = maths.V(corner.X, height, corner.Z)
	}
	return points
}

// level is the horizontal unit vector under a vector.
func level(vec maths.Vec) maths.Vec { return maths.V(vec.X, 0, vec.Z).Unit() }

// coverProbe is how many slacks out of a wall the place is that is asked
// whether a box hides a mount: two, so that it is two slacks inside a box
// that stands against the wall, which is more than covered asks.
const coverProbe = 2.0

// SignMounts is the places a design of masses offers: for each mass that
// is a box, along its top edges, roof signs, and on each of its walls, a
// screen and a blade sign, except where another box covers the place.
// Masses of any other form offer nothing.
func (design Design) SignMounts() ([]signMount, error) {
	drawing, err := newDrawer(signCulture, 0)
	if err != nil {
		return nil, err
	}
	var mounts []signMount
	slack := drawing.fixed("sign-cover-slack")
	for _, mass := range design.Massing {
		box, isBox := mass.Form.(Box)
		if !isBox {
			continue
		}
		for _, offered := range drawing.boxMounts(mass.Place, box) {
			// A mount is on the surface of its own box, which is on the
			// surface of a box beside it too, and neither hides it: the
			// place asked about is one probe out of the wall.
			probe := offered.Point.Add(offered.Out.Scale(coverProbe * slack))
			if !design.covered(probe, slack) {
				mounts = append(mounts, offered)
			}
		}
	}
	return mounts, drawing.trouble
}

// boxMounts is the places one box of a design offers, in the design's
// frame.
func (drawing *drawer) boxMounts(place maths.Subspace,
	box Box) []signMount {
	foot := []maths.Vec{place.Out(maths.V(0, 0, 0)),
		place.Out(maths.V(box.Width, 0, 0)),
		place.Out(maths.V(box.Width, 0, box.Depth)),
		place.Out(maths.V(0, 0, box.Depth))}
	high := place.Out(maths.V(0, box.Height, 0)).Y - foot[0].Y
	centre := place.Out(maths.V(box.Width/half, 0, box.Depth/half))
	var mounts []signMount
	for _, wall := range edgesOf(foot, centre, high) {
		mounts = append(mounts, drawing.wallMounts(wall)...)
		wall.mid = wall.mid.Add(maths.Up.Scale(high))
		mounts = append(mounts, drawing.roofMounts(wall)...)
	}
	return mounts
}

// covered is whether a point is inside any box of the design by more
// than the slack, in metres: a place that another mass hides. A point on
// the surface of its own box is not inside it, whatever the rounding.
func (design Design) covered(point maths.Vec, slack float64) bool {
	for _, mass := range design.Massing {
		box, isBox := mass.Form.(Box)
		if !isBox {
			continue
		}
		local := mass.Place.In(point)
		if local.X > slack && local.X < box.Width-slack &&
			local.Y > slack && local.Y < box.Height-slack &&
			local.Z > slack && local.Z < box.Depth-slack {
			return true
		}
	}
	return false
}
