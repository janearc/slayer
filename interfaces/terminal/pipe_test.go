package terminal

import (
	"os"
	"testing"
)

// pipeOf is a file that reads back keys once, as a keyboard's would.
func pipeOf(t *testing.T, keys string) *os.File {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString(keys); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	t.Cleanup(func() { reader.Close() })
	return reader
}
