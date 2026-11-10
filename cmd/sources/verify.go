package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/sources"
)

const verifyPage = `# verify

holds every measured number in slayer's research data to its source:
the source pinned in the list and on the shelf at its pin, the text the
data quotes on the page it names, and the value the data holds the one
that text says. a number that fails any of them fails the verb, and the
build that runs it first. a paged source, a pdf, is read a page at a
time by poppler's pdftotext; it reaches nothing outside and writes
nothing.

this is a text-and-value check. it does not infer units, range endpoints,
coefficient roles or powers from prose. those claims need source review
and the model's independent checks.`

// pdfSuffix is how a paged source's file is named.
const pdfSuffix = ".pdf"

// doubled is the bits a float64 holds, as strconv asks to be told.
const doubled = 64

// verify holds every measured number to its source, and says each.
func (at place) verify(call pzm.Call) error {
	list, err := sources.Read(at.list)
	if err != nil {
		return err
	}
	data, err := at.research()
	if err != nil {
		return err
	}
	var failed []error
	for _, number := range data {
		if err := at.verified(list, number); err != nil {
			failed = append(failed, err)
			continue
		}
		from := *number.From
		if _, err := fmt.Fprintf(call.Out(), "%s: %s, %s\n", number.Name, from.Text, whereIn(from)); err != nil {
			return errors.Join(append(failed, err)...)
		}
	}
	return errors.Join(failed...)
}

// unpaged is the page of a source that is not paged: none.
const unpaged = 0

// whereIn is where a number was found, as verify says it.
func whereIn(from numbers.Source) string {
	if from.Page == unpaged {
		return "in " + from.Key
	}
	return fmt.Sprintf("on page %d of %s", from.Page, from.Key)
}

// verified is nothing when a number's source is on the shelf as pinned,
// its page holds the text, and the text says the number; otherwise what
// is wrong.
func (at place) verified(list sources.List, number numbers.Number) error {
	if number.From == nil {
		return fmt.Errorf("%s: no source is named", number.Name)
	}
	from := *number.From
	source, found := list.Find(from.Key)
	if !found || source.Never() != "" {
		return fmt.Errorf("%s: %s is not a source on the shelf",
			number.Name, from.Key)
	}
	if _, err := at.shelf.Check(source); err != nil {
		return fmt.Errorf("%s: %w", number.Name, err)
	}
	page, err := pageOf(at.shelf.Path(source), from.Page)
	if err != nil {
		return fmt.Errorf("%s: %w", number.Name, err)
	}
	if from.Excerpt != "" {
		if !strings.Contains(page, from.Excerpt) {
			return fmt.Errorf("%s: source excerpt %q is not %s", number.Name, from.Excerpt, whereIn(from))
		}
		page = from.Excerpt
	}
	if !strings.Contains(page, from.Text) {
		return fmt.Errorf("%s: %q is not %s", number.Name, from.Text,
			whereIn(from))
	}
	said, err := strconv.ParseFloat(from.Text, doubled)
	if err != nil || said != number.Value {
		return fmt.Errorf("%s: the page says %s, the data holds %v",
			number.Name, from.Text, number.Value)
	}
	return nil
}

// pageOf is the text of one page of a pdf, as poppler lays it out, or the
// whole of a source that is not paged.
func pageOf(path string, page int) (string, error) {
	if !strings.HasSuffix(path, pdfSuffix) {
		whole, err := os.ReadFile(path)
		return string(whole), err
	}
	numbered := strconv.Itoa(page)
	laid, err := exec.Command("pdftotext", "-layout", "-f", numbered,
		"-l", numbered, path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext could not read page %d of %s "+
			"(brew install poppler): %w", page, path, err)
	}
	return string(laid), nil
}
