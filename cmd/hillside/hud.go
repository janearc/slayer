package main

import (
	"fmt"
	"image"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// overlay is a set's hud made ready to draw: the hud, the theme its
// roles come from, and how many rows of cells a picture is laid out in.
type overlay struct {
	made  hud.Hud
	theme hud.Theme
	rows  int
}

// overlaid is the set's hud, or nil when it has none: its theme read
// from the set file's own directory, and each label pinned where the
// thing it names was stood. it is made once everything is stood.
func (stage *staging) overlaid(read set.Set) (*overlay, error) {
	written, has := read.HudOf()
	if !has {
		return nil, nil
	}
	path := written.Theme()
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(stage.given.set), path)
	}
	theme, err := hud.ReadTheme(path)
	if err != nil {
		return nil, err
	}
	made := &overlay{theme: theme, rows: written.Rows()}
	for _, each := range written.Elements() {
		element, err := stage.element(each)
		if err != nil {
			name := each.Name()
			return nil, fmt.Errorf("hud element %s: %w", name, err)
		}
		made.made.Elements = append(made.made.Elements, element)
	}
	return made, nil
}

// aligned is each way words are aligned, by the word a set says it in.
var aligned = map[string]hud.Align{"left": hud.Left, "centre": hud.Centre,
	"right": hud.Right}

// element is one element of a set's hud as the hud draws it.
func (stage *staging) element(each set.HudElement) (hud.Element, error) {
	ink, ground, fill, empty := each.Roles()
	left, top, right, bottom := each.Box()
	box := hud.Box{Left: left, Top: top, Right: right, Bottom: bottom}
	across, down, align := each.At()
	switch each.Kind() {
	case "frame":
		return hud.Frame{Box: box, Ink: ink}, nil
	case "bar":
		return hud.Bar{Box: box, Value: gauged(each.Value()),
			Fill: fill, Empty: empty}, nil
	case "text":
		at := hud.Point{Across: across, Down: down}
		return hud.Text{Words: worded(each), At: at,
			Align: aligned[align], Ink: ink, Ground: ground}, nil
	}
	of, above, occluded := each.Pin()
	place, found := stage.places[of]
	if !found {
		return nil, fmt.Errorf("%q is not stood on the ground", of)
	}
	return hud.Label{Point: place.Add(maths.V(0, above, 0)),
		Words: worded(each), Align: aligned[align], Ink: ink,
		Ground: ground, Occluded: occluded}, nil
}

// gauged is a set's value as a gauge: its from until its start, then
// evenly to its to over its seconds.
func gauged(value set.Value) hud.Gauge {
	second := float64(time.Second)
	return hud.Ramp{From: value.From, To: value.To,
		Start: time.Duration(value.Start * second),
		Over:  time.Duration(value.Over * second)}
}

// worded is what a text or a label says: its words, or its count
// written out by its format.
func worded(each set.HudElement) hud.Wording {
	words, format, counted, count := each.Words()
	if counted {
		return hud.Count{Of: gauged(count), Format: format}
	}
	return hud.Words(words)
}

// overdraw draws the set's hud, if it has one, on its developed picture,
// as the picture was seen: through its eye and lens, at its moment, the
// world hiding the labels that ask.
func overdraw(img *image.RGBA, given asked, world *thing.World,
	across float64) error {
	if given.overlay == nil {
		return nil
	}
	view := hud.View{Eye: eyeFor(given, across), Sensor: lensFor(given),
		Width: given.width, Height: given.height,
		Blocker: hud.Seen(world, given.at)}
	return given.overlay.made.Picture(img, given.overlay.rows, given.at,
		given.overlay.theme, view)
}
