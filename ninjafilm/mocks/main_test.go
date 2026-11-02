package mocks_test

import (
	"flag"
	"fmt"
	"os"
	"runtime/metrics"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
)

// runawayNumbers is where the heap ceiling is, with its source.
const runawayNumbers = "testdata/numbers.yaml"

// runawayCeiling is the bytes of heap that stop the tests, read from the
// numbers file.
func runawayCeiling() (ceiling float64, err error) {
	raw, err := os.ReadFile(runawayNumbers)
	if err != nil {
		return ceiling, err
	}
	held, err := numbers.Read(runawayNumbers, raw)
	if err != nil {
		return ceiling, err
	}
	number, err := held.Get("heap-ceiling")
	if err != nil {
		return ceiling, err
	}
	return number.Value, nil
}

// TestMain runs the tests under the guard.
func TestMain(suite *testing.M) {
	ceiling, err := runawayCeiling()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(runawayFailed)
	}
	runawayGuard(ceiling)
	os.Exit(suite.Run())
}

// The numbers the guard works with. Each is defined.
const (
	// runawayNoTimeout is the -test.timeout of a run that is never stopped. It
	// is defined by go test: -timeout 0 turns the timeout off.
	runawayNoTimeout = 0
	// runawayQuarter divides the timeout into the part taken off it. It
	// is defined by the arithmetic.
	runawayQuarter = 4
	// runawayEveryStack is the debug level of runtime/pprof that writes a stack
	// for each goroutine, the running test among them. It is defined by
	// that package.
	runawayEveryStack = 2
	// runawayPoll is how often the heap is read. It's defined here, and a
	// runaway allocates gigabytes in far less than a second.
	runawayPoll = 100 * time.Millisecond
	// runawayMetric is the runtime metric for the bytes of live and
	// not yet freed objects. It is defined by runtime/metrics.
	runawayMetric = "/memory/classes/heap/objects:bytes"
	// runawayOnly is the place of the one sample asked of the runtime. It's
	// defined.
	runawayOnly = 0
	// runawayFailed is the exit code of a run the guard stopped. It's
	// defined by convention.
	runawayFailed = 1
)

// runawayTimeout is the -test.timeout this binary was started with. go test
// gives every test binary one, ten minutes unless told otherwise.
func runawayTimeout() time.Duration {
	got, _ := flag.Lookup("test.timeout").Value.(flag.Getter).Get().(time.Duration)
	return got
}

// runawayStop ends the run as failed and gives the reason. It then writes the
// stack of every goroutine, the running test among them.
func runawayStop(reason string) {
	fmt.Fprintln(os.Stderr, "FAIL: "+reason)
	fmt.Fprintln(os.Stderr, "the goroutines, the running test among them:")
	_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, runawayEveryStack)
	os.Exit(runawayFailed)
}

// runawayHeap is the bytes of heap objects now.
func runawayHeap() float64 {
	sample := []metrics.Sample{{Name: runawayMetric}}
	metrics.Read(sample)
	only := sample[runawayOnly]
	return float64(only.Value.Uint64())
}

// runawayWatch stops the run when the heap passes the ceiling. Code that
// allocates without end gets here before the machine is out of memory.
func runawayWatch(ceiling float64) {
	for range time.Tick(runawayPoll) {
		if used := runawayHeap(); used > ceiling {
			runawayStop(fmt.Sprintf("the heap passed %.0f bytes, at %.0f; "+
				"a test allocates without end", ceiling, used))
		}
	}
}

// runawayGuard makes a runaway fail fast, not hang or fill memory. It stops the
// run at three quarters of -test.timeout, and when the heap passes the
// ceiling it is given. Loops that do not end get the first. Code
// that allocates without end gets the second.
func runawayGuard(ceiling float64) {
	flag.Parse()
	if timeout := runawayTimeout(); timeout > runawayNoTimeout {
		time.AfterFunc(timeout-timeout/runawayQuarter, runawayOverrun)
	}
	go runawayWatch(ceiling)
}

// runawayOverrun stops a run that passed three quarters of its timeout.
func runawayOverrun() {
	runawayStop("the tests ran past three quarters of their -test.timeout; " +
		"a loop does not end")
}
