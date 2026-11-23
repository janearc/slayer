package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// demoLevel is the detail at which a facade is drawn part by part: the
// window's lights and bars, its sill and head, the shutters' slats, the
// door's panels and step, and a cornice. The hull draws only the wall,
// and the quick level the cut wall with each opening's frame, pane and
// shutters whole.
const demoLevel Detail = "demo"

// finish is the measures of the small parts a facade is drawn with at the
// demo level, all drawn from the culture's data.
type finish struct {
	// MuntinWidth is the width of the bars between a window's lights, and
	// LightsAcross and LightsUp how many lights a window has each way.
	MuntinWidth            float64
	LightsAcross, LightsUp int
	// SillDepth is how far a sill and a head stand out of the wall, and
	// SillThickness and HeadHeight how thick each is.
	SillDepth, SillThickness, HeadHeight float64
	// SlatPitch is how far apart the slats of a shutter are up it,
	// SlatChord how wide a slat is across its tilt, SlatLeaf how thick,
	// and SlatTilt how far, in degrees, it leans. StileWidth and
	// RailHeight are the widths of the bars of a shutter's frame.
	SlatPitch, SlatChord, SlatLeaf, SlatTilt float64
	StileWidth, RailHeight                   float64
	// SlatThickness is how far a shutter's frame stands out of the wall.
	SlatThickness float64
	// PanelInset is how far a door's raised panels are in from its edges,
	// HandleSize the size of its handle, and StepDepth and StepHeight the
	// size of the step before it.
	PanelInset, HandleSize, StepDepth, StepHeight float64
	// CorniceHeight and CorniceProud are the height of the cornice at the
	// top of the face and how far it stands out of the wall.
	CorniceHeight, CorniceProud float64
	// RoomDepth is how deep a room behind a window is, RoomSlab the
	// thickness of its floor, ceiling and walls, and BackWall the wall of
	// the house that must be left behind the rooms.
	RoomDepth, RoomSlab, BackWall float64
	// LampSize is the side of a lamp, LampDrop how far it hangs below the
	// ceiling, and CurtainThickness how thick a curtain is.
	LampSize, LampDrop, CurtainThickness float64
	// FrameDepth is how deep the bars of a window's frame are.
	FrameDepth float64
	// LampStrength is the luminance at a metre of the light in a lamp, in
	// the kit's relative light.
	LampStrength float64
}

// finish draws the measures of a facade's small parts from the culture.
func (drawing *drawer) finish() finish {
	return finish{MuntinWidth: drawing.fixed("muntin-width"),
		LightsAcross:  int(drawing.fixed("lights-across")),
		LightsUp:      int(drawing.fixed("lights-up")),
		SillDepth:     drawing.fixed("sill-depth"),
		SillThickness: drawing.fixed("sill-thickness"),
		HeadHeight:    drawing.fixed("head-height"),
		SlatPitch:     drawing.fixed("slat-pitch"),
		SlatChord:     drawing.fixed("slat-chord"),
		SlatLeaf:      drawing.fixed("slat-leaf"),
		SlatTilt:      drawing.fixed("slat-tilt"),
		RailHeight:    drawing.fixed("rail-height"),
		SlatThickness: drawing.fixed("slat-thickness"),
		StileWidth:    drawing.fixed("stile-width"),
		PanelInset:    drawing.fixed("panel-inset"),
		HandleSize:    drawing.fixed("handle-size"),
		StepDepth:     drawing.fixed("step-depth"),
		StepHeight:    drawing.fixed("step-height"),
		CorniceHeight: drawing.fixed("cornice-height"),
		CorniceProud:  drawing.fixed("cornice-proud")}.rooms(drawing)
}

// rooms is the finish with the measures of the rooms behind the windows
// drawn from the culture.
func (fine finish) rooms(drawing *drawer) finish {
	fine.RoomDepth = drawing.fixed("room-depth")
	fine.RoomSlab = drawing.fixed("room-slab")
	fine.BackWall = drawing.fixed("back-wall")
	fine.LampSize = drawing.fixed("lamp-size")
	fine.LampDrop = drawing.fixed("lamp-drop")
	fine.LampStrength = drawing.fixed("lamp-strength")
	fine.FrameDepth = drawing.fixed("frame-depth")
	fine.CurtainThickness = drawing.fixed("curtain-thickness")
	return fine
}

// recess is how deep the cuts in a wall are: at the demo level the wall's
// whole thickness round a window, the reveal, which the sun shades, and
// otherwise the shallow cut the frame fills.
func (drawing *drawer) recess() float64 {
	if drawing.level == demoLevel {
		return drawing.fixed("reveal-depth")
	}
	return drawing.fixed("recess")
}

// frameDepth is how deep the bars of a window frame are: the whole cut
// below the demo level, and at it a bar set at the back of the reveal, so
// that the reveal is wall.
func (facade facade) frameDepth() float64 {
	if facade.Level == demoLevel {
		return facade.Finish.FrameDepth
	}
	return facade.Recess
}

// slab is the face at the hull: the wall in one box, with nothing cut.
func (facade facade) slab() []placedElement {
	return []placedElement{put("wall", Box{facade.Width, facade.Height,
		facade.Recess}, facade.Wall, maths.V(0, 0, -facade.Recess))}
}

// lights are a recessed window's panes at the demo level: the lights of
// its sash, in a grid, with a muntin bar between each pair, the bars as
// deep as half the cut and the lights on the back of it.
func (facade facade) lights(opening facadeOpening) []placedElement {
	bar, fine := facade.FrameWidth, facade.Finish
	across, up := max(fine.LightsAcross, 1), max(fine.LightsUp, 1)
	width := opening.Width - 2*bar
	height := opening.Height - 2*bar
	cell := (width - float64(across-1)*fine.MuntinWidth) / float64(across)
	row := (height - float64(up-1)*fine.MuntinWidth) / float64(up)
	back := -facade.Recess
	var parts []placedElement
	for column := range across {
		left := opening.Left + bar + float64(column)*(cell+fine.MuntinWidth)
		for level := range up {
			foot := opening.Sill + bar + float64(level)*(row+fine.MuntinWidth)
			parts = append(parts, put("light", Box{cell, row,
				facade.GlassThickness}, opening.Pane, maths.V(left, foot,
				back)))
		}
		if column < across-1 {
			parts = append(parts, put("muntin", Box{fine.MuntinWidth, height,
				facade.Recess / half}, opening.Frame, maths.V(left+cell,
				opening.Sill+bar, back)))
		}
	}
	for level := range up - 1 {
		foot := opening.Sill + bar + row +
			float64(level)*(row+fine.MuntinWidth)
		parts = append(parts, put("muntin", Box{width, fine.MuntinWidth,
			facade.Recess / half}, opening.Frame, maths.V(opening.Left+bar,
			foot, back)))
	}
	return parts
}

// surround is the sill under a window and the head over it, each a slab a
// little wider than the frame, standing out of the wall. A head is left
// out where it would pass the top of its storey.
func (facade facade) surround(opening facadeOpening, storey float64) []placedElement {
	bar, fine := facade.FrameWidth, facade.Finish
	left, width := opening.Left-bar, opening.Width+2*bar
	var parts []placedElement
	if opening.Sill >= fine.SillThickness {
		parts = append(parts, put("sill", Box{width, fine.SillThickness,
			fine.SillDepth}, opening.Frame, maths.V(left,
			opening.Sill-fine.SillThickness, 0)))
	}
	head := opening.Sill + opening.Height
	if head+fine.HeadHeight <= storey {
		parts = append(parts, put("head", Box{width, fine.HeadHeight,
			fine.SillDepth}, opening.Frame, maths.V(left, head, 0)))
	}
	return parts
}

// leaf is a door at the demo level: the leaf on the back of the cut, a
// raised panel above and one below, a handle, and the step before it.
func (facade facade) leaf(opening facadeOpening) []placedElement {
	bar, fine := facade.FrameWidth, facade.Finish
	back := -facade.Recess
	width, height := opening.Width-2*bar, opening.Height-2*bar
	left, foot := opening.Left+bar, opening.Sill+bar
	panel := Box{width - 2*fine.PanelInset, height * doorPanelShare,
		facade.GlassThickness / half}
	parts := []placedElement{
		put("door leaf", Box{width, height, facade.GlassThickness},
			opening.Pane, maths.V(left, foot, back)),
		put("door panel", panel, opening.Pane, maths.V(left+fine.PanelInset,
			foot+fine.PanelInset, back+facade.GlassThickness)),
		put("door panel", panel, opening.Pane, maths.V(left+fine.PanelInset,
			foot+height-fine.PanelInset-panel.Height,
			back+facade.GlassThickness)),
		put("handle", Box{fine.HandleSize, fine.HandleSize * half,
			fine.HandleSize}, opening.Frame, maths.V(left+width-
			fine.PanelInset-fine.HandleSize, foot+height/half,
			back+facade.GlassThickness)),
		put("step", Box{opening.Width, fine.StepHeight, fine.StepDepth},
			opening.Frame, maths.V(opening.Left, 0, 0)),
	}
	return append(parts, facade.stoop(opening)...)
}

// spandrels fill the upper corners of an arched opening: for each of a
// run of slices up the arch, the wall between the opening's edge and the
// half circle, on each side, in front of the door's leaf.
func (facade facade) spandrels(opening facadeOpening) []placedElement {
	radius := opening.Width / half
	centre := opening.Sill + opening.Height - radius
	slice := radius / archSlices
	depth := facade.Recess - facade.GlassThickness
	var parts []placedElement
	for index := range archSlices {
		foot := centre + float64(index)*slice
		rise := foot + slice/half - centre
		fill := radius - math.Sqrt(radius*radius-rise*rise)
		if !(fill > archLeast) {
			continue
		}
		parts = append(parts,
			put("spandrel", Box{fill, slice, depth}, facade.Wall,
				maths.V(opening.Left, foot, -depth)),
			put("spandrel", Box{fill, slice, depth}, facade.Wall,
				maths.V(opening.Left+opening.Width-fill, foot, -depth)))
	}
	return parts
}

// archSlices is how many steps the half circle of an arch is cut into, and
// archLeast the least width in metres a corner filling has to be drawn.
// Choices of the detail study, by eye.
const (
	archSlices = 10
	archLeast  = 0.004
)

// sectional is a garage door at the demo level: sections one over another
// with a groove between each pair, and in each section two raised panels.
func (facade facade) sectional(opening facadeOpening) []placedElement {
	bar, glass := facade.FrameWidth, facade.GlassThickness
	count := int(facade.measured("garage-sections"))
	groove := facade.measured("garage-groove")
	inset := facade.measured("garage-inset")
	width, height := opening.Width-2*bar, opening.Height-2*bar
	section := (height - float64(count-1)*groove) / float64(count)
	back := -facade.Recess
	var parts []placedElement
	for index := range count {
		foot := opening.Sill + bar + float64(index)*(section+groove)
		parts = append(parts, put("garage section", Box{width, section, glass},
			opening.Pane, maths.V(opening.Left+bar, foot, back)))
		for _, side := range []float64{0, 1} {
			wide := (width - 3*inset) / half
			parts = append(parts, put("garage panel", Box{wide,
				section - 2*inset, glass / half}, opening.Pane, maths.V(
				opening.Left+bar+inset+side*(wide+inset), foot+inset,
				back+glass)))
		}
	}
	return parts
}

// stoop is the lower tread before a door, wider than the door and
// beyond its step.
func (facade facade) stoop(opening facadeOpening) []placedElement {
	fine, side := facade.Finish, facade.measured("stoop-side")
	return []placedElement{put("stoop", Box{opening.Width + 2*side,
		fine.StepHeight / half, fine.StepDepth}, opening.Frame,
		maths.V(opening.Left-side, 0, fine.StepDepth))}
}

// doorPanelShare is the share of a door's height a raised panel takes.
const doorPanelShare = 0.38

// slatted is one shutter panel at the demo level: a stile down each edge
// and slats between them, evenly down its height.
func (facade facade) slatted(left, foot, height float64,
	opening facadeOpening) []placedElement {
	fine := facade.Finish
	thick := facade.ShutterThickness
	width := opening.ShutterWidth
	parts := []placedElement{
		put("stile", Box{fine.StileWidth, height, thick}, opening.Shutter,
			maths.V(left, foot, 0)),
		put("stile", Box{fine.StileWidth, height, thick}, opening.Shutter,
			maths.V(left+width-fine.StileWidth, foot, 0)),
	}
	middle := foot + (height-fine.RailHeight)/half
	for _, rail := range []float64{foot, middle,
		foot + height - fine.RailHeight} {
		parts = append(parts, put("rail", Box{width, fine.RailHeight, thick},
			opening.Shutter, maths.V(left, rail, 0)))
	}
	return append(parts, facade.louvres(left+fine.StileWidth, foot, height,
		width-2*fine.StileWidth, opening)...)
}

// louvres are the slats of a shutter between its rails: thin boards the
// width of the frame's opening, each tilted so that its top leans in, a
// pitch apart up the shutter, none where the mid rail is.
func (facade facade) louvres(left, foot, height, across float64,
	opening facadeOpening) []placedElement {
	fine := facade.Finish
	middle := foot + (height-fine.RailHeight)/half
	tilt := maths.Rotation(maths.XAxis, -fine.SlatTilt*maths.Degree)
	var parts []placedElement
	for rise := foot + fine.RailHeight; rise+fine.SlatChord <=
		foot+height-fine.RailHeight; rise += fine.SlatPitch {
		if rise+fine.SlatChord > middle && rise < middle+fine.RailHeight {
			continue
		}
		parts = append(parts, placedElement{Element: solid{Name: "slat",
			Form:     Box{across, fine.SlatChord, fine.SlatLeaf},
			Material: opening.Shutter},
			Place: maths.NewSubspace(maths.V(left, rise,
				facade.ShutterThickness/half), tilt, 0)})
	}
	return parts
}

// cornice is the band at the top of a face, standing out of the wall.
func (facade facade) cornice() []placedElement {
	fine := facade.Finish
	return []placedElement{put("cornice", Box{facade.Width + 2*fine.CorniceProud,
		fine.CorniceHeight, fine.CorniceProud}, facade.Wall,
		maths.V(-fine.CorniceProud, facade.Height-fine.CorniceHeight, 0))}
}
