//go:build darwin && storageprobe

package storageprobe

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const (
	probeWrites = 3     // three observations, not a performance distribution
	probeBytes  = 4096  // a chosen small payload, not a host page-size claim
	privateFile = 0o600 // only the test owner reads or writes the fixture
)

// TestRegularFileSyncReportsBoundedCosts measures the actual Go operation
// on a small temporary file without installing a publisher or a service.
func TestRegularFileSyncReportsBoundedCosts(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "sync-probe"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	payload := bytes.Repeat([]byte("a"), probeBytes)
	for sample := range probeWrites {
		written, err := file.Write(payload)
		if err != nil || written != len(payload) {
			t.Fatalf("write %d bytes: %d, %v", len(payload), written, err)
		}
		started := time.Now()
		if err := file.Sync(); err != nil {
			t.Fatalf("File.Sync sample %d: %v", sample, err)
		}
		t.Logf("File.Sync sample %d: %s", sample, time.Since(started))
	}
}

// TestClosedFileSyncIsNotSuccessful is the anti-test: a refused flush
// cannot be mistaken for a successful sample.
func TestClosedFileSyncIsNotSuccessful(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "closed-probe"))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("sync closed file: %v, want closed refusal", err)
	}
}

// TestDirectorySyncReportsEachVariant reports Go's operation and direct
// fsync separately. Go may fall back from F_FULLFSYNC without reporting it.
// A successful call does not identify which internal variant succeeded.
func TestDirectorySyncReportsEachVariant(t *testing.T) {
	dir, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dir.Close(); err != nil {
			t.Error(err)
		}
	}()
	goEntry := filepath.Join(dir.Name(), "go-sync-entry")
	if err := os.WriteFile(goEntry, []byte("x"), privateFile); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	err = dir.Sync()
	t.Logf("Go directory File.Sync: elapsed=%s, error=%v",
		time.Since(started), err)
	unsupported := errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.ENOTSUP)
	if err != nil && !unsupported {
		t.Fatalf("unexpected Go directory-sync failure: %v", err)
	}
	directEntry := filepath.Join(dir.Name(), "direct-sync-entry")
	if err := os.WriteFile(directEntry, []byte("x"), privateFile); err != nil {
		t.Fatal(err)
	}
	started = time.Now()
	if err := syscall.Fsync(int(dir.Fd())); err != nil {
		t.Fatalf("direct directory fsync: %v", err)
	}
	t.Logf("direct directory fsync: elapsed=%s, success",
		time.Since(started))
}

// TestClosedDirectorySyncIsRefused is the anti-test: the Go directory
// variant must not report a closed descriptor as successful.
func TestClosedDirectorySyncIsRefused(t *testing.T) {
	dir, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := dir.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dir.Sync(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("sync closed directory: %v, want closed refusal", err)
	}
}

// TestReplacementKeepsAnExistingReaderOnTheOldFile checks namespace
// replacement and open-reader lifetime. It does not check crash or
// power-loss durability.
func TestReplacementKeepsAnExistingReaderOnTheOldFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "state")
	temporary := filepath.Join(dir, "next")
	if err := os.WriteFile(destination, []byte("old"), privateFile); err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := os.WriteFile(temporary, []byte("new"), privateFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporary, destination); err != nil {
		t.Fatal(err)
	}
	old, err := io.ReadAll(reader)
	if err != nil || string(old) != "old" {
		t.Fatalf("held reader: %q, %v, want old", old, err)
	}
	current, err := os.ReadFile(destination)
	if err != nil || string(current) != "new" {
		t.Fatalf("new reader: %q, %v, want new", current, err)
	}
}

// TestFailedRenameKeepsTheDestination is the anti-test: a missing source
// is rejected before any change. It says nothing about uncertain failures
// during a real rename after changes might have happened.
func TestFailedRenameKeepsTheDestination(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "state")
	if err := os.WriteFile(destination, []byte("old"), privateFile); err != nil {
		t.Fatal(err)
	}
	err := os.Rename(filepath.Join(dir, "absent"), destination)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("rename absent source: %v, want not-exist refusal", err)
	}
	current, err := os.ReadFile(destination)
	if err != nil || string(current) != "old" {
		t.Fatalf("after failed rename: %q, %v, want old", current, err)
	}
}
