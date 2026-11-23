package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/mock/gomock"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

// TestUnsafeRootsAndFiles rejects aliases, permissions and linked state.
func TestUnsafeRootsAndFiles(t *testing.T) {
	for _, kind := range []string{"relative", "alias", "directory-mode", "database-mode", "database-link", "database-directory", "database-absent"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := fixture(t)
			switch kind {
			case "relative":
				s = At(".")
			case "alias":
				alias := filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(s.root, alias); err != nil {
					t.Fatal(err)
				}
				s = At(alias)
			case "directory-mode":
				if err := os.Chmod(s.root, 0755); err != nil {
					t.Fatal(err)
				}
			case "database-mode":
				if err := os.Chmod(filepath.Join(s.root, "queue.db"), 0644); err != nil {
					t.Fatal(err)
				}
			case "database-link":
				if err := os.Link(filepath.Join(s.root, "queue.db"), filepath.Join(s.root, "other")); err != nil {
					t.Fatal(err)
				}
			case "database-directory":
				if err := os.Remove(filepath.Join(s.root, "queue.db")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(s.root, "queue.db"), 0600); err != nil {
					t.Fatal(err)
				}
			case "database-absent":
				if err := os.Remove(filepath.Join(s.root, "queue.db")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.Status(t.Context()); err == nil {
				t.Fatal("accepted unsafe root")
			}
		})
	}
}

// TestCorruptStateIsRefused preserves unusable records rather than resetting them.
func TestCorruptStateIsRefused(t *testing.T) {
	for _, data := range []string{"not-json", `{"version":2,"config":{"game":"/usr/bin/true"}}`,
		`{"version":3,"config":{"game":"relative"}}`,
		`{"version":3,"config":{"game":"/usr/bin/true"},"jobs":[{"sequence":2}]}`,
		`{"version":3,"config":{"game":"/usr/bin/true"},"jobs":[{"sequence":1,"request":{"agent":"a","key":"b"},"state":"unknown"}]}`,
		`{"version":3,"config":{"game":"/usr/bin/true"},"jobs":[{"sequence":1,"request":{"agent":"a","key":"b"},"state":"running"}]}`,
	} {
		s, _ := fixture(t)
		db, err := bolt.Open(filepath.Join(s.root, "queue.db"), 0600, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = db.Update(func(tx *bolt.Tx) error { return tx.Bucket(bucketName).Put(stateKey, []byte(data)) })
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
		if _, err = s.Status(t.Context()); err == nil {
			t.Fatal("accepted corrupt state", data)
		}
	}
}

// TestMissingJobsAndWrongAttemptsNeverReleaseWork checks every mutation boundary.
func TestMissingJobsAndWrongAttemptsNeverReleaseWork(t *testing.T) {
	s, r := fixture(t)
	if err := s.Cancel(t.Context(), 0, r.Agent); !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
	if err := s.finish(t.Context(), 1, "w", Result{State: Failed}); !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
	if err := s.finish(t.Context(), 1, "w", Result{State: Queued}); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
	if _, err := s.claim(t.Context(), ""); !errors.Is(err, ErrOwner) {
		t.Fatal(err)
	}
	if err := s.Pause(t.Context(), ""); err == nil {
		t.Fatal("accepted empty reason")
	}
	if err := s.Work(t.Context(), nil, false); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	executor := NewMockExecutor(gomock.NewController(t))
	if err := s.Work(ctx, executor, false); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.Init(ctx, Config{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := At(".").Init(t.Context(), Config{}); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
	if err := s.Init(t.Context(), Config{Game: "/missing"}); err == nil {
		t.Fatal("accepted nonexistent Game")
	}
	if err := s.Init(t.Context(), Config{Game: "/usr/bin/true"}); !errors.Is(err, os.ErrExist) {
		t.Fatal(err)
	}
}

// TestContendedDatabaseHonorsCancellation bounds cross-process admission waiting.
func TestContendedDatabaseHonorsCancellation(t *testing.T) {
	s, _ := fixture(t)
	db, err := bolt.Open(filepath.Join(s.root, "queue.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if _, err = s.Status(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

// TestInputRefusalsNeverStart protects command type and byte identity.
func TestInputRefusalsNeverStart(t *testing.T) {
	s, r := fixture(t)
	for _, path := range []string{"relative", "/nonexistent-azazel-input", r.Directory} {
		bad := r
		bad.Inputs = []string{path}
		if _, err := s.Submit(t.Context(), bad); err == nil {
			t.Fatal("accepted bad input", path)
		}
	}
	file := filepath.Join(r.Directory, "not-executable")
	if err := os.WriteFile(file, []byte("nothing"), 0600); err != nil {
		t.Fatal(err)
	}
	bad := r
	bad.Command = file
	if _, err := s.Submit(t.Context(), bad); err == nil {
		t.Fatal("accepted non-executable")
	}
	bad = r
	bad.Directory = file
	if _, err := s.Submit(t.Context(), bad); err == nil {
		t.Fatal("accepted regular cwd")
	}
	bad.Directory = filepath.Join(r.Directory, "absent")
	if _, err := s.Submit(t.Context(), bad); err == nil {
		t.Fatal("accepted absent cwd")
	}
	if err := (Job{}).verify(); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
	j, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	j.Files[0].Path = "/absent-executable"
	if err = j.verify(); err == nil {
		t.Fatal("accepted missing executable")
	}
}

// TestWorkerLockRefusals cannot replace another worker's identity.
func TestWorkerLockRefusals(t *testing.T) {
	s, _ := fixture(t)
	if err := os.Symlink("/dev/null", filepath.Join(s.root, "worker.lock")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.workerLock(); err == nil {
		t.Fatal("accepted linked lock")
	}
	executor := NewMockExecutor(gomock.NewController(t))
	if err := At("/nonexistent-queue").Work(t.Context(), executor, false); err == nil {
		t.Fatal("accepted missing queue")
	}
}

// TestServeStopsOnCancellation waits for submissions without occupying Game.
func TestServeStopsOnCancellation(t *testing.T) {
	s, _ := fixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 350*time.Millisecond)
	defer cancel()
	executor := NewMockExecutor(gomock.NewController(t))
	if err := s.Work(ctx, executor, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

// TestTerminalReceiptRetriesDoNotChangeResults handles lost acknowledgements.
func TestTerminalReceiptRetriesDoNotChangeResults(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	if _, err := s.claim(t.Context(), "worker"); err != nil {
		t.Fatal(err)
	}
	result := Result{State: Succeeded}
	if err := s.finish(t.Context(), 1, "worker", result); err != nil {
		t.Fatal(err)
	}
	if err := s.finish(t.Context(), 1, "worker", result); err != nil {
		t.Fatal(err)
	}
	if err := s.finish(t.Context(), 1, "worker", Result{State: Failed}); !errors.Is(err, ErrOwner) {
		t.Fatal(err)
	}
}

// TestSnapshotIsDetached proves callers cannot mutate persisted identities.
func TestSnapshotIsDetached(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Jobs[0].Request.Args[0] = "changed"
	again, err := s.Status(t.Context())
	if err != nil || again.Jobs[0].Request.Args[0] != "draw" {
		t.Fatal(again, err)
	}
}

// TestInternalExecutionCannotSkipGame leaves the attempt fenced on direct calls.
func TestInternalExecutionCannotSkipGame(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	j, err := s.claim(t.Context(), "w")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAME_LANES", "")
	if err = s.Execute(t.Context(), j.Sequence, j.Worker, os.Stdout); err == nil {
		t.Fatal("bypassed Game")
	}
}

// TestStoreRollbackPreservesThePredecessor exercises transaction failures.
func TestStoreRollbackPreservesThePredecessor(t *testing.T) {
	s, _ := fixture(t)
	sentinel := errors.New("transaction refused")
	err := s.transact(t.Context(), false, func(st *Snapshot) error { st.Paused = "must not commit"; return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	status, err := s.Status(t.Context())
	if err != nil || status.Paused != "" {
		t.Fatal(status, err)
	}
	err = s.transact(t.Context(), false, func(st *Snapshot) error { st.Version = recordVersion + 1; return nil })
	if !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	err = s.transact(ctx, false, func(st *Snapshot) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestEnvironmentIsBoundedAndDoesNotDuplicate carries only render settings.
func TestEnvironmentIsBoundedAndDoesNotDuplicate(t *testing.T) {
	env := setEnvironment([]string{"GOMAXPROCS=99", "X=y"}, "GOMAXPROCS", "2")
	encoded, _ := json.Marshal(env)
	if string(encoded) != `["X=y","GOMAXPROCS=2"]` {
		t.Fatal(env)
	}
}

// TestCompletionDoesNotConfuseAnUnknownGroupWithAnEmptyOne stays fenced.
func TestCompletionDoesNotConfuseAnUnknownGroupWithAnEmptyOne(t *testing.T) {
	group, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if result := completion(group, nil); !result.Uncertain {
		t.Fatal(result)
	}
}

// TestStoreCapacityAndSchemaFailureNeverDiscardExistingJobs keeps history intact.
func TestStoreCapacityAndSchemaFailureNeverDiscardExistingJobs(t *testing.T) {
	s, r := fixture(t)
	first, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.transact(t.Context(), false, func(st *Snapshot) error {
		st.Jobs = make([]Job, maximumJobs)
		for i := range st.Jobs {
			j := first
			j.Sequence = uint64(i + 1)
			j.Request.Key = fmt.Sprint(i)
			st.Jobs[i] = j
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r.Key = "one-new"
	if _, err = s.Submit(t.Context(), r); err == nil {
		t.Fatal("overflowed history")
	}
	db, err := bolt.Open(filepath.Join(s.root, "queue.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err = updateDatabase(ctx, db, false, func(*Snapshot) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err = db.Update(func(tx *bolt.Tx) error { return tx.DeleteBucket(bucketName) }); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err = s.Status(t.Context()); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
}

// TestEmptyOrForeignGroupIdentifiersAreNeverSignaled prevents broad kill targets.
func TestEmptyOrForeignGroupIdentifiersAreNeverSignaled(t *testing.T) {
	for _, id := range []int{-1, 0, 1} {
		if result := stopGroup(id, make(chan error), Result{State: Cancelled}); !result.Uncertain {
			t.Fatal(result)
		}
	}
}

// TestFingerprintIsBoundToTheExecutedCommand refuses substituted request paths.
func TestFingerprintIsBoundToTheExecutedCommand(t *testing.T) {
	s, r := fixture(t)
	j, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	j.Request.Command = "/usr/bin/false"
	if err = j.verify(); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
}

// TestDeferredAttemptCannotReleaseAnotherWorker checks prestart ownership.
func TestDeferredAttemptCannotReleaseAnotherWorker(t *testing.T) {
	s, j := executableFixture(t, "exit 0")
	bad := j
	bad.Sequence = 99
	if err := s.deferAttempt(t.Context(), bad, "held"); !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
	bad = j
	bad.Worker = "wrong"
	if err := s.deferAttempt(t.Context(), bad, "held"); !errors.Is(err, ErrOwner) {
		t.Fatal(err)
	}
}

// TestIdenticalConcurrentRetriesReturnOneAcknowledgement exercises the second check.
func TestIdenticalConcurrentRetriesReturnOneAcknowledgement(t *testing.T) {
	s, r := fixture(t)
	var wg sync.WaitGroup
	results := make(chan Job, 8)
	failures := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); j, err := s.Submit(t.Context(), r); results <- j; failures <- err }()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	for j := range results {
		if j.Sequence != 1 {
			t.Fatal(j)
		}
	}
	status, err := s.Status(t.Context())
	if err != nil || len(status.Jobs) != 1 {
		t.Fatal(status, err)
	}
}
