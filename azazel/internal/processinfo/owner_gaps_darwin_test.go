package processinfo

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// scriptedOwner is an ownerReader whose every answer is a function of how
// many times it has been asked, so a test can change the world between
// two reads.
type scriptedOwner struct {
	boot      func(n int) (string, error)
	usage     func(n int) usageObservation
	relations func() (processRelations, error)
	children  func() ([]int, error)
	group     func() ([]int, error)
	boots     int
	usages    int
}

// ReadBoot answers from the boot script, counting the call.
func (s *scriptedOwner) ReadBoot() (string, error) {
	s.boots++
	return s.boot(s.boots)
}

// ReadUsage answers from the usage script, counting the call.
func (s *scriptedOwner) ReadUsage(usageQuery) usageObservation {
	s.usages++
	return s.usage(s.usages)
}

// ReadRelations answers from the relations script.
func (s *scriptedOwner) ReadRelations(int) (processRelations, error) { return s.relations() }

// ListChildren answers from the children script.
func (s *scriptedOwner) ListChildren(int, int) ([]int, error) { return s.children() }

// ListGroup answers from the group script.
func (s *scriptedOwner) ListGroup(int, int) ([]int, error) { return s.group() }

// steadyOwner is a scriptedOwner for the process in ownerExpected, with a
// steady boot, a present process, one child and a group of two.
func steadyOwner() *scriptedOwner {
	return &scriptedOwner{
		boot:      func(int) (string, error) { return ownerTestBoot, nil },
		usage:     func(int) usageObservation { return usageObservation{pid: 42, born: 100, state: usageObserved} },
		relations: func() (processRelations, error) { return processRelations{pid: 42, parent: 1, group: 42}, nil },
		children:  func() ([]int, error) { return []int{43}, nil },
		group:     func() ([]int, error) { return []int{42, 43}, nil },
	}
}

// flippingContext reports no error for its first quiet calls to Err and
// context.Canceled from then on. It keeps the deadline of its parent.
type flippingContext struct {
	context.Context
	quiet int
	calls int
}

// Err counts the call and says whether the context is still alive.
func (c *flippingContext) Err() error {
	c.calls++
	if c.calls > c.quiet {
		return context.Canceled
	}
	return nil
}

// newFlipping makes a flippingContext that is cancelled after quiet calls.
func newFlipping(t *testing.T, quiet int) *flippingContext {
	t.Helper()
	parent, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	return &flippingContext{Context: parent, quiet: quiet}
}

// TestParseOwnerIdentityRefusals checks each way a token is turned away.
func TestParseOwnerIdentityRefusals(t *testing.T) {
	good := "process-v1." + ownerTestBoot + ".42.100"
	if id, err := ParseOwnerIdentity(good); err != nil || id != ownerExpected() {
		t.Fatalf("good token: %+v %v", id, err)
	}
	cases := map[string]string{
		"too long":           good + strings.Repeat("0", 80),
		"start not a number": "process-v1." + ownerTestBoot + ".42.abc",
		"start too large":    "process-v1." + ownerTestBoot + ".42.18446744073709551616",
		"pid not a number":   "process-v1." + ownerTestBoot + ".x.100",
		"wrong prefix":       "process-v2." + ownerTestBoot + ".42.100",
		"missing part":       "process-v1." + ownerTestBoot + ".42",
		"start zero":         "process-v1." + ownerTestBoot + ".42.0",
		"leading zero pid":   "process-v1." + ownerTestBoot + ".042.100",
		"plus sign in start": "process-v1." + ownerTestBoot + ".42.+100",
	}
	for name, token := range cases {
		if id, err := ParseOwnerIdentity(token); !errors.Is(err, os.ErrInvalid) || id != (OwnerIdentity{}) {
			t.Errorf("%s: %+v %v", name, id, err)
		}
	}
}

// TestValidBootChecksTheDashes checks that a boot id with a character other
// than a dash where a dash belongs is refused.
func TestValidBootChecksTheDashes(t *testing.T) {
	if !validBoot(ownerTestBoot) {
		t.Fatal("the test boot is refused")
	}
	for _, at := range []int{8, 13, 18, 23} {
		bad := []byte(ownerTestBoot)
		bad[at] = '_'
		if validBoot(string(bad)) {
			t.Errorf("a boot with %q at %d was accepted", '_', at)
		}
	}
	if validBoot(strings.ToUpper(ownerTestBoot)) {
		t.Error("an uppercase boot was accepted")
	}
}

// TestOwnerUsageRefusesACancelledContext checks that ownerUsage reads
// nothing for a context that is already done or missing.
func TestOwnerUsageRefusesACancelledContext(t *testing.T) {
	reader := steadyOwner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ownerUsage(ctx, reader, 42); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if _, err := ownerUsage(nil, reader, 42); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("nil context: %v", err)
	}
	if reader.usages != 0 {
		t.Errorf("read usage %d times for a context that was done", reader.usages)
	}
}

// TestCaptureOwnerRefusesWhatChangesBetweenReads checks the second usage
// read failing and the closing boot read failing or moving.
func TestCaptureOwnerRefusesWhatChangesBetweenReads(t *testing.T) {
	cases := []struct {
		name  string
		shape func(*scriptedOwner)
		want  error
	}{
		{"second usage read fails", func(s *scriptedOwner) {
			s.usage = func(n int) usageObservation {
				if n == 2 {
					return usageObservation{state: usageFailed, err: syscall.EPERM}
				}
				return usageObservation{pid: 42, born: 100, state: usageObserved}
			}
		}, syscall.EPERM},
		{"closing boot read fails", func(s *scriptedOwner) {
			s.boot = func(n int) (string, error) {
				if n == 2 {
					return "", syscall.EIO
				}
				return ownerTestBoot, nil
			}
		}, syscall.EIO},
		{"closing boot read is another boot", func(s *scriptedOwner) {
			s.boot = func(n int) (string, error) {
				if n == 2 {
					return otherBoot, nil
				}
				return ownerTestBoot, nil
			}
		}, ErrOwnerUnknown},
	}
	for _, c := range cases {
		reader := steadyOwner()
		c.shape(reader)
		id, err := captureOwner(context.Background(), reader, 42)
		if !errors.Is(err, c.want) || id != (OwnerIdentity{}) {
			t.Errorf("%s: %+v %v", c.name, id, err)
		}
	}
}

// TestObserveOwnerGivesUpWhenTheBootMoves checks that a boot that differs
// after the usage read leaves the owner unknown, not present.
func TestObserveOwnerGivesUpWhenTheBootMoves(t *testing.T) {
	reader := steadyOwner()
	reader.boot = func(n int) (string, error) {
		if n == 2 {
			return otherBoot, nil
		}
		return ownerTestBoot, nil
	}
	got := observeOwner(context.Background(), reader, ownerExpected())
	if got.State != OwnerUnknown || !errors.Is(got.Err, ErrOwnerUnknown) {
		t.Fatalf("moved boot: %+v", got)
	}
}

// TestCancellationAtEveryCheckIsReported cancels each of the three
// observations after every possible number of context checks. A context
// that is cancelled before the work ends must be reported as cancelled and
// must never produce an answer; one that is not must succeed.
func TestCancellationAtEveryCheckIsReported(t *testing.T) {
	type run func(ctx context.Context) error
	runs := map[string]run{
		"capture": func(ctx context.Context) error {
			_, err := captureOwner(ctx, steadyOwner(), 42)
			return err
		},
		"owner": func(ctx context.Context) error {
			return observeOwner(ctx, steadyOwner(), ownerExpected()).Err
		},
		"quiescence": func(ctx context.Context) error {
			got := observeQuiescence(ctx, steadyOwner(), ownerExpected())
			if !errors.Is(got.Err, ErrDescendantsUnknown) {
				t.Errorf("quiescence lost its standing error: %v", got.Err)
			}
			if got.Err == ErrDescendantsUnknown {
				return nil
			}
			return got.Err
		},
	}
	for name, work := range runs {
		whole := newFlipping(t, 1<<20)
		if err := work(whole); err != nil {
			t.Fatalf("%s: an uncancelled run failed: %v", name, err)
		}
		checks := whole.calls
		if checks < 4 {
			t.Fatalf("%s: only %d context checks; the sweep would prove little", name, checks)
		}
		for quiet := 0; quiet < checks; quiet++ {
			if err := work(newFlipping(t, quiet)); !errors.Is(err, context.Canceled) {
				t.Errorf("%s: cancelled after %d checks, got %v", name, quiet, err)
			}
		}
	}
}

// TestObserveQuiescenceRefusals checks each answer of the reader that
// makes the closure unknown, and that what was read is kept or dropped as
// documented.
func TestObserveQuiescenceRefusals(t *testing.T) {
	cases := []struct {
		name         string
		shape        func(*scriptedOwner)
		want         error
		children     []int
		group        []int
		ownerPresent bool
	}{
		{"relations unreadable", func(s *scriptedOwner) {
			s.relations = func() (processRelations, error) { return processRelations{}, syscall.EPERM }
		}, syscall.EPERM, nil, nil, true},
		{"relations name another pid", func(s *scriptedOwner) {
			s.relations = func() (processRelations, error) { return processRelations{pid: 7, group: 7}, nil }
		}, ErrOwnerUnknown, nil, nil, true},
		{"relations with no group", func(s *scriptedOwner) {
			s.relations = func() (processRelations, error) { return processRelations{pid: 42}, nil }
		}, ErrOwnerUnknown, nil, nil, true},
		{"a child with a bad pid", func(s *scriptedOwner) {
			s.children = func() ([]int, error) { return []int{43, 0}, nil }
		}, ErrOwnerUnknown, nil, nil, true},
		{"group unreadable", func(s *scriptedOwner) {
			s.group = func() ([]int, error) { return nil, syscall.EPERM }
		}, syscall.EPERM, []int{43}, nil, true},
		{"group fills the buffer", func(s *scriptedOwner) {
			s.group = func() ([]int, error) { return make([]int, quiescenceMemberLimit), nil }
		}, ErrOwnerUnknown, []int{43}, nil, true},
		{"a group member with a bad pid", func(s *scriptedOwner) {
			s.group = func() ([]int, error) { return []int{42, -5}, nil }
		}, ErrOwnerUnknown, []int{43}, nil, true},
		{"owner replaced during the reads", func(s *scriptedOwner) {
			s.usage = func(n int) usageObservation {
				born := uint64(100)
				if n >= 2 {
					born = 101
				}
				return usageObservation{pid: 42, born: born, state: usageObserved}
			}
		}, ErrOwnerUnknown, nil, nil, true},
		{"owner unknown at the start", func(s *scriptedOwner) {
			s.boot = func(int) (string, error) { return otherBoot, nil }
		}, ErrOwnerUnknown, nil, nil, false},
	}
	for _, c := range cases {
		reader := steadyOwner()
		c.shape(reader)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		got := observeQuiescence(ctx, reader, ownerExpected())
		cancel()
		if got.State != QuiescenceUnknown || !errors.Is(got.Err, ErrDescendantsUnknown) || !errors.Is(got.Err, c.want) {
			t.Errorf("%s: %+v", c.name, got)
		}
		if len(got.Children) != len(c.children) || len(got.Group) != len(c.group) {
			t.Errorf("%s: kept children %v and group %v, want %v and %v", c.name, got.Children, got.Group, c.children, c.group)
		}
		if (got.Owner.State == OwnerPresent) != c.ownerPresent && c.name != "owner replaced during the reads" {
			t.Errorf("%s: owner state %v", c.name, got.Owner.State)
		}
	}
}

// TestNativeRelationsAndLists checks the native reader's refusals with
// pids and limits the system cannot answer for.
func TestNativeRelationsAndLists(t *testing.T) {
	reader := darwinOwnerReader{}
	if _, err := reader.ReadRelations(0); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("relations of pid 0: %v", err)
	}
	own, err := reader.ReadRelations(os.Getpid())
	if err != nil || own.pid != os.Getpid() || own.parent != os.Getppid() || own.group != syscall.Getpgrp() {
		t.Errorf("own relations: %+v %v", own, err)
	}
	// Pid 2147483647 is the largest a 32-bit pid can be; no process has it.
	if _, err := reader.ReadRelations(2147483647); !errors.Is(err, ErrOwnerUnknown) {
		t.Errorf("relations of a pid nothing has: %v", err)
	}
	for _, limit := range []int{0, -1, quiescenceMemberLimit + 1} {
		if _, err := reader.ListChildren(os.Getpid(), limit); !errors.Is(err, os.ErrInvalid) {
			t.Errorf("children with limit %d: %v", limit, err)
		}
	}
	if _, err := reader.ListGroup(0, 10); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("group of pid 0: %v", err)
	}
	if _, err := reader.ListChildren(2147483647, 10); !errors.Is(err, ErrOwnerUnknown) {
		t.Errorf("children of a pid nothing has: %v", err)
	}
	// The group of this process holds at least this process, so a buffer
	// of one is full and is refused.
	if _, err := reader.ListGroup(syscall.Getpgrp(), 1); !errors.Is(err, ErrOwnerUnknown) {
		t.Errorf("a full buffer of one: %v", err)
	}
	members, err := reader.ListGroup(syscall.Getpgrp(), 64)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pid := range members {
		if pid == os.Getpid() {
			found = true
		}
	}
	if !found {
		t.Errorf("this process %s is not in its own group %v", strconv.Itoa(os.Getpid()), members)
	}
}
