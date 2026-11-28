package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

func TestResearchFetchGetsEachCitedSourceOnce(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	t.Cleanup(made.done)
	data, err := os.ReadFile(at.list)
	if err != nil {
		t.Fatal(err)
	}
	unused := "  - key: unused\n    from: " + made.server.URL +
		"/unused.txt\n    licence: public domain\n"
	if err := os.WriteFile(at.list, append(data, unused...), openFile); err != nil {
		t.Fatal(err)
	}
	claims, err := at.research()
	if err != nil {
		t.Fatal(err)
	}
	another := claims[0]
	another.Name = "another claim"
	at.research = held(append(claims, another)).read

	var out bytes.Buffer
	code := tool(at).Run([]string{"fetch", "--research"}, &out, &out)
	if code != 0 || strings.Count(out.String(), "candle:") != 1 ||
		strings.Contains(out.String(), "unused") {
		t.Fatalf("research fetch gave %d: %s", code, out.String())
	}
	out.Reset()
	if code := tool(at).Run([]string{"verify"}, &out, &out); code != 0 {
		t.Fatalf("verify after research fetch gave %d: %s", code, out.String())
	}
}

func TestResearchFetchRefusesExplicitKeys(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	t.Cleanup(made.done)
	var out bytes.Buffer
	code := tool(at).Run([]string{"fetch", "--research", "candle"},
		&out, &out)
	if code != 1 || !strings.Contains(out.String(), "explicit source keys") {
		t.Fatalf("research fetch with a key gave %d: %s", code, out.String())
	}
}

func TestResearchFetchRefusesUnknownCitation(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	t.Cleanup(made.done)
	at.research = held{{Name: "missing claim",
		From: &numbers.Source{Key: "unknown"}}}.read
	var out bytes.Buffer
	code := tool(at).Run([]string{"fetch", "--research"}, &out, &out)
	if code != 1 || !strings.Contains(out.String(), "unknown is not a source") {
		t.Fatalf("research fetch of an unknown citation gave %d: %s",
			code, out.String())
	}
}
