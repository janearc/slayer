package cities

import "github.com/janearc/slayer/maths"

// flat is the windows of one dwelling along a storey: its household,
// and where along the face its first window's left edge and its last
// window's right edge are.
type flat struct {
	home        *household
	left, right float64
}

// flats are the dwellings of a storey, each the run of windows that
// share a household, in order along the face.
func flats(openings []facadeOpening) []flat {
	var runs []flat
	for _, opening := range openings {
		if opening.Home == nil || opening.Door {
			continue
		}
		right := opening.Left + opening.Width
		last := len(runs) - 1
		if last >= 0 && runs[last].home == opening.Home {
			runs[last].right = right
			continue
		}
		runs = append(runs, flat{opening.Home, opening.Left, right})
	}
	return runs
}

// homes is what the households of a storey have chosen, drawn in the
// storey's frame, whose floor is at nought: each window's blind, and each
// dwelling's balcony with what stands on it.
func (facade facade) homes(openings []facadeOpening) []placedElement {
	var parts []placedElement
	for _, opening := range openings {
		if opening.Home != nil && opening.Home.Blind != nil {
			parts = append(parts, facade.blind(opening)...)
		}
	}
	for _, run := range flats(openings) {
		parts = append(parts, facade.balconyOf(run)...)
	}
	return parts
}

// blind is a roller blind just behind the glass, let down from the top of
// the window by the share of its height the household chose.
func (facade facade) blind(opening facadeOpening) []placedElement {
	bar := facade.FrameWidth
	inner := opening.Height - 2*bar
	drop := opening.Home.Drop * inner
	thick := facade.Finish.CurtainThickness
	return []placedElement{put("blind", Box{opening.Width - 2*bar, drop, thick},
		opening.Home.Blind, maths.V(opening.Left+bar,
			opening.Sill+bar+inner-drop, -facade.Recess-thick))}
}

// balconyOf is the balcony before a dwelling, if it has one: reaching
// past its windows by the culture's side, and no further than the depth
// the culture gives it.
func (facade facade) balconyOf(run flat) []placedElement {
	side := facade.measured("home-balcony-side")
	deep := facade.measured("home-balcony-depth")
	left, wide := run.left-side, run.right-run.left+2*side
	switch run.home.Balcony {
	case balconyOpen:
		return facade.openBalcony(run.home, left, wide, deep)
	case balconyEnclosed:
		return facade.enclosedBalcony(run.home, left, wide, deep)
	}
	return nil
}

// openBalcony is a slab under a rail, and what the household keeps on it.
func (facade facade) openBalcony(home *household, left, wide,
	deep float64) []placedElement {
	slab := facade.measured("balcony-slab")
	rail := facade.measured("rail-top")
	parts := []placedElement{put("balcony", Box{wide, slab, deep}, facade.Wall,
		maths.V(left, -slab, 0))}
	parts = append(parts, facade.rails(left, 0, wide, deep, rail)...)
	parts = append(parts, facade.uprights(left, 0, wide, deep, rail)...)
	return append(parts, facade.kept(home.Pieces, left, wide, deep)...)
}

// enclosedBalcony is a balcony glazed in: the slab, a roof slab over it,
// a post at each corner, a mid rail, and glass across the front.
func (facade facade) enclosedBalcony(home *household, left, wide,
	deep float64) []placedElement {
	slab := facade.measured("balcony-slab")
	high := facade.measured("enclosed-height")
	post := facade.measured("enclosed-bar")
	thin := facade.measured("glass-thickness")
	rail := facade.measured("rail-top")
	parts := []placedElement{
		put("balcony", Box{wide, slab, deep}, facade.Wall,
			maths.V(left, -slab, 0)),
		put("balcony roof", Box{wide, slab, deep}, facade.Wall,
			maths.V(left, high, 0)),
		put("enclosure glass", Box{wide - 2*post, high - slab, thin},
			home.Glass, maths.V(left+post, 0, deep-thin)),
		put("enclosure rail", Box{wide, post, post}, facade.Iron,
			maths.V(left, rail, deep-post)),
	}
	for _, along := range []float64{0, wide - post} {
		for _, back := range []float64{0, deep - post} {
			parts = append(parts, put("enclosure post", Box{post, high, post},
				facade.Iron, maths.V(left+along, 0, back)))
		}
	}
	return parts
}

// kept is the pieces a household keeps on its balcony, spread along it
// with the same gap before the first, between each and after the last.
// Pieces are dropped from the end until the gap is wide enough to walk.
func (facade facade) kept(pieces []keepsake, left, wide,
	deep float64) []placedElement {
	reach := facade.measured("home-balcony-side") / half
	for ; len(pieces) > 0; pieces = pieces[:len(pieces)-1] {
		widths := make([]float64, len(pieces))
		for index, piece := range pieces {
			widths[index] = facade.pieceWidth(piece)
		}
		lefts, gap := spread(left+reach, left+wide-reach, widths)
		if gap < facade.measured("piece-gap") {
			continue
		}
		var parts []placedElement
		back := deep - facade.measured("piece-gap") - facade.measured(
			"piece-reach")
		for index, piece := range pieces {
			parts = append(parts, facade.piece(piece, lefts[index], back)...)
		}
		return parts
	}
	return nil
}

// pieceWidth is how much of a balcony's length a piece takes.
func (facade facade) pieceWidth(piece keepsake) float64 {
	return facade.measured(map[string]string{piecePlant: "plant-width",
		pieceBike: "bike-length", pieceChair: "chair-width",
		pieceLaundry: "rack-length"}[piece.Kind])
}

// piece is one thing kept on a balcony, standing on its slab at a place
// along it and a distance out from the wall.
func (facade facade) piece(piece keepsake, along, out float64) []placedElement {
	switch piece.Kind {
	case piecePlant:
		return facade.plant(piece, along, out)
	case pieceBike:
		return facade.bike(piece, along, out)
	case pieceChair:
		return facade.chair(piece, along, out)
	}
	return facade.rack(piece, along, out)
}

// plant is a pot with a block of leaves over it.
func (facade facade) plant(piece keepsake, along, out float64) []placedElement {
	wide, pot := facade.measured("plant-width"), facade.measured("pot-width")
	high := facade.measured("pot-height")
	return []placedElement{
		put("pot", Box{pot, high, pot}, piece.Body,
			maths.V(along+(wide-pot)/half, 0, out)),
		put("foliage", Box{wide, facade.measured("foliage-height"), wide},
			piece.Soft, maths.V(along, high, out-(wide-pot)/half)),
	}
}

// bike is a bicycle standing along the rail: two wheels, the bar between
// them, a post and a seat, and the fork's post and handlebar.
func (facade facade) bike(piece keepsake, along, out float64) []placedElement {
	long, wheel := facade.measured("bike-length"), facade.measured(
		"wheel-size")
	tyre, bar := facade.measured("tyre-width"), facade.measured("bike-bar")
	seat := facade.measured("bike-seat")
	hub := wheel / half
	return []placedElement{
		put("wheel", Box{wheel, wheel, tyre}, piece.Body,
			maths.V(along, 0, out)),
		put("wheel", Box{wheel, wheel, tyre}, piece.Body,
			maths.V(along+long-wheel, 0, out)),
		put("bike bar", Box{long - wheel, bar, bar}, piece.Body,
			maths.V(along+hub, hub, out)),
		put("bike post", Box{bar, seat - hub, bar}, piece.Body,
			maths.V(along+long*bikeSeatShare, hub, out)),
		put("bike seat", Box{wheel / half, bar, wheel / half}, piece.Body,
			maths.V(along+long*bikeSeatShare-bar, seat, out-bar)),
		put("bike fork", Box{bar, seat - hub, bar}, piece.Body,
			maths.V(along+long-hub, hub, out)),
		put("handlebar", Box{bar, bar, wheel / half}, piece.Body,
			maths.V(along+long-hub, seat, out-bar)),
	}
}

// bikeSeatShare is how far along a bicycle its seat post is, as a share
// of its length. A choice, by eye.
const bikeSeatShare = 0.3

// chair is a seat on four legs with a back.
func (facade facade) chair(piece keepsake, along, out float64) []placedElement {
	wide, seat := facade.measured("chair-width"), facade.measured(
		"chair-seat")
	leg, thin := facade.measured("chair-leg"), facade.measured("chair-leg")
	parts := []placedElement{
		put("chair seat", Box{wide, thin, wide}, piece.Body,
			maths.V(along, seat, out)),
		put("chair back", Box{wide, facade.measured("chair-back"), thin},
			piece.Body, maths.V(along, seat+thin, out)),
	}
	for _, across := range []float64{0, wide - leg} {
		for _, back := range []float64{0, wide - leg} {
			parts = append(parts, put("chair leg", Box{leg, seat, leg},
				piece.Body, maths.V(along+across, 0, out+back)))
		}
	}
	return parts
}

// rack is a drying rack, two posts and a top bar, with two cloths hung
// over the bar.
func (facade facade) rack(piece keepsake, along, out float64) []placedElement {
	long, high := facade.measured("rack-length"), facade.measured(
		"rack-height")
	bar := facade.measured("bike-bar")
	wide, drop := facade.measured("cloth-width"), facade.measured(
		"cloth-drop")
	parts := []placedElement{
		put("rack bar", Box{long, bar, bar}, piece.Body,
			maths.V(along, high, out)),
		put("rack post", Box{bar, high, bar}, piece.Body,
			maths.V(along, 0, out)),
		put("rack post", Box{bar, high, bar}, piece.Body,
			maths.V(along+long-bar, 0, out)),
	}
	for _, at := range []float64{rackFirst, rackSecond} {
		parts = append(parts, put("washing", Box{wide, drop, bar / half},
			piece.Soft, maths.V(along+long*at, high-drop, out)))
	}
	return parts
}

// rackFirst and rackSecond are how far along a drying rack the two cloths
// hang, as shares of its length. A choice, by eye.
const (
	rackFirst  = 0.05
	rackSecond = 0.52
)
