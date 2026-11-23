package cities

import (
	"github.com/janearc/slayer/scene/thing"
)

// fineLooks are the materials the demo level of a tower draws its small
// parts in, drawn once for a tower. At the hull and quick levels they are
// not made: the zero value is never read there.
type fineLooks struct {
	// metal is the aluminium of caps, sills and vents, backpan the tray
	// behind a spandrel, slab the concrete edge of a floor, head the
	// precast stone of lintels, frame the metal of a window's frame.
	metal, backpan, slab, head, frame *thing.Material
	// spandrel is the glass of a spandrel panel, which lets the backpan
	// behind it show, and glazing the clear glass of a window.
	spandrel, glazing *thing.Material
	// wall and floor are the paint of the core wall and ceiling and the
	// boards of a room's floor, desk the laminate of desks and chairs,
	// and mirror the mirror glass of a mirror wall, which lets a little
	// of the room through.
	wall, floor, desk, mirror *thing.Material
}

// fined draws the materials of the demo level from the culture, given the
// looks the skin drew: a mirror wall's glass is its panel, let through.
func (tower *Tower) fined(made towerLooks) fineLooks {
	drawing := tower.drawing
	pick := func(name string) *thing.Material {
		return paint(name, drawing.pick(name, drawing.options(name),
			place{}))
	}
	glass := *pick("mirror-spandrel-colour")
	glass.Name += " spandrel glass"
	glass.Through = drawing.fixed("cw-spandrel-through")
	rooms := drawing.rooms()
	fine := fineLooks{metal: pick("fine-metal-colour"),
		backpan: pick("backpan-colour"), slab: pick("slab-edge-colour"),
		head: pick("precast-colour"), frame: pick("window-frame-colour"),
		spandrel: &glass, glazing: drawing.glazing(), wall: rooms.Wall,
		floor: rooms.Floor, desk: pick("desk-colour")}
	if made.skin.panel != nil {
		see := *made.skin.panel
		see.Name += " seen through"
		see.Through = drawing.fixed("mirror-through")
		fine.mirror = &see
	}
	return fine
}

// fineSpan is a span drawn part by part at the demo level, and whether
// the skin has a demo drawing. A skin that has none is drawn as at the
// quick level.
func (job faceJob) fineSpan(low, high, pitch float64,
	columns int) ([]placedElement, bool) {
	switch job.skin {
	case skinCurtain, skinMirror:
		return job.wallSpan(low, high, pitch, columns), true
	case skinGranite:
		if job.style != styleRibbon {
			return job.punchedFine(low, high, pitch, columns), true
		}
	case skinConcrete:
		return job.concreteFine(low, high, pitch, columns), true
	case skinGothic:
		return job.gothicFine(low, high, pitch, columns), true
	case skinCantilever:
		return job.cantileverFine(low, high, pitch, columns), true
	}
	return nil, false
}

// vision is the glass of a window at the demo level and the room behind
// it: clear glass, or on a mirror wall the panel, leaned as the quick
// level leans it, over a room lit or dark as its floor or household is.
func (job faceJob) vision(index, column int, start, end, floor, up,
	tall float64) []placedElement {
	glass := job.glass(index, column, start, end, up, tall)
	return append(glass, job.room(index, column, start, end, floor, up,
		tall)...)
}

// glass is the glass of one window at the demo level.
func (job faceJob) glass(index, column int, start, end, up,
	tall float64) []placedElement {
	if job.skin == skinMirror {
		return []placedElement{job.panel(index, column, start, end, up, tall)}
	}
	return []placedElement{job.frame.onFace("window", Box{end - start, tall,
		job.pane}, job.looks.fine.glazing, start, up, -job.recess)}
}

// deep is how far behind the face the core wall is: the recess of the
// glass and the rooms, which are nothing but the recess below the demo
// level.
func (job faceJob) deep() float64 { return job.recess + job.cavity }

// floorSlab is the slab of a storey across a run of face at the demo
// level, the floor of the rooms behind it and the ceiling of those below.
func (job faceJob) floorSlab(left, right, foot float64) []placedElement {
	if !(job.cavity > 0) {
		return nil
	}
	return []placedElement{job.frame.onFace("floor slab", Box{right - left,
		job.drawing.fixed("floor-thick"), job.cavity}, job.looks.fine.slab,
		left, foot, -job.deep())}
}

// sash is the frame of a window of a wall of stone or concrete: four bars
// round it, one up its middle and a transom across it, standing in the
// window's recess from the glass out by the frame's depth.
func (job faceJob) sash(at, width, up, tall float64) []placedElement {
	drawing, look := job.drawing, job.looks.fine.frame
	bar, deep := drawing.fixed("ps-frame-bar"), drawing.fixed("ps-frame-deep")
	across := up + tall*drawing.fixed("ps-transom-share")
	inner := tall - 2*bar
	onFace := func(name string, wide, high, along, rise float64) placedElement {
		return job.frame.onFace(name, Box{wide, high, deep}, look, along,
			rise, -job.recess)
	}
	return []placedElement{
		onFace("frame", width, bar, at, up),
		onFace("frame", width, bar, at, up+tall-bar),
		onFace("frame", bar, inner, at, up+bar),
		onFace("frame", bar, inner, at+width-bar, up+bar),
		onFace("frame", bar, inner, at+(width-bar)/2, up+bar),
		onFace("frame", width-2*bar, bar, at+bar, across)}
}
