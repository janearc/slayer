package queue

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/janearc/slayer/azazel/internal/footprint"
)

// bootTail is the part of a boot identifier after its first eight digits.
const bootTail = "-89ab-cdef-0123-456789abcdef"

// bootFixture is a boot identifier used by the leader metadata tests.
const bootFixture = "01234567" + bootTail

func TestLeaderMetadataIsOptionalAndRejectsUncertainIdentity(t *testing.T) {
	var old Job
	if err := json.Unmarshal([]byte(`{"process_group":7}`), &old); err != nil || old.ProcessStartSeconds != 0 || old.ProcessBootID != "" {
		t.Fatal(old, err)
	}
	valid := footprint.Leader{BootID: bootFixture, Identity: footprint.Identity{PID: 7, Group: 7, StartSeconds: 10, StartMicroseconds: 4}}
	for _, row := range []struct {
		name        string
		leader      footprint.Leader
		known, want bool
	}{
		{"valid", valid, true, true}, {"unavailable", valid, false, false},
		{"wrong pid", footprint.Leader{BootID: bootFixture, Identity: footprint.Identity{PID: 8, Group: 7, StartSeconds: 10}}, true, false},
		{"bad boot", footprint.Leader{BootID: "bad", Identity: valid.Identity}, true, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			job := Job{ProcessBootID: "old", ProcessStartSeconds: 5}
			recordLeader(&job, 7, row.leader, row.known)
			if job.ProcessGroup != 7 || (job.ProcessStartSeconds != 0) != row.want {
				t.Fatal(job)
			}
			if !row.want && (job.ProcessBootID != "" || job.ProcessStartMicroseconds != 0) {
				t.Fatal(job)
			}
		})
	}
}

func TestMissingLeaderObservationNeverFailsRender(t *testing.T) {
	store, job := executableFixture(t, "exit 0")
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	result := store.executeUnderLockWithLeader(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(int) (footprint.Leader, bool) { return footprint.Leader{}, false })
	if result.State != Succeeded || result.Uncertain {
		t.Fatal(result)
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || snapshot.Jobs[0].ProcessGroup <= 1 || snapshot.Jobs[0].ProcessStartSeconds != 0 || snapshot.Jobs[0].ProcessBootID != "" {
		t.Fatal(snapshot, err)
	}
}

func TestBlockedLeaderCaptureDoesNotBlockFastCompletionOrAttachLate(t *testing.T) {
	store, job := executableFixture(t, "exit 0")
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	release := make(chan struct{})
	returned := make(chan struct{})
	started := time.Now()
	result := store.executeUnderLockWithLeader(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(pid int) (footprint.Leader, bool) {
		<-release
		defer close(returned)
		return footprint.Leader{BootID: bootFixture, Identity: footprint.Identity{PID: pid, Group: pid, StartSeconds: 1}}, true
	})
	if result.State != Succeeded || time.Since(started) > time.Second {
		t.Fatal(result, time.Since(started))
	}
	close(release)
	<-returned
	snapshot, err := store.Status(t.Context())
	if err != nil || snapshot.Jobs[0].ProcessStartSeconds != 0 || snapshot.Jobs[0].ProcessBootID != "" {
		t.Fatal(snapshot, err)
	}
}

func TestBlockedLeaderCaptureHonorsCancellation(t *testing.T) {
	store, job := executableFixture(t, "exec /bin/sleep 30")
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	release := make(chan struct{})
	defer close(release)
	ctx, cancel := context.WithCancel(t.Context())
	capturing := make(chan struct{})
	go func() { <-capturing; cancel() }()
	defer cancel()
	started := time.Now()
	result := store.executeUnderLockWithLeader(ctx, job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(int) (footprint.Leader, bool) { close(capturing); <-release; return footprint.Leader{}, false })
	if result.State != Cancelled || result.Uncertain || time.Since(started) > time.Second {
		t.Fatal(result, time.Since(started))
	}
}

func TestLeaderCapturePanicDoesNotFailRender(t *testing.T) {
	store, job := executableFixture(t, "exit 0")
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	result := store.executeUnderLockWithLeader(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(int) (footprint.Leader, bool) { panic("sample failure") })
	if result.State != Succeeded || result.Uncertain {
		t.Fatal(result)
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || snapshot.Jobs[0].ProcessStartSeconds != 0 {
		t.Fatal(snapshot, err)
	}
}

func TestNativeLeaderMetadataIsRecordedOnLaunch(t *testing.T) {
	store, job := executableFixture(t, "exec /bin/sleep 0.2")
	result := store.execute(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil))
	if result.State != Succeeded || result.Uncertain {
		t.Fatal(result)
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || snapshot.Jobs[0].ProcessGroup <= 1 || snapshot.Jobs[0].ProcessStartSeconds == 0 || !footprint.ValidBootID(snapshot.Jobs[0].ProcessBootID) {
		t.Fatal(snapshot, err)
	}
}

func TestBothClaimPathsClearPreviousAttemptLeader(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy", false: "dispatcher"}[legacy], func(t *testing.T) {
			store, request := fixture(t)
			if _, err := store.Submit(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			if err := store.transact(t.Context(), false, func(state *Snapshot) error {
				state.Jobs[0].ProcessStartSeconds = 10
				state.Jobs[0].ProcessStartMicroseconds = 4
				state.Jobs[0].ProcessBootID = bootFixture
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			var got Job
			var err error
			if legacy {
				got, err = store.claim(t.Context(), "new")
			} else {
				got, err = store.claimAvailable(t.Context(), "new", Capacity{Ready: true, GPUKnown: true}, time.Now())
			}
			if err != nil || got.ProcessStartSeconds != 0 || got.ProcessStartMicroseconds != 0 || got.ProcessBootID != "" {
				t.Fatal(got, err)
			}
		})
	}
}
