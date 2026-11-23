package processinfo

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// bootOf spells 32 hex digits as a boot id, in the 8-4-4-4-12 form.
func bootOf(digits string) string {
	return digits[:8] + "-" + digits[8:12] + "-" + digits[12:16] + "-" +
		digits[16:20] + "-" + digits[20:]
}

// ownerTestBoot is the made-up boot the tests run under, otherBoot a
// different one, and zeroBoot the all-zero id that is no boot at all.
var (
	ownerTestBoot = bootOf("00112233445566778899aabbccddeeff")
	otherBoot     = bootOf("10112233445566778899aabbccddeeff")
	zeroBoot      = bootOf(strings.Repeat("0", 32))
)

type ownerMock struct {
	boot      string
	bootErr   error
	usage     usageObservation
	reads     int
	bootReads int
	mutate    func(*ownerMock)
	children  []int
	group     []int
	listErr   error
	listReads int
}

func (m *ownerMock) ReadBoot() (string, error) {
	m.bootReads++
	return m.boot, m.bootErr
}
func (m *ownerMock) ReadUsage(q usageQuery) usageObservation {
	m.reads++
	if m.mutate != nil {
		m.mutate(m)
	}
	return m.usage
}
func (m *ownerMock) ReadRelations(pid int) (processRelations, error) {
	return processRelations{pid: pid, parent: 1, group: pid}, nil
}
func (m *ownerMock) ListChildren(pid, limit int) ([]int, error) {
	m.listReads++
	return m.children, m.listErr
}
func (m *ownerMock) ListGroup(pid, limit int) ([]int, error) {
	m.listReads++
	return m.group, m.listErr
}
func ownerExample() *ownerMock {
	return &ownerMock{boot: ownerTestBoot, usage: usageObservation{pid: 42, born: 100, state: usageObserved}}
}
func ownerExpected() OwnerIdentity {
	return OwnerIdentity{boot: ownerTestBoot, pid: 42, start: 100}
}

func TestIdentityCaptureAndDurableIncarnation(t *testing.T) {
	m := ownerExample()
	got, err := captureOwner(context.Background(), m, 42)
	if err != nil || got != ownerExpected() {
		t.Fatalf("capture: %+v %v", got, err)
	}
	if got.Boot() != ownerTestBoot || got.PID() != 42 || got.Start() != 100 {
		t.Fatal("identity accessors")
	}
	if got.Incarnation() != "process-v1."+ownerTestBoot+".42.100" {
		t.Fatalf("incarnation: %q", got.Incarnation())
	}
	parsed, err := ParseOwnerIdentity(got.Incarnation())
	if err != nil || parsed != got {
		t.Fatalf("round trip: %+v %v", parsed, err)
	}
	if m.reads != 2 || m.bootReads != 2 {
		t.Fatalf("unfenced capture: %+v", m)
	}
}

func TestIdentityRejectsMalformedOrUnstableObservation(t *testing.T) {
	for _, name := range []string{"zero-start", "exited", "wrong-pid", "failed", "unknown", "boot-error", "bad-boot", "reuse", "boot-change", "exit-race"} {
		t.Run(name, func(t *testing.T) {
			m := ownerExample()
			switch name {
			case "zero-start":
				m.usage.born = 0
			case "exited":
				m.usage.exited = 101
			case "wrong-pid":
				m.usage.pid++
			case "failed":
				m.usage.state = usageFailed
				m.usage.err = syscall.EPERM
			case "unknown":
				m.usage = usageObservation{}
			case "boot-error":
				m.bootErr = syscall.EIO
			case "bad-boot":
				m.boot = "not-a-boot"
			case "reuse":
				m.mutate = func(m *ownerMock) {
					if m.reads == 2 {
						m.usage.born++
					}
				}
			case "boot-change":
				m.mutate = func(m *ownerMock) { m.boot = otherBoot }
			case "exit-race":
				m.mutate = func(m *ownerMock) {
					if m.reads == 2 {
						m.usage.exited = 101
					}
				}
			}
			got, err := captureOwner(context.Background(), m, 42)
			if err == nil || got != (OwnerIdentity{}) {
				t.Fatalf("unsafe capture: %+v %v", got, err)
			}
		})
	}
}

func TestIdentityRejectsInvalidInputsBeforeNativeReads(t *testing.T) {
	for _, pid := range []int{0, -1, math.MaxInt32 + 1, math.MaxInt} {
		m := ownerExample()
		got, err := captureOwner(context.Background(), m, pid)
		if !errors.Is(err, os.ErrInvalid) || got != (OwnerIdentity{}) || m.reads != 0 || m.bootReads != 0 {
			t.Fatalf("invalid %d: %+v %v %+v", pid, got, err, m)
		}
	}
	for _, token := range []string{"", "process-v1." + ownerTestBoot + ".042.100", "process-v1." + ownerTestBoot + ".42.0", "process-v1." + ownerTestBoot + ".2147483648.100", "process-v1." + strings.ToUpper(ownerTestBoot) + ".42.100", "process-v2." + ownerTestBoot + ".42.100"} {
		got, err := ParseOwnerIdentity(token)
		if err == nil || got != (OwnerIdentity{}) {
			t.Fatalf("invalid token %q: %+v %v", token, got, err)
		}
	}
	if (OwnerIdentity{}).Incarnation() != "" {
		t.Fatal("zero identity has incarnation")
	}
}

func TestOwnerObservationStatesAndErrors(t *testing.T) {
	for _, name := range []string{"present", "reused", "absent", "exited", "denied", "io", "unknown", "zero-start", "wrong-pid", "contradictory", "wrong-boot"} {
		t.Run(name, func(t *testing.T) {
			m := ownerExample()
			want := OwnerUnknown
			var cause error
			switch name {
			case "present":
				want = OwnerPresent
			case "reused":
				m.usage.born++
				want = OwnerDead
			case "absent":
				m.usage = usageObservation{state: usageFailed, err: errors.Join(errUsageRead, syscall.ESRCH)}
				want = OwnerDead
			case "exited":
				m.usage.exited = 101
				want = OwnerDead
			case "denied":
				cause = syscall.EPERM
				m.usage = usageObservation{state: usageFailed, err: cause}
			case "io":
				cause = syscall.EIO
				m.usage = usageObservation{state: usageFailed, err: cause}
			case "unknown":
				m.usage = usageObservation{}
			case "zero-start":
				m.usage.born = 0
			case "wrong-pid":
				m.usage.pid++
			case "contradictory":
				m.usage.err = syscall.ESRCH
			case "wrong-boot":
				m.boot = otherBoot
			}
			got := observeOwner(context.Background(), m, ownerExpected())
			if got.State != want || (want == OwnerUnknown) != (got.Err != nil) {
				t.Fatalf("observation: %+v", got)
			}
			if cause != nil && !errors.Is(got.Err, cause) {
				t.Fatalf("lost native cause: %v", got.Err)
			}
		})
	}
}

func TestOwnerGeneratedPIDReuseNeverMatches(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		m := ownerExample()
		m.usage.born = rapid.Uint64Range(101, math.MaxUint64).Draw(t, "replacement-start")
		got := observeOwner(context.Background(), m, ownerExpected())
		if got.State != OwnerDead || got.Err != nil {
			t.Fatalf("PID reuse: %+v", got)
		}
	})
}

func TestIdentityGeneratedRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := OwnerIdentity{boot: ownerTestBoot, pid: rapid.IntRange(1, math.MaxInt32).Draw(t, "pid"), start: rapid.Uint64Range(1, math.MaxUint64).Draw(t, "start")}
		parsed, err := ParseOwnerIdentity(id.Incarnation())
		if err != nil || parsed != id {
			t.Fatalf("round trip: %+v %v", parsed, err)
		}
	})
}

func TestQuiescenceNeverInfersDescendantClosure(t *testing.T) {
	for _, mode := range []string{"live", "dead", "reused", "child", "group", "denied", "full"} {
		t.Run(mode, func(t *testing.T) {
			m := ownerExample()
			switch mode {
			case "dead":
				m.usage = usageObservation{state: usageFailed, err: syscall.ESRCH}
			case "reused":
				m.usage.born++
			case "child":
				m.children = []int{43}
			case "group":
				m.group = []int{42, 44}
			case "denied":
				m.listErr = syscall.EPERM
			case "full":
				m.children = make([]int, quiescenceMemberLimit)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			got := observeQuiescence(ctx, m, ownerExpected())
			if got.State != QuiescenceUnknown || got.Err == nil {
				t.Fatalf("invented closure: %+v", got)
			}
			if m.listReads > 2 || m.reads > 4 {
				t.Fatalf("unbounded work: %+v", m)
			}
			if mode == "denied" && !errors.Is(got.Err, syscall.EPERM) {
				t.Fatalf("lost list cause: %v", got.Err)
			}
			if mode == "child" && len(got.Children) != 1 {
				t.Fatalf("missing child observation: %+v", got)
			}
			if mode == "group" && len(got.Group) != 2 {
				t.Fatalf("missing group observation: %+v", got)
			}
		})
	}
}

func TestOwnerAndQuiescenceCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := ownerExample()
	if got := observeOwner(ctx, m, ownerExpected()); got.State != OwnerUnknown || !errors.Is(got.Err, context.Canceled) {
		t.Fatalf("cancel: %+v", got)
	}
	if got := observeQuiescence(ctx, m, ownerExpected()); got.State != QuiescenceUnknown || !errors.Is(got.Err, context.Canceled) {
		t.Fatalf("cancel: %+v", got)
	}
	if m.reads != 0 || m.bootReads != 0 || m.listReads != 0 {
		t.Fatal("canceled observation read native data")
	}
	if got := observeQuiescence(context.Background(), m, ownerExpected()); got.Err == nil {
		t.Fatal("unbounded context accepted")
	}
	if got := observeOwner(never, m, ownerExpected()); got.Err == nil {
		t.Fatal("nil context accepted")
	}
	if got := observeOwner(context.Background(), m, OwnerIdentity{}); got.Err == nil {
		t.Fatal("zero identity accepted")
	}
}

func TestOwnerNativeIdentityAndReapedChild(t *testing.T) {
	deadline := time.Duration(usageExamples(t).ChildDeadlineNS)
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	own, err := CaptureOwner(ctx, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	again, err := CaptureOwner(ctx, os.Getpid())
	if err != nil || own != again {
		t.Fatalf("unstable identity: %+v %+v %v", own, again, err)
	}
	child := startUsageChild(t)
	id, err := CaptureOwner(ctx, child.command.Process.Pid)
	if err != nil || id == own || id.Start() <= own.Start() {
		t.Fatalf("child identity: %+v %v", id, err)
	}
	if got := ObserveOwner(ctx, id); got.State != OwnerPresent || got.Err != nil {
		t.Fatalf("live child: %+v", got)
	}
	child.reap()
	if got := ObserveOwner(ctx, id); got.State != OwnerDead || got.Err != nil {
		t.Fatalf("reaped child: %+v", got)
	}
	if got := ObserveQuiescence(ctx, id); got.State != QuiescenceUnknown || !errors.Is(got.Err, ErrDescendantsUnknown) {
		t.Fatalf("reaped child closure: %+v", got)
	}
}

func TestQuiescenceNativeEnumeratesOwnedChild(t *testing.T) {
	child := startUsageChild(t)
	deadline := time.Duration(usageExamples(t).ChildDeadlineNS)
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	id, err := CaptureOwner(ctx, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	got := ObserveQuiescence(ctx, id)
	if got.State != QuiescenceUnknown || got.Owner.State != OwnerPresent || !errors.Is(got.Err, ErrDescendantsUnknown) {
		t.Fatalf("native quiescence: %+v", got)
	}
	found := false
	for _, pid := range got.Children {
		if pid == child.command.Process.Pid {
			found = true
		}
	}
	if !found {
		t.Fatalf("owned child omitted: %+v", got)
	}
}

func TestOwnerGeneratedUnknownNeverBecomesDead(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		m := ownerExample()
		cause := []error{syscall.EPERM, syscall.EACCES, syscall.EIO, syscall.ENOSYS}[rapid.IntRange(0, 3).Draw(t, "cause")]
		m.usage = usageObservation{state: usageFailed, err: cause}
		got := observeOwner(context.Background(), m, ownerExpected())
		if got.State != OwnerUnknown || !errors.Is(got.Err, cause) {
			t.Fatalf("unknown converted to death: %+v", got)
		}
	})
}

func TestQuiescenceGeneratedEmptySnapshotsStayUnknown(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		m := ownerExample()
		if rapid.Bool().Draw(t, "owner-dead") {
			m.usage = usageObservation{state: usageFailed, err: syscall.ESRCH}
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		for range rapid.IntRange(1, 8).Draw(t, "observations") {
			got := observeQuiescence(ctx, m, ownerExpected())
			if got.State != QuiescenceUnknown || !errors.Is(got.Err, ErrDescendantsUnknown) {
				t.Fatalf("invented closure: %+v", got)
			}
		}
	})
}

func TestOwnerCancellationDuringNativeRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := ownerExample()
	m.mutate = func(m *ownerMock) { cancel() }
	got := observeOwner(ctx, m, ownerExpected())
	if got.State != OwnerUnknown || !errors.Is(got.Err, context.Canceled) {
		t.Fatalf("cancellation lost: %+v", got)
	}
	if m.reads != 1 {
		t.Fatalf("continued reads: %d", m.reads)
	}
}

func TestOwnerMalformedNativeAbsenceIsNotDeath(t *testing.T) {
	for _, field := range []string{"pid", "birth", "exit", "status"} {
		m := ownerExample()
		m.usage = usageObservation{state: usageFailed, err: syscall.ESRCH}
		switch field {
		case "pid":
			m.usage.pid = 42
		case "birth":
			m.usage.born = 100
		case "exit":
			m.usage.exited = 101
		case "status":
			m.usage.state = usageUnknown
		}
		got := observeOwner(context.Background(), m, ownerExpected())
		if got.State != OwnerUnknown || !errors.Is(got.Err, syscall.ESRCH) {
			t.Fatalf("malformed absence %s: %+v", field, got)
		}
	}
}

func TestIdentityRejectsInvalidBoots(t *testing.T) {
	for _, boot := range []string{"", zeroBoot, "001122334455-6677-8899-aabbccddeeff", ownerTestBoot + "x", "z0112233-4455-6677-8899-aabbccddeeff"} {
		m := ownerExample()
		m.boot = boot
		id, err := captureOwner(context.Background(), m, 42)
		if err == nil || id != (OwnerIdentity{}) {
			t.Fatalf("bad boot %q: %+v %v", boot, id, err)
		}
	}
}

func TestUsageNativeWitnesses(t *testing.T) {
	t.Run("current", TestNativeUsageReadsCurrentProcess)
	t.Run("invalid", TestNativeUsageRejectsNonpositivePIDs)
	t.Run("stable", TestNativeUsageIsStableForTheSameProcess)
	t.Run("narrowing", TestNativeUsageRefusesUnrepresentablePIDs)
	t.Run("child", TestNativeUsageDistinguishesAnOwnedChild)
	t.Run("reaped", TestNativeUsageNeverReturnsAReapedChildsOldObservation)
	t.Run("zero", TestNativeUsageZeroResultIsUnknown)
	t.Run("generated-stable", TestNativeUsageGeneratedStableCurrentReads)
	t.Run("generated-invalid", TestNativeUsageGeneratedInvalidInputsIsTheAntiProperty)
}
