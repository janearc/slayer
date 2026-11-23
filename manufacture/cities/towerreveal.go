package cities

// punchedFine is a span of stone with windows punched in it, at the demo
// level: piers between the windows, and in each bay of each storey an
// apron under the sill, a sill of precast stone, a lintel over the head, a
// spandrel of wall above it, and the window set deep, in a frame with a
// transom and a mullion. The piers stand out further than the wall between
// them, so the window is reached through a reveal as deep as a pier.
func (job faceJob) punchedFine(low, high, pitch float64,
	columns int) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	wide := drawing.fixed("granite-window-width")
	parts := job.bars("pier", low, pitch, columns, pitch-wide,
		drawing.fixed("granite-pier-proud"), stone)
	return append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		floor := job.floorSlab(left, right, foot)
		return append(floor, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				return job.punchedBay(index, column, (start+end)/2-wide/2,
					foot)
			})...)
	})...)
}

// punchedBay is one punched window of one storey, with the stone round
// it, its left edge at a place along the face and the floor foot up it.
func (job faceJob) punchedBay(index, column int, at, foot float64) []placedElement {
	drawing, stone, fine := job.drawing, job.looks.skin.stone,
		job.looks.fine
	wide, tall := drawing.fixed("granite-window-width"),
		drawing.fixed("granite-window-height")
	sill, lift := drawing.fixed("granite-sill"),
		drawing.fixed("granite-sill-height")
	wall := job.deep() + drawing.fixed("ps-spandrel-proud")
	head := drawing.fixed("ps-head-height")
	lap := drawing.fixed("ps-head-lap")
	hang := drawing.fixed("ps-sill-lap")
	top := foot + sill + tall
	parts := []placedElement{
		job.frame.onFace("apron", Box{wide, sill - lift, wall}, stone, at,
			foot, -job.deep()),
		job.frame.onFace("sill", Box{wide + 2*hang, lift, job.deep() +
			drawing.fixed("granite-sill-proud")}, fine.head, at-hang,
			foot+sill-lift, -job.deep()),
		job.frame.onFace("lintel", Box{wide + 2*lap, head, job.deep() +
			drawing.fixed("ps-head-proud")}, fine.head, at-lap, top,
			-job.deep())}
	if above := job.punchedAbove(foot) - top - head; above > 0 {
		parts = append(parts, job.frame.onFace("spandrel", Box{wide, above,
			wall}, stone, at, top+head, -job.deep()))
	}
	parts = append(parts, job.sash(at, wide, foot+sill, tall)...)
	parts = append(parts, job.glass(index, column, at, at+wide, foot+sill,
		tall)...)
	return append(parts, job.room(index, column, at, at+wide, foot,
		foot+sill, tall)...)
}

// punchedAbove is how far up the face the foot of the storey over the
// one that stands on a floor foot up the face is: a storey higher.
func (job faceJob) punchedAbove(foot float64) float64 {
	return foot + job.frame.slope/float64(job.storeys)
}
