package cities

// concreteFine is a span of poured concrete at the demo level, cut into
// what the formwork left: each pier a lift to a storey with a cold joint
// at its foot, each slab edge a panel to a bay with a gap between panels, a
// drip lip under it and a tie hole at each end, and between them a window
// in a frame, set deep. The joints and holes are in the colour of shadow.
func (job faceJob) concreteFine(low, high, pitch float64,
	columns int) []placedElement {
	return job.eachStorey(low, high, func(index int, foot, storey, left,
		right float64) []placedElement {
		parts := job.pierLifts(index, low, pitch, columns, foot, storey)
		parts = append(parts, job.floorSlab(left, right, foot)...)
		return append(parts, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				return job.concreteBay(index, column, start, end, foot,
					storey)
			})...)
	})
}

// pierLifts are the lifts of the piers at the ends of each bay of a
// storey, each with the cold joint at its foot. A pier the face has
// narrowed past by the top of the storey is left out.
func (job faceJob) pierLifts(index int, low, pitch float64, columns int,
	foot, storey float64) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	width := drawing.fixed("concrete-pier-width")
	proud := job.deep() + drawing.fixed("concrete-pier-proud")
	joint := drawing.fixed("cc-joint-height")
	half := job.frame.half((float64(index) + 1) / float64(job.storeys))
	var parts []placedElement
	for column := 0; column <= columns; column++ {
		along := low + float64(column)*pitch
		if along-width/2 < -half || along+width/2 > half {
			continue
		}
		parts = append(parts,
			job.frame.onFace("pier", Box{width, storey, proud}, skin.stone,
				along-width/2, foot, -job.deep()),
			job.frame.onFace("cold joint", Box{width, joint, proud +
				drawing.fixed("cc-tie-proud")}, skin.board, along-width/2,
				foot, -job.deep()))
	}
	return parts
}

// concreteBay is one bay of one storey between two piers: the panel of the
// slab edge with its drip lip and tie holes, and the window under it.
func (job faceJob) concreteBay(index, column int, start, end, foot,
	storey float64) []placedElement {
	drawing := job.drawing
	slab, width := drawing.fixed("concrete-slab-thick"),
		drawing.fixed("concrete-pier-width")
	gap := drawing.fixed("cc-panel-gap")
	left, right := start+width/2+gap/2, end-width/2-gap/2
	parts := job.slabPanel(left, right, foot)
	glass, tall := right-left-gap, storey-slab
	at := left + gap/2
	parts = append(parts, job.sash(at, glass, foot+slab, tall)...)
	parts = append(parts, job.glass(index, column, at, at+glass, foot+slab,
		tall)...)
	return append(parts, job.room(index, column, at, at+glass, foot,
		foot+slab, tall)...)
}

// slabPanel is one panel of a slab edge between two places along the face
// at a floor: the panel, the drip lip under its front edge and a tie hole
// near each end.
func (job faceJob) slabPanel(left, right, foot float64) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	thick := drawing.fixed("concrete-slab-thick")
	proud := job.deep() + drawing.fixed("concrete-slab-proud")
	lip, drip := drawing.fixed("cc-drip-height"),
		drawing.fixed("cc-drip-proud")
	size, stand := drawing.fixed("cc-tie-size"), drawing.fixed("cc-tie-proud")
	inset := drawing.fixed("cc-tie-inset")
	parts := []placedElement{
		job.frame.onFace("slab edge", Box{right - left, thick, proud},
			skin.stone, left, foot, -job.deep()),
		job.frame.onFace("drip", Box{right - left, lip, proud + drip},
			skin.board, left, foot, -job.deep())}
	for _, along := range []float64{left + inset, right - inset - size} {
		parts = append(parts, job.frame.onFace("tie hole", Box{size, size,
			stand}, skin.board, along, foot+(thick-size)/2,
			-job.deep()+proud))
	}
	return parts
}
