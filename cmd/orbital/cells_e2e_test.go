package main_test

// These tests run the built orbital as a window would: a program that
// writes to a pipe. They build it first. A scene takes seconds to build,
// so each test that builds one draws it small.

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// readable is who may read the golden file. It is defined as the owner
// writing and everyone reading.
const readable os.FileMode = 0o644

// update rewrites the golden from what the orbital writes now. No gate
// passes it: a golden is moved by hand, by a change that meant to move it.
var update = flag.Bool("update", false,
	"rewrite testdata/far-cells.golden from the orbital drawn now")

// outcome is what a run of the orbital left behind.
type outcome struct {
	stdout, stderr string
	err            error
}

// limit is how long a run is let go on before the test gives it up.
func limit() time.Duration {
	return time.Duration(picked("run-limit-seconds")) * time.Second
}

// binary builds the orbital and says where it is, or the test stops.
func binary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orbital")
	built, err := exec.Command("go", "build", "-o", path, ".").
		CombinedOutput()
	if err != nil {
		t.Fatalf("the orbital would not build: %v\n%s", err, built)
	}
	return path
}

// cellsArgs are the arguments that draw the picked number of frames at
// the picked size, with the rest given.
func cellsArgs(rest ...string) []string {
	size := fmt.Sprintf("%dx%d", picked("cells-cols"), picked("cells-rows"))
	frames := fmt.Sprint(picked("cells-frames"))
	rays := fmt.Sprint(picked("cells-rays"))
	return append([]string{"-cells", size, "-frames", frames, "-n", rays},
		rest...)
}

// run runs the orbital in a directory of its own and gives what it wrote.
// A run that goes past the limit fails the test.
func run(t *testing.T, program string, args []string) outcome {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit())
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = t.TempDir()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("%v did not finish in %v", args, limit())
	}
	return outcome{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

// Each row of the table is a command line the orbital refuses. It says so
// on standard error and writes nothing to standard output. It says it
// before it has built a scene, which would have said "ring 0".
func TestABadAskIsRefusedBeforeTheSceneIsBuilt(t *testing.T) {
	program := binary(t)
	for _, row := range casesOf(t, refusedFile) {
		args := strings.Fields(textIn(t, row, "args"))
		got := run(t, program, args)
		var exited *exec.ExitError
		if !errors.As(got.err, &exited) || exited.ExitCode() == nothing {
			t.Errorf("%v: exit %v; want a failure", args, got.err)
		}
		if !strings.Contains(got.stderr, textIn(t, row, "says")) {
			t.Errorf("%v: stderr %q; want %q", args, got.stderr,
				textIn(t, row, "says"))
		}
		if got.stdout != "" || strings.Contains(got.stderr, "ring") {
			t.Errorf("%v: wrote %q and built a scene", args, got.stdout)
		}
	}
}

// With -cells the orbital writes frames and nothing else to standard
// output. They are plain ANSI. What it says of the scene it built goes to
// standard error.
func TestTheOrbitalWritesPlainFramesAndKeepsItsChatterApart(t *testing.T) {
	got := run(t, binary(t), cellsArgs())
	if got.err != nil {
		t.Fatalf("%v\n%s", got.err, got.stderr)
	}
	if err := plain(got.stdout, picked("cells-rows")); err != nil {
		t.Error(err)
	}
	frames := strings.Count(got.stdout, frameBegin)
	if frames != picked("cells-frames") {
		t.Errorf("%d frames; want %d", frames, picked("cells-frames"))
	}
	for _, said := range []string{"ring 0:", "ring 1:"} {
		if !strings.Contains(got.stderr, said) {
			t.Errorf("stderr is %q; want %q", got.stderr, said)
		}
	}
}

// The far view, her favourite, is held byte for byte to its golden.
// go test -update rewrites the golden.
func TestTheFarViewIsHeldToItsGolden(t *testing.T) {
	got := run(t, binary(t), cellsArgs(farRendering))
	if got.err != nil {
		t.Fatalf("%v\n%s", got.err, got.stderr)
	}
	if *update {
		if err := os.WriteFile(farGolden, []byte(got.stdout), readable); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(farGolden)
	if err != nil {
		t.Fatal(err)
	}
	if got.stdout != string(want) {
		t.Errorf("the far view is not its golden: %d bytes, want %d; if the "+
			"change meant to move it, go test -update", len(got.stdout),
			len(want))
	}
}

// untilFrameEnd reads from the pipe until one whole frame has come.
func untilFrameEnd(pipe io.Reader) error {
	var seen []byte
	chunk := make([]byte, picked("read-chunk"))
	for !bytes.Contains(seen, []byte(frameEnd)) {
		read, err := pipe.Read(chunk)
		seen = append(seen, chunk[:read]...)
		if err != nil {
			return err
		}
	}
	return nil
}

// A stream with no end stops when its reader goes. The test takes one
// frame, closes the pipe and waits. The orbital must exit with a failure
// and not run on.
func TestTheOrbitalStopsWhenItsReaderGoes(t *testing.T) {
	program := binary(t)
	ctx, cancel := context.WithTimeout(context.Background(), limit())
	defer cancel()
	cmd := exec.CommandContext(ctx, program, cellsArgs("-frames", "0")...)
	cmd.Dir = t.TempDir()
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := untilFrameEnd(pipe); err != nil {
		t.Fatalf("no whole frame came: %v", err)
	}
	if err := pipe.Close(); err != nil {
		t.Fatal(err)
	}
	waited := cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("the orbital ran on after its reader went, for %v", limit())
	}
	if waited == nil {
		t.Error("the orbital exited cleanly with no one reading")
	}
}
