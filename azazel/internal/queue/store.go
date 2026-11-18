package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	bolt "go.etcd.io/bbolt"
)

const recordVersion = 3
const maximumJobs = 10000

var bucketName = []byte("azazel-v1")
var stateKey = []byte("state")

// private checks the final entry; cooperating users keep its ancestors stable.
func private(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	mode := os.FileMode(0600)
	if directory {
		mode = 0700
	}
	if !ok || stat.Uid != uint32(os.Geteuid()) || info.Mode().Perm() != mode || info.IsDir() != directory || info.Mode()&os.ModeSymlink != 0 || (!directory && (!info.Mode().IsRegular() || stat.Nlink != 1)) {
		return ErrUnsafe
	}
	return nil
}

// location refuses aliases and shared writable queue roots.
func (s Store) location() error {
	if !filepath.IsAbs(s.root) || filepath.Clean(s.root) != s.root {
		return ErrUnsafe
	}
	return private(s.root, true)
}

// Init creates the queue once, without replacing any existing state.
func (s Store) Init(ctx context.Context, cfg Config) error {
	return s.initialize(ctx, cfg, nativeFiles{})
}

// initialize retains filesystem errors instead of inventing an empty queue.
func (s Store) initialize(ctx context.Context, cfg Config, files initialFiles) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(s.root) || filepath.Clean(s.root) != s.root {
		return ErrUnsafe
	}
	if cfg.Game != "" {
		if _, err := fingerprint(cfg.Game, true); err != nil {
			return err
		}
	}
	if err := files.MakeDirectory(s.root); err != nil {
		return err
	}
	file, err := files.Create(filepath.Join(s.root, "queue.db"))
	if err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = s.transact(ctx, true, func(st *Snapshot) error {
		*st = Snapshot{Version: recordVersion, Config: cfg, Jobs: []Job{}}
		return nil
	}); err != nil {
		return err
	}
	if err = files.SyncDirectory(s.root); err != nil {
		return err
	}
	return files.SyncDirectory(filepath.Dir(s.root))
}

// transact opens bbolt briefly so independent processes can submit and observe.
func (s Store) transact(ctx context.Context, initialize bool, change func(*Snapshot) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.location(); err != nil {
		return err
	}
	path := filepath.Join(s.root, "queue.db")
	if err := private(path, false); err != nil {
		return err
	}
	db, err := openDatabase(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	return updateDatabase(ctx, db, initialize, change)
}

// updateDatabase applies one checked transition to an already opened database.
func updateDatabase(ctx context.Context, db *bolt.DB, initialize bool, change func(*Snapshot) error) error {
	return db.Update(func(tx *bolt.Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		bucket := tx.Bucket(bucketName)
		if initialize {
			var err error
			bucket, err = tx.CreateBucket(bucketName)
			if err != nil {
				return err
			}
		} else if bucket == nil {
			return ErrUnsafe
		}
		var state Snapshot
		if !initialize {
			if err := json.Unmarshal(bucket.Get(stateKey), &state); err != nil {
				return err
			}
			if err := state.validate(); err != nil {
				return err
			}
		}
		if err := change(&state); err != nil {
			return err
		}
		if err := state.validate(); err != nil {
			return err
		}
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if bytes.Equal(bucket.Get(stateKey), data) {
			return nil
		}
		return bucket.Put(stateKey, data)
	})
}

// validate refuses corrupt ordering or reused ownership across active attempts.
func (st Snapshot) validate() error {
	if !st.Tuning.valid() {
		return ErrUnsafe
	}
	if st.Version != recordVersion || st.Config.Game != "" && !filepath.IsAbs(st.Config.Game) || len(st.Jobs) > maximumJobs {
		return ErrUnsafe
	}
	active := map[string]bool{}
	for i, j := range st.Jobs {
		if j.Sequence != uint64(i+1) || j.Request.Agent == "" || j.Request.Key == "" {
			return ErrUnsafe
		}
		switch j.State {
		case Running:
			if active[j.Worker] || j.Worker == "" || j.ProcessGroup < 0 || j.ProcessGroup == 1 {
				return ErrUnsafe
			}
			active[j.Worker] = true
		case Queued, Succeeded, Failed, Cancelled, Interrupted:
		default:
			return ErrUnsafe
		}
	}
	return nil
}

// acknowledged finds an existing idempotency key without rereading its inputs.
func acknowledged(st *Snapshot, r Request) (Job, bool, error) {
	for _, j := range st.Jobs {
		if j.Request.Agent == r.Agent && j.Request.Key == r.Key {
			if !reflect.DeepEqual(normalizedRequest(j.Request), normalizedRequest(r)) {
				return Job{}, true, ErrConflict
			}
			return j, true, nil
		}
	}
	return Job{}, false, nil
}

// normalizedRequest matches the omitted empty fields in the durable encoding.
func normalizedRequest(request Request) Request {
	if len(request.Inputs) == 0 {
		request.Inputs = nil
	}
	if len(request.Environment) == 0 {
		request.Environment = nil
	}
	return request
}

// Submit fingerprints outside the transaction so large inputs cannot block status.
func (s Store) Submit(ctx context.Context, r Request) (Job, error) {
	var out Job
	if err := r.validate(); err != nil {
		return out, err
	}
	st, err := s.Status(ctx)
	if err != nil {
		return out, err
	}
	if existing, found, err := acknowledged(&st, r); found {
		return existing, err
	}
	paths := []string{r.Command}
	executables := 1
	if st.Config.Game != "" {
		paths = append(paths, st.Config.Game)
		executables++
	}
	paths = append(paths, r.Inputs...)
	files := make([]Fingerprint, 0, len(paths))
	for i, path := range paths {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		f, err := fingerprint(path, i < executables)
		if err != nil {
			return out, err
		}
		files = append(files, f)
	}
	err = s.transact(ctx, false, func(st *Snapshot) error {
		if existing, found, err := acknowledged(st, r); found {
			out = existing
			return err
		}
		if len(st.Jobs) == maximumJobs {
			return fmt.Errorf("queue history is full")
		}
		out = Job{Sequence: uint64(len(st.Jobs) + 1), Request: r, Files: files, State: Queued, Submitted: time.Now().UTC()}
		st.Jobs = append(st.Jobs, out)
		return nil
	})
	return out, err
}

// Status returns a stable snapshot independent of later submissions.
func (s Store) Status(ctx context.Context) (Snapshot, error) {
	var out Snapshot
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := s.location(); err != nil {
		return out, err
	}
	path := filepath.Join(s.root, "queue.db")
	if err := private(path, false); err != nil {
		return out, err
	}
	db, err := openDatabaseMode(ctx, path, true)
	if err != nil {
		return out, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return ErrUnsafe
		}
		if err := json.Unmarshal(bucket.Get(stateKey), &out); err != nil {
			return err
		}
		return out.validate()
	})
	return out, err
}

// job finds a sequence in its checked ordered ledger.
func (st *Snapshot) job(seq uint64) (*Job, error) {
	if seq == 0 || seq > uint64(len(st.Jobs)) {
		return nil, ErrMissing
	}
	return &st.Jobs[seq-1], nil
}

// Cancel only accepts the original agent and retains running ownership.
func (s Store) Cancel(ctx context.Context, seq uint64, agent string) error {
	return s.transact(ctx, false, func(st *Snapshot) error {
		j, err := st.job(seq)
		if err != nil {
			return err
		}
		if j.Request.Agent != agent {
			return ErrOwner
		}
		if j.State == Queued {
			j.State = Cancelled
			j.Result = Result{State: Cancelled, Reason: "cancelled before start"}
			j.Finished = time.Now().UTC()
		}
		if j.State == Running {
			j.CancelRequested = true
		}
		return nil
	})
}

// Pause records a reason and leaves pending jobs queued.
func (s Store) Pause(ctx context.Context, reason string) error {
	if reason == "" {
		return fmt.Errorf("pause needs a reason")
	}
	return s.transact(ctx, false, func(st *Snapshot) error { st.Paused = reason; return nil })
}

// Resume clears the latch only on an explicit request.
func (s Store) Resume(ctx context.Context) error {
	return s.transact(ctx, false, func(st *Snapshot) error { st.Paused = ""; return nil })
}

// claim atomically selects the oldest pending request.
func (s Store) claim(ctx context.Context, owner string) (Job, error) {
	var out Job
	if owner == "" {
		return out, ErrOwner
	}
	err := s.transact(ctx, false, func(st *Snapshot) error {
		if st.Paused != "" {
			return fmt.Errorf("%w: %s", ErrPaused, st.Paused)
		}
		for _, j := range st.Jobs {
			if j.State == Running {
				return ErrActive
			}
		}
		for i := range st.Jobs {
			j := &st.Jobs[i]
			if j.State == Queued {
				j.State = Running
				j.Attempt++
				j.ProcessStartSeconds, j.ProcessStartMicroseconds, j.ProcessBootID = 0, 0, ""
				j.Worker = owner
				j.Started = time.Now().UTC()
				out = *j
				return nil
			}
		}
		return ErrEmpty
	})
	return out, err
}

// finish preserves failures and refuses another worker's acknowledgement.
func (s Store) finish(ctx context.Context, seq uint64, owner string, result Result) error {
	if result.State != Succeeded && result.State != Failed && result.State != Cancelled && result.State != Interrupted {
		return ErrUnsafe
	}
	return s.transact(ctx, false, func(st *Snapshot) error {
		j, err := st.job(seq)
		if err != nil {
			return err
		}
		if j.Worker == owner && j.State == result.State && j.Result == result {
			return nil
		}
		if j.Worker != owner || j.State != Running {
			return ErrOwner
		}
		j.State = result.State
		j.Result = result
		j.Finished = time.Now().UTC()
		return nil
	})
}

// openDatabase retries brief cross-process contention within a bounded wait.
func openDatabase(ctx context.Context, path string) (*bolt.DB, error) {
	return openDatabaseMode(ctx, path, false)
}

// openDatabaseMode opens path with a five second bound on lock waits. readOnly
// opens the file for reading only.
func openDatabaseMode(ctx context.Context, path string, readOnly bool) (*bolt.DB, error) {
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if err := bounded.Err(); err != nil {
			return nil, err
		}
		db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 50 * time.Millisecond, ReadOnly: readOnly})
		if !errors.Is(err, bolt.ErrTimeout) {
			return db, err
		}
	}
}

// deferAttempt returns unstarted work to the queue after a durable host stop.
func (s Store) deferAttempt(ctx context.Context, j Job, reason string) error {
	return s.transact(ctx, false, func(st *Snapshot) error {
		current, err := st.job(j.Sequence)
		if err != nil {
			return err
		}
		if current.Worker != j.Worker || current.State != Running || current.Executing {
			return ErrOwner
		}
		current.State = Queued
		current.Worker = ""
		current.Held = reason
		current.Started = time.Time{}
		return nil
	})
}
