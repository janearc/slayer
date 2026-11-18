package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// every verb is declared soundly, and its example holds its output.
func TestTheExamplesHere(t *testing.T) {
	if err := tool(context.Background()).Examples(os.DirFS(".")); err != nil {
		t.Error(err)
	}
}

// write's example: one bar of crypt over one dark still, said and not
// written.
func Example_write() {
	tool(context.Background()).Run([]string{"write",
		"testdata/one-bar.yaml", "--dry-run"}, os.Stdout, os.Stdout)
	// Output: 4 frames, 1.85 s
}

// run says pan's verb with the arguments, and gives its code and what it
// wrote to out and to the error stream.
func run(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := tool(context.Background()).Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// TestWriteRefusesWhatItCannotDo checks a call with no script, a missing script, no film named and a film that cannot be made, each with a failing code.
func TestWriteRefusesWhatItCannotDo(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "no", "such", "film.mp4")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no script", []string{"write", "--dry-run"}, "write takes one script"},
		{"two scripts", []string{"write", "testdata/one-bar.yaml", "testdata/one-bar.yaml", "--dry-run"}, "write takes one script"},
		{"script not there", []string{"write", "testdata/missing.yaml", "--dry-run"}, "missing.yaml"},
		{"no film named", []string{"write", "testdata/one-bar.yaml"}, "say where the film goes: --out FILM.mp4"},
		{"film where it cannot go", []string{"write", "testdata/one-bar.yaml", "--out", gone}, ""},
	}
	for _, each := range cases {
		code, out, errOut := run(each.args...)
		if code == 0 || out != "" || !strings.Contains(errOut, each.want) || errOut == "" {
			t.Errorf("%s: code %d, out %q, errors %q; want a failure holding %q", each.name, code, out, errOut, each.want)
		}
	}
	if _, err := os.Stat(gone); err == nil {
		t.Error("a film was written to a place that does not exist")
	}
}

// TestWriteDryRunSaysTheFilmAndWritesNothing checks that a dry run says the frames and length and writes no film.
func TestWriteDryRunSaysTheFilmAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := run("write", "testdata/one-bar.yaml", "--dry-run", "--out", filepath.Join(dir, "film.mp4"))
	if code != 0 || out != "4 frames, 1.85 s\n" || errOut != "" {
		t.Errorf("code %d, out %q, errors %q", code, out, errOut)
	}
	if names, _ := os.ReadDir(dir); len(names) != 0 {
		t.Errorf("a dry run wrote %d files", len(names))
	}
}
