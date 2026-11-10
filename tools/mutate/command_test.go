package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
)

var coverageCommand string

// TestMain builds the real command once, with the test's coverage mode.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-mutator-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	coverageCommand = filepath.Join(directory, "mutate")
	if err = commandtest.Build(".", ".", coverageCommand, testing.CoverMode()); err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

func mutationFixture(t *testing.T, assertion string) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	source := []byte("package fixture\nfunc F() bool { return true }\n")
	for name, data := range map[string][]byte{
		"go.mod":    []byte("module example.invalid/mutation-fixture\n\ngo 1.26\n"),
		"f.go":      source,
		"f_test.go": []byte("package fixture\nimport \"testing\"\nfunc TestF(t *testing.T) {" + assertion + "}\n"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q"}, {"add", "go.mod", "f.go", "f_test.go"},
		{"-c", "user.name=Slayer test fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "commit", "--no-gpg-sign", "-qm", "fixture"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %q", args, err, out)
		}
	}
	return dir, source
}

func mutateIn(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, coverageCommand, args...)
	cmd.Dir = dir
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestMutatorKillsARealFaultAndRestoresTheTrackedFile(t *testing.T) {
	dir, original := mutationFixture(t, "if !F() {t.Fatal(\"false\")}")
	out, err := mutateIn(t, dir, "-procs", "3", "f.go")
	if err != nil || !strings.Contains(out, "1 killed, 0 survived") {
		t.Fatalf("%v %q", err, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "f.go"))
	if err != nil || string(got) != string(original) {
		t.Fatalf("original was not restored: %v %q", err, got)
	}
	cmd := exec.Command("git", "diff", "--exit-code")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal("mutation left a dirty tracked file")
	}
}

func TestMutatorReportsASurvivorAndStillRestoresTheFile(t *testing.T) {
	dir, original := mutationFixture(t, "")
	out, err := mutateIn(t, dir, "-procs", "3", "f.go")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(out, "SURVIVED f.go:2") || !strings.Contains(out, "0 killed, 1 survived") {
		t.Fatalf("%v %q", err, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "f.go"))
	if err != nil || string(got) != string(original) {
		t.Fatal("surviving mutant was left behind")
	}
}

func TestMutatorRejectsMissingUntrackedDirtyAndMalformedInputs(t *testing.T) {
	dir, _ := mutationFixture(t, "if !F() {t.Fatal(\"false\")}")
	for _, test := range []struct {
		args []string
		want string
	}{
		{nil, "no files"}, {[]string{"f.go:bad"}, "is not file:first-last"}, {[]string{"absent.go"}, "not tracked"},
	} {
		out, err := mutateIn(t, dir, test.args...)
		if err == nil || !strings.Contains(out, test.want) {
			t.Fatalf("%v %q", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte("package fixture\nfunc F() bool {return false}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := mutateIn(t, dir, "f.go")
	if err == nil || !strings.Contains(out, "uncommitted changes") {
		t.Fatalf("%v %q", err, out)
	}
}

// Interrupt a real mutant after its test has started. The public command
// must restore the tracked source and stop the nested Go test process.
func TestMutatorSignalRestoresSourceAndStopsTheMutant(t *testing.T) {
	dir, original := mutationFixture(t, "")
	marker := filepath.Join(dir, "mutant.pid")
	block := fmt.Sprintf(`package fixture
import ("os"; "strconv"; "testing")
func TestBlockingMutant(t *testing.T) {
 if !F() {
  if err:=os.WriteFile(%q,[]byte(strconv.Itoa(os.Getpid())+"\n"),0600);err!=nil {t.Fatal(err)}
  select {}
 }
}
`, marker)
	if err := os.WriteFile(filepath.Join(dir, "blocking_test.go"), []byte(block), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, coverageCommand, "-procs", "3", "f.go")
	cmd.Dir = dir
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 2 * time.Second
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			cmd.Process.Signal(syscall.SIGTERM)
			cmd.Wait()
		}
	}()
	pid := waitForPid(t, marker)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	waited = true
	var stopped *exec.ExitError
	if !errors.As(err, &stopped) || stopped.ExitCode() != 130 {
		t.Fatalf("%v %q", err, output.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "f.go"))
	if err != nil || string(got) != string(original) {
		t.Fatal("interrupt left the mutant in the tracked file")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("interrupt left its nested test process running")
}

func TestMutatorReportsWhenTrackedSourceCannotBeRestored(t *testing.T) {
	dir, original := mutationFixture(t, "if !F() {t.Fatal(\"false\")}")
	path := filepath.Join(dir, "f.go")
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)
	out, err := mutateIn(t, dir, "-procs", "3", "f.go")
	var stopped *exec.ExitError
	if !errors.As(err, &stopped) || stopped.ExitCode() != 3 || !strings.Contains(out, "puts it back") {
		t.Fatalf("%v %q", err, out)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatal("failed write changed the original")
	}
}
