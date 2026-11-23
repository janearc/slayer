package cities

import (
	"github.com/janearc/slayer/maths"
)

// pierCulture is the culture a pier is drawn from.
const pierCulture = "pier"

// Pier is a wharf pier: a deck on rows of piles reaching out over water,
// with a long low shed standing on it and a strip of deck clear on each
// side. It stands with its origin at the middle of its root, at the height
// of the sea, so that its deck is over the sea by the culture's deck
// height and its piles go below. It runs along positive z, from nought to
// its length, and is as wide as the culture draws, centred on x.
func Pier(seed Seed) (Design, error) {
	drawing, err := newDrawer(pierCulture, seed)
	if err != nil {
		return Design{}, err
	}
	at := place{}
	length, width := drawing.number("length", at), drawing.number("width", at)
	height := drawing.fixed("deck-height")
	deck := paint("deck", drawing.pick("deck-colour",
		drawing.options("deck-colour"), at))
	parts := []placedElement{put("deck", Box{width, drawing.fixed("deck-thickness"),
		length}, deck, maths.V(-width/half, height-
		drawing.fixed("deck-thickness"), 0))}
	parts = append(parts, pierPiles(drawing, length, width, height)...)
	parts = append(parts, pierShed(drawing, length, width, height)...)
	if drawing.trouble != nil {
		return Design{}, drawing.trouble
	}
	return Design{Name: "pier", Elements: parts}, nil
}

// pierPiles are the rows of piles under a deck, from below the sea up to
// the deck's underside, a row at each edge and the rest between, a pile
// every pitch along each.
func pierPiles(drawing *drawer, length, width, height float64) []placedElement {
	pile, depth := drawing.fixed("pile-width"), drawing.fixed("pile-depth")
	look := paint("pile", drawing.pick("pile-colour",
		drawing.options("pile-colour"), place{}))
	rows := drawing.count("pile-rows", place{})
	pitch := drawing.fixed("pile-pitch")
	var piles []placedElement
	for row := range rows {
		across := -width/half + pile/half
		if rows > 1 {
			across += float64(row) * (width - pile) / float64(rows-1)
		}
		for along := pitch / half; along < length; along += pitch {
			piles = append(piles, put("pile", Box{pile, depth + height -
				drawing.fixed("deck-thickness"), pile}, look,
				maths.V(across-pile/half, -depth, along-pile/half)))
		}
	}
	return piles
}

// pierShed is the shed on a deck: a box with a flat roof, a share of the
// pier's length, set a share of the way out, with the deck's apron clear
// on each side.
func pierShed(drawing *drawer, length, width, height float64) []placedElement {
	at := place{}
	apron := drawing.fixed("shed-apron")
	run := length * drawing.number("shed-share", at)
	start := length * drawing.number("shed-start", at)
	shape := massShape{width: width - 2*apron, depth: run}
	wall := wall(drawing.pick("palette", drawing.options("palette"), at),
		drawing.pick("cladding", drawing.options("cladding"), at))
	body := drawing.number("shed-height", at)
	roof, err := drawing.roof(at, shape, wall).parts()
	drawing.fail(err)
	origin := maths.V(-width/half+apron, height, start)
	parts := []placedElement{put("shed", Box{shape.width, body, run}, wall,
		maths.Vec{})}
	parts = append(parts, within(roof, standingAt(maths.V(0, body, 0)))...)
	return within(parts, standingAt(origin))
}
