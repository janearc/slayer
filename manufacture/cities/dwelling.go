package cities

import (
	"fmt"

	"github.com/janearc/slayer/scene/thing"
)

// The kinds of balcony a household may have before its windows.
const (
	// balconyOpen is a slab with a rail round it, and what the household
	// keeps on it.
	balconyOpen = "open"
	// balconyEnclosed is a balcony glazed in, made part of the flat.
	balconyEnclosed = "enclosed"
)

// The things a household keeps on an open balcony.
const (
	pieceBike    = "bike"
	pieceChair   = "chair"
	piecePlant   = "plant"
	pieceLaundry = "laundry"
)

// keepsake is one thing kept on a balcony: what it is, the material of
// its body, and of its soft part, the leaves of a plant or the cloth on
// a rack. Keepsakes of one kind and size are one mesh placed many times.
type keepsake struct {
	Kind       string
	Body, Soft *thing.Material
}

// household is what one household of a building has chosen for the
// windows of its flat and for what stands before them. The windows of a
// dwelling share one. It is drawn at the dwelling scope, so two
// households differ, and the same seed draws the same household.
type household struct {
	// Blind is the material of a roller blind, if the household has one,
	// and Drop how much of the window's height it is let down, from the
	// top. Curtain is the cloth of a curtain instead, and Cover how much
	// of the window's width it hangs over.
	Blind, Curtain *thing.Material
	Drop, Cover    float64
	// Balcony is the kind of balcony before the windows, or nothing.
	Balcony string
	// Pieces are what stands on an open balcony, in order along it.
	Pieces []keepsake
	// Glass is the glazing of an enclosed balcony.
	Glass *thing.Material
	// Unit is an air-conditioning unit hung under the first window, if
	// the household has one.
	Unit *thing.Material
	// Replaced is the frame of the household's replaced window, the last
	// of its windows, if one has been replaced.
	Replaced *thing.Material
}

// taste is the option a household chose from an element's options, for
// the thing of a name, at the dwelling's scope.
func (drawing *drawer) taste(name, salt string, at place) option {
	return drawing.choose(name, scopeDwelling, salt, drawing.options(name),
		at, nil)
}

// household draws the household of a dwelling at a place. It has no
// balcony on the ground storey.
func (drawing *drawer) household(at place) *household {
	home := &household{}
	switch {
	case drawing.chance("blind-share", at):
		home.Blind = paint("blind", drawing.taste("blind-colour", "", at))
		home.Drop = drawing.number("blind-drop", at)
	case drawing.chance("household-curtain", at):
		home.Curtain = paint("curtain", drawing.taste("curtain-colour",
			"home", at))
		home.Cover = drawing.number("household-cover", at)
	}
	deck := drawing.onDecks[place{Bay: at.Bay, Storey: at.Storey}]
	if at.Storey > 0 && !deck && drawing.chance("balcony-share", at) {
		home.balcony(drawing, at)
	}
	if home.Balcony == "" && drawing.chance("household-aircon", at) {
		home.Unit = paint("unit", drawing.taste("unit-colour", "home", at))
	}
	if drawing.chance("replaced-window", at) {
		home.Replaced = paint("frame", drawing.taste("replaced-frame", "", at))
	}
	return home
}

// balcony gives the household the balcony its taste chooses, and, if it
// is open, what it keeps there.
func (home *household) balcony(drawing *drawer, at place) {
	home.Balcony = drawing.taste("balcony-kind", "", at).Name
	if home.Balcony == balconyEnclosed {
		glass := drawing.glazing()
		home.Glass = glass
		return
	}
	for index := range drawing.count("clutter-count", at) {
		kind := drawing.taste("clutter-item", fmt.Sprint(index), at).Name
		home.Pieces = append(home.Pieces, drawing.piece(kind, index, at))
	}
}

// piece is a thing of a kind a household keeps on its balcony, in the
// materials its taste chooses; the index is how many came before it.
func (drawing *drawer) piece(kind string, index int, at place) keepsake {
	salt := fmt.Sprint(kind, index)
	piece := keepsake{Kind: kind}
	switch kind {
	case piecePlant:
		piece.Body = paint("pot", drawing.taste("pot-colour", salt, at))
		piece.Soft = paint("foliage", foliage)
	case pieceBike:
		piece.Body = paint("bike", drawing.taste("bike-colour", salt, at))
	case pieceChair:
		piece.Body = paint("chair", drawing.taste("chair-colour", salt, at))
	case pieceLaundry:
		piece.Body = paint("rack", drawing.taste("bike-colour", salt, at))
		piece.Soft = paint("cloth", drawing.taste("cloth-colour", salt, at))
	}
	return piece
}

// dwellings are the windows of a storey as households have them: as many
// windows as the culture draws, split into dwellings of the number the
// culture draws for the storey, each dwelling's windows dressed by its
// household's choices.
func (drawing *drawer) dwellings(at place) []facadeOpening {
	count := drawing.count("window-count", at)
	per := max(drawing.count("dwelling-windows", at), 1)
	windows := make([]facadeOpening, count)
	var home *household
	for index := range windows {
		at.Opening, at.Dwelling = index, index/per
		if index%per == 0 {
			home = drawing.household(at)
		}
		windows[index] = drawing.window(at)
		windows[index].Home = home
		home.dress(&windows[index], index%per == 0, index%per == per-1 ||
			index == count-1)
	}
	return windows
}

// dress gives a window what its household chose: its curtain or blind,
// the air-conditioning unit under the first window, and the replaced
// frame in the last.
func (home *household) dress(opening *facadeOpening, first, last bool) {
	opening.Curtain, opening.Cover = home.Curtain, home.Cover
	if home.Blind != nil {
		opening.Curtain, opening.Cover = nil, 0
	}
	if first && home.Unit != nil {
		opening.Extra = extra{Under: "aircon", Unit: home.Unit}
	}
	if last && home.Replaced != nil {
		opening.Frame = home.Replaced
	}
}
