package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The facade systems a building may be drawn in, each a method of
// building that shows on the face: render with nothing added, precast
// concrete panels with a joint between each pair, and a curtain wall of
// spandrel bands between ribbons of glass, held by mullions.
const (
	systemRender  = "stucco"
	systemPrecast = "precast"
	systemCurtain = "curtain-wall"
)

// renderCladding is the name of the cladding of a wall that is plastered:
// the one a facade system may be drawn over.
const renderCladding = "render"

// system is the method a face is built by, and the materials of what
// shows of it: the joint of a precast panel, the spandrel panel of a
// curtain wall, and the frame that holds its glass.
type system struct {
	Kind                     string
	Joint, Spandrel, Mullion *thing.Material
}

// system draws the facade system of a bay, from the culture, in the
// materials its data gives. A building whose cladding is brick shows
// its courses and is drawn in no system.
func (drawing *drawer) system(at place, base *thing.Material) system {
	pick := drawing.pick("facade-system", drawing.options("facade-system"),
		at)
	if claddingOf(base) != renderCladding || pick.Name == systemRender {
		return system{Kind: systemRender}
	}
	return system{Kind: pick.Name,
		Joint: paint("joint", drawing.pick("joint-colour",
			drawing.options("joint-colour"), at)),
		Spandrel: paint("spandrel", drawing.pick("spandrel-colour",
			drawing.options("spandrel-colour"), at)),
		Mullion: paint("mullion", drawing.pick("mullion-colour",
			drawing.options("mullion-colour"), at))}
}

// system is the face's method of building, drawn in one storey's frame,
// whose floor is at nought: the joints of precast panels, or the
// spandrel bands and mullions of a curtain wall.
func (facade facade) system(openings []facadeOpening, storey float64) []placedElement {
	switch facade.System.Kind {
	case systemPrecast:
		return facade.joints(openings, storey)
	case systemCurtain:
		return facade.curtainWall(openings, storey)
	}
	return nil
}

// joints are the seams between the precast panels of one storey: a
// horizontal one at its floor, and a vertical one a panel's width apart
// along the face, left out where an opening is, since a seam does not
// cross a window.
func (facade facade) joints(openings []facadeOpening, storey float64) []placedElement {
	panel := facade.measured("panel-width")
	seam := facade.measured("joint-width")
	proud := facade.measured("joint-proud")
	parts := []placedElement{put("joint", Box{facade.Width, seam, proud},
		facade.System.Joint, maths.Vec{})}
	for left := panel; left+seam < facade.Width; left += panel {
		if !crosses(openings, left, left+seam) {
			parts = append(parts, put("joint", Box{seam, storey, proud},
				facade.System.Joint, maths.V(left, 0, 0)))
		}
	}
	return parts
}

// crosses is whether a span along the face, between two places, is over
// an opening of the storey.
func crosses(openings []facadeOpening, from, to float64) bool {
	for _, opening := range openings {
		if from < opening.Left+opening.Width && to > opening.Left {
			return true
		}
	}
	return false
}

// curtainWall is the curtain wall of one storey: a spandrel panel under
// each window and another over it, and a mullion at the edge of each
// window.
func (facade facade) curtainWall(openings []facadeOpening, storey float64) []placedElement {
	proud := facade.measured("spandrel-proud")
	bar := facade.measured("mullion-width")
	lean := facade.measured("mullion-proud")
	var parts []placedElement
	for _, opening := range openings {
		if opening.Door {
			continue
		}
		parts = append(parts, put("spandrel", Box{opening.Width, opening.Sill,
			proud}, facade.System.Spandrel, maths.V(opening.Left, 0, 0)))
		head := opening.Sill + opening.Height
		if above := storey - head; above > 0 {
			parts = append(parts, put("spandrel", Box{opening.Width, above,
				proud}, facade.System.Spandrel, maths.V(opening.Left, head,
				0)))
		}
		parts = append(parts, facade.mullions(opening, storey, bar, lean)...)
	}
	return parts
}

// mullions are the two bars of a curtain wall at the sides of a window,
// as tall as its storey, standing out of the wall a little.
func (facade facade) mullions(opening facadeOpening, storey, bar,
	lean float64) []placedElement {
	return []placedElement{
		put("mullion", Box{bar, storey, lean}, facade.System.Mullion,
			maths.V(opening.Left-bar, 0, 0)),
		put("mullion", Box{bar, storey, lean}, facade.System.Mullion,
			maths.V(opening.Left+opening.Width, 0, 0)),
	}
}
