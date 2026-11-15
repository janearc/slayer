package main

import (
	"image"
	"math"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/improv/walk"
)

// The panes' names on the frame. The words are drafts, to be rewritten.
const (
	viewerWords = "The show, rendered (cropped to fill)"
	aboveWords  = "Above, north up: drawn"
	moonWords   = "Ridge, moon behind: drawn"
	// a traced pane's words: what is rendered and what is drawn on it
	renderedWords = "Rendered: the set by the show's light"
	// renderedMore is a traced pane's second line: what is drawn, and
	// what the show lays in from the viewer's eye alone and so is not
	// there from this camera
	renderedMore = "Drawn: beams. Not shown: water, moon"
	viewerSpot   = "you are here"
	northWord    = "N"
)

// captionSize and captionInset are the panes' names' type and place,
// nameSize a fixture kind's name, in pixels of a pane 960 across.
// Picked by the untraced film's author, 2026-11-12.
const (
	captionSize  = 26.0
	captionInset = 12
	nameSize     = 20.0
	nameRise     = 34
	// smallestType is the least size any pane's words are set at, so a
	// tall layout's small panes stay legible on a phone: 28 pixels of
	// 1080 is about 9 on a phone 360 across, the house's small-print
	smallestType = 28.0
)

// compose lays the four panes for a moment, each filling its pane: the
// film cropped to fill, the panel drawn for the pane, and the two
// stills with the rig drawn on them from the cue state at that moment.
func (made parts) compose(seconds float64, viewer *image.RGBA) (
	*image.RGBA, error) {
	frame, err := made.layout.Ground()
	if err != nil {
		return nil, err
	}
	moment, err := made.sheet.At(seconds, filmRate)
	if err != nil {
		return nil, err
	}
	state, err := cuesOf(moment.Lines())
	if err != nil {
		return nil, err
	}
	place, err := made.layout.Pane(walk.PaneViewer)
	if err != nil {
		return nil, err
	}
	walk.Laid(frame, viewer, place, walk.Crop)
	if err := caption(frame, viewerWords, place); err != nil {
		return nil, err
	}
	explaining, err := made.layout.Pane(walk.PaneExplain)
	if err != nil {
		return nil, err
	}
	if err := drawPanel(frame, explaining, made.explainer,
		seconds); err != nil {
		return nil, err
	}
	for _, pane := range []struct {
		name, words string
		drawn       view
		traced      *traced
	}{{walk.PaneAbove, aboveWords, made.above, made.aboveTraced},
		{walk.PaneMoon, moonWords, made.moon, made.moonTraced}} {
		if err := made.drawnPane(frame, pane.name, pane.words, pane.drawn,
			pane.traced, state, seconds); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

// drawnPane lays a drawn pane: its still's part graded down, the rig
// drawn on it, the kinds of fixture named, the viewer's spot marked,
// and the pane's name.
func (made parts) drawnPane(frame *image.RGBA, name, words string,
	pane view, rendered *traced, state cues, seconds float64) error {
	place, err := made.layout.Pane(name)
	if err != nil {
		return err
	}
	pane.part = partAt(pane.part, pane.moves, seconds)
	picture := pane.picture(place.Size())
	more := ""
	if rendered != nil && Rendered(state) {
		if picture, err = rendered.at(state, pane.part,
			place.Size()); err != nil {
			return err
		}
		words, more = renderedWords, renderedMore
	}
	names := drawRig(picture, pane, state, made.rigLayout, seconds,
		made.tones)
	walk.Laid(frame, picture, place, walk.Crop)
	size := math.Max(smallestType, nameSize*float64(place.Dx())/panelWidth)
	var taken []image.Rectangle
	for _, each := range names {
		box, err := edgeName(frame, place, each, size, taken)
		if err != nil {
			return err
		}
		taken = append(taken, box)
	}
	x, y, ahead := pane.onPane(rig.Point{}, place.Size())
	if ahead && (image.Point{int(x), int(y)}).In(image.Rectangle{
		Max: place.Size()}) {
		wide, err := walk.Measure(viewerSpot, size)
		if err != nil {
			return err
		}
		// to the left of the spot, so the backlights beside it keep
		// their name
		left := int(x) - wide - 3*captionInset
		if left < 0 {
			// no room to its left: to its right
			left = int(x) + 2*captionInset
		}
		at := place.Min.Add(image.Pt(left, int(y)-captionInset))
		height := int(size) + 2*captionInset
		for moved := true; moved; {
			moved = false
			for _, other := range taken {
				if (image.Rectangle{Min: at, Max: at.Add(image.Pt(
					wide+2*captionInset, height))}).Overlaps(other) {
					at.Y, moved = other.Max.Y+captionInset/2, true
				}
			}
		}
		if err := walk.Caption(frame, viewerSpot, at, size); err != nil {
			return err
		}
	}
	if name == walk.PaneAbove {
		if err := walk.Caption(frame, northWord+" up", place.Min.Add(
			image.Pt(place.Dx()-4*captionInset-int(4*size),
				captionInset)), size); err != nil {
			return err
		}
	}
	if err := caption(frame, words, place); err != nil || more == "" {
		return err
	}
	return caption(frame, more, place.Add(image.Pt(0,
		int(captionTypeFor(place))+2*captionInset)))
}

// captionTypeFor is the size a pane's name is set at.
func captionTypeFor(place image.Rectangle) float64 {
	return math.Max(smallestType, math.Min(captionSize,
		captionSize*float64(place.Dx())/panelWidth))
}

// caption names a pane at its top left.
func caption(frame *image.RGBA, words string, place image.Rectangle) error {
	return walk.Caption(frame, words, place.Min.Add(image.Pt(captionInset,
		captionInset)), captionTypeFor(place))
}

// tonesOf is the panel's lamps' colours as light, for the drawings.
func tonesOf() (lampTones, error) {
	var made lampTones
	slots := []*[3]float64{&made.lasers, &made.strobes, &made.floods,
		&made.search, nil, &made.ship}
	for i, kind := range lampKinds {
		if slots[i] == nil {
			continue
		}
		colour, err := explain.ThemeColour(kind.role)
		if err != nil {
			return lampTones{}, err
		}
		*slots[i] = [3]float64{float64(colour.R) / 255,
			float64(colour.G) / 255, float64(colour.B) / 255}
	}
	return made, nil
}

// Rendered is whether the traced panes show the rendered set at a
// moment: when the show's light falls on the set, which is when the
// backlights flash or the ship's light is on; the drawing otherwise.
// Under the moon alone the set from these cameras traces nearly black
// (the moon, 2.3 degrees up, grazes the land), so a rendered pane then
// would show nothing true that the drawing does not.
func Rendered(state cues) bool {
	return state.floodLeft > shown || state.floodRight > shown ||
		state.overhead > shown
}

// edgeName writes a kind of fixture's name at the pane's nearer side,
// level with the fixture, with a short leader to it, so the middle of
// the pane is left to the light.
func edgeName(frame *image.RGBA, place image.Rectangle, each named,
	size float64, taken []image.Rectangle) (image.Rectangle, error) {
	wide, err := walk.Measure(each.name, size)
	if err != nil {
		return image.Rectangle{}, err
	}
	plate := image.Pt(wide+2*captionInset, int(size)+2*captionInset)
	left := captionInset
	if each.at.X >= place.Dx()/2 {
		left = place.Dx() - captionInset - plate.X
	}
	top := max(nameTop, min(place.Dy()-plate.Y-captionInset,
		each.at.Y-plate.Y/2))
	at := place.Min.Add(image.Pt(left, top))
	// clear of the names already written: stepped down past each
	for moved := true; moved; {
		moved = false
		for _, other := range taken {
			if (image.Rectangle{Min: at, Max: at.Add(plate)}).Overlaps(
				other) {
				at.Y, moved = other.Max.Y+captionInset/2, true
			}
		}
	}
	foot := image.Pt(at.X, at.Y+plate.Y/2)
	if left == captionInset {
		foot.X += plate.X
	}
	point := place.Min.Add(each.at)
	stroke(frame, float64(foot.X), float64(foot.Y), float64(point.X),
		float64(point.Y), leaderColour, 1)
	return image.Rectangle{Min: at, Max: at.Add(plate)},
		walk.Caption(frame, each.name, at, size)
}

// nameTop keeps fixture names below the pane's own name, in pixels;
// leaderColour is their leaders' light. Picked by the untraced film's
// author, 2026-11-12.
const nameTop = 70

// leaderColour is a pale grey line.
var leaderColour = [3]float64{0.6, 0.6, 0.62}
