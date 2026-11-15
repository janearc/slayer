//go:build draw

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pending is the card of one entry at quick detail, not yet drawn.
func pending(test *testing.T, kind string) card {
	test.Helper()
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{kind})
	if err != nil {
		test.Fatal(err)
	}
	return card{Entry: entries[0], Detail: "quick", Folder: kind + "-quick",
		Status: "pending"}
}

// A drawn specimen that covers less of its picture than the plan asks,
// here more than all of it, is refused and leaves no folder; and one
// drawn for a book that is not there has nowhere to go.
func TestADrawnSpecimenMustBeSeenAndHaveABookToGoIn(test *testing.T) {
	root := test.TempDir()
	unseen := pending(test, "webb")
	strict := *defaultPlan(test)
	strict.MinCoverage = 2
	if err := strict.draw(root, &unseen, nil); err == nil ||
		!strings.Contains(err.Error(), "coverage") {
		test.Errorf("a specimen was seen past all of its picture: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, unseen.Folder)); !os.IsNotExist(err) {
		test.Error("an unseen specimen has a folder", err)
	}
	homeless := pending(test, "webb")
	if err := defaultPlan(test).draw(filepath.Join(root, "no-such-book"),
		&homeless, nil); err == nil {
		test.Error("a specimen was drawn into a book that is not there")
	}
}

// stopping is somewhere that takes what is written and stops the run
// as it does.
type stopping struct{ stop context.CancelFunc }

// A run that cannot say the card it has just drawn stops there, with
// the card drawn. A run cancelled after its first card draws no second:
// the first has its folder and the second has none.
func TestRenderStopsWhenItCannotSayACardOrIsCancelled(test *testing.T) {
	root := test.TempDir()
	unsaid := filepath.Join(root, "unsaid")
	var wrong bytes.Buffer
	if status := tool(context.Background()).Run([]string{"render", "webb",
		"--out", unsaid}, refusing{}, &wrong); status == successfulExit {
		test.Error("a run that could not say its card went on")
	}
	if _, err := os.Stat(filepath.Join(unsaid, "webb-quick", "specimen.png")); err != nil {
		test.Error("the card it could not say was not drawn", err)
	}
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	two, err := menu.selectEntries([]string{"webb", "fern"})
	if err != nil {
		test.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	cancelled := filepath.Join(root, "cancelled")
	if status := tool(ctx).Run([]string{"render", "webb", "fern", "--out",
		cancelled}, stopping{stop}, &wrong); status == successfulExit {
		test.Error("a cancelled run went on to its end")
	}
	first := filepath.Join(cancelled, two[0].Kind+"-quick")
	second := filepath.Join(cancelled, two[1].Kind+"-quick")
	if _, err := os.Stat(first); err != nil {
		test.Error("the card before the cancel was not drawn", err)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		test.Error("a card was drawn after the cancel", err)
	}
}

// Write takes every byte and cancels.
func (heard stopping) Write(said []byte) (int, error) {
	heard.stop()
	return len(said), nil
}
