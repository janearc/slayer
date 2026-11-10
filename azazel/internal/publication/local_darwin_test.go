package publication

import (
	"errors"
	"os"
	"testing"
)

// nativeHandleCleanup retains the fixture reporter and its selected close call.
type nativeHandleCleanup struct {
	reporter *testing.T
	close    func() error
}

// finish reports every close refusal for fixtures that must remain open.
func (owner nativeHandleCleanup) finish() {
	if err := owner.close(); err != nil {
		owner.reporter.Error(err)
	}
}

// finishIfOpen tolerates an earlier close of the owned fixture handle.
func (owner nativeHandleCleanup) finishIfOpen() {
	if err := owner.close(); err != nil && !errors.Is(err, os.ErrClosed) {
		owner.reporter.Error(err)
	}
}

// nativeTemporaryFile creates a tiny local fixture and retains cleanup
// even when the method under test leaves the descriptor open.
func nativeTemporaryFile(t *testing.T) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "local-fixture-*")
	if err != nil {
		t.Fatal(err)
	}
	owner := nativeHandleCleanup{reporter: t, close: file.Close}
	t.Cleanup(owner.finishIfOpen)
	return file
}

// TestLocalCloseEndsHandleUse needs a real file to observe Go refusing
// later writes on its closed handle before a syscall. No flush.
func TestLocalCloseEndsHandleUse(t *testing.T) {
	file := nativeTemporaryFile(t)
	if _, err := file.Write([]byte("before")); err != nil {
		t.Fatalf("fixture was not writable: %v", err)
	}
	if err := (localTemporary{file: file}).Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := file.Write([]byte("after")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("write after close: %v, want closed descriptor", err)
	}
}

// TestLocalCloseRetainsInvalidHandleErrors is the anti-test: an already
// closed or absent descriptor cannot become a successful close.
func TestLocalCloseRetainsInvalidHandleErrors(t *testing.T) {
	t.Run("already closed", checkLocalCloseAlreadyClosed)
	t.Run("absent", checkLocalCloseAbsent)
}

// checkLocalCloseAlreadyClosed retains the native already-closed refusal.
func checkLocalCloseAlreadyClosed(t *testing.T) {
	file := nativeTemporaryFile(t)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	err := (localTemporary{file: file}).Close()
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("second close: %v, want closed refusal", err)
	}
}

// checkLocalCloseAbsent retains the omitted-handle refusal.
func checkLocalCloseAbsent(t *testing.T) {
	if err := (localTemporary{}).Close(); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("absent close: %v, want invalid refusal", err)
	}
}
