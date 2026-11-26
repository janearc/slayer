package cities

import (
	"github.com/janearc/slayer/scene/thing"
)

// glowing is a light's material made to glow as much as a gain says of what
// it did, and opaque: the light of a ceiling panel or a screen, which is a
// small thing, where a lit window of the quick level was a whole pane.
func glowing(look *thing.Material, gain float64,
	name string) *thing.Material {
	lit := *look
	lit.Name += " " + name
	lit.Glow = look.Glow.Scale(gain)
	lit.Through = 0
	return &lit
}

// place is where a window is in the tower for the culture's draws.
func (job faceJob) place(index, column int) place {
	return place{Bay: job.shaft, Storey: job.firstStorey + index,
		Opening: job.face*faceStride + column}
}

// room is the room behind a window at the demo level, between the glass
// and the core wall: an office, with a ceiling of lights and desks and
// screens as silhouettes under them, or a home, with its side walls, its
// boards and its couch, a lamp and a curtain.
//
// It is lit by the same draw
// as the window of the quick level: an office floor at a time, a home
// alone. The glass is clear, so all of it is seen from outside. A window
// is given by where it stands along the face, the floor of its storey, and
// the foot and height of the glass.
func (job faceJob) room(index, column int, start, end, floor, up,
	tall float64) []placedElement {
	if !(job.cavity > 0) {
		return nil
	}
	ceiling := floor + job.frame.slope/float64(job.storeys)
	look, on := job.windowLight(index, column)
	if job.use == useOffice {
		parts := job.office(index, column, start, end, floor, ceiling, look,
			on)
		return append(parts, job.blind(index, column, start, end, up,
			tall)...)
	}
	return job.home(index, column, start, end, floor, ceiling, look, on, up,
		tall)
}

// wash is the core wall of a lit room as it looks lit: a sheet that glows
// across the wall, over the width of its window and up to the ceiling, in
// the room's light. A lamp every few windows lights the things in the
// rooms near it; this lights every lit room's wall, which a lamp that far
// off does not.
func (job faceJob) wash(start, end, floor, ceiling float64,
	look *thing.Material) placedElement {
	drawing := job.drawing
	slab := drawing.fixed("floor-thick")
	return job.frame.onFace("lit wall", Box{end - start,
		ceiling - floor - slab, drawing.fixed("wall-wash-thick")},
		glowing(look, drawing.fixed("wall-wash-gain"), "wash"), start,
		floor+slab, -job.deep())
}

// wall is a thin wall between a place along the face and the next, from
// a floor to a ceiling, as deep as the rooms.
func (job faceJob) wall(name string, along, floor, ceiling float64) placedElement {
	drawing := job.drawing
	slab := drawing.fixed("floor-thick")
	return job.frame.onFace(name, Box{drawing.fixed("partition-thick"),
		ceiling - floor - slab, job.cavity}, job.looks.fine.wall, along,
		floor+slab, -job.deep())
}

// backWall is the wall at the back of an office, in front of the core: a
// wall of the rooms' own colour, which a lamp in the room lights as it
// lights the partitions.
func (job faceJob) backWall(start, end, floor, ceiling float64) placedElement {
	drawing := job.drawing
	slab := drawing.fixed("floor-thick")
	return job.frame.onFace("back wall", Box{end - start,
		ceiling - floor - slab, drawing.fixed("wall-wash-thick")},
		job.looks.fine.wall, start, floor+slab, -job.deep())
}

// office is an office behind one window: a partition now and then and a
// back wall, and if its floor is lit the ceiling lights, a lamp every so
// many windows, and a desk with its screen and chair, by chance.
//
// A lit office with a lamp is lit by it, back wall and all. One without
// has the wash in place of its back wall, which glows as a lit wall would
// look and lights nothing.
func (job faceJob) office(index, column int, start, end, floor, ceiling float64,
	look *thing.Material, on bool) []placedElement {
	var parts []placedElement
	if column%int(job.drawing.fixed("partition-every")) == 0 {
		parts = append(parts, job.wall("partition", start, floor, ceiling))
	}
	if !on || look == nil {
		return append(parts, job.backWall(start, end, floor, ceiling))
	}
	parts = append(parts, job.ceilingLights(start, end, ceiling, look)...)
	if column%int(job.drawing.fixed("office-lamp-every")) == 0 {
		parts = append(parts, job.backWall(start, end, floor, ceiling),
			job.lamp((start+end)/2, ceiling, look,
				job.drawing.fixed("office-heat"),
				job.drawing.fixed("office-lamp-strength")))
	} else {
		parts = append(parts, job.wash(start, end, floor, ceiling, look))
	}
	if job.drawing.chance("desk-share", job.place(index, column)) {
		parts = append(parts, job.desk(start, end, floor, look)...)
	}
	return parts
}

// ceilingLights are the square lights of an office ceiling over one
// window: a few across it, and rows of them from the glass to the core.
func (job faceJob) ceilingLights(start, end, ceiling float64,
	look *thing.Material) []placedElement {
	drawing := job.drawing
	size, thick := drawing.fixed("ceiling-light-size"),
		drawing.fixed("ceiling-light-thick")
	across, rows := int(drawing.fixed("ceiling-lights-across")),
		int(drawing.fixed("ceiling-lights-deep"))
	panel := glowing(look, drawing.fixed("ceiling-light-gain"), "panel")
	var parts []placedElement
	for column := range across {
		along := start + (float64(column)+0.5)*(end-start)/float64(across)
		for row := range rows {
			deep := -job.recess - (float64(row)+0.5)*job.cavity/float64(rows)
			parts = append(parts, job.frame.onFace("ceiling light", Box{size,
				thick, size}, panel, along-size/2, ceiling-thick,
				deep-size/2))
		}
	}
	return parts
}

// lamp is a light hung at a place along the face from a ceiling, in the
// middle of the depth of the rooms: a box that glows, which is a light of
// the heat and luminance given as well.
func (job faceJob) lamp(along, ceiling float64, look *thing.Material,
	heat, strength float64) placedElement {
	size := job.drawing.fixed("lamp-box")
	return placedElement{Element: solid{Name: "lamp", Form: Box{size, size, size},
		Material: glowing(look, 1, "lamp"),
		Lamp:     &Lamp{Heat: heat, Luminance: strength}},
		Place: job.frame.at(along-size/2, ceiling-2*size,
			-job.recess-job.cavity/2)}
}

// desk is a desk of an office behind a window, standing back from the
// glass, with a screen on it that glows and a chair behind it.
func (job faceJob) desk(start, end, floor float64,
	look *thing.Material) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	middle := (start + end) / 2
	wide := min(drawing.fixed("desk-width"), end-start-drawing.fixed(
		"partition-thick")*2)
	deep, high := drawing.fixed("desk-depth"), drawing.fixed("desk-height")
	back := -job.recess - drawing.fixed("desk-set-back")
	top := floor + drawing.fixed("floor-thick")
	screen, tall := drawing.fixed("screen-width"),
		drawing.fixed("screen-height")
	seat, rise := drawing.fixed("chair-size"), drawing.fixed("chair-height")
	return []placedElement{
		job.frame.onFace("desk", Box{wide, high, deep}, fine.desk,
			middle-wide/2, top, back-deep),
		job.frame.onFace("screen", Box{screen, tall,
			drawing.fixed("screen-thick")}, glowing(look,
			drawing.fixed("screen-gain"), "screen"), middle-screen/2,
			top+high, back-deep/2),
		job.frame.onFace("chair", Box{seat, rise, seat}, fine.desk,
			middle-seat/2, top, back-deep-seat-seat/2)}
}

// blind is the blind drawn down some of the way over a window, if the
// window has one, each as far down as its place draws, in a fabric of its
// own, hanging just behind the glass.
func (job faceJob) blind(index, column int, start, end, up,
	tall float64) []placedElement {
	drawing := job.drawing
	at := job.place(index, column)
	if !drawing.chance("blind-share", at) {
		return nil
	}
	drop := drawing.number("blind-drop", at) * tall
	cloth := paint("blind", drawing.pick("blind-colour",
		drawing.options("blind-colour"), at))
	return []placedElement{job.frame.onFace("blind", Box{end - start, drop,
		drawing.fixed("blind-thick")}, cloth, start, up+tall-drop,
		-job.recess-drawing.fixed("blind-set-back"))}
}

// home is a home behind one window: the walls on either side of it and the
// boards under it, a couch by chance, a lamp every so many windows if it
// is lit, and a curtain behind the glass by the culture's chance.
func (job faceJob) home(index, column int, start, end, floor, ceiling float64,
	look *thing.Material, on bool, up, tall float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	at := job.place(index, column)
	slab, thin := drawing.fixed("floor-thick"),
		drawing.fixed("partition-thick")
	parts := []placedElement{
		job.wall("side wall", start, floor, ceiling),
		job.wall("side wall", end-thin, floor, ceiling),
		job.frame.onFace("room floor", Box{end - start, thin, job.cavity},
			fine.floor, start, floor+slab, -job.deep())}
	if on && look != nil {
		parts = append(parts, job.wash(start, end, floor, ceiling, look))
	}
	if drawing.chance("couch-share", at) {
		parts = append(parts, job.couch(at, start, end, floor+slab))
	}
	if on && look != nil && column%int(drawing.fixed("home-lamp-every")) == 0 {
		parts = append(parts, job.lamp((start+end)/2, ceiling, look,
			drawing.heat(at), drawing.fixed("home-lamp-strength")))
	}
	return append(parts, job.curtain(index, column, start, end-start, up,
		tall, -job.recess-drawing.fixed("blind-set-back"))...)
}

// couch is a couch in a home, standing back from the glass in the middle of
// its window, in the cloth its place draws.
func (job faceJob) couch(at place, start, end, top float64) placedElement {
	drawing := job.drawing
	wide := min(drawing.fixed("couch-width"), end-start-
		2*drawing.fixed("partition-thick"))
	cloth := paint("couch", drawing.pick("couch-colour",
		drawing.options("couch-colour"), at))
	deep := drawing.fixed("couch-depth")
	return job.frame.onFace("couch", Box{wide, drawing.fixed("couch-height"),
		deep}, cloth, (start+end)/2-wide/2, top,
		-job.recess-drawing.fixed("couch-set-back")-deep)
}
