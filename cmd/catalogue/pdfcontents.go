package main

import "strconv"

// The contents lists every specimen under its group, groups in
// alphabetical order as the HTML's filter has them, specimens in the
// book's order. Each row has the kind, a leader of dots and the number
// of the sheet the specimen starts on, and the whole row is a link to
// that sheet. The number of sheets the contents takes depends only on
// the rows, so it is known before any specimen is laid.

// A contentsRow is a group heading, or a specimen under one.
type contentsRow struct {
	group string
	card  int
	lines []string
}

// numberRoom is picked: the width kept free at the right of a row for
// the sheet number and its leader, in points.
const numberRoom = 64.0

// leaderGap is picked: the least room, in points, between the end of a
// label and the first dot of its leader, and between the leader and
// the number.
const leaderGap = 8.0

// isHeading says whether the row stands for a group rather than a card.
func (row contentsRow) isHeading() bool { return row.group != "" }

// height is the vertical room a row takes.
func (row contentsRow) height() float64 {
	if row.isHeading() {
		return sectionGap + headingStyle.leading
	}
	return float64(len(row.lines)) * entryStyle.leading
}

// contentsRows is the rows in order, with each label wrapped to the
// room before the numbers.
func (made *composer) contentsRows() []contentsRow {
	var rows []contentsRow
	for _, group := range made.loaded.Groups() {
		rows = append(rows, contentsRow{group: group})
		for index, each := range made.loaded.Cards {
			if each.Entry.Group != group {
				continue
			}
			rows = append(rows, contentsRow{card: index,
				lines: wrapText(entryStyle.set, entryStyle.size,
					made.rowLabel(each), columnWidth-numberRoom)})
		}
	}
	return rows
}

// rowLabel is a card's label with a note when it has no picture.
func (made *composer) rowLabel(each card) string {
	if each.Status != renderedStatus {
		return made.label(each) + " (did not render)"
	}
	return made.label(each)
}

// contentsTitleHeight is the room the title of the first contents
// sheet takes.
const contentsTitleHeight = 32.0 + paragraphGap

// paginate splits rows into sheets. A heading never ends a sheet: it
// goes over with the first row under it.
func paginate(rows []contentsRow) [][]contentsRow {
	var pages [][]contentsRow
	var open []contentsRow
	used := contentsTitleHeight
	room := textTop - textFloor
	for index, row := range rows {
		keep := row.height()
		if row.isHeading() && index+firstName < len(rows) {
			keep += rows[index+firstName].height()
		}
		if used+keep > room && len(open) > noEntries {
			pages = append(pages, open)
			open, used = nil, noEntries
		}
		open = append(open, row)
		used += row.height()
	}
	return append(pages, open)
}

// contents draws the pages paginate planned onto the sheets set aside
// for them, after the cover.
func (made *composer) contents(pages [][]contentsRow) error {
	for index, page := range pages {
		laid := newFlow(made.doc, made.doc.sheets[coverSheets+index],
			"Contents")
		if index == noEntries {
			laid.write(nameStyle, margin, "Contents")
			laid.gap(paragraphGap)
		}
		for _, row := range page {
			made.contentsRow(laid, row)
		}
		if laid.err != nil {
			return laid.err
		}
	}
	return nil
}

// contentsRow draws a heading or a specimen's lines.
func (made *composer) contentsRow(laid *flow, row contentsRow) {
	if row.isHeading() {
		laid.gap(sectionGap)
		laid.write(headingStyle, margin, row.group)
		return
	}
	target := made.first[row.card]
	top := laid.cursor
	for index, line := range row.lines {
		laid.write(entryStyle, margin, line)
		if index+firstName == len(row.lines) {
			made.leader(laid, line, strconv.Itoa(target+firstName))
		}
	}
	laid.sheet.jump(margin, laid.cursor, columnWidth, top-laid.cursor,
		target)
}

// leader fills from the end of a label to the sheet number with dots
// and sets the number flush right, on the line just written.
func (made *composer) leader(laid *flow, label, number string) {
	if laid.err != nil {
		return
	}
	look := entryStyle
	baseline := laid.cursor + look.leading -
		(look.leading+look.size*ascentShare)/2
	right := margin + columnWidth
	wide := look.set.measure(number, look.size)
	laid.err = laid.sheet.text(look.set, look.size, look.colour,
		right-wide, baseline, number)
	start := margin + look.set.measure(label, look.size) + leaderGap
	unit := look.set.measure(". ", look.size)
	dots := int((right - wide - leaderGap - start) / unit)
	if dots <= noEntries || laid.err != nil {
		return
	}
	laid.err = laid.sheet.text(look.set, look.size, inkFaint, right-wide-
		leaderGap-float64(dots)*unit, baseline,
		repeated(". ", dots))
}

// repeated is a string written count times.
func repeated(piece string, count int) string {
	made := ""
	for range count {
		made += piece
	}
	return made
}
