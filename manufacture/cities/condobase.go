package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// canopy is the roof over a stair core's entrance: a slab standing out of
// the core's front over the door, held by a post at each outer corner.
// The building's frame, its front on the plane at z equal to the depth.
func (drawing *drawer) canopy(front, left float64) []placedElement {
	rise, deep := drawing.fixed("canopy-rise"), drawing.fixed("canopy-depth")
	side, thick := drawing.fixed("canopy-side"), drawing.fixed(
		"canopy-thickness")
	post := drawing.fixed("canopy-post")
	wide := drawing.fixed("core-width") + 2*side
	steel := paint("canopy", drawing.pick("iron-colour",
		drawing.options("iron-colour"), place{}))
	parts := []placedElement{put("canopy", Box{wide, thick, deep}, steel,
		maths.V(left-side, rise, front))}
	for _, across := range []float64{left - side, left + wide - side - post} {
		parts = append(parts, put("canopy post", Box{post, rise, post}, steel,
			maths.V(across, 0, front+deep-post)))
	}
	return parts
}

// podium is the row of shops along the foot of a building's front, from
// past its stair core to its end: for each a fascia sign that glows, and
// a canopy of its own under it. The building's frame.
func (drawing *drawer) podium(at place, depth, from, length float64) []placedElement {
	count := drawing.count("shops", at)
	gap := drawing.fixed("min-gap")
	wide := (length - from - float64(count+1)*gap) / float64(count)
	rise, high := drawing.fixed("shop-fascia-rise"),
		drawing.fixed("shop-sign-height")
	var parts []placedElement
	for index := range count {
		left := from + gap + float64(index)*(wide+gap)
		look := drawing.choose("shop-colour", scopeBay, fmt.Sprint(index),
			drawing.options("shop-colour"), at, nil)
		parts = append(parts,
			put("shop sign", Box{wide, high, drawing.fixed("shop-sign-proud")},
				drawing.glowing("shop sign", look), maths.V(left, rise, depth)),
			put("shop canopy", Box{wide, drawing.fixed("canopy-thickness"),
				drawing.fixed("shop-canopy-depth")}, paint("canopy", look),
				maths.V(left, rise-drawing.fixed("canopy-thickness"), depth)))
	}
	return parts
}

// glowing is a material of an option's colour that is also a source of
// light, as a sign is, at the culture's strength.
func (drawing *drawer) glowing(role string, one option) *thing.Material {
	look := paint(role, one)
	look.Glow = colourOf(one).Scale(drawing.fixed("sign-glow"))
	return look
}

// court furnishings are drawn in the court's frame, its middle at the
// origin and its paving's top at the paving's thickness.

// furnish is the shared ground of the court: a raised plaza with a ramp
// up to it, two paths across, planting, benches, and a few parked cars.
func (shared court) furnish(drawing *drawer) []placedElement {
	thick := drawing.fixed("paving-thickness")
	parts := shared.plaza(drawing, thick)
	parts = append(parts, shared.paths(drawing, thick)...)
	parts = append(parts, shared.planting(drawing, thick)...)
	return append(parts, shared.cars(drawing, thick)...)
}

// plaza is a raised platform at the court's middle, with a ramp up to it
// from the east, and a bench on each of its long sides.
func (shared court) plaza(drawing *drawer, thick float64) []placedElement {
	wide, deep := drawing.fixed("plaza-width"), drawing.fixed("plaza-depth")
	rise := drawing.fixed("plaza-rise")
	stone := paint("plaza", drawing.pick("plaza-colour",
		drawing.options("plaza-colour"), place{}))
	parts := []placedElement{put("plaza", Box{wide, rise, deep}, stone,
		maths.V(-wide/half, thick, -deep/half))}
	parts = append(parts, shared.ramp(drawing, stone, thick, wide/half)...)
	bench := drawing.fixed("bench-length")
	wood := paint("bench", drawing.pick("bench-colour",
		drawing.options("bench-colour"), place{}))
	for _, side := range []float64{-1, 1} {
		parts = append(parts, put("bench", Box{bench, drawing.fixed(
			"bench-height"), drawing.fixed("bench-width")}, wood,
			maths.V(-bench/half, thick+rise, side*(deep/half-
				drawing.fixed("bench-width")*benchInset))))
	}
	return parts
}

// benchInset is how many bench widths in from the plaza's edge a bench
// stands. A choice, by eye.
const benchInset = 1.5

// ramp is a slab sloping down from the plaza's east edge to the court.
func (shared court) ramp(drawing *drawer, stone *thing.Material, thick,
	edge float64) []placedElement {
	rise, run := drawing.fixed("plaza-rise"), drawing.fixed("ramp-run")
	across, slab := drawing.fixed("ramp-width"), drawing.fixed("ramp-slab")
	slope := math.Atan2(rise, run)
	start := maths.V(edge, thick+rise-slab*math.Cos(slope), -across/half)
	return []placedElement{{Place: maths.NewSubspace(start, maths.Rotation(
		maths.ZAxis, -slope), 0), Element: solid{Name: "ramp", Material: stone,
		Form: Box{math.Hypot(run, rise), slab, across}}}}
}

// paths are two strips across the court, one along each of its axes,
// lighter than the paving, as wide as the culture says.
func (shared court) paths(drawing *drawer, thick float64) []placedElement {
	wide := drawing.fixed("path-width")
	laid := paint("path", drawing.pick("path-colour",
		drawing.options("path-colour"), place{}))
	height := drawing.fixed("path-height")
	return []placedElement{
		put("path", Box{shared.width, height, wide}, laid,
			maths.V(-shared.width/half, thick, -wide/half)),
		put("path", Box{wide, height, shared.depth}, laid,
			maths.V(-wide/half, thick, -shared.depth/half)),
	}
}

// planting is a bed at each corner of the plaza with a tree in it: a
// trunk and a cone of leaves.
func (shared court) planting(drawing *drawer, thick float64) []placedElement {
	wide, deep := drawing.fixed("plaza-width"), drawing.fixed("plaza-depth")
	rise, bed := drawing.fixed("plaza-rise"), drawing.fixed("bed-size")
	soil := paint("bed", drawing.pick("bed-colour",
		drawing.options("bed-colour"), place{}))
	var parts []placedElement
	for _, across := range []float64{-1, 1} {
		for _, along := range []float64{-1, 1} {
			corner := maths.V(across*(wide/half-bed), thick+rise,
				along*(deep/half-bed))
			parts = append(parts, put("bed", Box{bed, drawing.fixed(
				"bed-height"), bed}, soil, corner.Sub(maths.V(bed/half,
				0, bed/half))))
			parts = append(parts, shared.tree(drawing, corner.Add(
				maths.V(0, drawing.fixed("bed-height"), 0)))...)
		}
	}
	return parts
}

// tree is a trunk with a cone of leaves on it, standing at a point.
func (shared court) tree(drawing *drawer, foot maths.Vec) []placedElement {
	trunk, high := drawing.fixed("trunk-width"), drawing.fixed("trunk-height")
	radius := drawing.fixed("crown-width") / half
	leaves := paint("crown", drawing.pick("crown-colour",
		drawing.options("crown-colour"), place{}))
	wood := paint("trunk", drawing.pick("trunk-colour",
		drawing.options("trunk-colour"), place{}))
	return []placedElement{
		put("trunk", Box{trunk, high, trunk}, wood,
			foot.Sub(maths.V(trunk/half, 0, trunk/half))),
		{Place: standingAt(foot.Add(maths.V(0, high, 0))), Element: solid{
			Name: "crown", Material: leaves, Form: prism{
				Bottom: outlineCircle{radius}, Top: outlineCircle{radius * crownTip},
				Height: drawing.fixed("crown-height")}}},
	}
}

// crownTip is how wide the top of a tree's cone is against its foot. A
// choice, by eye.
const crownTip = 0.15

// cars are the few cars the court holds, parked in a row along its
// south edge, one car length and a gap apart.
func (shared court) cars(drawing *drawer, thick float64) []placedElement {
	count := drawing.count("cars", place{})
	long := drawing.fixed("car-length")
	margin := drawing.fixed("car-margin")
	lefts, _ := spread(-shared.width/half+margin, shared.width/half-margin,
		repeated(long, count))
	var parts []placedElement
	for index, left := range lefts {
		at := maths.V(left, thick, shared.depth/half-margin)
		parts = append(parts, shared.car(drawing, index, at)...)
	}
	return parts
}

// repeated is a number given that many times.
func repeated(value float64, count int) []float64 {
	run := make([]float64, count)
	for index := range run {
		run[index] = value
	}
	return run
}

// car is one car standing at a point: a body, a cabin over its middle,
// and four wheels.
func (shared court) car(drawing *drawer, index int, at maths.Vec) []placedElement {
	long, wide := drawing.fixed("car-length"), drawing.fixed("car-width")
	body, wheel := drawing.fixed("car-body"), drawing.fixed("wheel-diameter")
	lift := drawing.fixed("car-lift")
	paintwork := paint("car", drawing.choose("car-colour", scopeDistrict,
		fmt.Sprint(index), drawing.options("car-colour"), place{}, nil))
	tyre := paint("tyre", drawing.pick("tyre-colour",
		drawing.options("tyre-colour"), place{}))
	cabin := drawing.fixed("cabin-length")
	parts := []placedElement{
		put("car body", Box{long, body, wide}, paintwork,
			at.Add(maths.V(0, lift, 0))),
		put("car cabin", Box{cabin, drawing.fixed("cabin-height"),
			wide - 2*drawing.fixed("cabin-inset")}, paintwork, at.Add(
			maths.V((long-cabin)/half, lift+body, drawing.fixed(
				"cabin-inset")))),
	}
	for _, along := range []float64{long * wheelFront, long * wheelBack} {
		for _, across := range []float64{0, wide - drawing.fixed(
			"car-tyre")} {
			parts = append(parts, put("wheel", Box{wheel, wheel,
				drawing.fixed("car-tyre")}, tyre, at.Add(maths.V(
				along-wheel/half, 0, across))))
		}
	}
	return parts
}

// wheelFront and wheelBack are how far along a car its wheels' middles
// are, as shares of its length. Choices, by eye.
const (
	wheelFront = 0.2
	wheelBack  = 0.8
)
