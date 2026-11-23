package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"strings"
	"unicode/utf16"
)

// a document is the smallest PDF writer the catalogue needs: text in
// the three standard faces, deflated pictures, link annotations and a
// sidebar outline on A4 sheets, assembled with a correct
// cross-reference table. Text streams are left uncompressed, so the
// file can be read, and tested, as it is written.

// pointsPerMM is defined by the point and the millimetre: an inch is
// 72 points and 25.4 mm.
const pointsPerMM = 72.0 / 25.4

// pageWidth and pageHeight are defined by ISO 216's A4: 210 by 297 mm.
const pageWidth, pageHeight = 210 * pointsPerMM, 297 * pointsPerMM

// firstName is defined by PDF numbering: objects and picture names
// count from one.
const firstName = 1

// freeHead is defined by the cross-reference table: its zeroth entry
// is the free-list head, standing before every object.
const freeHead = 1

// channels is defined as a picture's red, green and blue.
const channels = 3

// highByte is defined as the shift from Go's 16-bit colour channel to
// its leading eight bits.
const highByte = 8

// lastPlace is defined as how far a slice's last entry stands back
// from its length.
const lastPlace = 1

// An ink is a colour for text and rules, each channel from 0 to 1.
type ink struct{ red, green, blue float64 }

type flatPicture struct {
	width, height int
	deflated      []byte
}

// A mark is a rectangle on a sheet that a reader can click: it goes
// to an address when the address is set, and otherwise to a sheet.
type mark struct {
	x, y, width, height float64
	address             string
	sheet               int
}

// An outlineEntry is a line in the reader's sidebar, with the entries
// indented under it.
type outlineEntry struct {
	title    string
	sheet    int
	children []outlineEntry
}

type document struct {
	title    string
	sheets   []*sheet
	pictures []flatPicture
	outline  []outlineEntry
}

type sheet struct {
	content bytes.Buffer
	marks   []mark
}

// newDocument opens an empty document with a title; pages and
// pictures follow.
func newDocument(title string) *document {
	return &document{title: title}
}

// addPage starts a fresh sheet; marks land in call order.
func (composed *document) addPage() *sheet {
	made := &sheet{}
	composed.sheets = append(composed.sheets, made)
	return made
}

// text sets one line at a point from the sheet's lower left corner.
// It refuses a character the face cannot set.
func (drawn *sheet) text(set face, size float64, colour ink,
	x, y float64, line string) error {
	coded, err := set.encode(line)
	if err != nil {
		return err
	}
	fmt.Fprintf(&drawn.content,
		"BT /%s %g Tf %g %g %g rg %g %g Td (%s) Tj ET\n",
		set.resource, size, colour.red, colour.green, colour.blue,
		x, y, escaped(coded))
	return nil
}

// rule draws a hairline of a width in points from a point to the
// right.
func (drawn *sheet) rule(colour ink, x, y, length, weight float64) {
	fmt.Fprintf(&drawn.content, "%g %g %g RG %g w %g %g m %g %g l S\n",
		colour.red, colour.green, colour.blue, weight, x, y, x+length, y)
}

// picture places a registered picture into a box of points.
func (drawn *sheet) picture(number int, x, y, width, height float64) {
	fmt.Fprintf(&drawn.content, "q %g 0 0 %g %g %g cm /Im%d Do Q\n",
		width, height, x, y, number)
}

// link makes a rectangle open an address, which must be plain ASCII
// so that it reaches the reader as written.
func (drawn *sheet) link(x, y, width, height float64,
	address string) error {
	for _, letter := range address {
		if letter < firstPrintable || letter > lastPrintable {
			return fmt.Errorf("the address %q holds the character %q,"+
				" which a PDF link cannot carry unescaped; write the"+
				" address in percent-encoded form", address, letter)
		}
	}
	drawn.marks = append(drawn.marks, mark{x: x, y: y, width: width,
		height: height, address: address})
	return nil
}

// jump makes a rectangle go to the sheet at a place in the document.
func (drawn *sheet) jump(x, y, width, height float64, sheet int) {
	drawn.marks = append(drawn.marks, mark{x: x, y: y, width: width,
		height: height, sheet: sheet})
}

// addPicture registers a decoded image and returns its number.
func (composed *document) addPicture(decoded image.Image) (int, error) {
	bounds := decoded.Bounds()
	var deflated bytes.Buffer
	press := zlib.NewWriter(&deflated)
	if _, err := press.Write(flatten(decoded)); err != nil {
		return noEntries, err
	}
	if err := press.Close(); err != nil {
		return noEntries, err
	}
	composed.pictures = append(composed.pictures, flatPicture{
		width: bounds.Dx(), height: bounds.Dy(),
		deflated: deflated.Bytes()})
	return len(composed.pictures), nil
}

// flatten resolves transparency onto the book's dark backdrop,
// leaving plain eight-bit red, green and blue rows.
func flatten(decoded image.Image) []byte {
	bounds := decoded.Bounds()
	flat := make([]byte, noEntries,
		bounds.Dx()*bounds.Dy()*channels)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, _ := decoded.At(x, y).RGBA()
			flat = append(flat, byte(red>>highByte),
				byte(green>>highByte), byte(blue>>highByte))
		}
	}
	return flat
}

// octalWidth is defined by the PDF string syntax: a byte written as a
// backslash escape takes three octal digits.
const octalWidth = 3

// escaped keeps encoded text inside PDF string delimiters: the
// delimiters are quoted and every byte outside printable ASCII is
// written as an octal escape, so the stream stays seven-bit.
func escaped(coded []byte) string {
	var quoted strings.Builder
	for _, code := range coded {
		switch {
		case code == '\\' || code == '(' || code == ')':
			quoted.WriteByte('\\')
			quoted.WriteByte(code)
		case code < firstPrintable || code > lastPrintable:
			fmt.Fprintf(&quoted, "\\%0*o", octalWidth, code)
		default:
			quoted.WriteByte(code)
		}
	}
	return quoted.String()
}

// hexTitle is a sidebar title as the UTF-16 hex string PDF accepts
// for text in any script.
func hexTitle(title string) string {
	var made strings.Builder
	made.WriteString("<FEFF")
	for _, unit := range utf16.Encode([]rune(title)) {
		fmt.Fprintf(&made, "%04X", unit)
	}
	made.WriteString(">")
	return made.String()
}

// objectTable numbers objects as they are added; a number may be
// reserved first and filled once the object that needs it is built.
type objectTable struct{ bodies [][]byte }

// reserve takes the next number and leaves the object for put.
func (table *objectTable) reserve() int {
	table.bodies = append(table.bodies, nil)
	return len(table.bodies)
}

// put fills a reserved number.
func (table *objectTable) put(number int, body []byte) {
	table.bodies[number-firstName] = body
}

// add numbers and stores a finished object.
func (table *objectTable) add(body []byte) int {
	number := table.reserve()
	table.put(number, body)
	return number
}

// assemble lays the header, every object, the cross-reference table
// and the trailer in one pass.
func (composed *document) assemble() ([]byte, error) {
	objects := composed.objects()
	var made bytes.Buffer
	made.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects))
	for index, object := range objects {
		offsets[index] = made.Len()
		fmt.Fprintf(&made, "%d 0 obj\n", index+firstName)
		made.Write(object)
		made.WriteString("\nendobj\n")
	}
	table := made.Len()
	fmt.Fprintf(&made, "xref\n0 %d\n", len(objects)+freeHead)
	made.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&made, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&made, "trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>"+
		"\nstartxref\n%d\n%%%%EOF\n", len(objects)+freeHead,
		catalogObject, infoObject, table)
	return made.Bytes(), nil
}

// catalogObject, pagesObject and infoObject are defined by the object
// order objects() keeps: the first three numbers are fixed.
const catalogObject, pagesObject, infoObject = 1, 2, 3

// objects is the document as numbered objects: the catalog, the page
// tree, the information record, the faces, the pictures, then each
// sheet with its content and links, then the outline.
func (composed *document) objects() [][]byte {
	table := &objectTable{}
	table.reserve()
	table.reserve()
	titleHex := hexTitle(composed.title)
	titleObject := fmt.Sprintf("<< /Title %s /Producer (catalogue pdf) >>",
		titleHex)
	table.add([]byte(titleObject))
	var fonts []string
	for _, set := range []face{helvetica, helveticaBold, courier} {
		fontObject := fmt.Sprintf("<< /Type /Font"+
			" /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>",
			set.base)
		number := table.add([]byte(fontObject))
		fonts = append(fonts, fmt.Sprintf("/%s %d 0 R", set.resource, number))
	}
	shown := make([]string, len(composed.pictures))
	for index, shot := range composed.pictures {
		pictureData := pictureObject(shot)
		shown[index] = fmt.Sprintf("/Im%d %d 0 R", index+firstName,
			table.add(pictureData))
	}
	pages := make([]int, len(composed.sheets))
	for index := range pages {
		pages[index] = table.reserve()
	}
	composed.fillSheets(table, pages)
	outlines := composed.fillOutline(table, pages)
	kids := make([]string, len(pages))
	for index, number := range pages {
		kids[index] = fmt.Sprintf("%d 0 R", number)
	}
	kidsText := strings.Join(kids, " ")
	fontsText := strings.Join(fonts, " ")
	shownText := strings.Join(shown, " ")
	pagesText := fmt.Sprintf(
		"<< /Type /Pages /Kids [%s] /Count %d /MediaBox [0 0 %g %g]"+
			" /Resources << /Font << %s >> /XObject << %s >> >> >>",
		kidsText, len(pages), pageWidth, pageHeight,
		fontsText, shownText)
	table.put(pagesObject, []byte(pagesText))
	catalogText := fmt.Sprintf(
		"<< /Type /Catalog /Pages %d 0 R /Outlines %d 0 R"+
			" /PageMode /UseOutlines >>", pagesObject, outlines)
	table.put(catalogObject, []byte(catalogText))
	return table.bodies
}

// pictureObject is a picture's image stream.
func pictureObject(shot flatPicture) []byte {
	heading := fmt.Sprintf("/Type /XObject /Subtype /Image"+
		" /Width %d /Height %d /ColorSpace /DeviceRGB"+
		" /BitsPerComponent 8 /Filter /FlateDecode ",
		shot.width, shot.height)
	return streamed(heading, shot.deflated)
}

// fillSheets stores each sheet's content stream, its links and its
// page object at the number reserved for it.
func (composed *document) fillSheets(table *objectTable, pages []int) {
	for index, drawn := range composed.sheets {
		contentBytes := drawn.content.Bytes()
		contents := table.add(streamed("", contentBytes))
		var annotations []string
		for _, each := range drawn.marks {
			number := table.add(each.annotation(pages))
			annotations = append(annotations, fmt.Sprintf("%d 0 R", number))
		}
		annotationText := strings.Join(annotations, " ")
		pageText := fmt.Sprintf(
			"<< /Type /Page /Parent %d 0 R /Contents %d 0 R"+
				" /Annots [%s] >>", pagesObject, contents,
			annotationText)
		table.put(pages[index], []byte(pageText))
	}
}

// annotation is the link object for a mark.
func (each mark) annotation(pages []int) []byte {
	action := fmt.Sprintf("/Dest [%d 0 R /Fit]", pages[each.sheet])
	if each.address != "" {
		addressBytes := []byte(each.address)
		action = fmt.Sprintf("/A << /S /URI /URI (%s) >>",
			escaped(addressBytes))
	}
	return []byte(fmt.Sprintf("<< /Type /Annot /Subtype /Link"+
		" /Rect [%g %g %g %g] /Border [0 0 0] %s >>", each.x, each.y,
		each.x+each.width, each.y+each.height, action))
}

// fillOutline stores the sidebar outline and returns its root's
// number. An empty outline still has a root.
func (composed *document) fillOutline(table *objectTable,
	pages []int) int {
	root := table.reserve()
	first, last, count := composed.outlineLevel(table, pages, root,
		composed.outline)
	body := fmt.Sprintf("<< /Type /Outlines /Count %d", count)
	if count > noEntries {
		body += fmt.Sprintf(" /First %d 0 R /Last %d 0 R", first, last)
	}
	table.put(root, []byte(body+" >>"))
	return root
}

// outlineLevel stores a run of sibling entries under a parent and
// returns the first and last numbers and how many entries are open
// beneath the run, itself included.
func (composed *document) outlineLevel(table *objectTable, pages []int,
	parent int, entries []outlineEntry) (first, last, count int) {
	numbers := make([]int, len(entries))
	for index := range entries {
		numbers[index] = table.reserve()
	}
	for index, each := range entries {
		body := fmt.Sprintf("<< /Title %s /Parent %d 0 R /Dest [%d 0 R /Fit]",
			hexTitle(each.title), parent, pages[each.sheet])
		if index > noEntries {
			body += fmt.Sprintf(" /Prev %d 0 R", numbers[index-lastPlace])
		}
		if index+lastPlace < len(entries) {
			body += fmt.Sprintf(" /Next %d 0 R", numbers[index+lastPlace])
		}
		inner := noEntries
		if len(each.children) > noEntries {
			begin, end, below := composed.outlineLevel(table, pages,
				numbers[index], each.children)
			body += fmt.Sprintf(" /First %d 0 R /Last %d 0 R /Count %d",
				begin, end, below)
			inner = below
		}
		table.put(numbers[index], []byte(body+" >>"))
		count += lastPlace + inner
	}
	if len(numbers) > noEntries {
		first, last = numbers[noEntries], numbers[len(numbers)-lastPlace]
	}
	return first, last, count
}

// streamed wraps a body as a stream object under a heading.
func streamed(heading string, body []byte) []byte {
	var made bytes.Buffer
	fmt.Fprintf(&made, "<< %s/Length %d >>\nstream\n", heading,
		len(body))
	made.Write(body)
	made.WriteString("\nendstream")
	return made.Bytes()
}
