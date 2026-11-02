package terminal

import (
	"os"
	"testing"
)

// pipeOf is a file that reads back keys once, as a keyboard's would.
func pipeOf(t *testing.T, keys string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(keys); err != nil {
		t.Fatal(err)
	}
	w.Close()
	t.Cleanup(func() { r.Close() })
	return r
}
