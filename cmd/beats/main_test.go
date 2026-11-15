package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewScoreDoesNotReplaceCorrections protects a person's edited score.
func TestNewScoreDoesNotReplaceCorrections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "score.yaml")
	if err := writeNew(path, []byte("corrected")); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(path, []byte("estimated")); err == nil {
		t.Fatal("replaced a score")
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "corrected" {
		t.Fatalf("correction changed: %s %v", raw, err)
	}
	// anti-test of TestNewScoreDoesNotReplaceCorrections: missing parent fails.
	if err := writeNew(filepath.Join(path, "missing", "x"), nil); err == nil {
		t.Fatal("accepted unwritable destination")
	}
}

// Example_analyse checks declaration without opening audio or writing output.
func Example_analyse() {
	var out, errs bytes.Buffer
	code := run([]string{"analyse", "song.opus", "--out", "score.yaml",
		"--dry-run"}, strings.NewReader(""), &out, &errs)
	fmt.Println(code, strings.TrimSpace(out.String()))
	// Output: 0 would analyse song.opus
}
