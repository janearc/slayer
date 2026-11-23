package cities

import "github.com/janearc/slayer/maths"

// guardUnder is what a stair core's landing window is given instead of
// anything under it: a guard rail across the window's foot.
const guardUnder = "guard"

// guard is the rail across a landing window of a stair core, as a window
// that opens to the stair would have: a top bar and uprights, standing
// out of the wall by the culture's depth, a little wider than the window.
func (facade facade) guard(opening facadeOpening) []placedElement {
	side := facade.measured("guard-side")
	deep := facade.measured("guard-depth")
	rail := facade.measured("guard-height")
	left, wide := opening.Left-side, opening.Width+2*side
	parts := facade.rails(left, opening.Sill, wide, deep, rail)
	return append(parts, facade.uprights(left, opening.Sill, wide, deep,
		rail)...)
}

// portal is the stone surround of a stair core's entrance: a jamb on each
// side, a lintel over the door, and beside it the panel of buttons and a
// speaker that lets a visitor ring a flat. It is nothing for a door that
// has no portal.
func (facade facade) portal(opening facadeOpening) []placedElement {
	if !opening.Extra.Portal {
		return nil
	}
	jamb := facade.measured("portal-jamb")
	proud := facade.measured("portal-proud")
	stone := facade.Fixtures.Stone
	high := opening.Sill + opening.Height
	wide := facade.measured("entry-panel-width")
	return []placedElement{
		put("portal jamb", Box{jamb, high, proud}, stone,
			maths.V(opening.Left-jamb, opening.Sill, 0)),
		put("portal jamb", Box{jamb, high, proud}, stone,
			maths.V(opening.Left+opening.Width, opening.Sill, 0)),
		put("portal lintel", Box{opening.Width + 2*jamb, jamb, proud}, stone,
			maths.V(opening.Left-jamb, high, 0)),
		put("entry panel", Box{wide, facade.measured("entry-panel-height"),
			facade.measured("entry-panel-proud")}, facade.Iron,
			maths.V(opening.Left+opening.Width+jamb+wide, facade.measured(
				"entry-panel-rise"), 0)),
	}
}
