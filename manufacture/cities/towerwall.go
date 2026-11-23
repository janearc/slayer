package cities

// wallSpan is a span of curtain wall at the demo level, drawn as its
// makers build it: mullions with pressure caps, and in every bay of every
// storey a spandrel glass over a folded backpan and the edge of the slab,
// a sill, the vision glass, a transom at its head, and now and then an
// operable vent. Both the plain curtain and the mirror wall are drawn so.
func (job faceJob) wallSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	proud := drawing.fixed("cw-mullion-proud")
	parts := job.bars("mullion", low, pitch, columns, job.mullion, proud,
		job.looks.grid)
	parts = append(parts, job.bars("mullion cap", low, pitch, columns,
		drawing.fixed("cw-cap-width"), proud+drawing.fixed("cw-cap-proud"),
		fine.metal)...)
	return append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		floor := job.floorSlab(left, right, foot)
		return append(floor, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				return job.wallBay(index, column, start+job.mullion/2,
					end-job.mullion/2, foot, storey)
			})...)
	})...)
}

// wallBay is one bay of curtain wall in one storey, from one place along
// the face to another, standing on a floor foot up the face.
func (job faceJob) wallBay(index, column int, start, end, foot,
	storey float64) []placedElement {
	drawing := job.drawing
	band := drawing.fixed("cw-spandrel-height")
	parts := job.spandrelPanel(start, end, foot, band)
	parts = append(parts, job.wallEdges(start, end, foot, band, storey)...)
	parts = append(parts, job.vision(index, column, start, end, foot,
		foot+band, storey-band)...)
	every := int(drawing.fixed("cw-vent-every"))
	if column%every == every-1 {
		parts = append(parts, job.vent(end, foot, band, storey)...)
	}
	return parts
}

// spandrelPanel is the opaque panel at the foot of a bay: a glass that
// lets the light through, the backpan behind it across a gap with a fold
// stiffening its middle, and the edge of the floor slab seen in front of
// the pan at the foot.
func (job faceJob) spandrelPanel(start, end, foot, band float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	gap, thick := drawing.fixed("cw-backpan-gap"),
		drawing.fixed("cw-backpan-thick")
	fold, stand := drawing.fixed("cw-backpan-fold"),
		drawing.fixed("cw-fold-proud")
	tall := band - drawing.fixed("cw-transom-height")
	back := -job.recess - gap - thick
	frame := job.frame
	return []placedElement{
		frame.onFace("spandrel", Box{end - start, tall, job.pane},
			fine.spandrel, start, foot, -job.recess),
		frame.onFace("backpan", Box{end - start, tall, thick}, fine.backpan,
			start, foot, back),
		frame.onFace("backpan fold", Box{end - start, fold, thick + stand},
			fine.metal, start, foot+tall/2, back),
		frame.onFace("slab edge", Box{end - start,
			drawing.fixed("cw-slab-edge-height"), gap}, fine.slab, start,
			foot, -job.recess-gap)}
}

// wallEdges are the bars that close a bay: the transom over the spandrel
// with the sill on it, at the foot of the vision glass, and the transom at
// the head of the glass, under the next storey's spandrel.
func (job faceJob) wallEdges(start, end, foot, band, storey float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	height, proud := drawing.fixed("cw-transom-height"),
		drawing.fixed("cw-transom-proud")
	frame := job.frame
	return []placedElement{
		frame.onFace("transom", Box{end - start, height, job.recess + proud},
			job.looks.grid, start, foot+band-height, -job.recess),
		frame.onFace("sill", Box{end - start, drawing.fixed("cw-sill-thick"),
			job.recess + drawing.fixed("cw-sill-proud")}, fine.metal, start,
			foot+band, -job.recess),
		frame.onFace("transom", Box{end - start, height, job.recess + proud},
			job.looks.grid, start, foot+storey-height, -job.recess)}
}

// vent is an operable vent at the head of a bay's vision glass, against
// its right-hand mullion: a frame of four bars standing out of the glass.
func (job faceJob) vent(end, foot, band, storey float64) []placedElement {
	drawing, metal := job.drawing, job.looks.fine.metal
	wide, tall := drawing.fixed("cw-vent-width"),
		drawing.fixed("cw-vent-height")
	bar, proud := drawing.fixed("cw-vent-bar"), drawing.fixed("cw-vent-proud")
	left, up := end-wide, foot+storey-tall-drawing.fixed("cw-transom-height")
	onGlass := func(name string, width, height, along, rise float64) placedElement {
		return job.frame.onFace(name, Box{width, height, proud}, metal,
			along, rise, -job.recess+job.pane)
	}
	return []placedElement{
		onGlass("vent", wide, bar, left, up),
		onGlass("vent", wide, bar, left, up+tall-bar),
		onGlass("vent", bar, tall-2*bar, left, up+bar),
		onGlass("vent", bar, tall-2*bar, end-bar, up+bar)}
}
