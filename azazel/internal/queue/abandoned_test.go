package queue

import (
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/azazel/internal/footprint"
)

// The leader identity a test records for the render's group, and its number.
const (
	orphanGroup        = 4242
	orphanStartSeconds = 1000
	orphanStartMicros  = 25
)

// unsupervised claims the fixture job as a render that runs, records its
// leader, holds its attempt lock as the render does, and ends its lease
// by the given time before now. The returned function lets the lock go.
func unsupervised(t *testing.T, leaseEnded time.Duration) (Store, Job, func()) {
	t.Helper()
	s, job := claimedFixture(t)
	lock, err := s.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.transact(t.Context(), false, func(state *Snapshot) error {
		current := &state.Jobs[0]
		current.Executing, current.ProcessGroup = true, orphanGroup
		current.ProcessStartSeconds, current.ProcessStartMicroseconds = orphanStartSeconds, orphanStartMicros
		current.ProcessBootID = thisBootForTest
		current.LeaseUntil = time.Now().Add(-leaseEnded)
		return nil
	}); err != nil {
		lock.Close()
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return s, state.Jobs[0], func() { lock.Close() }
}

// A held attempt lock with a lease that no one renews is written plainly
// in the ledger: which group, and since when.
func TestReconcileWritesThatTheSupervisorIsGone(t *testing.T) {
	s, _, release := unsupervised(t, 2*renderLeaseDuration)
	defer release()
	if err := s.reconcile(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	held := state.Jobs[0].Held
	if err != nil || !strings.HasPrefix(held, supervisorGonePrefix) || !strings.Contains(held, "process group 4242") || !strings.Contains(held, "lease ended at ") {
		t.Fatal("the ledger did not say the supervisor is gone", held, err)
	}
	if err := s.reconcile(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	again, err := s.Status(t.Context())
	if err != nil || again.Jobs[0].Held != held || again.Jobs[0].State != Running {
		t.Fatal("a second look changed the note or the job", again, err)
	}
}

// Anti-test: a lease that is current, or past its end by less than a
// lease, says nothing; and a note is withdrawn once the lease is renewed.
func TestReconcileSaysNothingWhileTheSupervisorMayStillRenew(t *testing.T) {
	for name, ended := range map[string]time.Duration{
		"current":      -renderLeaseDuration,
		"just expired": renderLeaseDuration / 2,
	} {
		t.Run(name, func(t *testing.T) {
			s, _, release := unsupervised(t, ended)
			defer release()
			if err := s.reconcile(t.Context(), nil); err != nil {
				t.Fatal(err)
			}
			state, err := s.Status(t.Context())
			if err != nil || state.Jobs[0].Held != "" {
				t.Fatal("a note was written for a supervisor that may live", state.Jobs[0].Held, err)
			}
		})
	}
	s, job, release := unsupervised(t, 2*renderLeaseDuration)
	defer release()
	if err := s.reconcile(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if err := s.renewHeldRenderLease(t.Context(), job, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcile(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].Held != "" {
		t.Fatal("the note outlived the renewal", state.Jobs[0].Held, err)
	}
}

// signalled records the group signals a cancel sends.
type signalled struct{ calls [][2]int }

// via is the means of signalling for a test: this boot is boot, the leader
// of the group is found as found with err, and kills are recorded.
func (s *signalled) via(boot string, found footprint.Identity, err error) groupSignaller {
	return groupSignaller{
		boot:     func() (string, error) { return boot, nil },
		identity: func(int) (footprint.Identity, error) { return found, err },
		kill: func(group int, signal syscall.Signal) error {
			s.calls = append(s.calls, [2]int{group, int(signal)})
			return nil
		},
	}
}

// the leader as the ledger recorded it.
var recordedLeader = footprint.Identity{PID: orphanGroup, Group: orphanGroup,
	StartSeconds: orphanStartSeconds, StartMicroseconds: orphanStartMicros}

// Cancel of a job whose supervisor is gone signals the render's group.
func TestCancelSignalsTheGroupOfAJobWhoseSupervisorIsGone(t *testing.T) {
	s, job, release := unsupervised(t, 2*renderLeaseDuration)
	defer release()
	if err := s.reconcile(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	var sent signalled
	if err := s.cancel(t.Context(), job.Sequence, job.Request.Agent, sent.via(thisBootForTest, recordedLeader, nil)); err != nil {
		t.Fatal(err)
	}
	if len(sent.calls) != 1 || sent.calls[0] != [2]int{-orphanGroup, int(syscall.SIGKILL)} {
		t.Fatal("the group was not signaled", sent.calls)
	}
	state, err := s.Status(t.Context())
	if err != nil || !state.Jobs[0].CancelRequested {
		t.Fatal(state, err)
	}
}

// Anti-test: nothing is signaled when the group may not be the render's,
// when its identity was never recorded, when the supervisor is not known to
// be gone, or when the caller is not the job's agent.
func TestCancelDoesNotSignalAGroupItCannotConfirm(t *testing.T) {
	later := recordedLeader
	later.StartSeconds++
	for name, c := range map[string]struct {
		boot    string
		found   footprint.Identity
		err     error
		agent   string
		forget  bool
		noted   bool
		wantErr error
	}{
		"another process holds the number": {thisBootForTest, later, nil, "studio", false, true, nil},
		"recorded under another boot":      {otherBootForTest, recordedLeader, nil, "studio", false, true, nil},
		"leader unreadable and alive":      {thisBootForTest, footprint.Identity{}, errors.New("unreadable"), "studio", false, true, errors.New("unreadable")},
		"identity not recorded":            {thisBootForTest, recordedLeader, nil, "studio", true, true, errors.New("not recorded")},
		"supervisor not known to be gone":  {thisBootForTest, recordedLeader, nil, "studio", false, false, nil},
		"another agent":                    {thisBootForTest, recordedLeader, nil, "intruder", false, true, ErrOwner},
	} {
		t.Run(name, func(t *testing.T) {
			s, job, release := unsupervised(t, 2*renderLeaseDuration)
			defer release()
			if err := s.transact(t.Context(), false, func(state *Snapshot) error {
				if c.noted {
					state.Jobs[0].Held = supervisorGoneNote(state.Jobs[0])
				}
				if c.forget {
					state.Jobs[0].ProcessStartSeconds = 0
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			var sent signalled
			via := sent.via(c.boot, c.found, c.err)
			kill := via.kill
			via.kill = func(group int, signal syscall.Signal) error {
				if signal == 0 {
					return nil
				}
				return kill(group, signal)
			}
			err := s.cancel(t.Context(), job.Sequence, c.agent, via)
			if (err == nil) != (c.wantErr == nil) || c.wantErr == ErrOwner && !errors.Is(err, ErrOwner) {
				t.Fatal("unexpected result", err)
			}
			if len(sent.calls) != 0 {
				t.Fatal("a group was signaled", sent.calls)
			}
		})
	}
}

// A group whose leader is gone is not an error: there is nothing to signal.
func TestCancelOfAGoneGroupIsQuiet(t *testing.T) {
	var sent signalled
	via := sent.via(thisBootForTest, footprint.Identity{}, errors.New("unreadable"))
	via.kill = func(int, syscall.Signal) error { return syscall.ESRCH }
	job := Job{ProcessGroup: orphanGroup, ProcessStartSeconds: orphanStartSeconds, ProcessBootID: thisBootForTest}
	if err := signalOrphan(job, via); err != nil {
		t.Fatal(err)
	}
}
