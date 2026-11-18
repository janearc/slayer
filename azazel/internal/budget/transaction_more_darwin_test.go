package budget

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// armedContext reports no error until arm is called with a count, then
// answers nil that many times and Canceled after.
type armedContext struct {
	context.Context
	armed bool
	skip  int
}

// Err answers nil until armed and the skipped answers are used up, then
// context.Canceled.
func (ctx *armedContext) Err() error {
	if !ctx.armed {
		return nil
	}
	if ctx.skip > 0 {
		ctx.skip--
		return nil
	}
	return context.Canceled
}

// TestAdmissionDirectoryRefusesBadPaths checks the paths that are refused
// before any directory is opened or that fail the ownership and mode check.
func TestAdmissionDirectoryRefusesBadPaths(t *testing.T) {
	for _, path := range []string{"relative/path", "", "/tmp/../tmp", "/tmp/", "/./tmp"} {
		if file, err := admissionDirectory(path); !errors.Is(err, errAdmissionUnsafe) || file != nil {
			t.Errorf("%q: %v", path, err)
		}
	}
	if os.Geteuid() != 0 {
		if file, err := admissionDirectory("/"); !errors.Is(err, errAdmissionUnsafe) || file != nil {
			t.Errorf("root directory: %v", err)
		}
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(base, "missing")
	if file, err := admissionDirectory(missing); !errors.Is(err, unix.ENOENT) || file != nil {
		t.Errorf("missing directory: %v", err)
	}
}

// TestStableLockFailures checks that a lock file that is closed, or no longer
// named lock, is reported.
func TestStableLockFailures(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		tx, _ := transactionCommitFixture(t)
		_ = tx.lock.Close()
		if err := tx.stableLock(); !errors.Is(err, unix.EBADF) {
			t.Errorf("got %v", err)
		}
	})
	t.Run("removed", func(t *testing.T) {
		tx, _ := transactionCommitFixture(t)
		if err := os.Remove(filepath.Join(tx.directory.Name(), "lock")); err != nil {
			t.Fatal(err)
		}
		if err := tx.stableLock(); !errors.Is(err, unix.ENOENT) {
			t.Errorf("got %v", err)
		}
	})
	t.Run("replaced", func(t *testing.T) {
		tx, _ := transactionCommitFixture(t)
		dir := tx.directory.Name()
		if err := os.Rename(filepath.Join(dir, "lock"), filepath.Join(dir, "old")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lock"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		if err := tx.stableLock(); !errors.Is(err, errAdmissionUnsafe) {
			t.Errorf("got %v", err)
		}
	})
}

// TestBeginRefusesMoreEntries checks a missing context, a lock file with
// contents, entries that are not files and an intent that is not ours.
func TestBeginRefusesMoreEntries(t *testing.T) {
	var noContext context.Context
	dir, binding := transactionFixture(t)
	if tx, err := beginAdmissionTransaction(noContext, dir, binding); !errors.Is(err, errAdmissionUnsafe) || tx != nil {
		t.Errorf("nil context: %v", err)
	}
	cases := map[string]struct {
		setup func(t *testing.T, dir string)
		want  []error
	}{
		"lock with contents": {func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "lock"), []byte("x"), 0600); err != nil {
				t.Fatal(err)
			}
		}, []error{errAdmissionUnsafe}},
		"pending name is a directory": {func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "state-next"), 0700); err != nil {
				t.Fatal(err)
			}
		}, []error{errAdmissionPending, errAdmissionUnsafe}},
		"intent is a directory": {func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "intent"), 0700); err != nil {
				t.Fatal(err)
			}
		}, []error{errAdmissionUnsafe}},
		"intent from another predecessor": {func(t *testing.T, dir string) {
			state := recordExample()
			intent := admissionIntent{version: 1, phase: intentObserving, boot: state.boot, owner: state.owners[0], predecessor: sha256.Sum256([]byte("other"))}
			encoded, err := encodeAdmissionIntent(intent)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "intent"), encoded, 0600); err != nil {
				t.Fatal(err)
			}
		}, []error{errAdmissionPending, errAdmissionRecord}},
	}
	for name, c := range cases {
		dir, binding := transactionFixture(t)
		c.setup(t, dir)
		tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
		if tx != nil {
			_ = tx.close()
			t.Errorf("%s: accepted", name)
			continue
		}
		for _, want := range c.want {
			if !errors.Is(err, want) {
				t.Errorf("%s: %v is not %v", name, err, want)
			}
		}
	}
}

// TestObserveAndCloseRefuseNil checks the nil receivers and an intent that is
// already there.
func TestObserveAndCloseRefuseNil(t *testing.T) {
	var tx *admissionTransaction
	if err := tx.observe(context.Background(), reservationOwner{}); !errors.Is(err, errAdmissionUnsafe) {
		t.Errorf("observe: %v", err)
	}
	if err := tx.close(); !errors.Is(err, errAdmissionUnsafe) {
		t.Errorf("close: %v", err)
	}
	if _, err := tx.admit(context.Background(), transactionAdmissionRequest{}, nil); !errors.Is(err, errAdmissionUnsafe) {
		t.Errorf("admit: %v", err)
	}
	if _, err := tx.release(context.Background(), transactionReleaseRequest{}); !errors.Is(err, errAdmissionUnsafe) {
		t.Errorf("release: %v", err)
	}
}

// TestObserveRefusesExistingIntent checks that an intent that appears after
// the entry check stops observe from writing over it.
func TestObserveRefusesExistingIntent(t *testing.T) {
	dir, binding := transactionFixture(t)
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.close()
	if err := os.WriteFile(filepath.Join(dir, "intent"), []byte("theirs"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.observe(context.Background(), recordExample().owners[0]); !errors.Is(err, unix.EEXIST) {
		t.Fatalf("got %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "intent")); string(got) != "theirs" {
		t.Errorf("intent was changed to %q", got)
	}
}

// TestPublishCheckFirstChecks checks that a done context and a replaced lock
// are caught before the publish hook runs.
func TestPublishCheckFirstChecks(t *testing.T) {
	tx, _ := transactionCommitFixture(t)
	hooked := false
	tx.publish = func(string) error { hooked = true; return nil }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tx.publishCheck(ctx, "point"); !errors.Is(err, context.Canceled) {
		t.Errorf("done context: %v", err)
	}
	if err := os.Remove(filepath.Join(tx.directory.Name(), "lock")); err != nil {
		t.Fatal(err)
	}
	if err := tx.publishCheck(context.Background(), "point"); !errors.Is(err, unix.ENOENT) {
		t.Errorf("removed lock: %v", err)
	}
	if hooked {
		t.Error("hook ran after a failed check")
	}
}

// TestAdmitFailsWhenFilesChange checks that admit fails and leaves the intent
// in place when a file it needs is gone or damaged.
func TestAdmitFailsWhenFilesChange(t *testing.T) {
	cases := map[string]func(t *testing.T, tx *admissionTransaction){
		"lock replaced": func(t *testing.T, tx *admissionTransaction) {
			dir := tx.directory.Name()
			if err := os.Rename(filepath.Join(dir, "lock"), filepath.Join(dir, "old")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "lock"), nil, 0600); err != nil {
				t.Fatal(err)
			}
		},
		"state removed": func(t *testing.T, tx *admissionTransaction) {
			if err := os.Remove(filepath.Join(tx.directory.Name(), "state")); err != nil {
				t.Fatal(err)
			}
		},
		"intent removed": func(t *testing.T, tx *admissionTransaction) {
			if err := os.Remove(filepath.Join(tx.directory.Name(), "intent")); err != nil {
				t.Fatal(err)
			}
		},
		"intent damaged": func(t *testing.T, tx *admissionTransaction) {
			if err := os.WriteFile(filepath.Join(tx.directory.Name(), "intent"), []byte("bad"), 0600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, damage := range cases {
		tx, request := transactionCommitFixture(t)
		damage(t, tx)
		measured := false
		_, err := tx.admit(context.Background(), request, func(context.Context) (admissionMeasurement, error) {
			measured = true
			return admissionMeasurement{}, nil
		})
		if err == nil || measured {
			t.Errorf("%s: err %v, measured %t", name, err, measured)
		}
	}
}

// TestReleaseFailsWhenFilesChange checks the same for release: a state or
// intent that is gone, or an intent that names a different owner.
func TestReleaseFailsWhenFilesChange(t *testing.T) {
	cases := map[string]func(t *testing.T, tx *admissionTransaction){
		"state removed": func(t *testing.T, tx *admissionTransaction) {
			if err := os.Remove(filepath.Join(tx.directory.Name(), "state")); err != nil {
				t.Fatal(err)
			}
		},
		"intent removed": func(t *testing.T, tx *admissionTransaction) {
			if err := os.Remove(filepath.Join(tx.directory.Name(), "intent")); err != nil {
				t.Fatal(err)
			}
		},
		"intent for another owner": func(t *testing.T, tx *admissionTransaction) {
			other := tx.state.owners[0]
			if other == tx.owner {
				other = tx.state.owners[1]
			}
			encoded, err := encodeAdmissionIntent(admissionIntent{version: admissionRecordVersion, phase: intentObserving, boot: tx.state.boot, owner: other, predecessor: tx.digest})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tx.directory.Name(), "intent"), encoded, 0600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, damage := range cases {
		tx, request := transactionReleaseFixture(t, false)
		damage(t, tx)
		if got, err := tx.release(context.Background(), request); err == nil || got != (transactionReleaseOutcome{}) {
			t.Errorf("%s: %+v %v", name, got, err)
		}
	}
}

// TestPublishStepsFailWhenFilesVanish checks that each rename, unlink and
// create in publishing reports a file that was removed or put in its way by
// the step before, and that an intent removed by then is written back.
func TestPublishStepsFailWhenFilesVanish(t *testing.T) {
	remove := func(name string) func(dir string) {
		return func(dir string) { _ = os.Remove(filepath.Join(dir, name)) }
	}
	create := func(name string) func(dir string) {
		return func(dir string) { _ = os.WriteFile(filepath.Join(dir, name), []byte("in the way"), 0600) }
	}
	cases := []struct {
		point      string
		act        func(dir string)
		want       error
		wantIntent bool
	}{
		{"prepared-write", create("intent-prepared"), unix.EEXIST, true},
		{"prepared-rename", remove("intent-prepared"), unix.ENOENT, true},
		{"state-write", create("state-next"), unix.EEXIST, true},
		{"state-rename", remove("state-next"), unix.ENOENT, true},
		{"retiring-write", create("retiring"), unix.EEXIST, true},
		{"intent-remove", remove("intent"), unix.ENOENT, false},
		{"retiring-remove", remove("retiring"), unix.ENOENT, true},
	}
	for _, c := range cases {
		tx, request := transactionCommitFixture(t)
		dir := tx.directory.Name()
		tx.publish = func(at string) error {
			if at == c.point {
				c.act(dir)
			}
			return nil
		}
		got, err := tx.admit(context.Background(), request, transactionMeasurement(request))
		if !errors.Is(err, c.want) || got != (transactionAdmissionOutcome{}) {
			t.Errorf("%s: %+v %v", c.point, got, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "intent")); (err == nil) != c.wantIntent && c.point != "intent-remove" {
			t.Errorf("%s: intent present is %t", c.point, err == nil)
		}
	}
}

// TestPublishRestoreFailureIsReported checks that when the intent cannot be
// written back, that failure is reported along with the first.
func TestPublishRestoreFailureIsReported(t *testing.T) {
	tx, request := transactionCommitFixture(t)
	dir := tx.directory.Name()
	injected := errors.New("publish fault")
	tx.publish = func(at string) error {
		if at != "retired-directory" {
			return nil
		}
		if err := os.WriteFile(filepath.Join(dir, "intent"), []byte("in the way"), 0600); err != nil {
			t.Error(err)
		}
		return injected
	}
	_, err := tx.admit(context.Background(), request, transactionMeasurement(request))
	if !errors.Is(err, injected) || !errors.Is(err, unix.EEXIST) {
		t.Errorf("got %v", err)
	}
}

// TestPublishEndsOnLateCancellation checks that a context that ends after
// the last step but before the final lock check is returned, and that the
// transition was already complete.
func TestPublishEndsOnLateCancellation(t *testing.T) {
	tx, request := transactionCommitFixture(t)
	ctx := &armedContext{Context: context.Background()}
	tx.publish = func(at string) error {
		if at == "complete-directory" {
			ctx.armed, ctx.skip = true, 1
		}
		return nil
	}
	_, err := tx.admit(ctx, request, transactionMeasurement(request))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	dir := tx.directory.Name()
	if _, statErr := os.Stat(filepath.Join(dir, "retiring")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("retiring file should be gone: %v", statErr)
	}
	state, readErr := os.ReadFile(filepath.Join(dir, "state"))
	if readErr != nil || sha256.Sum256(state) == tx.digest {
		t.Errorf("state was not replaced: %v", readErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "intent")); statErr != nil {
		t.Errorf("intent should be written back after the failure: %v", statErr)
	}
}

// TestAdmitReturnsCancellationAfterMeasure checks that a context that ends
// after the measurement stops admit before anything is written.
func TestAdmitReturnsCancellationAfterMeasure(t *testing.T) {
	tx, request := transactionCommitFixture(t)
	ctx := &armedContext{Context: context.Background()}
	measure := func(c context.Context) (admissionMeasurement, error) {
		ctx.armed = true
		return transactionMeasurement(request)(c)
	}
	if _, err := tx.admit(ctx, request, measure); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(tx.directory.Name(), "intent-prepared")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("wrote after cancellation: %v", err)
	}
}

// TestBeginAndObserveReturnLateCancellation checks that a context that ends
// at the last check of begin, or of observe, is returned and leaves nothing
// held.
func TestBeginAndObserveReturnLateCancellation(t *testing.T) {
	dir, binding := transactionFixture(t)
	begin := &armedContext{Context: context.Background(), armed: true, skip: 1}
	if tx, err := beginAdmissionTransaction(begin, dir, binding); !errors.Is(err, context.Canceled) || tx != nil {
		t.Fatalf("begin: %v", err)
	}
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	defer tx.close()
	observe := &armedContext{Context: context.Background(), armed: true, skip: 2}
	if err := tx.observe(observe, recordExample().owners[0]); !errors.Is(err, context.Canceled) {
		t.Fatalf("observe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "intent")); err != nil {
		t.Errorf("intent should be written before the last check: %v", err)
	}
	if tx.ready {
		t.Error("transaction is ready after a cancelled observe")
	}
}

// TestSampleReturnsCancellationBeforeDiskQuery checks that a context that
// ends after the last read, before the disk is queried, is returned.
func TestSampleReturnsCancellationBeforeDiskQuery(t *testing.T) {
	ctx := &armedContext{Context: context.Background(), armed: true, skip: 9}
	host := &fakeHost{statfs: func() (unix.Statfs_t, error) {
		t.Error("disk was queried")
		return unix.Statfs_t{}, nil
	}}
	if _, err := sampleAdmissionHost(ctx, openFakeDir(t), fakeBinding(), host); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}
