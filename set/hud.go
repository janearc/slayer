package set

import (
	"fmt"
	"math"
	"slices"

	yaml "go.yaml.in/yaml/v3"
)

// Hud is what a set draws on its pictures' own plane: the colourway its
// roles come from, how many rows of cells a picture is laid out in, and
// its elements, in the order written, each over the ones before.
type Hud struct {
	theme    string
	rows     float64
	elements []HudElement
}

// Theme is the colourway file the hud's roles come from, its path from
// the set file's own directory.
func (made Hud) Theme() string { return made.theme }

// Rows is how many rows of cells a picture is laid out in.
func (made Hud) Rows() int { return int(made.rows) }

// Elements is the hud's elements, in the order written.
func (made Hud) Elements() []HudElement { return slices.Clone(made.elements) }

// HudElement is one thing a hud draws, by its name and its kind: text, a frame,
// a bar, or a label pinned to a thing on the set. a part a kind does not take
// is left as zero.
type HudElement struct {
	name, kind    string
	words, format string
	counted       bool
	value         Value
	box           [4]float64
	across, down  float64
	align         string
	ink, ground   string
	fill, empty   string
	of            string
	above         float64
	occluded      bool
	said          []string
}

// Value is a number that may change from frame to frame: from, until
// start seconds, then evenly to to over so many seconds. a number
// written alone is from and to both.
type Value struct{ From, To, Start, Over float64 }

// Name is the element's name.
func (made HudElement) Name() string { return made.name }

// Kind is text, frame, bar or label.
func (made HudElement) Kind() string { return made.kind }

// Words is what a text or a label says: its words, or, when counted, a
// value written out by the format, as fmt writes a float64.
func (made HudElement) Words() (words, format string, counted bool,
	count Value) {
	return made.words, made.format, made.counted, made.value
}

// Box is a frame's or a bar's part of the screen, its left, top, right
// and bottom as shares of its width and height.
func (made HudElement) Box() (left, top, right, bottom float64) {
	return made.box[0], made.box[1], made.box[2], made.box[3]
}

// At is where a text stands, as shares across and down, and its
// alignment there: left, centre or right.
func (made HudElement) At() (across, down float64, align string) {
	return made.across, made.down, made.align
}

// Roles is the theme's roles the element is drawn in: ink and ground
// for words, ink for a frame, fill and empty for a bar. no ground is
// clear.
func (made HudElement) Roles() (ink, ground, fill, empty string) {
	return made.ink, made.ground, made.fill, made.empty
}

// Value is a bar's value, from zero to one.
func (made HudElement) Value() Value { return made.value }

// Pin is the thing on the set a label is pinned to, by name, how many
// metres above its place, and whether the scene hides it.
func (made HudElement) Pin() (of string, above float64, occluded bool) {
	return made.of, made.above, made.occluded
}

// HudOf is the set's hud, and whether it has one.
func (made Set) HudOf() (Hud, bool) {
	if made.hud == nil {
		return Hud{}, false
	}
	return *made.hud, true
}

// hud reads the set's hud.
func (reading reader) hud(_, value *yaml.Node) error {
	made := &Hud{rows: 24}
	err := reading.keys(value, map[string]step{
		"theme":    reading.text(&made.theme),
		"rows":     reading.whole(&made.rows, 2, 500),
		"elements": reading.hudElements(made),
	}, nil)
	if err == nil && made.theme == "" {
		err = reading.refuse(value, "a hud says its theme, the "+
			"colourway its roles come from")
	}
	reading.set.hud = made
	return err
}

// hudElements reads each element of a hud, by its name.
func (reading reader) hudElements(into *Hud) step {
	return func(_, value *yaml.Node) error {
		return reading.each(value, func(name, block *yaml.Node) error {
			made := HudElement{name: name.Value, align: "left"}
			err := reading.keys(block, reading.hudKeys(&made), nil)
			if err == nil {
				made.said = keysOf(block)
				err = reading.fullySaid(block, made)
			}
			into.elements = append(into.elements, made)
			return err
		})
	}
}

// hudKeys is every key an element may say.
func (reading reader) hudKeys(made *HudElement) map[string]step {
	share := func(at *float64) step { return reading.number(at, 0, 1) }
	kinds := []string{"text", "frame", "bar", "label"}
	aligns := []string{"left", "centre", "right"}
	return map[string]step{
		"kind":     reading.word(&made.kind, kinds...),
		"words":    reading.text(&made.words),
		"count":    reading.counted(made),
		"format":   reading.text(&made.format),
		"value":    reading.value(&made.value),
		"left":     share(&made.box[0]),
		"top":      share(&made.box[1]),
		"right":    share(&made.box[2]),
		"bottom":   share(&made.box[3]),
		"across":   share(&made.across),
		"down":     share(&made.down),
		"align":    reading.word(&made.align, aligns...),
		"ink":      reading.text(&made.ink),
		"ground":   reading.text(&made.ground),
		"fill":     reading.text(&made.fill),
		"empty":    reading.text(&made.empty),
		"of":       reading.text(&made.of),
		"above":    reading.number(&made.above, -1e4, 1e4),
		"occluded": reading.yes(&made.occluded),
	}
}

// counted reads a text's count: the value it writes out.
func (reading reader) counted(made *HudElement) step {
	read := reading.value(&made.value)
	return func(name, value *yaml.Node) error {
		made.counted = true
		return read(name, value)
	}
}

// value reads a value: a number alone, or a block of from, to, start and
// over, the last two in seconds and from and to said.
func (reading reader) value(into *Value) step {
	return func(name, value *yaml.Node) error {
		if value.Kind == yaml.ScalarNode {
			read := reading.number(&into.From, -1e12, 1e12)
			err := read(name, value)
			into.To = into.From
			return err
		}
		got := Value{From: math.NaN(), To: math.NaN()}
		err := reading.keys(value, map[string]step{
			"from":  reading.number(&got.From, -1e12, 1e12),
			"to":    reading.number(&got.To, -1e12, 1e12),
			"start": reading.number(&got.Start, 0, 1e9),
			"over":  reading.number(&got.Over, 0, 1e9),
		}, nil)
		if err == nil && (math.IsNaN(got.From) || math.IsNaN(got.To)) {
			err = reading.refuse(value, "%s says its from and its "+
				"to", name.Value)
		}
		*into = got
		return err
	}
}

// needs is what each kind of element must say.
var needs = map[string][]string{
	"text":  {"ink", "across", "down"},
	"frame": {"ink", "left", "top", "right", "bottom"},
	"bar": {"fill", "empty", "value", "left", "top", "right",
		"bottom"},
	"label": {"ink", "of"},
}

// fullySaid refuses an element with no kind, one that leaves out what its
// kind must say, a text or a label with neither words nor a count, and
// a count with no format.
func (reading reader) fullySaid(block *yaml.Node, made HudElement) error {
	if made.kind == "" {
		return reading.refuse(block, "hud element %s says its kind",
			made.name)
	}
	for _, key := range needs[made.kind] {
		if !slices.Contains(made.said, key) {
			return reading.refuse(block, "a %s says its %v",
				made.kind, needs[made.kind])
		}
	}
	worded := made.kind == "text" || made.kind == "label"
	switch {
	case worded && (made.words == "") == !made.counted:
		return reading.refuse(block, "a %s says its words or its "+
			"count, one of them", made.kind)
	case made.counted && made.format == "":
		return reading.refuse(block, "a count says its format")
	}
	return nil
}

// keysOf is every key a block says.
func keysOf(block *yaml.Node) []string {
	var said []string
	for i := 0; i+1 < len(block.Content); i += 2 {
		said = append(said, block.Content[i].Value)
	}
	return said
}

// pinned refuses a hud's label pinned to a name the set does not have.
func (reading reader) pinned() error {
	if reading.set.hud == nil {
		return nil
	}
	for _, each := range reading.set.hud.elements {
		if each.kind != "label" || reading.set.named(each.of) {
			continue
		}
		return fmt.Errorf("%s: label %s is pinned to %q, and the set "+
			"has no asset by that name", reading.file, each.name,
			each.of)
	}
	return nil
}
