package main

// A flow lays lines of text down a column, one sheet after another.
// A line that would cross the foot of a sheet opens a continuation
// sheet instead, headed with the name of what is being set, so that
// long notes run on rather than off the page. The flow keeps the first
// error it meets and does nothing after it; its owner reads err once.

// Page geometry, picked by legibility notes and 2026-11-12's
// first look at the pages: a three-quarter inch margin all round, and
// a foot band below the last line of text.
const (
	// margin is the picked outer edge of every sheet, in points.
	margin = 54.0
	// footBand is the picked room under the text for the running
	// foot and its rule, in points.
	footBand = 40.0
	// columnWidth is defined by the sheet less its margins.
	columnWidth = pageWidth - margin - margin
	// textTop is where the first line of a sheet starts.
	textTop = pageHeight - margin
	// textFloor is the lowest a line may end.
	textFloor = margin + footBand
)

// ascentShare is defined as a measured share of the size that a
// capital rises above its baseline in Helvetica (718 thousandths in
// Helvetica.afm), rounded up.
const ascentShare = 0.72

// A style is a face at a size with the spacing and ink it is set in.
type style struct {
	set     face
	size    float64
	leading float64
	colour  ink
}

// The inks are picked for contrast on white paper: all pass 7 to 1.
var (
	inkBody  = ink{0.08, 0.08, 0.08}
	inkQuiet = ink{0.33, 0.33, 0.33}
	inkLink  = ink{0.0, 0.25, 0.6}
	inkFaint = ink{0.55, 0.55, 0.55}
)

// The styles are picked: body text is 12 points on 19 points, a
// leading of about 1.6, and nothing on the page is below 11 points.
var (
	bodyStyle    = style{helvetica, 12, 19, inkBody}
	quietStyle   = style{helvetica, 12, 19, inkQuiet}
	headingStyle = style{helveticaBold, 13, 20, inkBody}
	nameStyle    = style{helveticaBold, 26, 32, inkBody}
	linkStyle    = style{helvetica, 12, 19, inkLink}
	hashStyle    = style{courier, 11, 19, inkBody}
	entryStyle   = style{helvetica, 12, 17, inkBody}
	footStyle    = style{helvetica, 11, 14, inkQuiet}
)

// paragraphGap and sectionGap are picked spaces, in points, between
// paragraphs and before a heading.
const paragraphGap, sectionGap = 6.0, 12.0

// orphanLines is picked: a heading is not left at the foot of a sheet
// unless this many lines go with it.
const orphanLines = 2

type flow struct {
	doc     *document
	sheet   *sheet
	cursor  float64
	running string
	err     error
}

// newFlow starts a flow on a sheet already in the document, with the
// running head a continuation sheet takes.
func newFlow(doc *document, open *sheet, running string) *flow {
	return &flow{doc: doc, sheet: open, cursor: textTop,
		running: running}
}

// room reports whether height more points fit above the foot.
func (laid *flow) room(height float64) bool {
	return laid.cursor-height >= textFloor
}

// need turns to a new sheet unless height points fit on this one.
func (laid *flow) need(height float64) {
	if laid.err == nil && !laid.room(height) {
		laid.turn()
	}
}

// turn opens a continuation sheet and heads it.
func (laid *flow) turn() {
	laid.sheet = laid.doc.addPage()
	laid.cursor = textTop
	laid.write(quietStyle, margin, laid.running+", continued")
}

// write sets one line at the cursor in a style and moves down.
func (laid *flow) write(look style, x float64, line string) {
	if laid.err != nil {
		return
	}
	drop := (look.leading + look.size*ascentShare) / 2
	laid.err = laid.sheet.text(look.set, look.size, look.colour, x,
		laid.cursor-drop, line)
	laid.cursor -= look.leading
}

// gap moves down without drawing.
func (laid *flow) gap(height float64) { laid.cursor -= height }

// paragraph wraps text to the column and sets it, flowing to a new
// sheet where one fills.
func (laid *flow) paragraph(look style, text string) {
	for _, line := range wrapText(look.set, look.size, text,
		columnWidth) {
		laid.need(look.leading)
		laid.write(look, margin, line)
	}
	laid.gap(paragraphGap)
}

// heading sets a heading with room for the lines that follow it.
func (laid *flow) heading(text string) {
	laid.gap(sectionGap)
	laid.need(headingStyle.leading + orphanLines*bodyStyle.leading)
	laid.write(headingStyle, margin, text)
}

// linked sets text as a link: coloured, clickable, and underlined when
// ruled is set (an address in the monospaced face is not, as the rule
// would hide its underscores). The address is the link's target and
// the lines are wrapped to the column, one click rectangle to a line.
func (laid *flow) linked(look style, text, address string, ruled bool) {
	for _, line := range wrapText(look.set, look.size, text,
		columnWidth) {
		laid.need(look.leading)
		top := laid.cursor
		laid.write(look, margin, line)
		laid.underline(look, top, line, address, ruled)
	}
}

// underlineDrop and underlineWeight are picked: the rule sits this far
// under the baseline, this thick, in points.
const underlineDrop, underlineWeight = 2.0, 0.6

// underline rules the line just written, if asked, and makes it a
// link.
func (laid *flow) underline(look style, top float64, line,
	address string, ruled bool) {
	if laid.err != nil {
		return
	}
	length := look.set.measure(line, look.size)
	base := top - (look.leading+look.size*ascentShare)/2
	if ruled {
		laid.sheet.rule(look.colour, margin, base-underlineDrop,
			length, underlineWeight)
	}
	laid.err = laid.sheet.link(margin, top-look.leading, length,
		look.leading, address)
}

// labelGap is picked: the space, in points, between a label and the
// value set beside it.
const labelGap = 8.0

// labelled sets a label and a value on one line when both fit the
// column, and the value under the label when they do not.
func (laid *flow) labelled(label, value string) {
	laid.need(bodyStyle.leading + hashStyle.leading)
	across := quietStyle.set.measure(label, quietStyle.size) + labelGap
	if hashStyle.set.measure(value, hashStyle.size) > columnWidth-across {
		laid.write(quietStyle, margin, label)
		laid.write(hashStyle, margin, value)
		return
	}
	top := laid.cursor
	laid.write(quietStyle, margin, label)
	laid.cursor = top
	laid.write(hashStyle, margin+across, value)
}
