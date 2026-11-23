package cities

import (
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// wallLampHeat is the colour temperature of a lamp beside a door, in
// kelvin: a tungsten bulb, as the lamps of a set are.
const wallLampHeat = 2700

// foliage is the green of the leaves in a flower box, an option as the
// data would write it. A choice of the detail study, by eye.
var foliage = option{Name: "foliage", Colour: []float64{0.04, 0.18, 0.05}}

// extra is what has been added to one opening of a facade, at the demo
// level: what stands under a window, a door's awning and the lamp beside
// it. The materials are those the culture drew for it.
type extra struct {
	// Under is what is under a window: nothing, a flower-box, a balcony
	// or an aircon.
	Under string
	// Planter, Bloom, Leaf and Unit are the materials of a flower box, its
	// flowers, its leaves, and an air-conditioning unit.
	Planter, Bloom, Leaf, Unit *thing.Material
	// Awning, if given, is the cloth of an awning over a door, and Lamp,
	// if given, the glowing box of a lamp beside it.
	Awning, Lamp *thing.Material
	// Portal says a door has a stone surround and an entry panel: it is
	// the entrance to a stair core.
	Portal bool
}

// fixtures is what has been added to a whole bay of a facade, at the demo
// level: each flag says it has the thing, and the materials are those the
// culture drew.
type fixtures struct {
	Dish, Pipe, Cable, Laundry, Sign bool
	Board, Receiver, Stone           *thing.Material
	Cloth                            []*thing.Material
}

// fitted gives a window at the demo level what the culture says is under
// it. A balcony on the ground floor is a flower box.
func (drawing *drawer) fitted(opening *facadeOpening, at place) {
	under := drawing.pick("under-window", drawing.options("under-window"),
		at).Name
	if under == "balcony" && at.Storey == 0 {
		under = "flower-box"
	}
	extra := extra{Under: under}
	switch under {
	case "flower-box":
		extra.Planter = paint("planter", drawing.pick("planter-colour",
			drawing.options("planter-colour"), at))
		extra.Bloom = paint("bloom", drawing.pick("bloom-colour",
			drawing.options("bloom-colour"), at))
		extra.Leaf = paint("foliage", foliage)
	case "aircon":
		extra.Unit = paint("unit", drawing.pick("unit-colour",
			drawing.options("unit-colour"), at))
	}
	opening.Extra = extra
}

// fittedDoor gives a door at the demo level its awning and its lamp, if
// the culture says it has them.
func (drawing *drawer) fittedDoor(opening *facadeOpening, at place) {
	if drawing.chance("awning", at) {
		opening.Extra.Awning = paint("awning", drawing.pick("awning-colour",
			drawing.options("awning-colour"), at))
	}
	if drawing.chance("wall-lamp", at) {
		glow := swatches.Light(swatch.Planckian(wallLampHeat)).Scale(
			drawing.fixed("window-glow"))
		opening.Extra.Lamp = &thing.Material{Name: "cities wall lamp",
			Colour: maths.RGB(1, 1, 1), Glow: glow.Scale(
				drawing.fixed("lamp-glow"))}
	}
}

// fixtures draws what a bay has added to it as a whole.
func (drawing *drawer) fixtures(at place) fixtures {
	fit := fixtures{Dish: drawing.chance("dish", at),
		Pipe: drawing.chance("pipe", at), Cable: drawing.chance("cable", at),
		Laundry: drawing.chance("laundry", at),
		Sign:    drawing.chance("shop-sign", at)}
	fit.Board = paint("sign", drawing.pick("sign-colour",
		drawing.options("sign-colour"), at))
	fit.Stone = paint("plinth", drawing.pick("plinth-colour",
		drawing.options("plinth-colour"), at))
	fit.Receiver = paint("dish", drawing.pick("unit-colour",
		drawing.options("unit-colour"), at))
	for _, cloth := range drawing.options("cloth-colour") {
		fit.Cloth = append(fit.Cloth, paint("cloth", cloth))
	}
	return fit
}
