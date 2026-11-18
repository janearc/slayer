package observability

import (
	"context"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestBoundedReporterAllowsNormalFinal(t *testing.T) {
	final := make(chan struct{}, 1)
	stop := StartBoundedReporter(context.Background(), time.Hour, 200*time.Millisecond, nil, nil, func(ctx context.Context) {
		if ctx.Err() == nil {
			final <- struct{}{}
		}
	})
	stop()
	select {
	case <-final:
	default:
		t.Fatal("normal final was not delivered")
	}
	stop()
}

func TestBoundedReporterDoesNotWaitForBlockedInitialOrSendLate(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var finals atomic.Int32
	stop := StartBoundedReporter(context.Background(), time.Millisecond, 30*time.Millisecond, func(context.Context) { close(entered); <-release }, nil, func(context.Context) { finals.Add(1) })
	<-entered
	started := time.Now()
	stop()
	if time.Since(started) > time.Second {
		t.Fatal("stop blocked")
	}
	close(release)
	time.Sleep(20 * time.Millisecond)
	if finals.Load() != 0 {
		t.Fatal("late final dispatch")
	}
}

func TestBoundedReporterSkipsFinalAfterParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var finals atomic.Int32
	stop := StartBoundedReporter(ctx, time.Hour, 30*time.Millisecond, nil, nil, func(context.Context) { finals.Add(1) })
	stop()
	if finals.Load() != 0 {
		t.Fatal("final after cancellation")
	}
}

func TestReporterCallbackPanicIsCountedAndDoesNotEscape(t *testing.T) {
	before := SummaryFailures()
	reporterCallback(context.Background(), func(context.Context) { panic("sampler failed") })
	if SummaryFailures() != before+1 {
		t.Fatal("panic was not counted")
	}
}

func TestReporterPanicDoesNotTerminateProcess(t *testing.T) {
	if os.Getenv("SUMMARY_PANIC_HELPER") == "1" {
		entered := make(chan struct{})
		stop := StartBoundedReporter(context.Background(), time.Hour, 100*time.Millisecond, func(context.Context) { close(entered); panic("optional sampler") }, nil, nil)
		<-entered
		stop()
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestReporterPanicDoesNotTerminateProcess$")
	command.Env = append(os.Environ(), "SUMMARY_PANIC_HELPER=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("reporter panic escaped: %v %s", err, output)
	}
}

func TestExclusiveReporterDoesNotAccumulateStuckWorkers(t *testing.T) {
	var gate atomic.Bool
	entered, release := make(chan struct{}), make(chan struct{})
	stop, ok := StartExclusiveReporter(&gate, context.Background(), time.Hour, 10*time.Millisecond, func(context.Context) { close(entered); <-release }, nil, nil)
	if !ok {
		t.Fatal("first reporter refused")
	}
	<-entered
	stop()
	for range 20 {
		if _, ok = StartExclusiveReporter(&gate, context.Background(), time.Hour, time.Millisecond, nil, nil, nil); ok {
			t.Fatal("started while prior callback stuck")
		}
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for gate.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if gate.Load() {
		t.Fatal("gate did not recover")
	}
	stop, ok = StartExclusiveReporter(&gate, context.Background(), time.Hour, 10*time.Millisecond, nil, nil, nil)
	if !ok {
		t.Fatal("gate did not admit new reporter")
	}
	stop()
}
