//go:build darwin

package locking

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"pgregory.net/rapid"
)

// authoredLockData stores witness limits outside the native adapter.
//
//go:embed testdata/locking.yaml
var authoredLockData []byte

// singleAttemptResult is defined by the one result each attempt sends.
const singleAttemptResult = 1

// noLockAttempts is defined as the cardinality of an empty attempt sequence.
const noLockAttempts = 0

// noAttemptWaitNanos is defined as zero nanoseconds before any wait elapses.
const noAttemptWaitNanos int64 = 0

// lockTestInputs preserves the original timeout and repetition domains.
type lockTestInputs struct {
	AttemptTimeoutNanos int64 `yaml:"attempt_timeout_ns"`
	MinimumAttempts     int   `yaml:"minimum_attempts"`
	MaximumAttempts     int   `yaml:"maximum_attempts"`
	RepeatedReleases    int   `yaml:"repeated_releases"`
}

// lockExamples checks limits once for each test before generated attempts.
func lockExamples(t *testing.T) lockTestInputs {
	t.Helper()
	var examples lockTestInputs
	if err := yamlfile.Read("testdata/locking.yaml", authoredLockData,
		&examples); err != nil {
		t.Fatal(err)
	}
	if examples.AttemptTimeoutNanos <= noAttemptWaitNanos ||
		examples.MinimumAttempts <= noLockAttempts ||
		examples.MaximumAttempts < examples.MinimumAttempts ||
		examples.RepeatedReleases <= noLockAttempts {
		t.Fatal("authored lock examples lost a positive timeout or repeat range")
	}
	return examples
}

// lockFixture owns a small file and closes each opened handle.
// Real file descriptors create contention without mocking the kernel.
func lockFixture(t *testing.T) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	keepLockHandle(t, file)
	if _, err := file.Write([]byte("owned fixture")); err != nil {
		t.Fatal(err)
	}
	return file
}

// keepLockHandle retains cleanup even when an assertion ends the test.
func keepLockHandle(t *testing.T, file *os.File) {
	t.Helper()
	owner := lockHandleOwner{reporter: t, file: file}
	t.Cleanup(owner.close)
}

// lockHandleOwner retains the original unit reporter and owned descriptor.
type lockHandleOwner struct {
	reporter *testing.T
	file     *os.File
}

// close tolerates a handle already closed by the witness under test.
func (owner lockHandleOwner) close() {
	if err := owner.file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		owner.reporter.Error(err)
	}
}

// unlock preserves the fixture's separately registered raw-lock cleanup.
func (owner lockHandleOwner) unlock() {
	unlockFixture(owner.reporter, owner.file)
}

// openLockHandle opens the same entry independently, without truncating it.
func openLockHandle(t *testing.T, file *os.File) *os.File {
	t.Helper()
	opened, err := os.OpenFile(file.Name(), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	keepLockHandle(t, opened)
	return opened
}

// rawLock needs nonblocking kernel behavior, with no production wrapper.
func rawLock(file *os.File) error {
	descriptor := int(file.Fd())
	return syscall.Flock(descriptor, syscall.LOCK_EX|syscall.LOCK_NB)
}

// unlockFixture changes only this owned fixture's advisory lock.
func unlockFixture(reporter rapid.TB, file *os.File) {
	reporter.Helper()
	descriptor := int(file.Fd())
	if err := syscall.Flock(descriptor, syscall.LOCK_UN); err != nil {
		reporter.Error(err)
	}
}

// holdFixture retains explicit unlock cleanup as well as descriptor cleanup.
func holdFixture(t *testing.T, file *os.File) {
	t.Helper()
	if err := rawLock(file); err != nil {
		t.Fatalf("fixture lock: %v", err)
	}
	owner := lockHandleOwner{reporter: t, file: file}
	t.Cleanup(owner.unlock)
}

// acquisitionAttempt sends one result to the same buffered completion channel.
type acquisitionAttempt struct {
	lock localLock
	done chan<- acquisition
}

// run keeps the selected lock and result send together for the goroutine.
func (attempt acquisitionAttempt) run() {
	attempt.done <- attempt.lock.tryAcquire()
}

// attemptWithin rejects a wrapper that blocks instead of reporting busy.
// Fixture cleanup releases the holder even after timeout assertion.
func (examples lockTestInputs) attemptWithin(reporter rapid.TB,
	lock localLock) acquisition {
	reporter.Helper()
	done := make(chan acquisition, singleAttemptResult)
	attempt := acquisitionAttempt{lock: lock, done: done}
	go attempt.run()
	timer := time.NewTimer(time.Duration(examples.AttemptTimeoutNanos))
	defer timer.Stop()
	select {
	case result := <-done:
		return result
	case <-timer.C:
		reporter.Fatal("nonblocking lock attempt exceeded two seconds")
		return acquisition{}
	}
}

// requireKernelBusy distinguishes actual exclusion from a returned enum.
func requireKernelBusy(reporter rapid.TB, file *os.File) {
	reporter.Helper()
	if err := rawLock(file); !errors.Is(err, syscall.EWOULDBLOCK) {
		reporter.Fatalf("independent handle: %v, want would-block", err)
	}
}

// requireFixtureBytes checks that taking a lock does not rewrite the inode.
func requireFixtureBytes(t *testing.T, file *os.File) {
	t.Helper()
	before, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.Stat(file.Name())
	if err != nil || !os.SameFile(before, current) {
		t.Fatalf("fixture entry changed: %v", err)
	}
	bytes, err := os.ReadFile(file.Name())
	if err != nil || string(bytes) != "owned fixture" {
		t.Fatalf("fixture bytes = %q, error %v", bytes, err)
	}
}

// TestNativeLockAcquires establishes exclusion through another open handle.
func TestNativeLockAcquires(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	result := examples.attemptWithin(t, localLock{file: file})
	if result.state != acquisitionHeld || result.err != nil {
		t.Fatalf("acquisition = %+v, want held", result)
	}
	requireKernelBusy(t, other)
	requireFixtureBytes(t, file)
}

// TestNativeLockKeepsAlreadyHeld pins the nonrecursive same-handle meaning.
func TestNativeLockKeepsAlreadyHeld(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	holdFixture(t, file)
	result := examples.attemptWithin(t, localLock{file: file})
	if result.state != acquisitionHeld || result.err != nil {
		t.Fatalf("held acquisition = %+v", result)
	}
	requireKernelBusy(t, other)
}

// TestNativeLockLeavesOtherFilesIndependent acquires a different entry.
func TestNativeLockLeavesOtherFilesIndependent(t *testing.T) {
	examples := lockExamples(t)
	held := lockFixture(t)
	heldProbe := openLockHandle(t, held)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	holdFixture(t, held)
	result := examples.attemptWithin(t, localLock{file: file})
	if result.state != acquisitionHeld || result.err != nil {
		t.Fatalf("independent acquisition = %+v", result)
	}
	requireKernelBusy(t, heldProbe)
	requireKernelBusy(t, other)
	requireFixtureBytes(t, held)
	requireFixtureBytes(t, file)
}

// TestNativeLockReportsContention refuses a different open of a held inode.
func TestNativeLockReportsContention(t *testing.T) {
	examples := lockExamples(t)
	holder := lockFixture(t)
	contender := openLockHandle(t, holder)
	holdFixture(t, holder)
	result := examples.attemptWithin(t, localLock{file: contender})
	if result.state != acquisitionBusy ||
		!errors.Is(result.err, syscall.EWOULDBLOCK) {
		t.Fatalf("contended acquisition = %+v, want busy cause", result)
	}
	requireFixtureBytes(t, contender)
}

// TestNativeLockRefusesMissingHandle prevents an omitted lock becoming held.
func TestNativeLockRefusesMissingHandle(t *testing.T) {
	examples := lockExamples(t)
	result := examples.attemptWithin(t, localLock{})
	if result.state != acquisitionFailed || !errors.Is(result.err, os.ErrInvalid) {
		t.Fatalf("missing acquisition = %+v, want invalid", result)
	}
}

// TestNativeLockRefusesClosedHandle retains the kernel's bad-descriptor cause.
func TestNativeLockRefusesClosedHandle(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	result := examples.attemptWithin(t, localLock{file: file})
	if result.state != acquisitionFailed || !errors.Is(result.err, syscall.EBADF) {
		t.Fatalf("closed acquisition = %+v, want bad descriptor", result)
	}
}

// TestNativeLockDoesNotReleaseTheHolder observes ownership after a refusal.
func TestNativeLockDoesNotReleaseTheHolder(t *testing.T) {
	examples := lockExamples(t)
	holder := lockFixture(t)
	contender := openLockHandle(t, holder)
	probe := openLockHandle(t, holder)
	holdFixture(t, holder)
	result := examples.attemptWithin(t, localLock{file: contender})
	if result.state != acquisitionBusy ||
		!errors.Is(result.err, syscall.EWOULDBLOCK) {
		t.Fatalf("refused acquisition = %+v, want busy cause", result)
	}
	requireKernelBusy(t, probe)
	unlockFixture(t, holder)
	if err := rawLock(probe); err != nil {
		t.Fatalf("holder released but probe could not acquire: %v", err)
	}
	requireKernelBusy(t, contender)
}

// lockOwnershipProperty retains the same three independent open descriptions.
type lockOwnershipProperty struct {
	holder, contender, probe *os.File
	examples                 lockTestInputs
}

// check reports and releases each trial through that actual Rapid context.
func (property lockOwnershipProperty) check(trial *rapid.T) {
	holder := property.holder
	contender := property.contender
	probe := property.probe
	competing := rapid.Bool().Draw(trial, "competing holder")
	repeats := rapid.IntRange(property.examples.MinimumAttempts,
		property.examples.MaximumAttempts).Draw(trial, "repeated attempts")
	defer unlockFixture(trial, holder)
	defer unlockFixture(trial, contender)
	if competing {
		if err := rawLock(holder); err != nil {
			trial.Fatal(err)
		}
	}
	for range repeats {
		result := property.examples.attemptWithin(trial,
			localLock{file: contender})
		if competing {
			if result.state != acquisitionBusy ||
				!errors.Is(result.err, syscall.EWOULDBLOCK) {
				trial.Fatalf("competing acquisition = %+v", result)
			}
		} else if result.state != acquisitionHeld || result.err != nil {
			trial.Fatalf("free acquisition = %+v", result)
		}
		requireKernelBusy(trial, probe)
	}
}

// TestNativeLockGeneratedOwnership compares repeated attempts with kernel state.
func TestNativeLockGeneratedOwnership(t *testing.T) {
	examples := lockExamples(t)
	holder := lockFixture(t)
	contender := openLockHandle(t, holder)
	probe := openLockHandle(t, holder)
	property := lockOwnershipProperty{
		holder: holder, contender: contender, probe: probe,
		examples: examples,
	}
	rapid.Check(t, property.check)
}
