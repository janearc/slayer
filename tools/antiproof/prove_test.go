package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// the fixture module is copied to a temporary directory for each test, so
// the checked-in copy is never touched
func fixture(t *testing.T) options {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"go.mod", "lib.go", "lib_test.go"} {
		data, err := os.ReadFile(filepath.Join("testdata/mod", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data,
			0o640); err != nil {
			t.Fatal(err)
		}
	}
	return options{file: filepath.Join(dir, "lib.go"), old: "x * 2",
		new: "x * 3", run: "TestDouble", pkg: ".", dir: dir, count: 1}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func quiet() chan os.Signal { return make(chan os.Signal, 1) }

func TestTestsFailUnderTheBreakAndFileIsRestored(t *testing.T) {
	o := fixture(t)
	before := read(t, o.file)
	out, _, err := prove(o, goTest, quiet())
	if err != nil || out != testsFailed {
		t.Fatalf("outcome %v, err %v, want failed", out, err)
	}
	if read(t, o.file) != before {
		t.Error("file not restored after a failing run")
	}
}

// anti-test of TestTestsFailUnderTheBreakAndFileIsRestored
func TestTestsPassUnderAnIrrelevantBreakAndFileIsRestored(t *testing.T) {
	o := fixture(t)
	o.old, o.new = "Double is", "Double is still"
	before := read(t, o.file)
	out, _, err := prove(o, goTest, quiet())
	if err != nil || out != testsPassed {
		t.Fatalf("outcome %v, err %v, want passed", out, err)
	}
	if read(t, o.file) != before {
		t.Error("file not restored after a passing run")
	}
}

func TestABreakThatDoesNotBuildIsAToolError(t *testing.T) {
	o := fixture(t)
	o.new = "x *"
	before := read(t, o.file)
	_, _, err := prove(o, goTest, quiet())
	if err == nil || !strings.Contains(err.Error(), "does not build") {
		t.Errorf("err %v, want a build refusal", err)
	}
	if read(t, o.file) != before {
		t.Error("file not restored after a build failure")
	}
}

// anti-test of TestTestsFailUnderTheBreakAndFileIsRestored
func TestARunThatMatchesNoTestsIsAToolError(t *testing.T) {
	o := fixture(t)
	o.run = "TestNothingIsCalledThis"
	_, _, err := prove(o, goTest, quiet())
	if err == nil || !strings.Contains(err.Error(), "no tests") {
		t.Errorf("err %v, want no tests matched", err)
	}
}

func TestAnInterruptedRunKillsTheTestsAndRestoresTheFile(t *testing.T) {
	o := fixture(t)
	o.run = "TestSlow"
	o.old, o.new = "Double is", "Double was"
	before := read(t, o.file)
	signals := quiet()
	go func() {
		time.Sleep(3 * time.Second)
		signals <- os.Interrupt
	}()
	start := time.Now()
	_, _, err := prove(o, goTest, signals)
	if err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Errorf("err %v, want interrupted", err)
	}
	if time.Since(start) > 30*time.Second {
		t.Error("the sleeping test was left to finish")
	}
	if read(t, o.file) != before {
		t.Error("file not restored after an interrupt")
	}
}

// anti-test of TestAnInterruptedRunKillsTheTestsAndRestoresTheFile
func TestWithoutAnInterruptTheSlowTestIsNotKilled(t *testing.T) {
	o := fixture(t)
	o.run = "TestSlow"
	fake := func(options, <-chan os.Signal) (string, bool, error) {
		return "", true, nil
	}
	if out, _, err := prove(o, fake, quiet()); err != nil ||
		out != testsPassed {
		t.Errorf("outcome %v, err %v", out, err)
	}
}

func TestAPanicInTheRunRestoresTheFile(t *testing.T) {
	o := fixture(t)
	before := read(t, o.file)
	boom := func(options, <-chan os.Signal) (string, bool, error) {
		if read(t, o.file) == before {
			t.Error("the file was not broken during the run")
		}
		panic("boom")
	}
	_, _, err := prove(o, boom, quiet())
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err %v, want the panic named", err)
	}
	if read(t, o.file) != before {
		t.Error("file not restored after a panic")
	}
}

// anti-test of TestAPanicInTheRunRestoresTheFile
func TestARunErrorRestoresTheFileToo(t *testing.T) {
	o := fixture(t)
	before := read(t, o.file)
	bad := func(options, <-chan os.Signal) (string, bool, error) {
		return "", false, errors.New("no go")
	}
	if _, _, err := prove(o, bad, quiet()); err == nil {
		t.Error("want the error back")
	}
	if read(t, o.file) != before {
		t.Error("file not restored after an error")
	}
}

func TestOldMustOccurExactlyOnce(t *testing.T) {
	o := fixture(t)
	before := read(t, o.file)
	for _, old := range []string{"nowhere in the file", "x"} {
		o.old = old
		if _, _, err := prove(o, goTest, quiet()); err == nil {
			t.Errorf("%q: want a refusal", old)
		}
		if read(t, o.file) != before {
			t.Errorf("%q: file changed", old)
		}
	}
}

// anti-test of TestOldMustOccurExactlyOnce
func TestOldOccurringOnceIsAccepted(t *testing.T) {
	o := fixture(t)
	if _, _, err := prove(o, goTest, quiet()); err != nil {
		t.Errorf("err %v", err)
	}
}

func TestFileModeIsKept(t *testing.T) {
	o := fixture(t)
	os.Chmod(o.file, 0o640)
	prove(o, goTest, quiet())
	info, _ := os.Stat(o.file)
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode %v", info.Mode().Perm())
	}
}

// anti-test of TestFileModeIsKept
func TestAFileThatIsNotThereIsRefused(t *testing.T) {
	o := fixture(t)
	o.file += ".missing"
	if _, _, err := prove(o, goTest, quiet()); err == nil {
		t.Error("want a refusal")
	}
}

func TestExitCodes(t *testing.T) {
	var out, errOut bytes.Buffer
	if report(testsFailed, "", nil, &out, &errOut) != 0 {
		t.Error("failed tests: the proof holds, exit 0")
	}
	if report(testsPassed, "", nil, &out, &errOut) != 1 {
		t.Error("passed tests: exit 1")
	}
	if report(0, "", errors.New("x"), &out, &errOut) != 2 {
		t.Error("tool error: exit 2")
	}
}

// anti-test of TestExitCodes
func TestAStuckFileIsExitThree(t *testing.T) {
	var out, errOut bytes.Buffer
	if report(0, "", stuckError{"f"}, &out, &errOut) != exitStuck {
		t.Error("a file left changed: exit 3")
	}
}

func TestFlagsAreChecked(t *testing.T) {
	good := []string{"-file", "f", "-old", "a", "-new", "b", "-run", "T", "p"}
	if o, err := parseFlags(good); err != nil || o.count != 1 ||
		o.pkg != "p" {
		t.Errorf("%+v, %v", o, err)
	}
}

// anti-test of TestFlagsAreChecked
func TestBadFlagsAreRefused(t *testing.T) {
	bad := [][]string{
		{"-file", "f", "-old", "a", "-new", "b", "p"},
		{"-file", "f", "-old", "a", "-new", "a", "-run", "T", "p"},
		{"-file", "f", "-old", "a", "-new", "b", "-run", "T"},
		{"-file", "f", "-old", "a", "-new", "b", "-run", "T",
			"-count", "0", "p"},
	}
	for _, args := range bad {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("%v: want a refusal", args)
		}
	}
}
