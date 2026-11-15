package main

import (
	"context"
	"os"
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
