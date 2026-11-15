package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

// Design note. A destination is one of three things. It is not there:
// the book is new. It holds a book, which is a catalogue.json this
// command wrote (one that names its producer): the run resumes it. It is
// anything else: the run refuses it by name and touches nothing.
//
// Resuming keeps a card that is rendered under the same plan, by the
// plan's settings hash, and draws every other card the run selected.
// Cards of kinds the menu no longer has are dropped from the index and
// said; their folders are left where they are. Cards the run did not
// select are left as the book has them. Each card keeps the record of
// the binary that drew it.

// The statuses a card has besides renderedStatus, which pdf.go defines.
const (
	pendingStatus = "pending"
	failedStatus  = "failed"
)

// drawer draws one specimen into a book's folder. The native one reaches
// Metal; the tests stand a mock in its place, so the book's bookkeeping
// is tried without a GPU.
type drawer interface {
	draw(settings plan, destination string, view *card,
		maker *producer) error
}

// nativeDrawer draws on the native Metal path.
type nativeDrawer struct{}

// draw is plan.draw, which has the plan for its receiver.
func (nativeDrawer) draw(settings plan, destination string, view *card,
	maker *producer,
) error {
	return settings.draw(destination, view, maker)
}

// hash is the SHA-256 of the plan's own bytes, the same hash a card's
// receipt records as its settings.
func (plan plan) hash() string {
	return fmt.Sprintf("%x", sha256.Sum256(plan.raw))
}

// drawnUnder says whether the card is already drawn, and under this plan.
func (view card) drawnUnder(settings string) bool {
	return view.Status == renderedStatus && view.Settings == settings
}

// open makes the destination ready and returns the folders of the cards
// this run selected. A new destination gets a new book; a book is
// resumed; anything else is refused with nothing touched.
func (fresh *book) open(said io.Writer, destination string,
	settings *plan,
) (map[string]bool, error) {
	selected := map[string]bool{}
	for _, view := range fresh.Cards {
		selected[view.Folder] = true
	}
	previous, err := existingBook(destination)
	if err != nil {
		return nil, err
	}
	if fresh.Producer, err = compiledProducer(); err != nil {
		return nil, err
	}
	if previous == nil {
		return selected, fresh.start(destination)
	}
	menu, err := readMenu()
	if err != nil {
		return nil, err
	}
	dropped, kept := fresh.resume(previous, menu, settings.hash())
	_, err = fmt.Fprintf(said, "resuming %s: %d cards kept, %d dropped\n",
		destination, kept, len(dropped))
	for _, folder := range dropped {
		if err == nil {
			_, err = fmt.Fprintf(said,
				"dropped %s: its kind is no longer in the menu\n", folder)
		}
	}
	return selected, err
}

// start creates a new book in a destination that is not there.
func (fresh *book) start(destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination),
		directoryMode); err != nil {
		return err
	}
	if err := os.Mkdir(destination, directoryMode); err != nil {
		return err
	}
	if err := writeIcons(destination); err != nil {
		return err
	}
	return fresh.write(destination)
}

// existingBook is nil for a destination that is not there, the book a
// destination holds, or a refusal naming what the destination is.
func existingBook(destination string) (*book, error) {
	if _, err := os.Stat(destination); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	var held book
	raw, err := os.ReadFile(filepath.Join(destination, "catalogue.json"))
	if err == nil {
		err = json.Unmarshal(raw, &held)
	}
	if err != nil || held.Producer == nil {
		return nil, fmt.Errorf("%s exists and is not a catalogue book"+
			" (it has no catalogue.json this command wrote); nothing was"+
			" changed; give --out a new directory, or a book to resume",
			destination)
	}
	return &held, nil
}

// resume folds a previous book into this run's cards. A card the book
// holds rendered under settings is kept; one the run selected and the
// book does not hold so is left pending to be drawn; a card the run did
// not select stays as the book has it; a card of a kind the menu no
// longer has is dropped. It returns the folders dropped and the count
// kept.
func (fresh *book) resume(previous *book, offered *menu,
	settings string) ([]string, int) {
	order := map[string]int{}
	for position, entry := range offered.Entries {
		order[entry.Kind] = position
	}
	wanted := map[string]card{}
	for _, view := range fresh.Cards {
		wanted[view.Folder] = view
	}
	var merged []card
	var dropped []string
	kept := noEntries
	for _, old := range previous.Cards {
		_, current := order[old.Entry.Kind]
		redraw, selected := wanted[old.Folder]
		switch {
		case !current:
			dropped = append(dropped, old.Folder)
		case selected && !old.drawnUnder(settings):
			merged = append(merged, redraw)
			delete(wanted, old.Folder)
		default:
			if selected {
				kept++
			}
			merged = append(merged, old)
			delete(wanted, old.Folder)
		}
	}
	for _, view := range fresh.Cards {
		if _, added := wanted[view.Folder]; added {
			merged = append(merged, view)
		}
	}
	slices.SortStableFunc(merged, func(left, right card) int {
		return order[left.Entry.Kind] - order[right.Entry.Kind]
	})
	fresh.Cards = merged
	return dropped, kept
}
