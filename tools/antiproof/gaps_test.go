package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestUnknownFlagIsRefused checks that a flag the tool does not have is an
// error from parseFlags.
func TestUnknownFlagIsRefused(t *testing.T) {
	args := []string{"-file", "f", "-old", "a", "-new", "b", "-run", "T", "-bogus", "p"}
	if _, err := parseFlags(args); err == nil {
		t.Error("an unknown flag was accepted")
	}
}

// TestJudge checks how a finished go test is read: build and setup failures
// prove nothing, a clean run passes, an exit of one is a failed test, and any
// other error is passed on.
func TestJudge(t *testing.T) {
	other := errors.New("signal: killed")
	cases := []struct {
		name    string
		text    string
		err     error
		passed  bool
		wantErr string
	}{
		{"build failed", "FAIL pkg [build failed]", nil, false, "does not build"},
		{"setup failed", "FAIL pkg [setup failed]", nil, false, "does not build"},
		{"no tests", "ok pkg [no tests to run]", nil, false, "matched no tests"},
		{"passed", "ok pkg 0.1s", nil, true, ""},
		{"other error", "partial output", other, false, "killed"},
	}
	for _, c := range cases {
		text, passed, err := judge(c.text, c.err)
		if text != c.text || passed != c.passed {
			t.Errorf("%s: text %q passed %t", c.name, text, passed)
		}
		if (c.wantErr == "") != (err == nil) || err != nil && !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: error %v, want one containing %q", c.name, err, c.wantErr)
		}
	}
}

// TestGoTestThatCannotStartIsAnError checks that a missing go command is
// returned as an error and not read as a test result.
func TestGoTestThatCannotStartIsAnError(t *testing.T) {
	o := fixture(t)
	t.Setenv("PATH", t.TempDir())
	text, passed, err := goTest(o, quiet())
	if err == nil || passed || text != "" {
		t.Errorf("got %q, %t, %v", text, passed, err)
	}
}

// TestAFileThatCannotBeWrittenIsRefusedAndKept checks that a file that is
// read-only is refused before the run and is left as it was.
func TestAFileThatCannotBeWrittenIsRefusedAndKept(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not bind the superuser")
	}
	o := fixture(t)
	before := read(t, o.file)
	if err := os.Chmod(o.file, 0o444); err != nil {
		t.Fatal(err)
	}
	ran := false
	spy := func(options, <-chan os.Signal) (string, bool, error) {
		ran = true
		return "", true, nil
	}
	if _, _, err := prove(o, spy, quiet()); err == nil || ran {
		t.Errorf("err %v, ran %t", err, ran)
	}
	if read(t, o.file) != before {
		t.Error("file changed")
	}
}

// TestAFileThatCannotBePutBackIsStuck checks that when the restore fails the
// result is a stuckError naming the file, and the file is left broken.
func TestAFileThatCannotBePutBackIsStuck(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not bind the superuser")
	}
	o := fixture(t)
	before := read(t, o.file)
	lock := func(options, <-chan os.Signal) (string, bool, error) {
		return "output", true, os.Chmod(o.file, 0o444)
	}
	_, _, err := prove(o, lock, quiet())
	var stuck stuckError
	if !errors.As(err, &stuck) || stuck.file != o.file {
		t.Fatalf("err %v, want stuck on %s", err, o.file)
	}
	if !strings.Contains(err.Error(), "git checkout") {
		t.Errorf("the message does not say how to put it back: %v", err)
	}
	if read(t, o.file) == before {
		t.Error("the file was restored despite being read-only")
	}
}

// TestAStuckFileOutranksAPanic checks that a panic in the run does not hide
// that the file could not be put back.
func TestAStuckFileOutranksAPanic(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not bind the superuser")
	}
	o := fixture(t)
	boom := func(options, <-chan os.Signal) (string, bool, error) {
		if err := os.Chmod(o.file, 0o444); err != nil {
			t.Error(err)
		}
		panic("boom")
	}
	_, _, err := prove(o, boom, quiet())
	if _, stuck := err.(stuckError); !stuck {
		t.Errorf("err %v, want a stuckError", err)
	}
}
