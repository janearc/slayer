//go:build unix

package sources

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestFreshSourceMetadataHasDataFilePermissions(t *testing.T) {
	// Umask belongs to this test process, and is restored before another
	// sequential or parallel test can run.
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)
	server, _ := served(t)
	source := one(t, server, "")
	shelf := NewShelf(t.TempDir(), server.Client())
	if _, err := shelf.Fetch(context.Background(), source, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(shelf.Path(source) + ".source")
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("metadata permissions %04o, want 0644", got)
	}
}
