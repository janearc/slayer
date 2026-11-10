package processinfo

import (
	"bufio"
	"context"
	_ "embed"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
)

const (
	// usageFixtureEnvironment selects this test binary's owned child fixture.
	usageFixtureEnvironment = "SLAYER_AZAZEL_USAGE_CHILD"
	usageFixtureChild       = "owned-usage-fixture"
	usageFixtureReady       = "ready\n"
	// fixtureSuccess is defined as the successful process exit status.
	fixtureSuccess = 0
	// fixtureFailure is defined as the least positive unsuccessful exit status.
	fixtureFailure = 1
	// noFixtureReads is defined as the size of an empty read sequence.
	noFixtureReads = 0
)

// usageFixtureData holds picked test bounds outside the native implementation.
//
//go:embed testdata/usage.yaml
var usageFixtureData []byte

// usageBounds decodes the fixture deadline without a duration parser.
type usageBounds struct {
	ChildDeadlineNS int64 `yaml:"child_deadline_ns"`
	MinimumReads    int   `yaml:"minimum_reads"`
	MaximumReads    int   `yaml:"maximum_reads"`
}

// usageExamples refuses missing bounds instead of running an unbounded child.
func usageExamples(t *testing.T) usageBounds {
	t.Helper()
	var bounds usageBounds
	if err := yamlfile.Read("testdata/usage.yaml", usageFixtureData,
		&bounds); err != nil {
		t.Fatal(err)
	}
	if bounds.ChildDeadlineNS <= noFixtureReads ||
		bounds.MinimumReads <= noFixtureReads ||
		bounds.MaximumReads < bounds.MinimumReads {
		t.Fatal("invalid native usage fixture bounds")
	}
	return bounds
}

// TestMain handles the owned-child mode before the normal test harness.
// That mode writes a ready line and waits for input closure.
// It spawns no work and exits without running the tests.
func TestMain(main *testing.M) {
	if os.Getenv(usageFixtureEnvironment) == usageFixtureChild {
		if _, err := io.WriteString(os.Stdout, usageFixtureReady); err != nil {
			os.Exit(fixtureFailure)
		}
		if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
			os.Exit(fixtureFailure)
		}
		os.Exit(fixtureSuccess)
	}
	os.Exit(main.Run())
}

// usageChild owns exactly the one process started for a native witness.
type usageChild struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  io.ReadCloser
	cancel  context.CancelFunc
	started bool
	waited  bool
	report  *testing.T
}

// startUsageChild waits for a bounded ready handshake before returning its PID.
func startUsageChild(t *testing.T) *usageChild {
	t.Helper()
	deadline := time.Duration(usageExamples(t).ChildDeadlineNS)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	command := exec.CommandContext(ctx, executable)
	fixtureMode := usageFixtureEnvironment + "=" + usageFixtureChild
	command.Env = append(os.Environ(), fixtureMode)
	command.Stderr = os.Stderr
	command.WaitDelay = deadline
	input, err := command.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		if closeErr := input.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		cancel()
		t.Fatal(err)
	}
	child := &usageChild{command: command, input: input, output: output,
		cancel: cancel, report: t}
	t.Cleanup(child.finish)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	child.started = true
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || line != usageFixtureReady {
		t.Fatalf("native child ready handshake: %q, %v", line, err)
	}
	return child
}

// reap closes this child's input and observes its bounded successful exit once.
func (child *usageChild) reap() {
	child.report.Helper()
	input := child.input
	child.input = nil
	if err := input.Close(); err != nil {
		child.report.Fatal(err)
	}
	child.waited = true
	if err := child.command.Wait(); err != nil {
		child.report.Fatal(err)
	}
}

// finish reaps every started fixture, including one whose assertion failed.
func (child *usageChild) finish() {
	if child.input != nil {
		if err := child.input.Close(); err != nil {
			child.report.Error(err)
		}
	}
	if child.started && !child.waited {
		child.waited = true
		if err := child.command.Wait(); err != nil {
			child.report.Error(err)
		}
	} else if !child.started {
		if err := child.output.Close(); err != nil {
			child.report.Error(err)
		}
	}
	child.cancel()
}
