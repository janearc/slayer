package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

// drawn is what a run said and how it ended.
type drawn struct {
	status      int
	said, wrong string
}

// runDrawing runs the tool with a drawer in place of Metal.
func runDrawing(test *testing.T, drawing drawer, args ...string) drawn {
	test.Helper()
	var said, wrong bytes.Buffer
	status := toolDrawing(context.Background(), drawing).Run(args, &said,
		&wrong)
	return drawn{status, said.String(), wrong.String()}
}

// drawsEvery expects the given number of draws, each making the folder a
// real draw makes, and failing the kinds named with their reason.
func drawsEvery(test *testing.T, times int, failing map[string]string,
) *Mockdrawer {
	test.Helper()
	made := NewMockdrawer(gomock.NewController(test))
	made.EXPECT().draw(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any()).DoAndReturn(func(_ plan, destination string,
		view *card, _ *producer) error {
		if why, broken := failing[view.Entry.Kind]; broken {
			return errors.New(why)
		}
		return os.Mkdir(filepath.Join(destination, view.Folder),
			directoryMode)
	}).Times(times)
	return made
}

// loadBook reads the book a run left.
func loadBook(test *testing.T, root string) book {
	test.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "catalogue.json"))
	if err != nil {
		test.Fatal(err)
	}
	var held book
	if err := json.Unmarshal(raw, &held); err != nil {
		test.Fatal(err)
	}
	return held
}

// cardOf is the book's card for a kind.
func cardOf(test *testing.T, held book, kind string) card {
	test.Helper()
	for _, view := range held.Cards {
		if view.Entry.Kind == kind {
			return view
		}
	}
	test.Fatalf("the book has no card for %s", kind)
	return card{}
}

// A specimen that fails does not stop the next: both are tried, the
// failed card says why, the other is rendered, the run says the failure
// on its last lines and exits non-zero. The anti-test: a run with no
// failure exits zero and says none.
func TestAFailedSpecimenDoesNotStopTheNext(test *testing.T) {
	book := filepath.Join(test.TempDir(), "book")
	got := runDrawing(test, drawsEvery(test, twoCards,
		map[string]string{"webb": "no light came"}),
		"render", "webb", "fern", "--out", book)
	if got.status == successfulExit {
		test.Fatal("a run with a failure exited zero")
	}
	if !strings.Contains(got.wrong, "webb-quick: no light came") {
		test.Errorf("the run did not name the failure: %q", got.wrong)
	}
	held := loadBook(test, book)
	if failed := cardOf(test, held, "webb"); failed.Status != failedStatus ||
		failed.Failure != "no light came" || failed.Settings != "" {
		test.Errorf("webb's card: %+v", failed)
	}
	if kept := cardOf(test, held, "fern"); kept.Status != renderedStatus {
		test.Errorf("the specimen after the failure was not drawn: %+v", kept)
	}
	clean := filepath.Join(test.TempDir(), "clean")
	got = runDrawing(test, drawsEvery(test, twoCards, nil),
		"render", "webb", "fern", "--out", clean)
	if got.status != successfulExit || got.wrong != "" {
		test.Errorf("a run with no failure: %d, %q", got.status, got.wrong)
	}
}

// A second run keeps a card rendered under the same plan and draws a
// card that failed. The anti-test: the kept card is not drawn again,
// which the mock enforces by allowing one draw only.
func TestASecondRunDrawsOnlyWhatIsNotRendered(test *testing.T) {
	book := filepath.Join(test.TempDir(), "book")
	runDrawing(test, drawsEvery(test, twoCards,
		map[string]string{"webb": "no light came"}),
		"render", "webb", "fern", "--out", book)
	got := runDrawing(test, drawsEvery(test, oneCard, nil),
		"render", "webb", "fern", "--out", book)
	if got.status != successfulExit {
		test.Fatalf("the resume failed: %q", got.wrong)
	}
	if !strings.Contains(got.said, "1 cards kept") {
		test.Errorf("the resume did not say what it kept: %q", got.said)
	}
	held := loadBook(test, book)
	for _, kind := range []string{"webb", "fern"} {
		if view := cardOf(test, held, kind); view.Status != renderedStatus ||
			view.Failure != "" {
			test.Errorf("%s after the resume: %+v", kind, view)
		}
	}
	if len(held.Cards) != twoCards {
		test.Errorf("the resume holds %d cards", len(held.Cards))
	}
}

// A card drawn under another plan is drawn again; the anti-test is the
// same plan, which draws nothing.
func TestACardDrawnUnderAnotherPlanIsDrawnAgain(test *testing.T) {
	book := filepath.Join(test.TempDir(), "book")
	runDrawing(test, drawsEvery(test, oneCard, nil),
		"render", "webb", "--out", book)
	before := cardOf(test, loadBook(test, book), "webb").Settings
	other := altered(test, "face-limit: 1500000", "face-limit: 1400000")
	got := runDrawing(test, drawsEvery(test, oneCard, nil),
		"render", "webb", "--plan", other, "--out", book)
	after := cardOf(test, loadBook(test, book), "webb").Settings
	if got.status != successfulExit || before == "" || after == before {
		test.Errorf("the card was not drawn under the new plan: %q, %q, %q",
			got.wrong, before, after)
	}
	runDrawing(test, drawsEvery(test, noEntries, nil),
		"render", "webb", "--plan", other, "--out", book)
}

// A kind the menu does not have is dropped from the index and said, its
// folder left; a kind the book lacks is added and drawn. The anti-test:
// a kind the run did not name, still in the menu, stays as it is.
func TestResumeDropsWhatTheMenuLostAndAddsWhatItGained(test *testing.T) {
	book := filepath.Join(test.TempDir(), "book")
	runDrawing(test, drawsEvery(test, twoCards, nil),
		"render", "webb", "fern", "--out", book)
	held := loadBook(test, book)
	gone := card{Entry: entry{Kind: "zebra", Group: "Tests"},
		Detail: "quick", Folder: "zebra-quick", Status: renderedStatus}
	held.Cards = []card{held.Cards[0], gone}
	if err := writeJSON(filepath.Join(book, "catalogue.json"),
		held); err != nil {
		test.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(book, gone.Folder),
		directoryMode); err != nil {
		test.Fatal(err)
	}
	only := held.Cards[0].Entry.Kind
	added := "fern"
	if only == "fern" {
		added = "webb"
	}
	got := runDrawing(test, drawsEvery(test, oneCard, nil),
		"render", added, "--out", book)
	if got.status != successfulExit ||
		!strings.Contains(got.said, "dropped zebra-quick") {
		test.Fatalf("the resume: %d, %q, %q", got.status, got.said, got.wrong)
	}
	after := loadBook(test, book)
	if len(after.Cards) != twoCards {
		test.Errorf("the book holds %d cards: %+v", len(after.Cards), after)
	}
	if view := cardOf(test, after, only); view.Status != renderedStatus {
		test.Errorf("the card not named was changed: %+v", view)
	}
	if _, err := os.Stat(filepath.Join(book, gone.Folder)); err != nil {
		test.Error("a dropped card's folder was removed", err)
	}
}

// Each card keeps the record of the binary that drew it: a card kept
// from another binary keeps that binary's, and a card drawn now has this
// one's.
func TestEachCardKeepsTheBinaryThatDrewIt(test *testing.T) {
	book := filepath.Join(test.TempDir(), "book")
	runDrawing(test, drawsEvery(test, twoCards,
		map[string]string{"fern": "no light came"}),
		"render", "webb", "fern", "--out", book)
	held := loadBook(test, book)
	for index := range held.Cards {
		if held.Cards[index].Entry.Kind == "webb" {
			held.Cards[index].Producer = &producer{BinarySHA256: "elder"}
		}
	}
	if err := writeJSON(filepath.Join(book, "catalogue.json"),
		held); err != nil {
		test.Fatal(err)
	}
	runDrawing(test, drawsEvery(test, oneCard, nil),
		"render", "webb", "fern", "--out", book)
	after := loadBook(test, book)
	if got := cardOf(test, after, "webb").Producer; got == nil ||
		got.BinarySHA256 != "elder" {
		test.Errorf("webb lost the binary that drew it: %+v", got)
	}
	now, err := compiledProducer()
	if got := cardOf(test, after, "fern").Producer; err != nil ||
		got == nil || got.BinarySHA256 != now.BinarySHA256 {
		test.Errorf("fern does not carry this binary: %+v, %v", got, err)
	}
}

// A destination that is not a book is refused by name and left as it
// was, whatever it is: a folder of other files, one with a
// catalogue.json this command did not write, a file. The mock allows no
// draw at all.
func TestADestinationThatIsNotABookIsRefusedAndUntouched(test *testing.T) {
	root := test.TempDir()
	stranger := filepath.Join(root, "stranger")
	if err := os.Mkdir(stranger, directoryMode); err != nil {
		test.Fatal(err)
	}
	foreign := filepath.Join(root, "foreign")
	if err := os.Mkdir(foreign, directoryMode); err != nil {
		test.Fatal(err)
	}
	alone := filepath.Join(root, "a-file")
	for place, name := range map[string]string{stranger: "keep.txt",
		foreign: "catalogue.json", alone: ""} {
		path := place
		if name != "" {
			path = filepath.Join(place, name)
		}
		if err := os.WriteFile(path, []byte(`{"cards": []}`),
			artifactMode); err != nil {
			test.Fatal(err)
		}
	}
	for _, place := range []string{stranger, foreign, alone} {
		got := runDrawing(test, drawsEvery(test, noEntries, nil),
			"render", "webb", "--out", place)
		if got.status == successfulExit ||
			!strings.Contains(got.wrong, place+" exists and is not a catalogue"+
				" book") {
			test.Errorf("%s was not refused by name: %q", place, got.wrong)
		}
	}
	for _, place := range []string{stranger, foreign} {
		entries, err := os.ReadDir(place)
		if err != nil || len(entries) != singleArtifact {
			test.Errorf("%s was touched: %v, %v", place, entries, err)
		}
	}
	if raw, err := os.ReadFile(alone); err != nil ||
		string(raw) != `{"cards": []}` {
		test.Error("the file was changed", err)
	}
}

// pdf refuses a book that is not finished and names every card that is
// not, with the reason a failed one gives. The anti-test is a finished
// book, which it takes.
func TestPDFNamesTheCardsThatAreNotFinished(test *testing.T) {
	root := writeTestBook(test, twoCards)
	held := loadBook(test, root)
	held.Cards[0].Status, held.Cards[0].Failure = failedStatus, "no light"
	held.Cards[1].Status = pendingStatus
	if err := writeJSON(filepath.Join(root, "catalogue.json"),
		held); err != nil {
		test.Fatal(err)
	}
	_, err := readBook(root, false)
	for _, want := range []string{"kind-0-quick is failed (no light)",
		"kind-1-quick is pending", "catalogue render --out " + root} {
		if err == nil || !strings.Contains(err.Error(), want) {
			test.Errorf("the refusal lacks %q: %v", want, err)
		}
	}
	if _, err := readBook(writeTestBook(test, twoCards), false); err != nil {
		test.Error("a finished book was refused", err)
	}
}
