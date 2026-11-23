package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The kinds of clutter on a roof, as the culture names them.
const (
	clutterTank    = "tank"
	clutterAntenna = "antenna"
	clutterShack   = "shack"
)

// clutter is one piece standing on a roof: what it is, the footprint
// its volume takes, and its height. Its parts are drawn about a point on
// the roof.
type clutter struct {
	kind                 string
	width, depth, height float64
	radius               float64
	material             *thing.Material
	barLength, mastWidth float64
}

// roofs is everything that stands on the roof of a building: a berth if
// the set brings parts and the gate says so, a yard if the gate says so,
// then the clutter, as many pieces as the accretion reaches, each piece
// strapped to the next with the scavenged band. Parts are added things,
// so a planned block, at nought, keeps no berth.
func (growing *accretion) roofs(index int) {
	drawing := growing.drawing
	at := place{Bay: index}
	if growing.ask.Berths && drawing.gate("berth-share", at, growing.grown,
		nil) {
		growing.berth(index)
	}
	if drawing.gate("yard-share", at, growing.grown, nil) {
		growing.yard(index)
	}
	var centres []maths.Vec
	for piece := range drawing.reach("clutter-count", at, growing.grown) {
		if centre, stood := growing.stand(index, piece); stood {
			centres = append(centres, centre)
		}
	}
	growing.strap(index, centres)
}

// roofOf is the ground of a roof's top: the lot less the parapet round
// it, and the height of the slab the clutter stands on.
func (growing *accretion) roofOf(index int) (rect, float64) {
	drawing := growing.drawing
	plot := growing.out.Plots[index]
	wall := drawing.fixed("parapet-wall")
	lot := plot.Lot
	return rect{lot.X0 + wall, lot.Z0 + wall, lot.X1 - wall, lot.Z1 - wall},
		plot.Height + drawing.fixed("slab-thickness")
}

// berth keeps a place on a roof for a part from the set's mix, as long
// as the culture's berth along the roof's longer side, if the roof is
// big enough for it. Nothing else stands there.
func (growing *accretion) berth(index int) {
	drawing := growing.drawing
	roof, top := growing.roofOf(index)
	length, width := drawing.fixed("berth-length"),
		drawing.fixed("berth-width")
	sizeX, sizeZ := length, width
	if roof.width() < roof.depth() {
		sizeX, sizeZ = width, length
	}
	if sizeX > roof.width() || sizeZ > roof.depth() {
		return
	}
	low := [3]float64{(roof.X0+roof.X1)/half - sizeX/half, top,
		(roof.Z0+roof.Z1)/half - sizeZ/half}
	berth := Volume{Kind: "berth", Plot: index, Low: low, High: [3]float64{
		low[0] + sizeX, top + drawing.fixed("berth-height"), low[2] + sizeZ}}
	if growing.clear(berth) {
		growing.solids = append(growing.solids, berth)
		growing.out.Berths = append(growing.out.Berths, berth)
	}
}

// stand tries places on a roof for one piece of clutter, as many as the
// culture allows, and gives the centre of the one that is clear.
func (growing *accretion) stand(index, piece int) (maths.Vec, bool) {
	drawing := growing.drawing
	roof, top := growing.roofOf(index)
	one := growing.piece(place{Bay: index, Storey: piece, Opening: piece})
	if one.width > roof.width() || one.depth > roof.depth() {
		return maths.Vec{}, false
	}
	for try := range int(drawing.fixed("clutter-tries")) {
		at := place{Bay: index, Storey: try, Opening: piece}
		centre := maths.V(
			roof.X0+one.width/half+drawing.number("clutter-place", at)*
				(roof.width()-one.width), top,
			roof.Z0+one.depth/half+drawing.number("clutter-place",
				place{Row: 1, Bay: index, Storey: try, Opening: piece})*
				(roof.depth()-one.depth))
		if growing.settle(one.volume(index, centre)) {
			growing.parts = append(growing.parts, growing.lay(one, centre)...)
			return centre, true
		}
	}
	return maths.Vec{}, false
}

// piece is a piece of clutter drawn, as the culture says its kind, its
// size and its look.
func (growing *accretion) piece(at place) clutter {
	drawing := growing.drawing
	kind := drawing.pick("clutter-kind", drawing.options("clutter-kind"),
		at).Name
	switch kind {
	case clutterTank:
		radius := drawing.number("tank-radius", at)
		return clutter{kind: kind, width: 2 * radius, depth: 2 * radius,
			height: drawing.number("tank-height", at), radius: radius,
			material: paint(kind, drawing.pick("tank-colour",
				drawing.options("tank-colour"), at))}
	case clutterAntenna:
		bar, mast := drawing.number("mast-bar", at),
			drawing.fixed("mast-width")
		return clutter{kind: kind, width: bar, depth: mast,
			height: drawing.number("mast-height", at), barLength: bar,
			mastWidth: mast, material: paint("mast", drawing.pick(
				"cage-colour", drawing.options("cage-colour"), at))}
	}
	side := drawing.number("shack-width", at)
	return clutter{kind: clutterShack, width: side, depth: side,
		height: drawing.number("shack-height", at),
		material: paint("shack", drawing.pick("box-colour",
			drawing.options("box-colour"), at))}
}

// volume is the air a piece takes, standing on a roof at a centre.
func (one clutter) volume(index int, centre maths.Vec) Volume {
	return Volume{Kind: "clutter", Plot: index,
		Low: [3]float64{centre.X - one.width/half, centre.Y,
			centre.Z - one.depth/half},
		High: [3]float64{centre.X + one.width/half, centre.Y + one.height,
			centre.Z + one.depth/half}}
}

// lay draws a piece, its bands where it has them, standing on a roof.
func (growing *accretion) lay(one clutter, centre maths.Vec) []placedElement {
	switch one.kind {
	case clutterTank:
		return growing.drum(one, centre)
	case clutterAntenna:
		return growing.mast(one, centre)
	}
	return growing.hut(one, centre)
}

// drum is a water tank: a cylinder with two bands round it, drawn a
// band's thickness larger so that they stand out of it.
func (growing *accretion) drum(one clutter, centre maths.Vec) []placedElement {
	thick, wide := growing.straps.Thickness, growing.straps.Width
	parts := []placedElement{put("tank", prism{outlineCircle{one.radius},
		outlineCircle{one.radius}, one.height}, one.material, centre)}
	for _, share := range growing.bandShares() {
		parts = append(parts, put("band", prism{outlineCircle{one.radius + thick},
			outlineCircle{one.radius + thick}, wide}, growing.lashing,
			centre.Add(maths.V(0, one.height*share, 0))))
	}
	return parts
}

// bandShares are the shares of a piece's height its two bands stand at:
// the culture's, and the share the other side of the middle.
func (growing *accretion) bandShares() [2]float64 {
	high := growing.drawing.fixed("band-height")
	return [2]float64{1 - high, high}
}

// mast is an antenna: a thin mast with a cross bar near its top.
func (growing *accretion) mast(one clutter, centre maths.Vec) []placedElement {
	thin := one.mastWidth
	bar := growing.drawing.fixed("bar-height")
	return []placedElement{
		put("mast", Box{thin, one.height, thin}, one.material,
			centre.Add(maths.V(-thin/half, 0, -thin/half))),
		put("mast", Box{one.barLength, thin, thin}, one.material,
			centre.Add(maths.V(-one.barLength/half, one.height*bar,
				-thin/half)))}
}

// hut is a shack: a box with two bands round it.
func (growing *accretion) hut(one clutter, centre maths.Vec) []placedElement {
	thick, wide := growing.straps.Thickness, growing.straps.Width
	corner := centre.Add(maths.V(-one.width/half, 0, -one.depth/half))
	parts := []placedElement{put("shack", Box{one.width, one.height, one.depth},
		one.material, corner)}
	for _, share := range growing.bandShares() {
		parts = append(parts, put("band", Box{one.width + 2*thick, wide,
			one.depth + 2*thick}, growing.lashing,
			corner.Add(maths.V(-thick, one.height*share, -thick))))
	}
	return parts
}

// strap lays a band flat on the roof from each piece of clutter to the
// next, the scavenged lashing that holds a roof's things together.
func (growing *accretion) strap(index int, centres []maths.Vec) {
	thick, wide := growing.straps.Thickness, growing.straps.Width
	for at := 1; at < len(centres); at++ {
		from, to := centres[at-1], centres[at]
		run := math.Hypot(to.X-from.X, to.Z-from.Z)
		turn := maths.Rotation(maths.YAxis, math.Atan2(from.Z-to.Z, to.X-from.X))
		growing.parts = append(growing.parts, placedElement{Element: solid{
			Name: "band", Form: Box{run, thick, wide}, Material: growing.lashing},
			Place: maths.NewSubspace(from, turn, 0)})
	}
}
