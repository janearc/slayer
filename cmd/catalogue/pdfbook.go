package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Design of the PDF, in the order a reader meets it. It is written
// from the same book record as the HTML, so it holds what the HTML
// holds for each card.
//
// The cover names the book, the date it was drawn, how many specimens
// and groups it holds, and the binary that drew it. The contents list
// every specimen under its group, alphabetically as the HTML's filter
// does, each with the sheet it is on.
//
// Then one sheet for each specimen in the book's order: the picture,
// the group and kind, the description, the notes, the references as
// links and the card's provenance. A card with more words than a sheet
// holds runs on to a continuation sheet. One specimen to a sheet,
// because the picture is the point and at a readable size two would
// not fit.
//
// Every sheet after the cover has the book's title and its number in
// the foot, and the reader's sidebar lists the contents, the groups
// and the kinds.

// bookTitle is defined by the HTML page's own title.
const bookTitle = "The manufactory"

// bookTagline and bookCaveat are the HTML page's words, kept the same.
const (
	bookTagline = "A field guide to the things we can make."
	bookScenery = "These are authored scenery. Source notes say which" +
		" dimensions have references and which are picked."
	bookCaveat = "A rendered specimen proves that view reached the GPU." +
		" It does not establish numerical agreement for every scene" +
		" or physical accuracy."
)

// coverStyle is picked: the cover's one large line.
var coverStyle = style{helveticaBold, 40, 48, inkBody}

// dateLayout is defined by Go's reference time: day, month name, year.
const dateLayout = "2 January 2006"

// unknownDevice is what a card says of its device when its render
// record is not in the book.
const unknownDevice = "not recorded"

// composer holds what laying a book needs.
type composer struct {
	loaded   *book
	root     string
	drawn    time.Time
	failures bool
	doc      *document
	first    []int
}

// compose lays the cover, the contents and then every card, one to a
// sheet, in the book's own order. Cards that did not render are laid
// as pages saying so when failures is set; the caller has already
// refused them otherwise.
func compose(loaded *book, root string, drawn time.Time,
	failures bool) (*document, error) {
	made := &composer{loaded: loaded, root: root, drawn: drawn,
		failures: failures, doc: newDocument(bookTitle)}
	for _, each := range loaded.Cards {
		if err := each.encodable(); err != nil {
			return nil, fmt.Errorf("specimen %s (%s): %w", each.Folder,
				each.Entry.Kind, err)
		}
	}
	if err := made.cover(); err != nil {
		return nil, err
	}
	rows := made.contentsRows()
	pages := paginate(rows)
	for range pages {
		made.doc.addPage()
	}
	for _, each := range loaded.Cards {
		made.first = append(made.first, len(made.doc.sheets))
		if err := made.specimen(each); err != nil {
			return nil, fmt.Errorf("specimen %s (%s): %w", each.Folder,
				each.Entry.Kind, err)
		}
	}
	if err := made.contents(pages); err != nil {
		return nil, err
	}
	made.outline()
	return made.doc, made.feet()
}

// encodable refuses a card with a character the PDF's fonts lack,
// before anything is laid, so that the refusal is about the card and
// not about the sheet it happened to fall on.
func (each card) encodable() error {
	texts := []string{each.Entry.Kind, each.Entry.Group, each.Detail,
		each.Entry.About, each.Entry.Notes, each.Failure, each.Status}
	texts = append(texts, each.Entry.Details...)
	for _, reference := range each.Entry.References {
		texts = append(texts, reference.Title, reference.URL)
	}
	for _, text := range texts {
		if _, err := helvetica.encode(text); err != nil {
			return err
		}
	}
	return nil
}

// feet sets the running foot on every sheet after the cover.
func (made *composer) feet() error {
	for index, each := range made.doc.sheets[coverSheets:] {
		number := strconv.Itoa(index + coverSheets + firstName)
		err := each.text(footStyle.set, footStyle.size, footStyle.colour,
			margin, footBaseline, bookTitle)
		if err != nil {
			return err
		}
		width := footStyle.set.measure("Page "+number, footStyle.size)
		err = each.text(footStyle.set, footStyle.size, footStyle.colour,
			pageWidth-margin-width, footBaseline, "Page "+number)
		if err != nil {
			return err
		}
		each.rule(inkFaint, margin, footRule, columnWidth, footRuleWeight)
	}
	return nil
}

// coverSheets is defined as the one sheet before the contents.
const coverSheets = 1

// footBaseline, footRule and footRuleWeight are picked: where the foot
// text and its rule sit above the sheet's edge, in points.
const footBaseline, footRule, footRuleWeight = 30.0, 50.0, 0.5

// The cover.
func (made *composer) cover() error {
	laid := newFlow(made.doc, made.doc.addPage(), bookTitle)
	laid.write(quietStyle, margin, "Slayer specimen book")
	laid.write(coverStyle, margin, bookTitle)
	laid.gap(paragraphGap)
	laid.paragraph(bodyStyle, bookTagline)
	laid.paragraph(bodyStyle, made.tally())
	laid.heading("Groups")
	laid.paragraph(bodyStyle, made.groupList())
	laid.paragraph(quietStyle, bookScenery)
	laid.paragraph(quietStyle, bookCaveat)
	made.producerLines(laid)
	if laid.err == nil && len(made.doc.sheets) != coverSheets {
		return errors.New("the cover does not fit one sheet")
	}
	return laid.err
}

// tally is the cover's line of counts and the date.
func (made *composer) tally() string {
	groupCount := len(made.loaded.Groups())
	line := fmt.Sprintf("Drawn %s. %d specimens in %d groups.",
		made.drawn.Format(dateLayout), len(made.loaded.Cards),
		groupCount)
	if missing := made.unrendered(); missing > noEntries {
		line += fmt.Sprintf(" %d did not render.", missing)
	}
	return line
}

// unrendered counts the cards without a picture.
func (made *composer) unrendered() int {
	missing := noEntries
	for _, each := range made.loaded.Cards {
		if each.Status != renderedStatus {
			missing++
		}
	}
	return missing
}

// groupList is every group with its count of specimens.
func (made *composer) groupList() string {
	counts := map[string]int{}
	for _, each := range made.loaded.Cards {
		counts[each.Entry.Group]++
	}
	var named []string
	for _, group := range made.loaded.Groups() {
		named = append(named, fmt.Sprintf("%s (%d)", group, counts[group]))
	}
	return strings.Join(named, "; ") + "."
}

// producerLines names the binary that drew the book.
func (made *composer) producerLines(laid *flow) {
	if made.loaded.Producer == nil {
		return
	}
	laid.heading("Made by")
	laid.write(quietStyle, margin, "Compiled revision "+
		made.loaded.Producer.Revision+".")
	laid.write(quietStyle, margin, "Binary sha256:")
	laid.write(hashStyle, margin, made.loaded.Producer.BinarySHA256)
}

// pictureSheet is a card's picture file inside its folder.
const pictureSheet = "specimen.png"

// specimen lays one card: its picture or the reason it has none, its
// words, its references and its provenance.
func (made *composer) specimen(each card) error {
	laid := newFlow(made.doc, made.doc.addPage(), each.Entry.Kind)
	laid.write(quietStyle, margin, each.Entry.Group+", "+each.Detail+
		" detail")
	laid.write(nameStyle, margin, each.Entry.Kind)
	laid.gap(paragraphGap)
	if err := made.pictureOrReason(laid, each); err != nil {
		return err
	}
	if err := made.words(laid, each); err != nil {
		return err
	}
	return laid.err
}

// words sets everything on a specimen's sheet below its picture.
func (made *composer) words(laid *flow, each card) error {
	if each.Cutaway {
		laid.paragraph(quietStyle, "Cutaway view: enclosure hidden.")
	}
	laid.paragraph(bodyStyle, each.Entry.About)
	if each.Entry.Notes != "" {
		laid.heading("Recipe and source notes")
		laid.paragraph(bodyStyle, each.Entry.Notes)
	}
	laid.paragraph(quietStyle, "Available details: "+
		strings.Join(each.Entry.Details, ", ")+".")
	made.references(laid, each)
	return made.provenance(laid, each)
}

// smallestPicture is picked: the least share of the column's width a
// picture is shrunk to so that its card's words fit under it. A card
// with more words than that leaves room for runs on to a second sheet
// instead, with its picture at full size.
const smallestPicture = 0.6

// pictureHeight is how tall a card's picture is set: the full width of
// the column, or less when that would push the card's words onto a
// second sheet, by measuring those words on a sheet of their own.
func (made *composer) pictureHeight(laid *flow, each card,
	full float64) (float64, error) {
	scratch := newDocument(bookTitle)
	measure := newFlow(scratch, scratch.addPage(), each.Entry.Kind)
	if err := made.words(measure, each); err != nil {
		return noEntries, err
	}
	if measure.err != nil {
		return noEntries, measure.err
	}
	used := textTop - measure.cursor
	room := laid.cursor - textFloor - used - sectionGap
	if len(scratch.sheets) > firstName || room < full*smallestPicture {
		return full, nil
	}
	return min(full, room), nil
}

// pictureOrReason sets the picture, as large as the column and the
// card's words allow and centred, or for a card that did not render,
// saying so and why.
func (made *composer) pictureOrReason(laid *flow, each card) error {
	if each.Status != renderedStatus {
		reason := each.Status
		if each.Failure != "" {
			reason += ": " + each.Failure
		}
		laid.write(headingStyle, margin, "This specimen did not render.")
		laid.paragraph(bodyStyle, "Reason: "+reason)
		return nil
	}
	decoded, err := readSpecimen(filepath.Join(made.root, each.Folder,
		pictureSheet))
	if err != nil {
		return err
	}
	number, err := made.doc.addPicture(decoded)
	if err != nil {
		return err
	}
	bounds := decoded.Bounds()
	ratio := float64(bounds.Dx()) / float64(bounds.Dy())
	height, err := made.pictureHeight(laid, each, columnWidth/ratio)
	if err != nil {
		return err
	}
	laid.need(height)
	across := height * ratio
	laid.sheet.picture(number, margin+(columnWidth-across)/halved,
		laid.cursor-height, across, height)
	laid.gap(height + sectionGap)
	return nil
}

// halved is defined as half, for centring.
const halved = 2.0

// references sets each reference's title as a link, with its address
// under it in the monospaced face.
func (made *composer) references(laid *flow, each card) {
	if len(each.Entry.References) == noEntries {
		return
	}
	laid.heading("References")
	for _, reference := range each.Entry.References {
		laid.need(linkStyle.leading + hashStyle.leading)
		laid.linked(linkStyle, reference.Title, reference.URL, true)
		laid.linked(style{courier, hashStyle.size, hashStyle.leading,
			inkLink}, reference.URL, reference.URL, false)
		laid.gap(paragraphGap)
	}
}

// provenance sets the settings hash, the binary and the device.
func (made *composer) provenance(laid *flow, each card) error {
	maker := each.Producer
	if maker == nil {
		maker = made.loaded.Producer
	}
	device, err := deviceOf(made.root, each)
	if err != nil {
		return err
	}
	laid.heading("Provenance")
	if each.Settings != "" {
		laid.labelled("Settings", each.Settings)
	}
	revision := "unknown"
	if maker != nil {
		laid.labelled("Binary", maker.BinarySHA256)
		revision = maker.Revision
	}
	laid.write(quietStyle, margin, "Hashes are sha256. Compiled revision "+
		revision+"; device: "+device+".")
	return nil
}

// deviceOf reads the GPU a card was drawn on from its render record.
// A book without the record says so; a record that cannot be read is
// an error.
func deviceOf(root string, each card) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, each.Folder,
		"render.json"))
	if errors.Is(err, os.ErrNotExist) {
		return unknownDevice, nil
	}
	if err != nil {
		return "", err
	}
	var record struct {
		Profile struct{ Device string } `json:"gpu-profile"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return "", fmt.Errorf("%s's render record: %w", each.Folder, err)
	}
	if record.Profile.Device == "" {
		return unknownDevice, nil
	}
	return record.Profile.Device, nil
}

// readSpecimen decodes one card's picture.
func readSpecimen(path string) (image.Image, error) {
	opened, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	return png.Decode(opened)
}

// outline lists the contents, then each group with its kinds, in the
// reader's sidebar.
func (made *composer) outline() {
	made.doc.outline = append(made.doc.outline,
		outlineEntry{title: "Contents", sheet: coverSheets})
	for _, group := range made.loaded.Groups() {
		entry := outlineEntry{title: group}
		for index, each := range made.loaded.Cards {
			if each.Entry.Group != group {
				continue
			}
			if entry.children == nil {
				entry.sheet = made.first[index]
			}
			entry.children = append(entry.children, outlineEntry{
				title: made.label(each), sheet: made.first[index]})
		}
		made.doc.outline = append(made.doc.outline, entry)
	}
}

// label names a card in the contents and the sidebar: its kind, with
// its detail when the book holds more than one.
func (made *composer) label(each card) string {
	for _, other := range made.loaded.Cards {
		if other.Detail != each.Detail {
			return each.Entry.Kind + " (" + each.Detail + ")"
		}
	}
	return each.Entry.Kind
}
