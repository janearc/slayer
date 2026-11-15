package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A run is one line of text as the writer put it on a sheet.
type run struct {
	resource string
	size     float64
	x, y     float64
	text     string
}

// A sheetText is every run on one sheet, in the order drawn.
type sheetText struct{ runs []run }

// words is the sheet's text with each run's words joined by spaces.
func (read sheetText) words() string {
	var all []string
	for _, each := range read.runs {
		all = append(all, each.text)
	}
	return strings.Join(strings.Fields(strings.Join(all, " ")), " ")
}

var (
	objectPattern = regexp.MustCompile(`(?s)(\d+) 0 obj\n(.*?)\nendobj\n`)
	runPattern    = regexp.MustCompile(`BT /(F\d) ([\d.]+) Tf ` +
		`[-\d.e+]+ [-\d.e+]+ [-\d.e+]+ rg ([-\d.e+]+) ([-\d.e+]+) Td ` +
		`\((.*)\) Tj ET`)
	pageContents = regexp.MustCompile(`/Type /Page /Parent \d+ 0 R ` +
		`/Contents (\d+) 0 R`)
	octalEscape = regexp.MustCompile(`^[0-7]{3}`)
)

// winAnsiRune is the character a byte stands for in the encoding.
func winAnsiRune(code byte) rune {
	for letter, held := range winAnsiByte {
		if held == code {
			return letter
		}
	}
	return '?'
}

// unescape reads a PDF string body back into the characters it says.
func unescape(body string) string {
	var made strings.Builder
	for index := noEntries; index < len(body); index++ {
		if body[index] != '\\' {
			made.WriteRune(winAnsiRune(body[index]))
			continue
		}
		rest := body[index+firstName:]
		if code := octalEscape.FindString(rest); code != "" {
			value, _ := strconv.ParseUint(code, octalBase, byteBits)
			made.WriteRune(winAnsiRune(byte(value)))
			index += len(code)
			continue
		}
		made.WriteByte(rest[noEntries])
		index++
	}
	return made.String()
}

// octalBase and byteBits are defined by the escape syntax.
const octalBase, byteBits = 8, 8

// readSheets reads a document's sheets back, in page order.
func readSheets(test *testing.T, raw []byte) []sheetText {
	test.Helper()
	bodies := map[int]string{}
	var order []int
	for _, found := range objectPattern.FindAllSubmatch(raw, -1) {
		number, _ := strconv.Atoi(string(found[firstMatch]))
		bodies[number] = string(found[firstMatch+firstName])
		order = append(order, number)
	}
	var sheets []sheetText
	for _, number := range order {
		page := pageContents.FindStringSubmatch(bodies[number])
		if page == nil {
			continue
		}
		stream, _ := strconv.Atoi(page[firstMatch])
		sheets = append(sheets, sheetText{runs: runsOf(bodies[stream])})
	}
	return sheets
}

// runsOf reads the runs out of one content stream.
func runsOf(stream string) []run {
	var runs []run
	for _, found := range runPattern.FindAllStringSubmatch(stream, -1) {
		size, _ := strconv.ParseFloat(found[firstMatch+firstName], 64)
		x, _ := strconv.ParseFloat(found[firstMatch+firstName*2], 64)
		y, _ := strconv.ParseFloat(found[firstMatch+firstName*3], 64)
		runs = append(runs, run{resource: found[firstMatch], size: size,
			x: x, y: y, text: unescape(found[firstMatch+firstName*4])})
	}
	return runs
}

// titleSheet is the first sheet after the contents with a specimen's
// kind set as a name, or -1.
func titleSheet(sheets []sheetText, kind string) int {
	for index, each := range sheets {
		for _, shown := range each.runs {
			if shown.resource == helveticaBold.resource &&
				shown.size == nameStyle.size && shown.text == kind &&
				index > coverSheets {
				return index
			}
		}
	}
	return -firstName
}

func TestPDFHasACoverContentsAndOneSheetPerSpecimen(test *testing.T) {
	// cards is defined as a book that fits one contents sheet.
	const cards = 5
	sheets := readSheets(test, composeDocument(test,
		writeTestBook(test, cards)))
	if want := coverSheets + firstName + cards; len(sheets) != want {
		test.Fatalf("got %d sheets, want %d", len(sheets), want)
	}
	if !strings.Contains(sheets[noEntries].words(), bookTitle) {
		test.Fatal("the cover does not carry the title")
	}
	if !strings.Contains(sheets[coverSheets].words(), "Contents") {
		test.Fatal("the sheet after the cover is not the contents")
	}
	for index := range cards {
		want := coverSheets + firstName + index
		kind := fmt.Sprintf("kind-%d", index)
		if got := titleSheet(sheets, kind); got != want {
			test.Fatalf("%s is on sheet %d, want %d (menu order)", kind,
				got, want)
		}
	}
}

func TestEveryCardsWordsAreOnItsOwnSheet(test *testing.T) {
	// cards is defined as a few cards, enough to mix up.
	const cards = 3
	sheets := readSheets(test, composeDocument(test,
		writeTestBook(test, cards)))
	for index := range cards {
		kind := fmt.Sprintf("kind-%d", index)
		words := sheets[titleSheet(sheets, kind)].words()
		for _, want := range []string{kind, testAbout, testNotes,
			testTitle + strconv.Itoa(index), "Provenance", "Settings 5e771ed5",
			"quick, demo", "tests, quick detail"} {
			if !strings.Contains(words, want) {
				test.Fatalf("the page of %s lacks %q: %s", kind, want,
					words)
			}
		}
		other := fmt.Sprintf("kind-%d", (index+firstName)%cards)
		if strings.Contains(words, other+" ") ||
			strings.Contains(words, testTitle+strconv.Itoa(
				(index+firstName)%cards)) {
			test.Fatalf("the page of %s holds words of %s", kind, other)
		}
	}
}

func TestContentsNumbersAreTheSheetsSpecimensAreOn(test *testing.T) {
	// cards is defined as enough specimens that the contents runs to
	// a second sheet.
	const cards = 70
	sheets := readSheets(test, composeDocument(test,
		writeTestBook(test, cards)))
	if first := titleSheet(sheets, "kind-0"); first <= coverSheets+
		firstName {
		test.Fatalf("the contents did not run to a second sheet (kind-0"+
			" is on sheet %d)", first)
	}
	for index := range cards {
		kind := fmt.Sprintf("kind-%d", index)
		number := contentsNumber(sheets, kind)
		if got := titleSheet(sheets, kind) + firstName; number != got {
			test.Fatalf("contents say %s is on page %d, it is on %d",
				kind, number, got)
		}
	}
}

// contentsNumber is the page number the contents print against a
// kind, or -1.
func contentsNumber(sheets []sheetText, kind string) int {
	for _, each := range sheets[coverSheets:] {
		for index, shown := range each.runs {
			if shown.resource == helvetica.resource &&
				shown.text == kind && index+firstName < len(each.runs) {
				number, err := strconv.Atoi(each.runs[index+firstName].text)
				if err == nil {
					return number
				}
			}
		}
	}
	return -firstName
}

func TestLongNotesRunOnToASecondSheetInsideTheMargins(test *testing.T) {
	// repeats is defined as enough sentences to fill several sheets.
	const repeats = 200
	made := testBook(twoCards)
	made.Cards[noEntries].Entry.Notes = strings.Repeat(testNotes+" ",
		repeats)
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	if len(sheets) <= coverSheets+firstName+twoCards {
		test.Fatal("long notes did not run on to another sheet")
	}
	var continued int
	for _, each := range sheets {
		if strings.Contains(each.words(), "kind-0, continued") {
			continued++
		}
		for _, shown := range each.runs {
			assertInside(test, shown)
		}
	}
	if continued == noEntries {
		test.Fatal("no continuation sheet names what it continues")
	}
	// proportions ends each sentence once, so a lost word shows here
	// even where a sentence is split across sheets.
	if strings.Count(strings.Join(wordsOf(sheets), " "),
		"proportions.") != repeats+firstName {
		test.Fatal("the long notes lost words in the wrapping")
	}
}

// wordsOf is each sheet's words.
func wordsOf(sheets []sheetText) []string {
	var all []string
	for _, each := range sheets {
		all = append(all, each.words())
	}
	return all
}

// facesByResource names each face by its resource.
var facesByResource = map[string]face{helvetica.resource: helvetica,
	helveticaBold.resource: helveticaBold, courier.resource: courier}

// assertInside fails a run that crosses a margin or the foot band, or
// is set smaller than the page allows.
func assertInside(test *testing.T, shown run) {
	test.Helper()
	// slack is defined as the rounding the written numbers carry.
	const slack = 0.01
	width := facesByResource[shown.resource].measure(shown.text,
		shown.size)
	if shown.x < margin-slack ||
		shown.x+width > pageWidth-margin+slack {
		test.Fatalf("%q runs past the margins: x %g width %g", shown.text,
			shown.x, width)
	}
	if shown.y > pageHeight-margin || shown.y < footBaseline-slack {
		test.Fatalf("%q is off the sheet at %g", shown.text, shown.y)
	}
	if shown.size < minimumSize {
		test.Fatalf("%q is set at %g points", shown.text, shown.size)
	}
}

// minimumSize is defined by the slice's rule: no text on the page is
// smaller than 11 points.
const minimumSize = 11.0

func TestAShortBookStaysOnOneSheetEach(test *testing.T) {
	made := testBook(oneCard)
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	for _, each := range sheets {
		if strings.Contains(each.words(), "continued") {
			test.Fatal("a short card ran on to a second sheet")
		}
		for _, shown := range each.runs {
			assertInside(test, shown)
		}
	}
}

func TestACharacterTheEncodingLacksIsRefusedByName(test *testing.T) {
	made := testBook(twoCards)
	made.Cards[oneCard].Entry.Notes = "Reaches the star → and beyond."
	_, wrong := runPDF(test, lay(test, made))
	for _, want := range []string{"→", "U+2192", "kind-1-quick"} {
		if !strings.Contains(wrong, want) {
			test.Fatalf("the refusal does not name %q: %s", want, wrong)
		}
	}
}

func TestAnAccentTheEncodingHasIsWrittenAndNotAQuestionMark(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Entry.Notes = "A café at 45°, a" +
		" dash – and “quotes”."
	raw := composeDocument(test, lay(test, made))
	if !strings.Contains(string(raw), `caf\351 at 45\260`) {
		test.Fatal("the accent and the degree sign are not in WinAnsi")
	}
	words := readSheets(test, raw)[coverSheets+firstName].words()
	want := "A café at 45°, a dash – and “quotes”."
	if !strings.Contains(words, want) {
		test.Fatalf("the page does not read back: %s", words)
	}
	if strings.Contains(words, "?") {
		test.Fatal("a question mark stands in for a character")
	}
}

func TestAFailedCardIsRefusedWithoutTheFlagAndPagedWithIt(test *testing.T) {
	made := testBook(twoCards)
	made.Cards[oneCard].Status = "failed"
	made.Cards[oneCard].Failure = "device lost"
	root := lay(test, made)
	if _, wrong := runPDF(test, root); !strings.Contains(wrong,
		"kind-1-quick") {
		test.Fatal("the refusal does not name the card:", wrong)
	}
	sheets := readSheets(test, composeDocument(test, root,
		"--with-failures"))
	if want := coverSheets + firstName + twoCards; len(sheets) != want {
		test.Fatalf("got %d sheets, want %d", len(sheets), want)
	}
	words := sheets[titleSheet(sheets, "kind-1")].words()
	for _, want := range []string{"kind-1", "did not render",
		"failed: device lost"} {
		if !strings.Contains(words, want) {
			test.Fatalf("the failure page lacks %q: %s", want, words)
		}
	}
	if !strings.Contains(sheets[noEntries].words(), "1 did not render") {
		test.Fatal("the cover does not count the card that failed")
	}
	if strings.Contains(sheets[titleSheet(sheets, "kind-0")].words(),
		"did not render") {
		test.Fatal("a rendered card says it did not render")
	}
}

func TestThePictureAndWordsOfARenderedCardAreNotAFailurePage(test *testing.T) {
	raw := composeDocument(test, writeTestBook(test, twoCards),
		"--with-failures")
	if strings.Count(string(raw), "/Subtype /Image") != twoCards {
		test.Fatal("the flag changed how rendered cards are laid")
	}
}

func TestEveryLaterSheetHasTheTitleAndItsNumberInTheFoot(test *testing.T) {
	// cards is defined as a few cards.
	const cards = 3
	sheets := readSheets(test, composeDocument(test,
		writeTestBook(test, cards)))
	if strings.Contains(sheets[noEntries].words(), "Page ") {
		test.Fatal("the cover carries a page number")
	}
	for index, each := range sheets[coverSheets:] {
		last := each.runs[len(each.runs)-firstName]
		want := fmt.Sprintf("Page %d", index+coverSheets+firstName)
		if last.text != want {
			test.Fatalf("sheet %d ends with %q, want %q", index+
				coverSheets+firstName, last.text, want)
		}
		if !strings.Contains(each.words(), bookTitle+" Page") {
			test.Fatal("the foot does not carry the book's title")
		}
	}
}

func TestTheCoverCarriesTheDateTheCountsAndTheProducer(test *testing.T) {
	root := writeTestBook(test, twoCards)
	drawn := time.Date(2026, time.November, 12, 11, 0, 0, 0, time.UTC)
	record := filepath.Join(root, bookRecord)
	if err := os.Chtimes(record, drawn, drawn); err != nil {
		test.Fatal(err)
	}
	cover := readSheets(test, composeDocument(test, root))[noEntries].words()
	for _, want := range []string{"Drawn 12 November 2026",
		"2 specimens in 1 groups", "tests (2)", "Compiled revision cafe",
		"feed"} {
		if !strings.Contains(cover, want) {
			test.Fatalf("the cover lacks %q: %s", want, cover)
		}
	}
	if strings.Contains(cover, "did not render") {
		test.Fatal("a finished book's cover says a card did not render")
	}
}

func TestProvenanceNamesTheBinaryAndTheDevice(test *testing.T) {
	made := testBook(twoCards)
	root := lay(test, made)
	record := `{"gpu-profile": {"Device": "Test device 9"}}`
	err := os.WriteFile(filepath.Join(root, "kind-0-quick", "render.json"),
		[]byte(record), artifactMode)
	if err != nil {
		test.Fatal(err)
	}
	sheets := readSheets(test, composeDocument(test, root))
	first := sheets[titleSheet(sheets, "kind-0")].words()
	second := sheets[titleSheet(sheets, "kind-1")].words()
	if !strings.Contains(first, "device: Test device 9.") ||
		!strings.Contains(first, "Binary feed") {
		test.Fatal("the provenance lacks the device or binary:", first)
	}
	if !strings.Contains(second, "device: "+unknownDevice+".") {
		test.Fatal("a card without a render record claims a device:",
			second)
	}
}

func TestReferencesAreLinksAndAnOutlineListsGroupsAndKinds(test *testing.T) {
	raw := string(composeDocument(test, writeTestBook(test, twoCards)))
	if strings.Count(raw, "/URI ("+testURL+")") < twoCards {
		test.Fatal("the references are not link annotations")
	}
	if !strings.Contains(raw, "/Type /Outlines") ||
		!strings.Contains(raw, "/PageMode /UseOutlines") {
		test.Fatal("the document has no outline")
	}
	for _, title := range []string{"Contents", "tests", "kind-0", "kind-1"} {
		if !strings.Contains(raw, "/Title "+hexTitle(title)) {
			test.Fatalf("the outline lacks %q", title)
		}
	}
	if !strings.Contains(raw, "/Dest [") {
		test.Fatal("the contents rows do not go to their sheets")
	}
}

func TestACardWithoutReferencesOrNotesHasNoEmptyHeadings(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Entry.References = nil
	made.Cards[noEntries].Entry.Notes = ""
	raw := composeDocument(test, lay(test, made))
	if strings.Contains(string(raw), "/URI") {
		test.Fatal("a card with no references has a link")
	}
	words := readSheets(test, raw)[coverSheets+firstName].words()
	if strings.Contains(words, "References") ||
		strings.Contains(words, "Recipe and source notes") {
		test.Fatal("empty sections have headings:", words)
	}
}

func TestALinkAddressOutsideASCIIIsRefused(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Entry.References[noEntries].URL =
		"https://example.org/café"
	_, wrong := runPDF(test, lay(test, made))
	if !strings.Contains(wrong, "percent-encoded") {
		test.Fatal("a non-ASCII address was not refused:", wrong)
	}
}

func TestContentsHeadingsAreNeverLeftAtTheFootOfASheet(test *testing.T) {
	// groups is defined as many one-card groups, so headings recur.
	const groups = 90
	var rows []contentsRow
	for index := range groups {
		rows = append(rows, contentsRow{group: fmt.Sprint("g", index)},
			contentsRow{card: index, lines: []string{"a"}})
	}
	pages := paginate(rows)
	if len(pages) < twoCards {
		test.Fatal("the rows fit one sheet")
	}
	for _, page := range pages {
		if page[len(page)-lastPlace].isHeading() {
			test.Fatal("a heading ends a sheet")
		}
	}
}

func TestAGroupNameWithAMissingCharacterIsRefusedByCard(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Entry.Group = "Tests \u2192"
	_, wrong := runPDF(test, lay(test, made))
	for _, want := range []string{"U+2192", "kind-0-quick"} {
		if !strings.Contains(wrong, want) {
			test.Fatalf("the refusal does not name %q: %s", want, wrong)
		}
	}
}

func TestACoverThatCannotFitOneSheetIsRefused(test *testing.T) {
	// groups is defined as far more groups than a cover can list.
	const groups = 400
	made := testBook(groups)
	for index := range made.Cards {
		made.Cards[index].Entry.Group = fmt.Sprintf("group-%d is a name"+
			" long enough to take room", index)
	}
	_, wrong := runPDF(test, lay(test, made))
	if !strings.Contains(wrong, "cover") {
		test.Fatal("an overfull cover was not refused:", wrong)
	}
}

func TestADetailIsNamedInTheContentsOnlyWhenTheBookHasMore(test *testing.T) {
	made := testBook(twoCards)
	one := readSheets(test, composeDocument(test, lay(test, made)))
	if strings.Contains(one[coverSheets].words(), "(quick)") {
		test.Fatal("a book of one detail names it on every row")
	}
	made.Cards[oneCard].Detail = "demo"
	made.Cards[oneCard].Folder = "kind-1-demo"
	two := readSheets(test, composeDocument(test, lay(test, made)))
	if !strings.Contains(two[coverSheets].words(), "kind-1 (demo)") {
		test.Fatal("two details are not told apart:",
			two[coverSheets].words())
	}
}

func TestABookWithoutAProducerIsComposedAndSaysUnknown(test *testing.T) {
	made := testBook(oneCard)
	made.Producer = nil
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	if strings.Contains(sheets[noEntries].words(), "Made by") {
		test.Fatal("the cover names a producer the book lacks")
	}
	if !strings.Contains(sheets[coverSheets+firstName].words(),
		"Compiled revision unknown") {
		test.Fatal("the provenance claims a revision it does not have")
	}
}

func TestACardsOwnBinaryIsNamedBeforeTheBooks(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Producer = &producer{BinarySHA256: "0ddba11",
		Revision: "beef"}
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	words := sheets[coverSheets+firstName].words()
	if !strings.Contains(words, "Binary 0ddba11") ||
		!strings.Contains(words, "Compiled revision beef") {
		test.Fatal("the card's own binary is not the one named:", words)
	}
}

func TestAHashTooWideToShareALineGoesUnderItsLabel(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Settings = strings.Repeat("ab", int(columnWidth/
		courier.measure("a", hashStyle.size)))
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	var label, value run
	for _, shown := range sheets[coverSheets+firstName].runs {
		if shown.text == "Settings" {
			label = shown
		}
		if shown.resource == courier.resource &&
			strings.HasPrefix(shown.text, "abab") {
			value = shown
		}
	}
	if value.x != margin || value.y >= label.y {
		test.Fatalf("the hash is not under its label: %v %v", label, value)
	}
}

func TestAKindTooLongForTheContentsRowWrapsInsteadOfCrossingTheNumber(
	test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Entry.Kind = strings.Repeat("long-kind ", 12)
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	for _, shown := range sheets[coverSheets].runs {
		assertInside(test, shown)
	}
}

func TestACardWithNoFailureTextStillSaysWhyItIsMissing(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Status = "pending"
	root := lay(test, made)
	sheets := readSheets(test, composeDocument(test, root,
		"--with-failures"))
	words := sheets[coverSheets+firstName].words()
	if !strings.Contains(words, "Reason: pending") ||
		!strings.Contains(sheets[coverSheets].words(),
			"(did not render)") {
		test.Fatal("a pending card is not explained:", words)
	}
}

func TestAnUnreadableRenderRecordIsRefusedByName(test *testing.T) {
	root := writeTestBook(test, oneCard)
	err := os.Mkdir(filepath.Join(root, "kind-0-quick", "render.json"),
		directoryMode)
	if err != nil {
		test.Fatal(err)
	}
	if _, wrong := runPDF(test, root); wrong == "" {
		test.Fatal("a render record that is a directory was accepted")
	}
}

func TestACardWithNoSettingsHashNamesNone(test *testing.T) {
	made := testBook(oneCard)
	made.Cards[noEntries].Settings = ""
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	if strings.Contains(sheets[coverSheets+firstName].words(),
		"Settings") {
		test.Fatal("a card without a hash shows one")
	}
}

func TestACutawayCardSaysSo(test *testing.T) {
	made := testBook(twoCards)
	made.Cards[noEntries].Cutaway = true
	sheets := readSheets(test, composeDocument(test, lay(test, made)))
	if !strings.Contains(sheets[coverSheets+firstName].words(),
		"Cutaway view: enclosure hidden.") ||
		strings.Contains(sheets[coverSheets+firstName+firstName].words(),
			"Cutaway") {
		test.Fatal("the cutaway note is missing or on the wrong card")
	}
}

func TestAReferenceAndItsAddressAreNotParted(test *testing.T) {
	made := &composer{doc: newDocument(bookTitle)}
	laid := newFlow(made.doc, made.doc.addPage(), "kind-0")
	// room is defined as one line's worth at the foot of the sheet.
	laid.cursor = textFloor + linkStyle.leading
	each := testBook(oneCard).Cards[noEntries]
	made.references(laid, each)
	if laid.err != nil {
		test.Fatal(laid.err)
	}
	var title, address *sheet
	for _, held := range made.doc.sheets {
		text := held.content.String()
		if strings.Contains(text, testTitle) {
			title = held
		}
		if strings.Contains(text, testURL) {
			address = held
		}
	}
	if title == nil || title != address {
		test.Fatal("the reference was parted from its address")
	}
}
