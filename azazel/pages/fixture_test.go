package pages

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fixtureDirectory holds the small renders tree the scripts were run on:
// times.txt lists every file and its time, tree/ the text files.
const fixtureDirectory = "testdata/fixture"

// scriptsDirectory holds what the scripts wrote from one run on the
// fixture, on 2026-11-15, with TZ=UTC.
const scriptsDirectory = "testdata/scripts"

// The numbers a fixture is laid with.
const (
	pictureSide = 2     // picked: the smallest picture that is still one
	grey        = 128   // picked: a mid grey, the one colour of it
	listing     = 0o644 // defined: what a file is written with here
	folders     = 0o755 // defined: what a folder is made with here
)

// solid is a small generated picture of one colour; nothing is drawn.
func solid(t *testing.T) []byte {
	t.Helper()
	return sized(t, pictureSide, pictureSide)
}

// lay writes one fixture file at its time: the committed text when there
// is one, else a solid picture for a png and a few bytes for the rest.
func lay(t *testing.T, root string, when int64, path string) {
	t.Helper()
	target := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(target), folders); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(fixtureDirectory, "tree", path))
	switch {
	case errors.Is(err, fs.ErrNotExist) && strings.HasSuffix(path, ".png"):
		data = solid(t)
	case errors.Is(err, fs.ErrNotExist):
		data = []byte("fixture")
	case err != nil:
		t.Fatal(err)
	}
	stamp := time.Unix(when, 0)
	err = errors.Join(os.WriteFile(target, data, listing),
		os.Chtimes(target, stamp, stamp))
	if err != nil {
		t.Fatal(err)
	}
}

// layLine lays one line of times.txt: a file, a numbered run of files
// one second apart, or a link.
func layLine(t *testing.T, root string, fields []string) {
	t.Helper()
	if fields[0] == "link" {
		target := filepath.Join(root, fields[1])
		if err := os.Symlink(fields[2], target); err != nil {
			t.Fatal(err)
		}
		return
	}
	when, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) == 2 {
		lay(t, root, when, fields[1])
		return
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		t.Fatal(err)
	}
	for index := range count {
		lay(t, root, when+int64(index), fmt.Sprintf(fields[1], index))
	}
}

// fixture lays the fixture's renders tree in a temporary folder, as the
// scripts' run did, and returns its root.
func fixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "renders")
	data, err := os.ReadFile(filepath.Join(fixtureDirectory, "times.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.Lines(string(data)) {
		layLine(t, root, strings.Split(strings.TrimSuffix(line, "\n"), "\t"))
	}
	return root
}

// made matches the line that says when a gallery page was made.
var made = regexp.MustCompile(`Made \d{4}-\d\d-\d\d \d\d:\d\d\.`)

// withoutMade is a page with the time it was made taken out.
func withoutMade(page string) string {
	return made.ReplaceAllString(page, "Made TIME.")
}

// expected is one of the scripts' pages.
func expected(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(scriptsDirectory, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// read is a file's text, or the test's failure.
func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// clock is a test's time, moved by hand.
type clock struct {
	lock sync.Mutex
	at   time.Time
}

// fixtureNoon is when a test's clock starts: after every fixture file.
var fixtureNoon = time.Unix(1794050000, 0) // picked: after the fixture's times

// now is the clock's time.
func (c *clock) now() time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.at
}

// advance moves the clock on.
func (c *clock) advance(by time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.at = c.at.Add(by)
}

// settingsFor are the settings of a test: its renders root, its encoder
// and its clock, with times shown in UTC as the scripts' run had them.
func settingsFor(root string, encoder Encoder, hands *clock) Settings {
	return Settings{Renders: root, Encoder: encoder, Now: hands.now,
		Location: time.UTC, Every: TickEvery}
}
