package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// keyScale turns a place on a face into a whole number of millimetres, so
// that two anchors a rounding error apart are one.
const keyScale = 1000

// cantileverFine is a span of cantilevered floors at the demo level: at
// every storey a plate thrown out of the glass with a lining under it, a
// glass balustrade on its edge, and between the plates the glass in slim
// mullions. The cable net is anchored: each cable runs from a boss on the
// edge of one plate to a boss on the edge of the plate two storeys up.
func (job faceJob) cantileverFine(low, high, pitch float64,
	columns int) []placedElement {
	drawing := job.drawing
	parts := job.bars("mullion", low, pitch, columns,
		drawing.fixed("cl-mullion"), drawing.fixed("cl-mullion")/2,
		job.looks.grid)
	parts = append(parts, job.eachStorey(low, high, func(index int, foot,
		storey, left, right float64) []placedElement {
		parts := job.plate(index, foot, left, right)
		return append(parts, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				return job.bay(index, column, start, end, foot, storey)
			})...)
	})...)
	return append(parts, job.anchoredCables(low, pitch, columns)...)
}

// bay is the glass between two plates over one bay, from the top of the
// lower plate to the foot of the next, and the room behind it.
func (job faceJob) bay(index, column int, start, end, foot,
	storey float64) []placedElement {
	thick := job.drawing.fixed("cantilever-thick")
	return job.vision(index, column, start, end, foot, foot+thick,
		storey-thick)
}

// reachOf is how far the plate of a storey is thrown out of its face.
func (job faceJob) reachOf(index int) float64 {
	return job.drawing.number("cantilever-reach", place{Bay: job.shaft,
		Storey: job.firstStorey + index, Opening: job.face * faceStride})
}

// plate is the floor plate of a storey across a span, with the lining
// under it and the balustrade along its edge.
func (job faceJob) plate(index int, foot, left, right float64) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	thick, reach := drawing.fixed("cantilever-thick"), job.reachOf(index)
	lining := drawing.fixed("cl-soffit-thick")
	back := drawing.fixed("cl-soffit-inset")
	parts := []placedElement{
		job.frame.onFace("floor plate", Box{right - left, thick,
			job.deep() + reach}, skin.plate, left, foot, -job.deep())}
	if reach > back {
		parts = append(parts, job.frame.onFace("soffit", Box{right - left,
			lining, reach - back}, job.looks.fine.slab, left, foot-lining,
			0))
	}
	return append(parts, job.balustrade(left, right, foot+thick, reach)...)
}

// balustrade is a glass balustrade along the edge of a plate from one
// place to another: a post at each end and every pitch between, a panel of
// glass between each pair, and a handrail along the top.
func (job faceJob) balustrade(left, right, foot, reach float64) []placedElement {
	drawing, fine := job.drawing, job.looks.fine
	size, pitch := drawing.fixed("cl-post-size"),
		drawing.fixed("cl-post-pitch")
	high, rail := drawing.fixed("cl-rail-height"),
		drawing.fixed("cl-rail-size")
	out := reach - drawing.fixed("cl-edge-inset")
	count := max(1, int((right-left)/pitch))
	each := (right - left) / float64(count)
	parts := []placedElement{job.frame.onFace("handrail", Box{right - left, rail,
		rail}, fine.metal, left, foot+high-rail, out)}
	for post := 0; post <= count; post++ {
		along := left + float64(post)*each
		parts = append(parts, job.frame.onFace("post", Box{size, high - rail,
			size}, fine.metal, along-size/2, foot, out))
		if post < count {
			parts = append(parts, job.frame.onFace("balustrade glass",
				Box{each - size, high - rail, job.pane}, fine.glazing,
				along+size/2, foot, out))
		}
	}
	return parts
}

// anchoredCables are the cables of the net across a span, each pair
// crossing in a bay, running from an anchor on the edge of the plate at
// their foot to an anchor on the edge of the plate at their head. Where
// plates are thrown out by different amounts the cable leans out or in.
func (job faceJob) anchoredCables(low, pitch float64, columns int) []placedElement {
	drawing, frame := job.drawing, job.frame
	rise := int(drawing.fixed("cantilever-cable-storeys"))
	storey := frame.slope / float64(job.storeys)
	var parts []placedElement
	anchored := map[[2]float64]bool{}
	for index := 0; index+rise <= job.storeys; index += rise {
		half := frame.half(float64(index+rise) / float64(job.storeys))
		for column := range columns {
			start := low + float64(column)*pitch
			if start+pitch > half || start < -half {
				continue
			}
			foot, head := float64(index)*storey, float64(index+rise)*storey
			from := job.reachOf(index)
			to := job.reachOf(min(index+rise, job.storeys-1))
			parts = append(parts,
				job.tie(start, foot, from, start+pitch, head, to),
				job.tie(start+pitch, foot, from, start, head, to))
			for _, along := range []float64{start, start + pitch} {
				for _, end := range [][3]float64{{foot, from, 0},
					{head, to, -1}} {
					key := [2]float64{math.Round(along * keyScale),
						math.Round(end[0] * keyScale)}
					if !anchored[key] {
						anchored[key] = true
						parts = append(parts, job.anchor(along, end[0],
							end[1], end[2]))
					}
				}
			}
		}
	}
	return parts
}

// anchor is the boss a cable ends in at a place on the face: along it, up
// its slope and out of it, standing up from the place by a share of its
// own side, nought to stand on it and minus one to hang from it.
func (job faceJob) anchor(along, up, out, share float64) placedElement {
	boss := job.drawing.fixed("cl-anchor")
	return job.frame.onFace("anchor", Box{boss, boss, boss},
		job.looks.fine.metal, along-boss/2, up+share*boss, out)
}

// tie is one cable from a place on the face to another, each given along
// the face, up its slope and out of it.
func (job faceJob) tie(fromAlong, fromUp, fromOut, toAlong, toUp,
	toOut float64) placedElement {
	frame, drawing := job.frame, job.drawing
	run := frame.along.Scale(toAlong - fromAlong).Add(
		frame.up.Scale(toUp - fromUp)).Add(frame.out.Scale(toOut - fromOut))
	along := run.Unit()
	rise := frame.along.Cross(along).Unit()
	if rise.Len() == 0 {
		rise = frame.along
	}
	across := along.Cross(rise).Unit()
	thick := drawing.fixed("cantilever-cable")
	origin := frame.at(fromAlong, fromUp, fromOut).Origin
	return placedElement{Element: solid{Name: "cable", Form: Box{run.Len(), thick,
		thick}, Material: job.looks.skin.cable}, Place: maths.NewSubspace(
		origin, maths.Orientation{X: along, Y: rise, Z: across}, 0)}
}
