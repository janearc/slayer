package main

import (
	"fmt"
)

// The document sets its text in three of the PDF standard fonts:
// Helvetica for prose, Helvetica-Bold for names and Courier for
// hashes and addresses. A reader supplies these faces, so nothing is
// embedded. Wrapping needs each face's advance widths, which are
// data below, and the bytes written must be in WinAnsi encoding,
// which covers Latin-1 and the common typographic marks.

// A face is one standard font: its resource name on the page, its
// base font name and the advance of each character it can set.
type face struct {
	resource string
	base     string
	advances map[rune]int
	fixed    int
}

// widthGroup is a width, in thousandths of the font size, shared by
// every character in letters.
type widthGroup struct {
	width   int
	letters string
}

// em is defined by the font format: advances are in thousandths of
// the font size.
const em = 1000.0

// courierAdvance is defined by the Courier AFM file: every character
// advances 600 thousandths of the size.
const courierAdvance = 600

// The three faces. The advance widths are transcribed from Adobe's
// Core 14 font metrics (Helvetica.afm and Helvetica-Bold.afm, the
// "Core 14 AFM Files" published by Adobe), as the widths of the
// WinAnsi characters. An accented letter has the width of its base
// letter in these files, and is grouped with it.
var (
	helvetica     = newFace("F1", "Helvetica", helveticaWidths, noFixed)
	helveticaBold = newFace("F2", "Helvetica-Bold", boldWidths, noFixed)
	courier       = newFace("F3", "Courier", nil, courierAdvance)
)

// noFixed is defined as a face whose characters differ in width.
const noFixed = 0

// winAnsiHigh is defined by the WinAnsi encoding (Windows code page
// 1252): the characters at bytes 0x80 to 0x9f, with the Latin-1
// control code standing where the encoding has none.
const winAnsiHigh = "€\u0081‚ƒ„…†‡ˆ‰Š‹Œ\u008dŽ\u008f" +
	"\u0090‘’“”•–—˜™š›œ\u009džŸ"

// firstHigh and firstLatin are defined by the encoding: where the
// WinAnsi block of typographic marks starts, and where its Latin-1
// block does.
const firstHigh, firstLatin = 0x80, 0xa0

// firstPrintable and lastPrintable are defined by ASCII: the space
// and the tilde.
const firstPrintable, lastPrintable = 0x20, 0x7e

// lastByte is defined by the encoding: it is one byte wide.
const lastByte = 0xff

// winAnsiByte maps each character the encoding has to its byte.
var winAnsiByte = buildEncoding()

// buildEncoding lays out the printable ASCII block, the typographic
// marks and the Latin-1 block, leaving out the unassigned bytes.
func buildEncoding() map[rune]byte {
	table := map[rune]byte{}
	for code := firstPrintable; code <= lastPrintable; code++ {
		table[rune(code)] = byte(code)
	}
	for offset, letter := range []rune(winAnsiHigh) {
		if letter >= firstHigh && letter < firstLatin {
			continue
		}
		table[letter] = byte(firstHigh + offset)
	}
	for code := firstLatin; code <= lastByte; code++ {
		table[rune(code)] = byte(code)
	}
	return table
}

// newFace builds a face from its width groups. A fixed width applies
// to every character in the encoding.
func newFace(resource, base string, groups []widthGroup,
	fixed int) face {
	advances := map[rune]int{}
	for _, group := range groups {
		for _, letter := range group.letters {
			advances[letter] = group.width
		}
	}
	return face{resource: resource, base: base, advances: advances,
		fixed: fixed}
}

// advance is a character's width in thousandths of the size, and
// whether the face can set it.
func (set face) advance(letter rune) (int, bool) {
	if _, held := winAnsiByte[letter]; !held {
		return noFixed, false
	}
	if set.fixed != noFixed {
		return set.fixed, true
	}
	width, known := set.advances[letter]
	return width, known
}

// measure is the width in points of a line of text at a size.
// Characters the face cannot set count as no width; encode refuses
// them.
func (set face) measure(text string, size float64) float64 {
	total := 0
	for _, letter := range text {
		width, _ := set.advance(letter)
		total += width
	}
	return float64(total) * size / em
}

// encode is text as WinAnsi bytes. It refuses the first character the
// encoding lacks, by name, rather than writing a stand-in for it.
func (set face) encode(text string) ([]byte, error) {
	made := make([]byte, noEntries, len(text))
	for _, letter := range text {
		code, held := winAnsiByte[letter]
		if !held {
			return nil, fmt.Errorf("the character %q (U+%04X) is not in"+
				" the PDF's WinAnsi encoding; replace it in the menu"+
				" with a plain spelling", letter, letter)
		}
		made = append(made, code)
	}
	return made, nil
}

// helveticaWidths are Helvetica.afm's widths for WinAnsi characters.
var helveticaWidths = []widthGroup{
	{278, " !,./:;I[\\]ft\u00a0·ÌÍÎÏìíîï"},
	{355, `"`},
	{556, "#$0123456789?L_abdeghnopqu" + "àáâãäåèéêëðñòóôõöùúûüþ" +
		"¢£¤¥§µ€ƒ†‡–«»"},
	{889, "%æ"},
	{667, "&ABEKPSVXYÀÁÂÃÄÅÈÉÊËÝŸŠÞ"},
	{191, "'"},
	{333, "()-`r¨¯´¸¡¹²³ˆ˜‹›„“”\u00ad"},
	{389, "*"},
	{584, "+<=>~¬±×÷"},
	{1015, "@"},
	{722, "CDHNRUwÇÐÑÙÚÛÜ"},
	{778, "GOQÒÓÔÕÖØ"},
	{611, "FTZß¿øŽ"},
	{500, "Jcksvxyzçýÿšž"},
	{833, "Mm"},
	{944, "Wœ"},
	{469, "^"},
	{222, "ijl‘’‚"},
	{334, "{}"},
	{260, "|¦"},
	{737, "©®"},
	{370, "ª"},
	{400, "°"},
	{537, "¶"},
	{365, "º"},
	{834, "¼½¾"},
	{1000, "ÆŒ…‰™—"},
	{350, "•"},
}

// boldWidths are Helvetica-Bold.afm's widths for WinAnsi characters.
var boldWidths = []widthGroup{
	{278, " ,./\\Iijl\u00a0·ÌÍÎÏìíîï‘’‚"},
	{333, "!()-:;[]`ft¡¨¯´¸¹²³ˆ˜‹›\u00ad"},
	{474, `"`},
	{556, "#$0123456789_Jaceksvxyçàáâãäåèéêëýÿ¢£¤¥§€ƒ†‡–¶š«»"},
	{611, "?LbdghnopquðñòóôõöùúûüþßøFTZŽ¿µ"},
	{889, "%mæ"},
	{722, "&ACDHNRUÀÁÂÃÄÅÇÐÑÙÚÛÜ"},
	{238, "'"},
	{389, "*{}r"},
	{584, "+<=>^~¬±×÷"},
	{975, "@"},
	{667, "BEKPSVXYÈÉÊËÝŸŠÞ"},
	{778, "GOQwÒÓÔÕÖØ"},
	{944, "Wœ"},
	{500, "zž„“”"},
	{280, "|¦"},
	{737, "©®"},
	{370, "ª"},
	{400, "°"},
	{365, "º"},
	{834, "¼½¾"},
	{1000, "ÆŒ…‰™—"},
	{350, "•"},
	{833, "M"},
}
