package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// oneCard and twoCards are defined as the smallest books the refusal
// tests need.
const oneCard, twoCards = 1, 2

// once is defined as the first occurrence alone.
const once = 1

// testPictureWide and testPictureHigh are defined as the book's
// frame in the plan, four columns to three rows, at its smallest.
const testPictureWide, testPictureHigh = 4, 3

// opaque and transparent are defined as a channel's full and empty.
const opaque, transparent = 255, 0

// backdrop is defined as the book's black backdrop, in every channel.
const backdrop = 0

// redAt, greenAt and blueAt are defined as the channel order flatten
// writes.
const redAt, greenAt, blueAt = 0, 1, 2

// firstMatch is defined by regexp submatches: the whole match stands
// at zero, the first group after it.
const firstMatch = 1

// testAbout is a short specimen description that fits a few lines.
const testAbout = "A specimen made for the composition tests."

// testNotes and testTitle are words with a recognisable shape, so a
// test can find them again in a page's text.
const (
	testNotes = "Requested aperture and the authored proportions."
	testTitle = "A reference title"
	testURL   = "https://example.org/reference"
)

// testBook is a finished book of cards, each with notes and one
// reference, ready to be written with its pictures.
func testBook(cards int) book {
	made := book{Producer: &producer{BinarySHA256: "feed",
		Revision: "cafe", Modified: "false", GoVersion: "go0"}}
	for index := range cards {
		made.Cards = append(made.Cards, card{
			Entry: entry{Kind: fmt.Sprintf("kind-%d", index),
				Group: "tests", About: testAbout, Notes: testNotes,
				Details: []string{"quick", "demo"},
				References: []reference{{Title: testTitle +
					fmt.Sprint(index), URL: testURL}}},
			Detail: "quick",
			Folder: fmt.Sprintf("kind-%d-quick", index),
			Status: renderedStatus, Settings: "5e771ed5"})
	}
	return made
}

// lay writes a book's record and the pictures of its rendered cards
// through the production types, so the fixture cannot drift from what
// render actually writes.
func lay(test *testing.T, made book) string {
	test.Helper()
	root := test.TempDir()
	for _, each := range made.Cards {
		folder := filepath.Join(root, each.Folder)
		if err := os.Mkdir(folder, directoryMode); err != nil {
			test.Fatal(err)
		}
		if each.Status == renderedStatus {
			writeTestPicture(test, filepath.Join(folder, "specimen.png"))
		}
	}
	err := writeJSON(filepath.Join(root, bookRecord), made)
	if err != nil {
		test.Fatal(err)
	}
	return root
}

// writeTestBook lays a finished book of cards.
func writeTestBook(test *testing.T, cards int) string {
	test.Helper()
	return lay(test, testBook(cards))
}

// writeTestPicture writes the smallest useful specimen: four pixels.
func writeTestPicture(test *testing.T, path string) {
	test.Helper()
	picture := image.NewNRGBA(image.Rect(noEntries, noEntries,
		testPictureWide, testPictureHigh))
	picture.SetNRGBA(noEntries, noEntries,
		color.NRGBA{R: opaque, A: opaque})
	var raw bytes.Buffer
	if err := png.Encode(&raw, picture); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(path, raw.Bytes(), artifactMode); err != nil {
		test.Fatal(err)
	}
}

// runPDF runs the pdf verb and returns the document, or the refusal's
// words when it refuses.
func runPDF(test *testing.T, root string, flags ...string) ([]byte, string) {
	test.Helper()
	out := filepath.Join(test.TempDir(), "catalogue.pdf")
	args := append([]string{"pdf", "--book", root, "--out", out}, flags...)
	var said, wrong bytes.Buffer
	status := tool(context.Background()).Run(args, &said, &wrong)
	if status != successfulExit {
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			test.Fatal("a refused composition left a document", err)
		}
		return nil, wrong.String()
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		test.Fatal(err)
	}
	return raw, ""
}

// composeDocument runs the pdf verb against a book and returns the
// bytes, failing if it refuses.
func composeDocument(test *testing.T, root string, flags ...string) []byte {
	test.Helper()
	raw, wrong := runPDF(test, root, flags...)
	if raw == nil {
		test.Fatal("pdf refused a finished book", wrong)
	}
	return raw
}

func TestPDFComposesEveryRenderedCard(test *testing.T) {
	// paginated is defined as enough cards for several sheets.
	const paginated = 5
	raw := composeDocument(test, writeTestBook(test, paginated))
	if !bytes.HasPrefix(raw, []byte("%PDF-")) {
		test.Fatal("output does not begin as a PDF")
	}
	if !bytes.HasSuffix(bytes.TrimSpace(raw), []byte("%%EOF")) {
		test.Fatal("output does not end as a PDF")
	}
	images := bytes.Count(raw, []byte("/Subtype /Image"))
	if images != paginated {
		test.Fatalf("composed %d pictures, want %d", images, paginated)
	}
}

func TestEveryCrossReferenceOffsetLandsOnItsObject(test *testing.T) {
	// paginated is defined as enough cards for several sheets.
	const paginated = 5
	raw := composeDocument(test, writeTestBook(test, paginated))
	tail := regexp.MustCompile(`startxref\n(\d+)\n%%EOF`).
		FindSubmatch(raw)
	if tail == nil {
		test.Fatal("document has no startxref")
	}
	start, err := strconv.Atoi(string(tail[firstMatch]))
	if err != nil || !bytes.HasPrefix(raw[start:], []byte("xref")) {
		test.Fatal("startxref does not reach the table", err)
	}
	entries := regexp.MustCompile(`(?m)^(\d{10}) 00000 n`).
		FindAllSubmatch(raw[start:], -1)
	if len(entries) == noEntries {
		test.Fatal("the table holds no objects")
	}
	for position, entry := range entries {
		offset, err := strconv.Atoi(string(entry[firstMatch]))
		if err != nil {
			test.Fatal(err)
		}
		opening := fmt.Sprintf("%d 0 obj", position+firstName)
		if !bytes.HasPrefix(raw[offset:], []byte(opening)) {
			test.Fatalf("offset %d misses %q", offset, opening)
		}
	}
}

func TestPDFRefusesAnUnfinishedBook(test *testing.T) {
	made := testBook(twoCards)
	made.Cards[oneCard].Status = "failed"
	made.Cards[oneCard].Failure = "no GPU"
	_, wrong := runPDF(test, lay(test, made))
	for _, want := range []string{made.Cards[oneCard].Folder, "failed",
		"no GPU", "--with-failures"} {
		if !strings.Contains(wrong, want) {
			test.Fatalf("the refusal does not say %q: %s", want, wrong)
		}
	}
}

func TestPDFNeverOverwritesAnExistingDocument(test *testing.T) {
	root := writeTestBook(test, oneCard)
	out := filepath.Join(test.TempDir(), "keep.pdf")
	expected := []byte("keep this document")
	if err := os.WriteFile(out, expected, artifactMode); err != nil {
		test.Fatal(err)
	}
	refuseInvocation(test, out, "pdf", "--book", root, "--out", out)
	actual, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(actual, expected) {
		test.Fatal("existing document changed", err)
	}
}

func TestPDFRefusesABookWithoutItsRecord(test *testing.T) {
	out := filepath.Join(test.TempDir(), "catalogue.pdf")
	refuseInvocation(test, out,
		"pdf", "--book", test.TempDir(), "--out", out)
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		test.Fatal("a recordless book composed a document", err)
	}
}

func TestPDFRefusesACardWithoutItsPicture(test *testing.T) {
	root := writeTestBook(test, twoCards)
	gone := filepath.Join(root, "kind-1-quick", "specimen.png")
	if err := os.Remove(gone); err != nil {
		test.Fatal(err)
	}
	if _, wrong := runPDF(test, root); wrong == "" {
		test.Fatal("a pictureless rendered card composed a document")
	}
}

func TestCancelledPDFWritesNoDocument(test *testing.T) {
	root := writeTestBook(test, oneCard)
	ctx, stop := context.WithCancel(context.Background())
	stop()
	out := filepath.Join(test.TempDir(), "catalogue.pdf")
	var said, wrong bytes.Buffer
	status := tool(ctx).Run(
		[]string{"pdf", "--book", root, "--out", out}, &said, &wrong)
	if status == successfulExit {
		test.Fatal("cancelled composition succeeded")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		test.Fatal("cancelled invocation wrote a document", err)
	}
}

func TestPDFRefusesAnEmptyBook(test *testing.T) {
	root := test.TempDir()
	record := `{"cards": [], "producer": null}`
	err := os.WriteFile(filepath.Join(root, bookRecord),
		[]byte(record), artifactMode)
	if err != nil {
		test.Fatal(err)
	}
	if _, wrong := runPDF(test, root); wrong == "" {
		test.Fatal("an empty book composed a document")
	}
}

func TestPDFRefusesAGarbledRecord(test *testing.T) {
	root := test.TempDir()
	err := os.WriteFile(filepath.Join(root, bookRecord),
		[]byte("not a record {"), artifactMode)
	if err != nil {
		test.Fatal(err)
	}
	if _, wrong := runPDF(test, root); wrong == "" {
		test.Fatal("a garbled record composed a document")
	}
}

func TestPDFRefusesAGarbledRenderRecord(test *testing.T) {
	root := writeTestBook(test, oneCard)
	err := os.WriteFile(filepath.Join(root, "kind-0-quick", "render.json"),
		[]byte("{ not json"), artifactMode)
	if err != nil {
		test.Fatal(err)
	}
	_, wrong := runPDF(test, root)
	if !strings.Contains(wrong, "kind-0-quick") {
		test.Fatal("a garbled render record was not refused by name:",
			wrong)
	}
}

func TestTransparentPixelsFlattenOntoTheBackdrop(test *testing.T) {
	// onePixel is defined as all the picture this test reads.
	const onePixel = 1
	picture := image.NewNRGBA(image.Rect(noEntries, noEntries,
		onePixel, onePixel))
	picture.SetNRGBA(noEntries, noEntries,
		color.NRGBA{R: opaque, G: opaque, B: opaque, A: transparent})
	flat := flatten(picture)
	if flat[redAt] != backdrop || flat[greenAt] != backdrop ||
		flat[blueAt] != backdrop {
		test.Fatal("a clear pixel did not flatten onto the backdrop")
	}
	// mixedRed, mixedGreen and mixedBlue are defined as a picked
	// colour that tells the channels apart.
	const mixedRed, mixedGreen, mixedBlue = 200, 100, 50
	mixed := color.NRGBA{R: mixedRed, G: mixedGreen, B: mixedBlue,
		A: opaque}
	picture.SetNRGBA(noEntries, noEntries, mixed)
	flat = flatten(picture)
	if flat[redAt] != mixed.R || flat[greenAt] != mixed.G ||
		flat[blueAt] != mixed.B {
		test.Fatal("an opaque pixel did not survive flattening")
	}
}

func TestTextDelimitersAreEscapedIntoTheStream(test *testing.T) {
	// anySize and atOrigin are defined as a text size and a place for
	// tests that only read the stream back.
	const anySize, atOrigin = 12.0, 0.0
	composed := newDocument("t")
	sheet := composed.addPage()
	err := sheet.text(helvetica, anySize, inkBody, atOrigin, atOrigin,
		`a (parenthetical\) caption`)
	if err != nil {
		test.Fatal(err)
	}
	raw, err := composed.assemble()
	if err != nil {
		test.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`a \(parenthetical\\\) caption`)) {
		test.Fatal("delimiters reached the stream unescaped")
	}
}

func Example_pdf() {
	tool(context.Background()).Run([]string{"pdf",
		"--book", filepath.Join("testdata", "book"), "--dry-run"},
		os.Stdout, os.Stderr)
	// Output:
	// 2 cards -> out/catalogue.pdf
}
