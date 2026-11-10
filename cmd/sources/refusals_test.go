package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/sources"
)

func TestSourceVerbsIdentifyMissingCataloguesAndUnknownKeys(t *testing.T) {
	at, done := demo("")
	defer done()
	for _, verb := range []string{"fetch", "check", "verify", "path"} {
		t.Run(verb, func(t *testing.T) {
			missing := at
			missing.list = filepath.Join(t.TempDir(), "absent.yaml")
			var out bytes.Buffer
			args := []string{verb}
			if verb == "path" {
				args = append(args, "candle")
			}
			if code := tool(missing).Run(args, &out, &out); code != 1 || !strings.Contains(out.String(), "absent.yaml") {
				t.Fatalf("%s with missing catalogue: %d %s", verb, code, out.String())
			}
		})
	}
	var out bytes.Buffer
	if code := tool(at).Run([]string{"path", "missing"}, &out, &out); code != 1 || !strings.Contains(out.String(), "not in the list") {
		t.Fatalf("unknown path: %d %s", code, out.String())
	}
	list, err := sources.Read(at.list)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := chosen(list, []string{"candle"})
	if err != nil || len(rows) != 1 || rows[0].Key() != "candle" {
		t.Fatalf("selected source: %v %v", rows, err)
	}
}

func TestVerifyRefusesUnreadableResearchAndMissingSourceRoles(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	want := errors.New("research was unreadable")
	at.research = func() (numbers.Numbers, error) { return nil, want }
	var out bytes.Buffer
	if code := tool(at).Run([]string{"verify"}, &out, &out); code != 1 || !strings.Contains(out.String(), want.Error()) {
		t.Fatalf("research failure: %d %s", code, out.String())
	}
	list, err := sources.Read(at.list)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []numbers.Number{
		{Name: "missing"},
		{Name: "unknown", From: &numbers.Source{Key: "absent"}},
	} {
		if err := at.verified(list, row); err == nil || !strings.Contains(err.Error(), row.Name) {
			t.Fatalf("unverifiable row: %v", err)
		}
	}
	if got := whereIn(numbers.Source{Key: "paper", Page: 24}); got != "on page 24 of paper" {
		t.Fatal(got)
	}
}

func TestPageReaderReportsMissingTextAndUnusablePDF(t *testing.T) {
	for _, extension := range []string{".txt", ".pdf"} {
		path := filepath.Join(t.TempDir(), "absent"+extension)
		if _, err := pageOf(path, 1); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("missing page: %v", err)
		}
	}
	// Poppler is the real boundary. A text file with a PDF suffix must
	// fail there, rather than passing because a value occurs in its bytes.
	path := filepath.Join(t.TempDir(), "broken.pdf")
	if err := os.WriteFile(path, []byte("a number, but not a PDF: 6.84"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := pageOf(path, 1); err == nil || !strings.Contains(err.Error(), "pdftotext") {
		t.Fatalf("bad PDF: %v", err)
	}
}
