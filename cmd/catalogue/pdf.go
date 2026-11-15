package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
)

// renderedStatus is defined by render: the one status a finished
// book's cards carry.
const renderedStatus = "native GPU render"

// bookRecord is defined by render: the file that holds a book's cards.
const bookRecord = "catalogue.json"

const pdfHelp = `# pdf

Compose a finished specimen book into one PDF, to send to someone who
has not got the book. It holds a cover, with the date the book was
drawn, the counts and the binary that drew it; a contents page with
each specimen's page number; then one page for each specimen, in the
menu's order. A specimen's page has its picture across the page, its
group and kind, the description, the recipe and source notes, the
references as links, and where the picture came from: the settings
hash, the binary and the device. Notes too long for one page run on to
the next. The reader's sidebar lists the groups and kinds.

The book is what render wrote, and the date is when its record was
last written. A book holding pending or failed cards is refused, with
the cards named; render it to completion first, or give --with-failures
to compose it anyway, with a page for each card that did not render,
saying why. A character the PDF's fonts lack is refused, naming it
and the card. The destination must be new.
`

// pdf composes the book named by --book into the one file named by
// --out, newly.
func (app application) pdf(call pzm.Call) error {
	if err := app.context.Err(); err != nil {
		return err
	}
	root := filepath.Join("out", "catalogue")
	if call.Given("book") {
		root = call.Value("book")
	}
	loaded, err := readBook(root, call.On("with-failures"))
	if err != nil {
		return err
	}
	destination := filepath.Join("out", "catalogue.pdf")
	if call.Given("out") {
		destination = call.Value("out")
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "%d cards -> %s\n",
			len(loaded.Cards), destination)
		return err
	}
	return app.write(call, loaded, root, destination)
}

// write composes the book and writes the document to a new file.
func (app application) write(call pzm.Call, loaded *book, root,
	destination string) error {
	info, err := os.Stat(filepath.Join(root, bookRecord))
	if err != nil {
		return err
	}
	composed, err := compose(loaded, root, info.ModTime(),
		call.On("with-failures"))
	if err != nil {
		return err
	}
	raw, err := composed.assemble()
	if err != nil {
		return err
	}
	made, err := os.OpenFile(destination,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL, artifactMode)
	if err != nil {
		return err
	}
	if _, err := made.Write(raw); err != nil {
		made.Close()
		return err
	}
	if err := made.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(call.Out(), destination)
	return err
}

// readBook refuses a book with no cards, and unless failures are
// allowed, a book with any card not rendered: the pdf is for sending
// someone, and a partial book is not a full catalogue.
func readBook(root string, failures bool) (*book, error) {
	raw, err := os.ReadFile(filepath.Join(root, bookRecord))
	if err != nil {
		return nil, err
	}
	var loaded book
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, err
	}
	if len(loaded.Cards) == noEntries {
		return nil, fmt.Errorf("%s holds no cards", root)
	}
	var unfinished []string
	for _, card := range loaded.Cards {
		if card.Status != renderedStatus {
			unfinished = append(unfinished, card.unfinished())
		}
	}
	if len(unfinished) > noEntries && !failures {
		return nil, fmt.Errorf("%s is not finished: %s; run catalogue"+
			" render --out %s to finish it before composing it, or give"+
			" --with-failures to compose it with a page for each card"+
			" that did not render", root, strings.Join(unfinished, "; "),
			root)
	}
	return &loaded, nil
}

// unfinished names a card that is not rendered, with its reason if it
// failed.
func (view card) unfinished() string {
	if view.Failure == "" {
		return view.Folder + " is " + view.Status
	}
	return view.Folder + " is " + view.Status + " (" + view.Failure + ")"
}
