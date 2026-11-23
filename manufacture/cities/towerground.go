package cities

import (
	"math"

	"github.com/janearc/slayer/scene/thing"
)

// The kinds of ground floor a face of the first segment of a shaft has,
// by the face's number modulo how many kinds there are: the face that is
// the way in, two that are shops, and the face with the loading bay.
const (
	faceEntrance = iota
	faceShops
	faceLoading
	faceMoreShops
	groundKinds
)

// ground is a face's ground floor at the demo level, which is a storey of
// its own: a plinth under a wall of glass, behind which a lobby or shops
// are lit; on the face that is the way in a canopy and a revolving door
// between two swing doors; on another a loading bay with its roller door;
// signs and awnings over the shops, and bollards along the kerb. The
// skin's storeys start over it, and its piers run down beside it.
func (job faceJob) ground(low, high, pitch float64, columns int) []placedElement {
	if !job.demo || job.firstStorey != 0 {
		return nil
	}
	parts := job.plinth(low, high)
	parts = append(parts, job.floorOf(low, high)...)
	for column := range columns {
		start := low + float64(column)*pitch
		parts = append(parts, job.groundBay(column, columns, start,
			start+pitch)...)
	}
	return append(parts, job.bollards(low, high)...)
}

// storeyOf is the height of a storey of the face, and so of the ground
// floor.
func (job faceJob) storeyOf() float64 {
	return job.frame.slope / float64(job.storeys)
}

// plinth is the course of stone, in the lobby's stone, that the glass of
// the ground floor stands on, across a span.
func (job faceJob) plinth(low, high float64) []placedElement {
	drawing := job.drawing
	return []placedElement{job.frame.onFace("plinth", Box{high - low,
		drawing.fixed("ground-plinth-height"), job.deep() +
			drawing.fixed("ground-plinth-proud")}, job.looks.lobby, low,
		0, -job.deep())}
}

// lobbyOf is the stone of a tower's ground floor plinth: a mirror tower's
// lobby is polished stone of its own and a granite tower's is its granite
// polished, so a writer can tell them; any other's is its podium.
func (tower *Tower) lobbyOf(made towerLooks) *thing.Material {
	var stone *thing.Material
	switch tower.Skin {
	case skinMirror:
		stone = tower.stone(paintOf(tower.drawing, "lobby-stone-colour",
			place{}), "granite")
	case skinGranite:
		stone = made.skin.stone
	default:
		return made.podium
	}
	polished := laid(stone, "lobby polished", stone.Texture)
	polished.Gloss = max(stone.Gloss, tower.drawing.fixed("lobby-gloss"))
	return polished
}

// shutter is the roller shutter pulled down over a dark shop's glass, of
// slats, between two places along the face and from a foot up as tall
// as given.
func (job faceJob) shutter(at place, left, wide, foot,
	tall float64) []placedElement {
	look := paintOf(job.drawing, "shutter-colour", at)
	parts := []placedElement{job.frame.onFace("shutter", Box{wide, tall, job.pane},
		look, left, foot, 0)}
	return append(parts, job.slats(left, wide, foot, tall,
		int(job.drawing.fixed("shutter-slats")))...)
}

// slats are the grooves across a roller door or shutter between two
// places along the face, from a foot up as tall as given, so many of
// them, standing on its face.
func (job faceJob) slats(left, wide, foot, tall float64,
	count int) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	groove := drawing.fixed("loading-slat")
	var parts []placedElement
	for slat := 1; slat < count; slat++ {
		parts = append(parts, job.frame.onFace("roller slat", Box{wide,
			groove, groove}, fine.metal, left,
			foot+float64(slat)*tall/float64(count), job.pane))
	}
	return parts
}

// floorOf is the slab of the ground floor across a span, on the ground.
func (job faceJob) floorOf(low, high float64) []placedElement {
	return job.floorSlab(low, high, 0)
}

// groundBay is one bay of the ground floor, between two places along the
// face: the way in, the loading bay or a shop, as its face and its place
// along it say.
func (job faceJob) groundBay(column, columns int, start, end float64) []placedElement {
	middle := columns / 2
	switch job.face % groundKinds {
	case faceEntrance:
		if column == middle {
			return job.entrance(start, end)
		}
	case faceLoading:
		if column == middle || column == middle-1 {
			return job.loading(column, middle, start, end)
		}
	}
	return job.shopfront(column, start, end)
}

// shopfront is a bay of glass over the plinth, the shop or lobby behind it
// lit or dark by chance, and a sign and an awning over it on the faces
// that are shops.
func (job faceJob) shopfront(column int, start, end float64) []placedElement {
	drawing := job.drawing
	at := job.place(0, column)
	up := drawing.fixed("ground-plinth-height")
	tall := drawing.fixed("ground-glass-height")
	wide := end - start - job.mullion
	parts := []placedElement{job.frame.onFace("shop glass", Box{wide, tall,
		job.pane}, job.looks.fine.glazing, start+job.mullion/2, up,
		-job.recess)}
	parts = append(parts, job.shopInside(at, start, end)...)
	if kind := job.face % groundKinds; kind == faceShops ||
		kind == faceMoreShops {
		parts = append(parts, job.shopSign(at, start, end, up+tall)...)
		if !drawing.chance("ground-lit", at) {
			parts = append(parts, job.shutter(at, start+job.mullion/2, wide,
				up, tall)...)
		}
	}
	return parts
}

// shopInside is the room behind a bay of glass on the ground: shelves along
// its back, and if its light is on a glowing wall, a strip of light along
// the ceiling and a warm light.
func (job faceJob) shopInside(at place, start, end float64) []placedElement {
	drawing := job.drawing
	storey := job.storeyOf()
	wide := end - start - job.mullion
	shelf := job.frame.onFace("shelf", Box{wide * drawing.fixed(
		"ground-shelf-share"), drawing.fixed("ground-shelf-height"),
		drawing.fixed("ground-shelf-deep")}, job.looks.fine.desk,
		start+job.mullion/2+wide*(1-drawing.fixed("ground-shelf-share"))/2,
		drawing.fixed("floor-thick"), -job.deep())
	if !drawing.chance("ground-lit", at) {
		return []placedElement{shelf}
	}
	light := glowing(drawing.lamp(drawing.fixed("ground-heat"), 1),
		drawing.fixed("ground-wash-gain"), "ground")
	strip := drawing.fixed("ground-strip")
	return []placedElement{shelf,
		job.frame.onFace("lit wall", Box{wide, storey -
			drawing.fixed("floor-thick"), drawing.fixed("wall-wash-thick")},
			light, start+job.mullion/2, drawing.fixed("floor-thick"),
			-job.deep()),
		job.frame.onFace("ceiling strip", Box{wide, drawing.fixed(
			"ceiling-light-thick"), strip}, light, start+job.mullion/2,
			storey-drawing.fixed("ceiling-light-thick"), -job.recess-
				job.cavity/half)}
}

// shopSign is the sign over a shop's glass, in a colour and glowing, and an
// awning under it that reaches out over the pavement.
func (job faceJob) shopSign(at place, start, end, up float64) []placedElement {
	drawing := job.drawing
	middle := (start + end) / 2
	wide := drawing.fixed("sign-width")
	paint := paint("sign", drawing.pick("sign-colour",
		drawing.options("sign-colour"), at))
	sign := *paint
	sign.Name += " lit"
	sign.Glow = paint.Colour.Scale(drawing.fixed("ground-sign-glow"))
	cloth := paintOf(drawing, "awning-colour", at)
	reach := drawing.fixed("awning-depth")
	return []placedElement{
		job.frame.onFace("shop sign", Box{wide, drawing.fixed("sign-height"),
			drawing.fixed("sign-thick")}, &sign, middle-wide/2,
			up+drawing.fixed("sign-rise"), 0),
		job.frame.onFace("awning", Box{end - start - job.mullion,
			drawing.fixed("awning-thick"), reach}, cloth,
			start+job.mullion/2, up-drawing.fixed("awning-thick"), 0)}
}

// paintOf is the material of a role named for an option drawn at a place.
func paintOf(drawing *drawer, name string, at place) *thing.Material {
	return paint(name, drawing.pick(name, drawing.options(name), at))
}

// entrance is the way in: a revolving door in a drum of glass, a swing door
// on each side of it, and the canopy over all three, hung on rods.
func (job faceJob) entrance(start, end float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	middle := (start + end) / 2
	radius := drawing.fixed("door-drum-radius")
	tall := drawing.fixed("door-drum-height")
	sides := int(drawing.fixed("door-drum-sides"))
	up := drawing.fixed("ground-plinth-height")
	round := circle(radius, sides)
	drum := prism{Bottom: polygon{round}, Top: polygon{round}, Height: tall}
	leaf := drawing.fixed("door-leaf-thick")
	parts := []placedElement{
		{Element: solid{Name: "revolving drum", Form: drum,
			Material: fine.glazing}, Place: job.frame.at(middle, up,
			radius/2)},
		job.frame.onFace("revolving leaf", Box{2 * radius, tall, leaf},
			fine.metal, middle-radius, up, radius/2),
		job.frame.onFace("revolving leaf", Box{leaf, tall, 2 * radius},
			fine.metal, middle-leaf/2, up, -radius/2)}
	side := drawing.fixed("door-swing-width")
	for _, sign := range []float64{-1, 1} {
		at := middle + sign*(radius+side/2) - side/2
		parts = append(parts, job.frame.onFace("swing door", Box{side, tall,
			job.pane}, fine.glazing, at, up, -job.recess),
			job.frame.onFace("door frame", Box{leaf, tall, drawing.fixed(
				"ps-frame-deep")}, fine.frame, at, up, -job.recess))
	}
	return append(parts, job.canopy(middle)...)
}

// canopy is the canopy over the way in, a slab standing out of the face
// over the doors, with a strip of light along its edge and a rod at each
// outer corner up to the face.
func (job faceJob) canopy(middle float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	wide, deep := drawing.fixed("canopy-width"), drawing.fixed("canopy-depth")
	thick, high := drawing.fixed("canopy-thick"), drawing.fixed("canopy-height")
	rod, strip := drawing.fixed("canopy-rod"), drawing.fixed("canopy-strip")
	light := glowing(drawing.lamp(drawing.fixed("ground-heat"), 1),
		drawing.fixed("ground-wash-gain"), "canopy")
	parts := []placedElement{
		job.frame.onFace("canopy", Box{wide, thick, deep}, fine.metal,
			middle-wide/2, high, 0),
		job.frame.onFace("canopy light", Box{wide, strip, strip}, light,
			middle-wide/2, high-strip, deep-strip)}
	for _, sign := range []float64{-1, 1} {
		parts = append(parts, job.frame.onFace("canopy rod", Box{rod, high,
			rod}, fine.metal, middle+sign*(wide/2-rod)-rod/2, 0, deep-rod))
	}
	return parts
}

// loading is a bay of the loading door, over two bays: a roller door of
// slats in a frame, and a lamp over it. The first of its two bays draws
// the door, the second only its frame's other half.
func (job faceJob) loading(column, middle int, start, end float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	if column != middle {
		return job.shopfront(column, start, end)
	}
	tall := drawing.fixed("loading-height")
	frame := drawing.fixed("loading-frame")
	wide := end - start + (end - start)
	left := start - (end - start)
	parts := []placedElement{
		job.frame.onFace("loading frame", Box{wide, tall + frame,
			job.recess + drawing.fixed("ps-frame-deep")}, fine.frame,
			left, 0, -job.recess),
		job.frame.onFace("roller door", Box{wide - 2*frame, tall,
			job.pane}, job.looks.plant, left+frame, 0, 0)}
	return append(parts, job.slats(left+frame, wide-2*frame, 0, tall,
		int(drawing.fixed("loading-slats")))...)
}

// bollards are the bollards along the kerb in front of a span: a post of
// steel every so often, standing out from the face.
func (job faceJob) bollards(low, high float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	radius, tall := drawing.fixed("bollard-radius"),
		drawing.fixed("bollard-height")
	round := circle(radius, int(drawing.fixed("bollard-sides")))
	post := prism{Bottom: polygon{round}, Top: polygon{round}, Height: tall}
	pitch, out := drawing.fixed("bollard-pitch"), drawing.fixed("bollard-out")
	var parts []placedElement
	for along := low + pitch/2; along < high; along += pitch {
		parts = append(parts, placedElement{Element: solid{Name: "bollard",
			Form: post, Material: fine.metal},
			Place: job.frame.at(along, 0, out)})
	}
	return parts
}

// circle is the corners of a circle of a radius, cut into a number of flat
// sides, about its middle.
func circle(radius float64, sides int) []outlineCorner {
	corners := make([]outlineCorner, sides)
	for index := range corners {
		angle := 2 * math.Pi * float64(index) / float64(sides)
		corners[index] = outlineCorner{radius * math.Cos(angle),
			radius * math.Sin(angle)}
	}
	return corners
}
