package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/sources"
)

// humanBytes says sizes in decimal units.
func TestHumanBytes(t *testing.T) {
	for count, want := range map[int64]string{0: "0 B", 999: "999 B",
		1000: "1.0 KB", 812_400_000: "812.4 MB", 1_710_000_000: "1.71 GB",
		3_200_000_000_000: "3.20 TB"} {
		if got := humanBytes(count); got != want {
			t.Errorf("%d is %q, want %q", count, got, want)
		}
	}
}

// the two lines say the source, the bytes of the total with a percent,
// and how many are done.
func TestTheStatusLines(t *testing.T) {
	count := tally{sources: 62, fetched: 11, kept: 2, failed: 1}
	got := statusLines(1, "candle", 812_400_000, 1_710_000_000, count, 120)
	if got[0] != "/ fetching candle  812.4 MB of 1.71 GB  47%" {
		t.Errorf("first line %q", got[0])
	}
	if got[1] != "  14 of 62: 11 fetched, 2 already here, 1 failed" {
		t.Errorf("second line %q", got[1])
	}
}

// with no length announced the first line has the bytes received and no
// total or percent.
func TestTheStatusWithNoLength(t *testing.T) {
	got := statusLines(0, "candle", 2048, 0, tally{sources: 3}, 80)
	if got[0] != "| fetching candle  2.0 KB" {
		t.Errorf("first line %q", got[0])
	}
}

// a line is cut to less than the width, so it never wraps, and a width
// of nothing gives nothing.
func TestTheStatusIsCutToTheWidth(t *testing.T) {
	count := tally{sources: 62}
	for _, width := range []int{40, 10, 2, 1, 0} {
		for _, line := range statusLines(2, "candle", 5, 10, count, width) {
			if len([]rune(line)) > max(width-1, 0) {
				t.Errorf("width %d: %q is too long", width, line)
			}
		}
	}
}

// the summary says what came, with its size, what was here and what
// failed, and the shelf.
func TestTheSummary(t *testing.T) {
	count := tally{fetched: 11, kept: 2, failed: 1, bytes: 2_310_000_000}
	want := "fetched 11 (2.31 GB), 2 already here, 1 failed; shelf /shelf"
	if got := summary(count, "/shelf", false); got != want {
		t.Errorf("summary %q", got)
	}
	if got := summary(count, "/shelf", true); got != "stopped early: "+want {
		t.Errorf("stopped summary %q", got)
	}
}

// papers is two sources on a local server, one that is there and one
// that is not, a list naming them, and a place with a shelf of its own.
func papers(t *testing.T, names ...string) (place, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(
		answer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "missing") {
			http.NotFound(answer, request)
			return
		}
		fmt.Fprint(answer, "bytes of "+request.URL.Path)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	text := "sources:\n"
	for _, name := range names {
		text += "  - key: " + name + "\n    from: " + server.URL + "/" +
			name + ".pdf\n    licence: public domain\n"
	}
	list := filepath.Join(dir, "sources.yaml")
	if err := os.WriteFile(list, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	clock := func() time.Time { return time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC) }
	return place{list: list, now: clock, shelf: sources.NewShelf(
		filepath.Join(dir, "shelf"), server.Client())}, dir
}

// sumOf is the sha256 of a string, as the shelf says it.
func sumOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// off a terminal the output is one line for each source, as it was
// before the live status, and no control sequence is in it.
func TestNotOnATerminalTheOutputIsOneLineEach(t *testing.T) {
	at, _ := papers(t, "alpha", "beta")
	var out bytes.Buffer
	if code := tool(at).Run([]string{"fetch"}, &out, &out); code != 0 {
		t.Fatalf("code %d: %s", code, out.String())
	}
	want := ""
	for _, name := range []string{"alpha", "beta"} {
		body := "bytes of /" + name + ".pdf"
		want += fmt.Sprintf("%s: fetched, not pinned: sha256 %s, %d bytes\n",
			name, sumOf(body), len(body))
	}
	if out.String() != want {
		t.Errorf("output\n%q\nwant\n%q", out.String(), want)
	}
	if strings.Contains(out.String(), "\x1b") || strings.Contains(out.String(),
		"fetching") {
		t.Error("the status was drawn off a terminal")
	}
}

// a buffer is not a terminal.
func TestABufferIsNotATerminal(t *testing.T) {
	if _, live := terminalColumns(&bytes.Buffer{}); live {
		t.Error("a buffer is live")
	}
}

// the live fetch ends with one summary line in place of the status, with
// the cursor shown again.
func TestTheLiveFetchEndsWithTheSummary(t *testing.T) {
	at, dir := papers(t, "alpha", "beta")
	list, _ := sources.Read(at.list)
	var out bytes.Buffer
	err := at.fetchLive(context.Background(), &out, func() int { return 60 },
		list.Sources())
	text := out.String()
	want := "fetched 2 (37 B), 0 already here, 0 failed; shelf " +
		filepath.Join(dir, "shelf") + "\n"
	if err != nil || !strings.HasSuffix(text, showCursor+want) {
		t.Errorf("err %v, output %q", err, text)
	}
	if !strings.HasPrefix(text, hideCursor) {
		t.Errorf("the cursor was not hidden: %q", text)
	}
}

// a failure is printed above the status, which is drawn again after it,
// and the fetch carries on to the next source.
func TestAFailureIsPrintedAboveTheStatus(t *testing.T) {
	at, _ := papers(t, "missing", "beta")
	list, _ := sources.Read(at.list)
	var out bytes.Buffer
	err := at.fetchLive(context.Background(), &out, func() int { return 60 },
		list.Sources())
	text := out.String()
	line := strings.Index(text, "missing: sources: not fetched")
	after := strings.Index(text[max(line, 0):], "\n"+lineStart+eraseLine)
	if line < 0 || after < 0 || err == nil {
		t.Fatalf("err %v, the failure line is not above a status: %q", err,
			text)
	}
	if !strings.Contains(text, "1 failed; shelf") ||
		!strings.Contains(text, "fetched 1 (") {
		t.Errorf("the fetch did not carry on: %q", text)
	}
}

// a fetch that is stopped before it starts says so, shows the cursor,
// and fetches nothing.
func TestAStoppedFetchLeavesTheTerminalClean(t *testing.T) {
	at, dir := papers(t, "alpha")
	list, _ := sources.Read(at.list)
	stopped, stop := context.WithCancel(context.Background())
	stop()
	var out bytes.Buffer
	err := at.fetchLive(stopped, &out, func() int { return 60 }, list.Sources())
	if err == nil || !strings.Contains(out.String(), showCursor+
		"stopped early: fetched 0") {
		t.Errorf("err %v, output %q", err, out.String())
	}
	if _, found := os.Stat(filepath.Join(dir, "shelf", "alpha.pdf")); found == nil {
		t.Error("a stopped fetch fetched")
	}
}

// the status is wiped by going to the start of the line, erasing it and
// the one below, and coming back up.
func TestTheStatusIsWipedInPlace(t *testing.T) {
	var out bytes.Buffer
	screen := newDisplay(&out, func() int { return 40 }, 1)
	screen.mu.Lock()
	screen.draw()
	out.Reset()
	screen.wipe()
	screen.mu.Unlock()
	if want := "\r\x1b[2K\x1b[B\x1b[2K\x1b[A\r"; out.String() != want {
		t.Errorf("wipe wrote %q", out.String())
	}
}
