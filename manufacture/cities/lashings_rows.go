package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// PlainRows are two rows of plain blocks facing each other across a
// street, for lashings to be drawn on where a set brings no buildings of
// its own. The street runs along z with its middle on the line x is
// nought, and the blocks stand on the ground, y nought, with their
// walls plain but for the panes of their windows, each lit or dark.
// A block is a volume of the group the lashings are drawn over.

// plainRowsKind is the kind the volumes of a plain row are given.
const plainRowsKind = "block"

// lastShare is how many widths of block the street may have left for its
// last block to take all of it, rather than leave one too short to build.
const lastShare = 1.5

// The two sides of the street, by the sign of x: the one on the west, and
// the one on the east.
const (
	westRow = iota
	eastRow
)

// RowsAsk is what a set asks of the rows: the width of the street and its
// length in metres, either of them nought for the culture's own.
type RowsAsk struct{ Width, Length float64 }

// PlainRows are the blocks of a street, as a design to stand in the
// street's frame and as the volumes lashings are drawn over.
type PlainRows struct {
	// Volumes are the blocks, west row first, each row from the south.
	Volumes []Volume

	design Design
}

// Design is the blocks and their windows.
func (rows *PlainRows) Design() Design { return rows.design }

// NewPlainRows is the rows of a street drawn from the seed. It refuses a
// width or a length below nought.
func NewPlainRows(ask RowsAsk, seed Seed) (*PlainRows, error) {
	if ask.Width < 0 || ask.Length < 0 {
		return nil, fmt.Errorf("cities: a street has a width and a length "+
			"that are not below nought, not %v by %v", ask.Width,
			ask.Length)
	}
	drawing, err := newDrawer(lashingsCulture, seed)
	if err != nil {
		return nil, err
	}
	if ask.Width == 0 {
		ask.Width = drawing.fixed("default-street-width")
	}
	if ask.Length == 0 {
		ask.Length = drawing.fixed("default-street-length")
	}
	rows := &PlainRows{design: Design{Name: "plain rows"}}
	for _, side := range []int{westRow, eastRow} {
		rows.row(drawing, ask, side)
	}
	return rows, drawing.trouble
}

// row builds the blocks of one side of the street, one after another
// from the south end, each as wide as the culture draws for it, and the
// last all that is left if what is left is not much more than one.
func (rows *PlainRows) row(drawing *drawer, ask RowsAsk, side int) {
	south, end := -ask.Length/2, ask.Length/2
	for index, last := 0, false; !last && drawing.trouble == nil; index++ {
		at := place{Row: side, Bay: index}
		width := drawing.number("block-width", at)
		if last = end-south < width*lastShare; last {
			width = end - south
		}
		rows.block(drawing, ask, at, south, width)
		south += width
	}
}

// block is one block of a side of the street, from a south end and a
// width along the street: its wall, its volume and the windows of its
// face on the street.
func (rows *PlainRows) block(drawing *drawer, ask RowsAsk, at place,
	south, width float64) {
	depth := drawing.fixed("block-depth")
	height := drawing.fixed("block-ground") +
		(drawing.fixed("block-storeys")-1)*drawing.fixed("block-storey")
	face, back := ask.Width/2, ask.Width/2+depth
	if at.Row == westRow {
		face, back = -face, -back
	}
	west, east := min(face, back), max(face, back)
	rows.Volumes = append(rows.Volumes, Volume{Kind: plainRowsKind,
		Plot: len(rows.Volumes), Low: [3]float64{west, 0, south},
		High: [3]float64{east, height, south + width}})
	wall := paint("plain wall", drawing.pick("block-wall",
		drawing.options("block-wall"), at))
	rows.design.Elements = append(rows.design.Elements, placedElement{
		Element: solid{Name: "plain block", Form: Box{east - west, height,
			width}, Material: wall},
		Place: standingAt(maths.V(west, 0, south))})
	rows.windows(drawing, at, face, south, width)
}

// windows are the panes of a block's face on the street: a row of them
// on every storey, evenly spaced and centred along the block, each lit or
// dark as the culture's chance says.
func (rows *PlainRows) windows(drawing *drawer, at place, face, south,
	width float64) {
	pitch, wide := drawing.fixed("window-pitch"), drawing.fixed("window-width")
	high, proud := drawing.fixed("window-height"),
		drawing.fixed("window-proud")
	count := int(width / pitch)
	first := south + (width-float64(count)*pitch+pitch-wide)/2
	if at.Row == eastRow {
		proud = -proud
	}
	for storey := range int(drawing.fixed("block-storeys")) {
		floor := drawing.fixed("block-ground")
		if storey > 0 {
			floor += float64(storey-1) * drawing.fixed("block-storey")
		}
		for opening := range count {
			glass, err := drawing.glass(place{Row: at.Row, Bay: at.Bay,
				Storey: storey, Opening: opening})
			drawing.fail(err)
			rows.design.Elements = append(rows.design.Elements, placedElement{
				Element: solid{Name: "pane", Form: Box{max(proud, -proud),
					high, wide}, Material: glass},
				Place: standingAt(maths.V(min(face, face+proud),
					floor+drawing.fixed("window-sill"),
					first+float64(opening)*pitch))})
		}
	}
}

// DefaultDensity is the density a street is hung at when a set does not
// say, from the culture.
func DefaultDensity() (float64, error) {
	drawing, err := newDrawer(lashingsCulture, 0)
	if err != nil {
		return 0, err
	}
	density := drawing.fixed("default-density")
	return density, drawing.trouble
}
