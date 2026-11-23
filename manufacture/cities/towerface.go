package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// faceStride is how many columns of windows a face may have, so that a
// window's place can number its face and its column in one counter.
const faceStride = 1000

// faceFrame is one flat face of a segment: where it stands, the way it
// runs, how wide it is at its foot and at its top, and the stretches of
// it that no other part of the plan covers. Its local x runs along the
// face from its middle, y runs up the slope of the face, and z runs out
// of it. A face that leans in, as the faces of a taper do, is a frame
// that leans, so what stands on it leans with it.
type faceFrame struct {
	origin, along, up, out maths.Vec
	length, topLength      float64
	slope                  float64
	spans                  [][2]float64
}

// at is a place on the face: along from its middle, up its slope, and
// out of it.
func (frame faceFrame) at(along, up, out float64) maths.Subspace {
	point := frame.origin.Add(frame.along.Scale(along)).
		Add(frame.up.Scale(up)).Add(frame.out.Scale(out))
	return maths.NewSubspace(point, maths.Orientation{X: frame.along,
		Y: frame.up, Z: frame.out}, 0)
}

// half is half the width of the face at a share of its height.
func (frame faceFrame) half(share float64) float64 {
	return (frame.length + (frame.topLength-frame.length)*share) / 2
}

// onFace is a box standing on the face, its corner at a place on it.
func (frame faceFrame) onFace(name string, size Box, look *thing.Material,
	along, up, out float64) placedElement {
	return placedElement{Element: solid{Name: name, Form: size, Material: look},
		Place: frame.at(along, up, out)}
}

// frames are the flat faces of a segment: one for each edge of each part
// of its plan, from the bottom edge to the matching edge of the top. A
// part is seen in the middle height of the others, so a face covered by
// another part of the plan is left with only what shows.
func frames(bottom, top [][]outlineCorner, base, height float64) []faceFrame {
	var found []faceFrame
	for index, part := range bottom {
		var others [][]outlineCorner
		for other := range bottom {
			if other != index {
				others = append(others, halfway(bottom[other], top[other]))
			}
		}
		for edge := range part {
			next := (edge + 1) % len(part)
			found = append(found, buildFrame(part, top[index], edge, next,
				base, height, others))
		}
	}
	return found
}

// halfway is the plan midway between a part's bottom and top.
func halfway(bottom, top []outlineCorner) []outlineCorner {
	middle := make([]outlineCorner, len(bottom))
	for index := range bottom {
		middle[index] = outlineCorner{(bottom[index].X + top[index].X) / 2,
			(bottom[index].Z + top[index].Z) / 2}
	}
	return middle
}

// buildFrame is the face on the edge of a part from one corner to the
// next, and on the same edge of the top.
func buildFrame(part, top []outlineCorner, from, to int, base, height float64,
	others [][]outlineCorner) faceFrame {
	foot0 := maths.V(part[from].X, base, part[from].Z)
	foot1 := maths.V(part[to].X, base, part[to].Z)
	head0 := maths.V(top[from].X, base+height, top[from].Z)
	head1 := maths.V(top[to].X, base+height, top[to].Z)
	along := foot1.Sub(foot0).Unit()
	origin := foot0.Add(foot1).Scale(0.5)
	lean := head0.Add(head1).Scale(0.5).Sub(origin)
	up := lean.Sub(along.Scale(lean.Dot(along))).Unit()
	out := along.Cross(up)
	centre := centroid(part)
	away := maths.V(origin.X-centre.X, 0, origin.Z-centre.Z)
	start, end := part[from], part[to]
	if out.Dot(away) < 0 {
		along, start, end = along.Neg(), end, start
		out = along.Cross(up)
	}
	length := foot1.Sub(foot0).Len()
	frame := faceFrame{origin: origin, along: along, up: up, out: out,
		length: length, topLength: head1.Sub(head0).Len(),
		slope: lean.Len()}
	for _, share := range spans(start, end, others) {
		frame.spans = append(frame.spans, [2]float64{
			(share[0] - 0.5) * length, (share[1] - 0.5) * length})
	}
	return frame
}

// centroid is the mean of the corners.
func centroid(corners []outlineCorner) outlineCorner {
	var sum outlineCorner
	for _, corner := range corners {
		sum.X += corner.X
		sum.Z += corner.Z
	}
	count := float64(len(corners))
	return outlineCorner{sum.X / count, sum.Z / count}
}

// faceJob is everything needed to dress one face: the measures the
// culture gives, the materials, and where the face is in its shaft.
type faceJob struct {
	drawing                                       *drawer
	looks                                         towerLooks
	frame                                         faceFrame
	shaft, face                                   int
	storeys, firstStorey, shaftStoreys, mechEvery int
	module, mullion, spandrel                     float64
	recess, margin, pane                          float64
	// skin is what the face is made of, segment the segment of the shaft
	// it is on, and use and style what that segment is for and how a
	// granite one is windowed.
	skin    Skin
	segment int
	use     Use
	style   string
	// last says the segment is the top of the shaft, ledge how far its
	// plan steps in from the one below, nought if it does not, and
	// seconds is the time the signs are drawn at.
	last           bool
	ledge, seconds float64
	// demo says the face is drawn at the demo level, part by part, and
	// cavity how deep the rooms behind its glass are, nought below it.
	demo   bool
	cavity float64
	// beneath is how much of its segment stands under the face, when the
	// face is one band of the segment's storeys, so that what hangs from
	// its lowest balcony hangs as far as it would on the whole face.
	beneath float64
}

// mechanical is whether a storey of the shaft, counted from nought, is a
// plant floor: every so many storeys, never the top one, which the crown
// is on.
func (job faceJob) mechanical(storey int) bool {
	return (storey+1)%job.mechEvery == 0 && storey+1 < job.shaftStoreys
}

// dress is a face dressed: a grid of mullions standing out of the glass
// by the recess, a spandrel at the foot of every storey, a band of
// louvres in place of glass and spandrel on a plant floor, and a lit
// pane in every window that is alight. The glass itself is the face of
// the body behind, drawn in by the recess, so every window is recessed.
func (job faceJob) dress() []placedElement {
	var parts []placedElement
	for _, span := range job.frame.spans {
		low, high := span[0]+job.margin, span[1]-job.margin
		if !(high-low >= job.least()) {
			continue
		}
		columns := max(1, int(math.Floor((high-low)/job.module)))
		pitch := (high - low) / float64(columns)
		parts = append(parts, job.span(low, high, pitch, columns)...)
	}
	return parts
}

// least is the least a run of face may be to be dressed: a window module,
// but for a ribbon of glass, which needs only the width of a bar, so that
// the narrow faces of an ellipse or an arc are dressed too.
func (job faceJob) least() float64 {
	if job.skin == skinStainless ||
		job.skin == skinGranite && job.style == styleRibbon {
		return job.mullion
	}
	return job.module
}

// span is one run of a face dressed in the face's skin.
func (job faceJob) span(low, high, pitch float64, columns int) []placedElement {
	parts := job.skinSpan(low, high, pitch, columns)
	if job.demo {
		parts = append(parts, job.ground(low, high, pitch, columns)...)
		parts = append(parts, job.fasteners(low, pitch, columns)...)
	}
	return parts
}

// skinSpan is a span of a face in the face's skin, above its ground floor.
func (job faceJob) skinSpan(low, high, pitch float64,
	columns int) []placedElement {
	if job.demo {
		if parts, drawn := job.fineSpan(low, high, pitch,
			columns); drawn {
			return parts
		}
	}
	switch job.skin {
	case skinMirror:
		return job.mirrorSpan(low, high, pitch, columns)
	case skinGranite:
		if job.style == styleRibbon {
			return job.ribbonSpan(low, high, pitch, columns)
		}
		return job.punchedSpan(low, high, pitch, columns)
	case skinStainless:
		return job.ribbonSpan(low, high, pitch, columns)
	case skinConcrete:
		return job.concreteSpan(low, high, pitch, columns)
	case skinGothic:
		return job.gothicSpan(low, high, pitch, columns)
	case skinCantilever:
		return job.cantileverSpan(low, high, pitch, columns)
	case skinVaporwave:
		return job.vaporSpan(low, high, pitch, columns)
	}
	parts := job.mullions(low, pitch, columns)
	parts = append(parts, job.floors(low, high)...)
	return append(parts, job.panes(low, pitch, columns)...)
}

// mullions are the vertical bars between the windows of a span, one at
// each end of each window, as tall as the face is wide enough for.
func (job faceJob) mullions(low, pitch float64, columns int) []placedElement {
	var bars []placedElement
	frame := job.frame
	for column := 0; column <= columns; column++ {
		along := low + float64(column)*pitch
		tall := 1.0
		if frame.topLength < frame.length {
			tall = math.Min(1, (frame.length/2-math.Abs(along))/
				((frame.length-frame.topLength)/2))
		}
		if !(tall > 0) {
			continue
		}
		bars = append(bars, frame.onFace("mullion", Box{job.mullion,
			tall * frame.slope, job.recess}, job.looks.grid,
			along-job.mullion/2, 0, -job.recess))
	}
	return bars
}

// floors are the spandrels at the foot of each storey of a span, and the
// plant bands of the plant floors, each as wide as the face is there.
func (job faceJob) floors(low, high float64) []placedElement {
	var parts []placedElement
	frame := job.frame
	storey := frame.slope / float64(job.storeys)
	for index := range job.storeys {
		half := frame.half((float64(index) + 0.5) / float64(job.storeys))
		left, right := math.Max(low, -half), math.Min(high, half)
		if !(right > left) {
			continue
		}
		foot := float64(index) * storey
		if job.mechanical(job.firstStorey + index) {
			parts = append(parts, job.plant(left, right, foot, storey)...)
			continue
		}
		parts = append(parts, frame.onFace("spandrel", Box{right - left,
			job.spandrel, job.recess}, job.looks.grid, left, foot,
			-job.recess))
	}
	return parts
}

// plant is a plant floor on a span: a slab over the whole storey and
// the louvres stood on it, in the dark plant material and never lit.
func (job faceJob) plant(left, right, foot, storey float64) []placedElement {
	drawing, frame := job.drawing, job.frame
	proud := drawing.fixed("plant-proud")
	slab := frame.onFace("plant", Box{right - left, storey,
		job.deep() + proud}, job.looks.plant, left, foot, -job.deep())
	parts := []placedElement{slab}
	count := int(drawing.fixed("louvre-count"))
	slat := drawing.fixed("louvre-height")
	for index := range count {
		up := foot + (float64(index)+0.5)*storey/float64(count) - slat/2
		parts = append(parts, frame.onFace("louvre", Box{right - left, slat,
			drawing.fixed("louvre-proud")}, job.looks.plant, left, up,
			proud))
	}
	return parts
}

// panes are the lit windows of a span: for each column and each storey
// that is not a plant floor, a pane of the window's size in the colour
// of its light if the culture says its light is on. A window that
// would stand off the face at the storey's top is left out.
func (job faceJob) panes(low, pitch float64, columns int) []placedElement {
	var lit []placedElement
	frame := job.frame
	storey := frame.slope / float64(job.storeys)
	for index := range job.storeys {
		global := job.firstStorey + index
		if job.mechanical(global) {
			continue
		}
		top := (float64(index) + 1) / float64(job.storeys)
		for column := range columns {
			left := low + float64(column)*pitch + job.mullion/2
			right := left + pitch - job.mullion
			if right > frame.half(top) || left < -frame.half(top) {
				continue
			}
			lit = append(lit, job.lit(index, column, left, right-left,
				float64(index)*storey+job.spandrel, storey-job.spandrel,
				-job.recess)...)
		}
	}
	return lit
}
