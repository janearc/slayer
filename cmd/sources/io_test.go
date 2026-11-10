package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/sources"
)

// failedSourceOutput models a closed pipe at the command's output boundary.
type failedSourceOutput struct{ err error }

func (out failedSourceOutput) Write([]byte) (int, error) { return 0, out.err }

func TestSourceCommandsPropagateOutputFailures(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	var out bytes.Buffer
	if code := tool(at).Run([]string{"fetch"}, &out, &out); code != 0 {
		t.Fatal(out.String())
	}
	for _, args := range [][]string{{"fetch", "--dry-run"}, {"fetch"}, {"check"}, {"path", "candle"}, {"verify"}} {
		var wrong bytes.Buffer
		want := errors.New("the source output pipe closed")
		code := tool(at).Run(args, failedSourceOutput{want}, &wrong)
		if code != 1 || !strings.Contains(wrong.String(), want.Error()) {
			t.Fatalf("%v: %d %s", args, code, wrong.String())
		}
	}
}

func TestSourceFetchIdentifiesANetworkRefusal(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	made.server.Close()
	var out bytes.Buffer
	if code := tool(at).Run([]string{"fetch"}, &out, &out); code != 1 || !strings.Contains(out.String(), "Get \""+made.server.URL) || !strings.Contains(out.String(), "connection refused") {
		t.Fatalf("network failure: %d %s", code, out.String())
	}
}

func TestSourceVerificationRejectsWrongNumericTextAndUnreadablePinnedPDF(t *testing.T) {
	for _, pdf := range []bool{false, true} {
		at, made := verifying("candle", "6.84")
		defer made.done()
		if pdf {
			raw, err := os.ReadFile(at.list)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(at.list, []byte(strings.Replace(string(raw), "/candle.txt", "/candle.pdf", 1)), 0600); err != nil {
				t.Fatal(err)
			}
		}
		var out bytes.Buffer
		if code := tool(at).Run([]string{"fetch"}, &out, &out); code != 0 {
			t.Fatal(out.String())
		}
		out.Reset()
		want := "the page says candle"
		if pdf {
			want = "pdftotext could not read page"
		}
		if code := tool(at).Run([]string{"verify"}, &out, &out); code != 1 || !strings.Contains(out.String(), want) {
			t.Fatalf("verification boundary: %d %s", code, out.String())
		}
	}
}

func TestWithdrawnSourcesCannotValidateNumbers(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	if err := os.WriteFile(at.list, []byte("sources:\n  - key: candle\n    never: withdrawn\n"), 0600); err != nil {
		t.Fatal(err)
	}
	list, err := sources.Read(at.list)
	if err != nil {
		t.Fatal(err)
	}
	data, err := at.research()
	if err != nil {
		t.Fatal(err)
	}
	if err := at.verified(list, data[0]); err == nil || !strings.Contains(err.Error(), "candle is not a source on the shelf") {
		t.Fatalf("withdrawn source: %v", err)
	}
}
