// Package conifers constructs the recursive shoots used by conifer foliage.
package conifers

import (
	"math"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
)

const SpruceStyle = 3

// Axis is one continuing woody axis and its recursive parent.
type Axis struct {
	At, Direction, Normal maths.Vec
	Length, Radius        float64
	Parent, Order         int
}

// Needle keeps physical leaf dimensions separate from shoot size.
type Needle struct {
	Root, Tip, Across, Normal maths.Vec
	Width, Thickness          float64
	Axis, Rank                int
}

// Plan is the botanical scaffold. Detail changes only its surfaces.
type Plan struct {
	Axes    []Axis
	Needles []Needle
}

// ShootProfile holds only species-specific geometry, not provenance.
type ShootProfile struct {
	Low, High, Width, Thickness, Spacing, Angle float64
	Rows, WoodSides                             int
}

// AxisSpace retains a spray plane through successive lateral branches.
func AxisSpace(axis Axis) maths.Subspace {
	side := axis.Direction.Cross(axis.Normal).Unit()
	return maths.Subspace{Origin: axis.At, Orientation: maths.Orientation{
		X: side, Y: axis.Direction, Z: side.Cross(axis.Direction),
	}}
}

// PlanShoot adapts ABOP chapter 2, figure 2.6: a continuing axis produces
// contracting lateral axes, alternating left and right in the spray plane.
// Child radii use Weber and Penn (1995), section 4.4. This is an adaptation,
// not their complete tree model. Choices live in needled-shoot.yml.
func PlanShoot(style int, length float64, from branching.Seed, foliage ShootProfile, value func(string) float64, normalOf func(maths.Vec) maths.Vec) Plan {
	plan := Plan{}
	var grow func(Axis)
	grow = func(axis Axis) {
		index := len(plan.Axes)
		plan.Axes = append(plan.Axes, axis)
		plan.addNeedles(style, axis, index, from.Sprout(index, 0), foliage, value, normalOf)
		if axis.Order >= int(value("shoot-orders")) {
			return
		}
		count := int(value("shoot-first-laterals"))
		if axis.Order > 0 {
			count = int(value("shoot-second-laterals"))
		}
		contraction := value("shoot-internode-contraction")
		total := (1 - math.Pow(contraction, float64(count))) / (1 - contraction)
		position := 0.0
		space := AxisSpace(axis)
		for node := range count {
			internode := math.Pow(contraction, float64(node))
			u := (position + internode/2) / total
			position += internode
			childLength := axis.Length * value("shoot-lateral-contraction") * (1 - u)
			if childLength < 2*foliage.Spacing {
				continue
			}
			side := space.Orientation.X
			if style == 2 {
				side = maths.About(axis.Direction, float64(node)*math.Pi*(3-math.Sqrt(5))).Out(side)
			} else if node%2 != 0 {
				side = side.Neg()
			}
			angle := (value("shoot-lateral-angle") +
				value("shoot-angle-variation")*from.Swing(index, node, 0)) * maths.Degree
			direction := axis.Direction.Scale(math.Cos(angle)).Add(
				side.Scale(math.Sin(angle))).Unit()
			// Twig radius follows the length ratio.
			radius := axis.Radius * math.Pow(childLength/axis.Length,
				value("shoot-radius-power"))
			normal := axis.Normal
			if style == 2 {
				normal = side.Cross(axis.Direction).Unit()
			}
			grow(Axis{
				At:        axis.At.Add(axis.Direction.Scale(u * axis.Length)),
				Direction: direction, Normal: normal, Length: childLength, Radius: radius,
				Parent: index, Order: axis.Order + 1,
			})
		}
	}
	grow(Axis{
		Direction: maths.YAxis, Normal: maths.ZAxis, Length: length,
		Radius: value("shoot-base-radius") * length / value("shoot-cell-length"), Parent: -1,
	})
	return plan
}

// needleSpacing is an authored spacing, not implied by a published length.

// addNeedles uses redwood's flat ranks, sequoia's appressed scale rows, or
// spruce needles around the stem. Redwood/sequoia lengths are in research
// data. Spruce dimensions, rows and alternating phases are authored choices.
func (plan *Plan) addNeedles(style int, axis Axis, index int, from branching.Seed, foliage ShootProfile, value func(string) float64, normalOf func(maths.Vec) maths.Vec) {
	rows, low, high := foliage.Rows, foliage.Low, foliage.High
	count := max(1, int(axis.Length/foliage.Spacing))
	space := AxisSpace(axis)
	angle := foliage.Angle * maths.Degree
	for node := range count {
		u := (float64(node) + 0.5) / float64(count)
		for rank := range rows {
			out := space.Orientation.X
			if style == 2 || style == SpruceStyle {
				phase := 0.0
				if style == SpruceStyle && node%2 != 0 {
					phase = math.Pi / float64(rows)
				}
				out = maths.Rotation(axis.Direction,
					(2*math.Pi)*float64(rank)/float64(rows)+phase).Out(out)
			} else if rank != 0 {
				out = out.Neg()
			}
			// Follow the woody taper, including its nonzero tip.
			radius := axis.Radius * (1 - (1-value("shoot-tip-radius"))*u)
			root := axis.At.Add(axis.Direction.Scale(u * axis.Length)).Add(
				out.Scale(radius))
			way := out.Scale(math.Sin(angle)).Add(axis.Direction.Scale(math.Cos(angle)))
			long := low + (high-low)*from.Draw(node, rank, 0)
			normal := axis.Normal
			if style == 2 || style == SpruceStyle {
				normal = normalOf(axis.Direction.Cross(out))
			}
			plan.Needles = append(plan.Needles, Needle{
				Root: root, Tip: root.Add(way.Scale(long)),
				Across: normal.Cross(way).Unit(), Normal: normal,
				Width: foliage.Width, Thickness: foliage.Thickness, Axis: index, Rank: rank,
			})
		}
	}
}

// NeedleSkin resolves a needle as an open diamond, a closed tetrahedron, or a
// thin closed diamond. No representation contains a green woody sleeve.
func NeedleSkin[F geometry.Triangle](needle Needle, detail string) []F {
	middle := needle.Root.Add(needle.Tip).Scale(0.5)
	left, right := middle.Add(needle.Across.Scale(needle.Width/2)), middle.Sub(needle.Across.Scale(needle.Width/2))
	if detail == "clumps" {
		return []F{{needle.Root, right, left}, {left, right, needle.Tip}}
	}
	var skin []F
	if detail == "quick" {
		tip := needle.Tip.Add(needle.Normal.Scale(needle.Thickness / 2))
		skin = []F{{needle.Root, left, right}, {needle.Root, tip, left}, {left, tip, right}, {right, tip, needle.Root}}
	} else {
		top, bottom := middle.Add(needle.Normal.Scale(needle.Thickness/2)), middle.Sub(needle.Normal.Scale(needle.Thickness/2))
		ring := []maths.Vec{needle.Root, left, needle.Tip, right}
		for i, point := range ring {
			next := ring[(i+1)%4]
			skin = append(skin, F{top, point, next}, F{bottom, next, point})
		}
	}
	// The same construction serves both ranks and all three scale rows.
	volume := 0.0
	origin := skin[0][0]
	for _, face := range skin {
		volume += face[0].Sub(origin).Dot(face[1].Sub(origin).Cross(face[2].Sub(origin)))
	}
	if volume < 0 {
		for i := range skin {
			skin[i][1], skin[i][2] = skin[i][2], skin[i][1]
		}
	}
	return skin
}
