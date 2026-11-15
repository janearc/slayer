package instrument

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestBuildCopiesInputsAndExposesTrustedCommand(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	args := []string{"--flag", "value"}
	env := []string{"A=B", "C=D"}

	plan, err := Build(Options{Executable: binary, Args: args, Env: env})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() {
		if err := plan.Cleanup(); err != nil && !strings.Contains(err.Error(), "already cleaned") {
			t.Fatalf("Cleanup() error = %v", err)
		}
	}()

	args[0] = "mutated"
	env[0] = "mutated"

	if got := plan.Executable(); got != binary {
		t.Fatalf("Executable() = %q, want %q", got, binary)
	}
	if got := plan.Args(); !reflect.DeepEqual(got, []string{"--flag", "value"}) {
		t.Fatalf("Args() = %#v", got)
	}
	if got := plan.Env(); !reflect.DeepEqual(got, []string{"A=B", "C=D"}) {
		t.Fatalf("Env() = %#v", got)
	}

	gotArgs := plan.Args()
	gotArgs[0] = "changed"
	gotEnv := plan.Env()
	gotEnv[0] = "changed"
	if !reflect.DeepEqual(plan.Args(), []string{"--flag", "value"}) {
		t.Fatalf("Args() leaked mutation")
	}
	if !reflect.DeepEqual(plan.Env(), []string{"A=B", "C=D"}) {
		t.Fatalf("Env() leaked mutation")
	}

	argv, gotEnv2, timingPath := plan.TrustedCommand()
	wantArgv := []string{"/usr/bin/time", "-l", "-o", timingPath, binary, "--flag", "value"}
	if !reflect.DeepEqual(argv, wantArgv) {
		t.Fatalf("TrustedCommand() argv = %#v, want %#v", argv, wantArgv)
	}
	if !reflect.DeepEqual(gotEnv2, []string{"A=B", "C=D"}) {
		t.Fatalf("TrustedCommand() env = %#v", gotEnv2)
	}
	if timingPath != plan.TimingPath() {
		t.Fatalf("TrustedCommand() timing path = %q, want %q", timingPath, plan.TimingPath())
	}

	argv[0] = "changed"
	gotEnv2[0] = "changed"
	if !reflect.DeepEqual(plan.Args(), []string{"--flag", "value"}) {
		t.Fatalf("TrustedCommand() leaked args mutation")
	}
	if !reflect.DeepEqual(plan.Env(), []string{"A=B", "C=D"}) {
		t.Fatalf("TrustedCommand() leaked env mutation")
	}
}

func TestBuildRejectsInvalidExecutable(t *testing.T) {
	if _, err := Build(Options{}); err == nil {
		t.Fatalf("Build() error = nil")
	}
	if _, err := Build(Options{Executable: "relative"}); err == nil {
		t.Fatalf("Build() relative executable error = nil")
	}
}

func TestCleanupRemovesOnlyOwnedPrivateDir(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	plan, err := Build(Options{Executable: binary})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	timingPath := plan.TimingPath()
	timingDir := filepath.Dir(timingPath)
	if _, err := os.Stat(timingDir); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", timingDir, err)
	}
	if err := plan.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(timingPath); !os.IsNotExist(err) {
		t.Fatalf("timing file exists after cleanup: %v", err)
	}
	if _, err := os.Stat(timingDir); !os.IsNotExist(err) {
		t.Fatalf("timing dir exists after cleanup: %v", err)
	}
	if err := plan.Cleanup(); err == nil {
		t.Fatalf("Cleanup() second call error = nil")
	}
}

func TestCleanupReportsUnexpectedDirContent(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	plan, err := Build(Options{Executable: binary})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	unexpected := filepath.Join(filepath.Dir(plan.TimingPath()), "unexpected")
	if err := os.WriteFile(unexpected, []byte("x"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", unexpected, err)
	}
	if err := plan.Cleanup(); err == nil {
		t.Fatalf("Cleanup() error = nil")
	}
	if _, err := os.Stat(unexpected); err != nil {
		t.Fatalf("unexpected file missing after failed cleanup: %v", err)
	}
	_ = os.Remove(unexpected)
	_ = os.Remove(plan.TimingPath())
	_ = os.Remove(filepath.Dir(plan.TimingPath()))
}

func TestCleanupRejectsMissingOwnership(t *testing.T) {
	if err := (&Plan{}).Cleanup(); err == nil {
		t.Fatalf("Cleanup() on empty plan error = nil")
	}
}

func TestCleanupFailsClosedAccessors(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	plan, err := Build(Options{Executable: binary, Args: []string{"--flag"}, Env: []string{"A=B"}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := plan.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if got := plan.Executable(); got != "" {
		t.Fatalf("Executable() after cleanup = %q, want empty", got)
	}
	if got := plan.Args(); got != nil {
		t.Fatalf("Args() after cleanup = %#v, want nil", got)
	}
	if got := plan.Env(); got != nil {
		t.Fatalf("Env() after cleanup = %#v, want nil", got)
	}
	if argv, env, timing := plan.TrustedCommand(); argv != nil || env != nil || timing != "" {
		t.Fatalf("TrustedCommand() after cleanup = %#v, %#v, %q", argv, env, timing)
	}
	if got := plan.TimingPath(); got != "" {
		t.Fatalf("TimingPath() after cleanup = %q, want empty", got)
	}
	if err := plan.Cleanup(); err != errAlreadyCleaned {
		t.Fatalf("Cleanup() after cleanup error = %v, want %v", err, errAlreadyCleaned)
	}
}

func TestCleanupIsSafeUnderConcurrentAccess(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	plan, err := Build(Options{Executable: binary, Args: []string{"--flag"}, Env: []string{"A=B"}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 1000; j++ {
				_ = plan.Executable()
				_ = plan.Args()
				_ = plan.Env()
				_, _, _ = plan.TrustedCommand()
				_ = plan.TimingPath()
			}
		}()
	}
	close(start)
	if err := plan.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	wg.Wait()
}
