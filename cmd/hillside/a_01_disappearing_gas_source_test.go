//go:build darwin

package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const gasSourceProbe = "SLAYER_DISAPPEARING_GAS_SOURCE_PROBE"

// The pipe removes its own name before EOF. Hash verification succeeds on
// the opened descriptor; reopening that checked name must report ENOENT.
// A child contains a broken read/reopen sequence without leaving a reader
// blocked in the ordinary test process.
func TestGasTableDisappearingAfterPinCheckKeepsItsOpenRefusal(t *testing.T) {
	if os.Getenv(gasSourceProbe) == "1" {
		probeDisappearingGasSource(t)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	args := []string{"-test.run=^TestGasTableDisappearingAfterPinCheckKeepsItsOpenRefusal$"}
	if dir := flag.Lookup("test.gocoverdir"); testing.CoverMode() != "" &&
		dir != nil && dir.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+dir.Value.String())
	}
	child := exec.CommandContext(ctx, os.Args[0], args...)
	child.Env = append(os.Environ(), gasSourceProbe+"=1")
	out, err := child.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("gas source read/reopen did not finish: %v", ctx.Err())
	}
	if err != nil {
		t.Fatalf("gas source read/reopen: %v\n%s", err, out)
	}
}

func probeDisappearingGasSource(t *testing.T) {
	t.Helper()
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	stage := staging{given: given}
	if _, err := stage.gases(); err != nil {
		t.Fatalf("ordinary pinned gas table: %v", err)
	}
	path := filepath.Join(os.Getenv("SLAYER_SHELF"),
		"karkoschka-1994-methane.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		var writer *os.File
		for {
			var err error
			writer, err = os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
			if err == nil {
				break
			}
			if !errors.Is(err, syscall.ENXIO) || time.Now().After(deadline) {
				done <- err
				return
			}
			time.Sleep(time.Millisecond)
		}
		_, writeErr := writer.Write(raw)
		removeErr := os.Remove(path)
		closeErr := writer.Close()
		done <- errors.Join(writeErr, removeErr, closeErr)
	}()
	_, err = stage.gases()
	if writerErr := <-done; writerErr != nil {
		t.Fatal(writerErr)
	}
	// A pin-check failure wraps the path error with "not on the shelf".
	// This unwrapped error must come from the consumer's later os.Open.
	opened, ok := err.(*os.PathError)
	if !ok || opened.Op != "open" || opened.Path != path ||
		!errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lost post-verification open refusal: %T %v", err, err)
	}
}
