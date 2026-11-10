package processinfo

import (
	"errors"
	"math"
	"os"
	"syscall"
	"testing"

	"pgregory.net/rapid"
)

const (
	// noObservedPID is defined as an omitted process identifier.
	noObservedPID = 0
	// noObservedTick is defined as an omitted native timestamp.
	noObservedTick uint64 = 0
	// onePIDStep is defined as the next representable integer PID value.
	onePIDStep = 1
)

// requireOwnObservation needs real libproc data from this running test process.
func requireOwnObservation(t *testing.T) usageObservation {
	t.Helper()
	query := usageQuery{pid: os.Getpid()}
	got := (darwinUsageReader{}).ReadUsage(query)
	if got.state != usageObserved || got.err != nil || got.pid != query.pid ||
		got.born == noObservedTick || got.exited != noObservedTick {
		t.Fatalf("current-process native observation: %+v", got)
	}
	return got
}

// TestNativeUsageReadsCurrentProcess needs the actual SDK and library.
func TestNativeUsageReadsCurrentProcess(t *testing.T) {
	requireOwnObservation(t)
}

// TestNativeUsageRejectsNonpositivePIDs is the paired native-call refusal.
func TestNativeUsageRejectsNonpositivePIDs(t *testing.T) {
	for _, pid := range []int{noObservedPID, -onePIDStep, math.MinInt32} {
		got := (darwinUsageReader{}).ReadUsage(usageQuery{pid: pid})
		if got.state != usageFailed || got.err != os.ErrInvalid ||
			got.pid != noObservedPID || got.born != noObservedTick ||
			got.exited != noObservedTick {
			t.Fatalf("invalid PID %d produced an observation: %+v", pid, got)
		}
	}
}

// TestNativeUsageIsStableForTheSameProcess checks repeated actual native reads.
func TestNativeUsageIsStableForTheSameProcess(t *testing.T) {
	first := requireOwnObservation(t)
	again := requireOwnObservation(t)
	if again.pid != first.pid || again.born != first.born {
		t.Fatalf("unchanged process changed start: %+v, %+v", first, again)
	}
}

// TestNativeUsageRefusesUnrepresentablePIDs checks PID narrowing is refused.
// The installed Darwin SDK declares its PID argument as a 32-bit C int.
func TestNativeUsageRefusesUnrepresentablePIDs(t *testing.T) {
	for _, pid := range []int{math.MaxInt32 + onePIDStep, math.MaxInt} {
		got := (darwinUsageReader{}).ReadUsage(usageQuery{pid: pid})
		if got.state != usageFailed || got.err != os.ErrInvalid ||
			got.pid != noObservedPID || got.born != noObservedTick ||
			got.exited != noObservedTick {
			t.Fatalf("unrepresentable PID %d was narrowed: %+v", pid, got)
		}
	}
}

// TestNativeUsageDistinguishesAnOwnedChild needs a real child at its handshake.
// It compares native creation order without wall-clock conversion or tolerance.
func TestNativeUsageDistinguishesAnOwnedChild(t *testing.T) {
	parent := requireOwnObservation(t)
	child := startUsageChild(t)
	query := usageQuery{pid: child.command.Process.Pid}
	got := (darwinUsageReader{}).ReadUsage(query)
	if got.state != usageObserved || got.err != nil || got.pid != query.pid ||
		got.pid == parent.pid || got.born <= parent.born ||
		got.exited != noObservedTick {
		t.Fatalf("owned child observation: %+v, parent %+v", got, parent)
	}
}

// TestNativeUsageNeverReturnsAReapedChildsOldObservation pairs child lifetime.
// A reused PID may return a different observation, never the old one.
func TestNativeUsageNeverReturnsAReapedChildsOldObservation(t *testing.T) {
	child := startUsageChild(t)
	query := usageQuery{pid: child.command.Process.Pid}
	reader := darwinUsageReader{}
	before := reader.ReadUsage(query)
	if before.state != usageObserved || before.err != nil ||
		before.born == noObservedTick {
		t.Fatalf("live owned child observation: %+v", before)
	}
	child.reap()
	after := reader.ReadUsage(query)
	if after.state == usageObserved {
		if after.err != nil || after.pid != query.pid ||
			after.born == noObservedTick || after.born == before.born {
			t.Fatalf("reaped child returned a stale observation: %+v", after)
		}
		return
	}
	var native syscall.Errno
	if after.state != usageFailed || !errors.Is(after.err, errUsageRead) ||
		!errors.As(after.err, &native) || after.pid != noObservedPID ||
		after.born != noObservedTick || after.exited != noObservedTick {
		t.Fatalf("native refusal lost cause or returned data: %+v", after)
	}
}

// TestNativeUsageZeroResultIsUnknown covers the omitted-result boundary.
func TestNativeUsageZeroResultIsUnknown(t *testing.T) {
	zero := usageObservation{}
	if zero.state == usageObserved || zero.pid != noObservedPID ||
		zero.born != noObservedTick || zero.exited != noObservedTick {
		t.Fatal("zero result claims a native observation")
	}
}

// usageStableProperty repeats current-process reads without spawning children.
type usageStableProperty struct{ bounds usageBounds }

// check compares all observed stamps across a picked bounded number of reads.
func (property usageStableProperty) check(trial *rapid.T) {
	reader := darwinUsageReader{}
	query := usageQuery{pid: os.Getpid()}
	first := reader.ReadUsage(query)
	if first.state != usageObserved || first.err != nil ||
		first.pid != query.pid || first.born == noObservedTick {
		trial.Fatalf("current-process property observation: %+v", first)
	}
	reads := rapid.IntRange(property.bounds.MinimumReads,
		property.bounds.MaximumReads).Draw(trial, "reads")
	for range reads {
		got := reader.ReadUsage(query)
		if got.state != usageObserved || got.err != nil || got.pid != first.pid ||
			got.born != first.born || got.exited != noObservedTick {
			trial.Fatalf("stable native observation changed: %+v, %+v", first, got)
		}
	}
}

// TestNativeUsageGeneratedStableCurrentReads uses bounded real libproc calls.
func TestNativeUsageGeneratedStableCurrentReads(t *testing.T) {
	property := usageStableProperty{bounds: usageExamples(t)}
	rapid.Check(t, property.check)
}

// usageInvalidProperty varies malformed PIDs without spawning any process.
type usageInvalidProperty struct{}

// check requires argument refusal rather than a narrowed native query.
func (usageInvalidProperty) check(trial *rapid.T) {
	pid := rapid.IntRange(math.MinInt32, noObservedPID).Draw(trial, "nonpositive")
	if rapid.Bool().Draw(trial, "outside C int") {
		pid = rapid.IntRange(math.MaxInt32+onePIDStep, math.MaxInt).
			Draw(trial, "unrepresentable")
	}
	got := (darwinUsageReader{}).ReadUsage(usageQuery{pid: pid})
	if got.state != usageFailed || got.err != os.ErrInvalid ||
		got.pid != noObservedPID || got.born != noObservedTick ||
		got.exited != noObservedTick {
		trial.Fatalf("invalid generated PID %d: %+v", pid, got)
	}
}

// TestNativeUsageGeneratedInvalidInputsIsTheAntiProperty spans bad inputs.
func TestNativeUsageGeneratedInvalidInputsIsTheAntiProperty(t *testing.T) {
	rapid.Check(t, usageInvalidProperty{}.check)
}
