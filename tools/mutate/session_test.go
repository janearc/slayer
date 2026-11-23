package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// a session over a file of its own, holding "before"
func newTestSession(t *testing.T) *session {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f.go")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &session{path: path, original: []byte("before")}
}

func TestStopPutsTheFileBackAndNothingIsWrittenAfter(t *testing.T) {
	s := newTestSession(t)
	if err := s.write([]byte("changed")); err != nil {
		t.Fatal(err)
	}
	if err := s.stop(); err != nil {
		t.Fatal(err)
	}
	if err := s.write([]byte("late")); !errors.Is(err, errStopped) {
		t.Errorf("a write after stop gave %v, want errStopped", err)
	}
	if _, err := s.run("true"); !errors.Is(err, errStopped) {
		t.Errorf("a run after stop gave %v, want errStopped", err)
	}
	if got, _ := os.ReadFile(s.path); string(got) != "before" {
		t.Errorf("the file holds %q after stop, want %q", got, "before")
	}
}

func TestARestoreThatFailsIsAnError(t *testing.T) {
	s := newTestSession(t)
	if err := os.Chmod(s.path, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := s.stop(); err == nil {
		t.Error("a file that could not be put back gave no error")
	}
}

// stop kills the tests and what they started: the shell here stands in
// for go test, and its sleep for the test binary
func TestStopKillsTheTestsAndTheirChildren(t *testing.T) {
	s := newTestSession(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	done := make(chan error, 1)
	go func() {
		script := "sleep 30 & echo $! > " + pidFile + "; wait"
		_, err := s.run("sh", "-c", script)
		done <- err
	}()
	child := waitForPid(t, pidFile)
	if err := s.stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, errStopped) {
			t.Errorf("a stopped run gave %v, want errStopped", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the run did not end when stopped")
	}
	for range 50 {
		if syscall.Kill(child, 0) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	syscall.Kill(child, syscall.SIGKILL)
	t.Errorf("the tests' child %d outlived the stop", child)
}

// waitForPid reads the pid a shell wrote, once it has written it
func waitForPid(t *testing.T, path string) int {
	t.Helper()
	for range 250 {
		said, err := os.ReadFile(path)
		if err == nil && strings.HasSuffix(string(said), "\n") {
			text := strings.TrimSpace(string(said))
			pid, err := strconv.Atoi(text)
			if err != nil {
				t.Fatal(err)
			}
			return pid
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the shell never said its child's pid")
	return 0
}

func TestCleanRefusesUntrackedAndChangedFiles(t *testing.T) {
	t.Chdir(t.TempDir())
	git := func(args ...string) {
		t.Helper()
		all := append([]string{"-c", "user.name=t",
			"-c", "user.email=t@t", "-c", "commit.gpgsign=false"},
			args...)
		said, err := exec.Command("git", all...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, said)
		}
	}
	git("init", "-q")
	if err := os.WriteFile("f.go", []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean("f.go") == nil {
		t.Error("an untracked file was taken as clean")
	}
	git("add", "f.go")
	git("commit", "-q", "-m", "f")
	if err := clean("f.go"); err != nil {
		t.Errorf("a committed file was refused: %v", err)
	}
	if err := os.WriteFile("f.go", []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean("f.go") == nil {
		t.Error("a changed file was taken as clean")
	}
}

func TestASpecIsAFileAndItsLines(t *testing.T) {
	cases := map[string][3]any{
		"a.go":       {"a.go", 0, 1 << 30},
		"a.go:3-9":   {"a.go", 3, 9},
		"d/a.go:1-1": {"d/a.go", 1, 1},
	}
	for spec, want := range cases {
		path, first, last, err := parseSpec(spec)
		if err != nil || path != want[0] || first != want[1] ||
			last != want[2] {
			t.Errorf("%q gave %q %d %d %v, want %v", spec, path,
				first, last, err, want)
		}
	}
	for _, spec := range []string{"a.go:3", "a.go:x-9", "a.go:3-"} {
		if _, _, _, err := parseSpec(spec); err == nil {
			t.Errorf("%q was not refused", spec)
		}
	}
}

// TestRunRefusesACommandThatCannotStart checks the start error comes back
// and no tests are left recorded as running.
func TestRunRefusesACommandThatCannotStart(t *testing.T) {
	s := newTestSession(t)
	missing := filepath.Join(t.TempDir(), "no-such-command")
	if _, err := s.run(missing); err == nil {
		t.Fatal("a command that is not there ran")
	}
	if s.tests != nil {
		t.Error("the session kept a command that never started")
	}
}

// TestTryStopsWhenTheFileCannotBeWritten checks that a file that cannot be
// written ends the run with the write error, not with a count.
func TestTryStopsWhenTheFileCannotBeWritten(t *testing.T) {
	s := newTestSession(t)
	if err := os.Chmod(s.path, 0o444); err != nil {
		t.Fatal(err)
	}
	survived, err := try(s, []change{{offset: 0, from: "b", to: "c", line: 1}}, []string{"version"})
	if err == nil || survived != 0 {
		t.Errorf("try gave %d and %v", survived, err)
	}
}

// TestTryEndsWithTheStopWhenTheSessionIsStopped checks a stop that comes
// while a change is under test ends the loop with errStopped and counts
// nothing.
func TestTryEndsWithTheStopWhenTheSessionIsStopped(t *testing.T) {
	s := newTestSession(t)
	program := filepath.Join(t.TempDir(), "slow.go")
	text := "package main\nimport \"time\"\nfunc main() { time.Sleep(time.Minute) }\n"
	if err := os.WriteFile(program, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		survived int
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		survived, err := try(s, []change{{offset: 0, from: "b", to: "c", line: 1}}, []string{"run", program})
		done <- outcome{survived, err}
	}()
	for range 500 {
		s.lock.Lock()
		running := s.tests != nil
		s.lock.Unlock()
		if running {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := s.stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if !errors.Is(got.err, errStopped) || got.survived != 0 {
			t.Errorf("try gave %d and %v", got.survived, got.err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("try did not end when the session stopped")
	}
}
