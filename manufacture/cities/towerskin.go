package cities

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// Skin is what the faces of a tower are made of. The curtain is the plain
// dress a tower has without one: a grid of mullions, spandrels and glass.
// The others each replace that grid with a wall of their own. A skin is
// drawn once for a tower, from the culture and the irregularity, and a
// set may pin it.
type Skin string

// The skins a tower may wear.
const (
	// skinCurtain is the plain grid of mullions, spandrels and glass.
	skinCurtain Skin = "curtain"
	// skinMirror is a mirror glass curtain wall: unitized panels, caps on
	// the mullions, opaque bands at each slab, panels a little out of true.
	skinMirror Skin = "mirror"
	// skinGranite is stone, polished or flamed, with windows punched in
	// it or ribbon windows between bands of it.
	skinGranite Skin = "granite"
	// skinConcrete is board-marked poured concrete: slabs expressed on the
	// face, piers standing out, windows set deep between them.
	skinConcrete Skin = "concrete"
	// skinGothic is stone buttresses unbroken to the top, pinnacles on
	// them, tracery under the crown, and figures on the ledges.
	skinGothic Skin = "gothic"
	// skinCantilever is floor plates thrown out past the glass, and a
	// cable net across it.
	skinCantilever Skin = "cantilever"
	// skinVaporwave is pastel bands that run from one colour at the foot to
	// another at the top, marble, and glass block.
	skinVaporwave Skin = "vaporwave"
	// skinStainless is stainless steel, with a ribbon of glass and a
	// horizontal fin at the head of every window.
	skinStainless Skin = "stainless"
)

// SkinNames are the skins there are.
func SkinNames() []Skin {
	return []Skin{skinCurtain, skinMirror, skinGranite, skinConcrete,
		skinGothic, skinCantilever, skinVaporwave, skinStainless}
}

// Use is what a segment of a shaft is for, which says how its windows are
// lit at night: an office a floor at a time, cool and even; a home a
// window at a time, warm and uneven, with curtains.
type Use string

// The uses a segment may have.
const (
	// useOffice lights whole floors, in a cool white.
	useOffice Use = "office"
	// useHome lights single windows, in a warm light, some curtained.
	useHome Use = "home"
)

// UseNames are the uses there are.
func UseNames() []Use { return []Use{useOffice, useHome} }

// The two ways a segment of granite is windowed.
const (
	// stylePunched is windows cut into a wall of stone, between piers.
	stylePunched = "punched"
	// styleRibbon is a ribbon of glass between bands of stone.
	styleRibbon = "ribbon"
)

// Tree is a tree a planted tower holds: the kind the catalogue calls it,
// the seed that makes it one of a few of its kind, where it stands in the
// tower's own space and how tall it is to be made. The tower draws where
// and what; whoever has the catalogue grows it.
type Tree struct {
	Kind   string
	Seed   uint64
	At     maths.Vec
	Height float64
}

// Trees are the trees a planted tower holds, after its design is drawn:
// those of each segment as it was last drawn, the lowest segment first,
// and in a segment drawn in bands the lowest band first. A tower that is
// not planted holds none, and nor does a segment drawn at the hull.
func (tower *Tower) Trees() []Tree {
	var all []Tree
	for _, bands := range tower.grown {
		for _, band := range slices.Sorted(maps.Keys(bands)) {
			all = append(all, bands[band]...)
		}
	}
	return all
}

// dressed is the skin, the layers over it and the pinned measures of a
// tower, drawn or taken from the pins. It refuses a skin, a use, a style
// or a colour there is not.
func (tower *Tower) dressed(pins Pins) error {
	drawing, reach := tower.drawing, tower.Irregularity
	if pins.Skin != "" && !slices.Contains(SkinNames(), pins.Skin) {
		return fmt.Errorf("cities: no skin %q, only %v", pins.Skin,
			SkinNames())
	}
	if pins.Use != "" && !slices.Contains(UseNames(), pins.Use) {
		return fmt.Errorf("cities: no use %q, only %v", pins.Use, UseNames())
	}
	if pins.Style != "" && pins.Style != stylePunched &&
		pins.Style != styleRibbon {
		return fmt.Errorf("cities: no style %q, only %s and %s", pins.Style,
			stylePunched, styleRibbon)
	}
	tower.Skin = tower.skinOf(pins)
	tower.Cyber = drawing.gate("cyber", place{}, reach, pins.Cyber)
	tower.Planted = drawing.gate("planted", place{}, reach, pins.Planted)
	tower.colour, tower.style, tower.width = pins.Colour, pins.Style,
		pins.Width
	if _, err := tower.paletteOf(); err != nil {
		return err
	}
	return drawing.trouble
}

// skinOf is the pinned skin, or one drawn if the gate lets the tower
// stray from the plain curtain.
func (tower *Tower) skinOf(pins Pins) Skin {
	drawing := tower.drawing
	if pins.Skin != "" {
		return pins.Skin
	}
	if !drawing.gate("skin-change", place{}, tower.Irregularity, nil) {
		return skinCurtain
	}
	return Skin(drawing.pick("skin", drawing.options("skin"), place{}).Name)
}

// paletteElements are the elements that name the main colour of a skin.
var paletteElements = map[Skin]string{skinMirror: "mirror-colour",
	skinGranite: "granite-colour", skinConcrete: "concrete-colour",
	skinGothic: "gothic-stone", skinVaporwave: "vapor-low"}

// paletteOf is the main colour of the skin: the option the pin names, or
// the one the draw gives. A skin with no colour to choose has none, and a
// pin for it is not read. A pin that names no option is refused.
func (tower *Tower) paletteOf() (option, error) {
	name, found := paletteElements[tower.Skin]
	if !found {
		return option{}, nil
	}
	drawing := tower.drawing
	options := drawing.options(name)
	if tower.colour == "" {
		return drawing.pick(name, options, place{}), drawing.trouble
	}
	for _, one := range options {
		if one.Name == tower.colour {
			return one, nil
		}
	}
	return option{}, fmt.Errorf("cities: no %s called %q, only %v", name,
		tower.colour, namesOf(options))
}

// used draws what each segment of a shaft is used for, and how a granite
// segment is windowed, unless a pin says.
func (tower *Tower) used(shaft *shaft, at place, pins Pins) {
	drawing := tower.drawing
	for index := range shaft.Segments {
		at.Storey = index
		segment := &shaft.Segments[index]
		segment.Use = Use(drawing.pick("use", drawing.options("use"),
			at).Name)
		if pins.Use != "" {
			segment.Use = pins.Use
		}
		segment.Style = drawing.pick("granite-style",
			drawing.options("granite-style"), at).Name
		if pins.Style != "" {
			segment.Style = pins.Style
		}
	}
}

// spiral is a shaft cut into blocks of a few storeys, each turned a few
// degrees on the one below, so its floors rotate as it rises.
func (tower *Tower) spiral(storeys int) []segment {
	drawing := tower.drawing
	block := int(drawing.fixed("spiral-storeys"))
	turn := drawing.fixed("spiral-turn")
	shape := tower.baseShape(tower.Plan)
	count := max(1, storeys/block)
	made := make([]segment, count)
	left := storeys
	for index := range made {
		each := storeys / count
		if index == count-1 {
			each = left
		}
		made[index] = segment{Bottom: shape, Top: shape, Storeys: each,
			Turn: turn}
		left -= each
	}
	return made
}

// skinLooks are the materials a skin is made of, drawn once for a tower.
// Which of them a tower has depends on its skin: the rest are nil.
type skinLooks struct {
	panel, spandrel, cap, board, plate, cable, stone, figure *thing.Material
	// window is the opaque dark glass set in a wall of stone, steel or
	// concrete.
	window *thing.Material
	// steps are the bands of a gradient, from the foot to the top, and
	// blocks the glass block that goes with each.
	steps, blocks []*thing.Material
}

// skinned draws the materials of the tower's skin into the looks, and
// sets the material its body is made of.
func (tower *Tower) skinned(made *towerLooks) error {
	drawing := tower.drawing
	main, err := tower.paletteOf()
	if err != nil {
		return err
	}
	pick := func(name string) *thing.Material {
		return paint(name, drawing.pick(name, drawing.options(name),
			place{}))
	}
	made.body = made.glass
	switch tower.Skin {
	case skinMirror:
		made.body = paint("mirror", main)
		made.skin = skinLooks{spandrel: pick("mirror-spandrel-colour"),
			cap: pick("mirror-cap-colour"), panel: made.body}
	case skinGranite:
		made.body = tower.granite(main)
		made.skin = skinLooks{stone: made.body,
			window: pick("window-dark")}
	case skinStainless:
		made.body = pick("stainless-colour")
		made.skin = skinLooks{stone: made.body,
			window: pick("window-dark")}
	case skinConcrete:
		made.body = paint("concrete", main)
		made.skin = skinLooks{stone: made.body,
			board:  pick("concrete-board-colour"),
			window: pick("window-dark")}
	case skinGothic:
		made.body = paint("gothic", main)
		made.skin = skinLooks{stone: made.body, figure: made.body,
			window: pick("window-dark")}
	case skinCantilever:
		made.skin = skinLooks{plate: pick("cantilever-colour"),
			cable: pick("cantilever-cable-colour")}
	case skinVaporwave:
		made.body = paint("marble", drawing.pick("vapor-marble",
			drawing.options("vapor-marble"), place{}))
		made.skin = tower.pastels(main, made.body)
	}
	return drawing.trouble
}

// granite is the stone of a granite tower: the colour it was cut in, in
// the finish it was given, named by both so that every ask is the same.
func (tower *Tower) granite(colour option) *thing.Material {
	drawing := tower.drawing
	finish := drawing.pick("granite-finish",
		drawing.options("granite-finish"), place{})
	return &thing.Material{Name: fmt.Sprintf("cities granite %s %s",
		colour.Name, finish.Name), Colour: colourOf(colour),
		Gloss: finish.Gloss}
}

// pastels are the bands and the glass block of a vaporwave tower, a few
// steps from its low pastel to its high one, each a mix of the two.
func (tower *Tower) pastels(low option, marble *thing.Material) skinLooks {
	drawing := tower.drawing
	high := drawing.pick("vapor-high", drawing.options("vapor-high"),
		place{})
	steps := drawing.count("vapor-steps", place{})
	glow, tint := drawing.fixed("vapor-glow"), drawing.fixed("vapor-pane")
	looks := skinLooks{stone: marble}
	for step := range steps {
		share := float64(step) / float64(max(1, steps-1))
		var mixed [3]float64
		for channel := range mixed {
			mixed[channel] = low.Colour[channel] + (high.Colour[channel]-
				low.Colour[channel])*share
		}
		colour := maths.RGB(mixed[0], mixed[1], mixed[2])
		name := fmt.Sprintf("%s %s %d", low.Name, high.Name, step)
		looks.steps = append(looks.steps, &thing.Material{
			Name: "cities vapor band " + name, Colour: colour,
			Glow: colour.Scale(glow), Gloss: marble.Gloss / 2})
		looks.blocks = append(looks.blocks, &thing.Material{
			Name: "cities vapor block " + name, Colour: colour.Scale(tint),
			Gloss: marble.Gloss})
	}
	return looks
}

// wet is a material as it is when wet: the same, with the gloss of a
// wet surface if its own is less.
func (tower *Tower) wet(look *thing.Material) *thing.Material {
	if look == nil {
		return nil
	}
	wetted := *look
	wetted.Name += " wet"
	wetted.Gloss = math.Max(look.Gloss,
		tower.drawing.fixed("cyber-wet-gloss"))
	if tower.drawing.level == demoLevel {
		return tower.grimed(&wetted)
	}
	return &wetted
}

// wetted is the looks with every surface of the building wet, which is
// the cyberpunk layer's wet look. The glass and the lights stay as they
// are.
func (tower *Tower) wetted(made towerLooks) towerLooks {
	made.grid, made.plant = tower.wet(made.grid), tower.wet(made.plant)
	made.podium, made.panel = tower.wet(made.podium), tower.wet(made.panel)
	if made.body != made.glass {
		made.body = tower.wet(made.body)
	}
	for _, each := range []**thing.Material{&made.skin.stone,
		&made.skin.plate, &made.skin.spandrel, &made.skin.cap,
		&made.skin.figure, &made.lobby} {
		*each = tower.wet(*each)
	}
	return made
}

// lamp is the material of a window lit at a colour temperature, brighter
// or dimmer by the share given: the glazing, and the light of that
// temperature glowing from it. Windows lit alike have one material, so a
// tower has a few and not a thousand.
func (drawing *drawer) lamp(heat, share float64) *thing.Material {
	dark, err := material(roleGlazing)
	if err != nil {
		drawing.fail(err)
		return nil
	}
	lit := *dark
	lit.Name = fmt.Sprintf("cities lit window %.0fK by %.2f", heat, share)
	lit.Glow = swatches.Light(swatch.Planckian(heat)).Scale(
		drawing.fixed("window-glow") * share)
	return &lit
}

// spot is a draw from nought to one for a place, which is a place, a size
// or whether a cell is lit. The salt makes the stream of one use another's.
func (drawing *drawer) spot(salt int, at place) float64 {
	at.Row = salt
	return drawing.number("cyber-spot", at)
}
