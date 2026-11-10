//go:build darwin

package locking

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// releaseAttempt retains the lock and the same buffered completion channel.
type releaseAttempt struct {
	lock localLock
	done chan<- error
}

// run sends the single release outcome without losing the selected handle.
func (attempt releaseAttempt) run() {
	attempt.done <- attempt.lock.release()
}

// releaseWithin bounds a native attempt without changing fixture ownership.
func (examples lockTestInputs) releaseWithin(reporter rapid.TB,
	lock localLock) error {
	reporter.Helper()
	done := make(chan error, singleAttemptResult)
	attempt := releaseAttempt{lock: lock, done: done}
	go attempt.run()
	timer := time.NewTimer(time.Duration(examples.AttemptTimeoutNanos))
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		reporter.Fatal("lock release exceeded two seconds")
		return syscall.ETIMEDOUT
	}
}

// requireKernelAcquisition proves freedom through a different open handle.
func requireKernelAcquisition(reporter rapid.TB, owner, probe *os.File) {
	reporter.Helper()
	if err := rawLock(owner); err != nil {
		reporter.Fatalf("independent acquisition after release: %v", err)
	}
	requireKernelBusy(reporter, probe)
	unlockFixture(reporter, owner)
}

// TestNativeReleaseUnlocks leaves the handle, inode and contents intact.
func TestNativeReleaseUnlocks(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	probe := openLockHandle(t, file)
	holdFixture(t, file)
	requireKernelBusy(t, other)
	if err := examples.releaseWithin(t, localLock{file: file}); err != nil {
		t.Fatalf("release: %v", err)
	}
	requireKernelAcquisition(t, other, probe)
	requireFixtureBytes(t, file)
}

// TestNativeReleaseAcceptsUnlocked handles an open descriptor with no lock.
func TestNativeReleaseAcceptsUnlocked(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	probe := openLockHandle(t, file)
	if err := examples.releaseWithin(t, localLock{file: file}); err != nil {
		t.Fatalf("unlocked release: %v", err)
	}
	requireKernelAcquisition(t, other, probe)
	requireFixtureBytes(t, file)
}

// TestNativeReleaseCanRepeat pins idempotence without closing the handle.
func TestNativeReleaseCanRepeat(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	probe := openLockHandle(t, file)
	holdFixture(t, file)
	for range examples.RepeatedReleases {
		if err := examples.releaseWithin(t, localLock{file: file}); err != nil {
			t.Fatalf("repeated release: %v", err)
		}
		requireKernelAcquisition(t, other, probe)
	}
	requireFixtureBytes(t, file)
}

// TestNativeReleaseRefusesMissing retains the omitted-handle cause.
func TestNativeReleaseRefusesMissing(t *testing.T) {
	examples := lockExamples(t)
	err := examples.releaseWithin(t, localLock{})
	if !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("missing release = %v, want invalid", err)
	}
}

// TestNativeReleaseRefusesClosed retains the native bad-descriptor cause.
func TestNativeReleaseRefusesClosed(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	err := examples.releaseWithin(t, localLock{file: file})
	if !errors.Is(err, syscall.EBADF) {
		t.Fatalf("closed release = %v, want bad descriptor", err)
	}
}

// TestNativeReleaseLeavesAnotherOpenHeld cannot unlock a competing owner.
func TestNativeReleaseLeavesAnotherOpenHeld(t *testing.T) {
	examples := lockExamples(t)
	holder := lockFixture(t)
	borrowed := openLockHandle(t, holder)
	probe := openLockHandle(t, holder)
	holdFixture(t, holder)
	if err := examples.releaseWithin(t, localLock{file: borrowed}); err != nil {
		t.Fatalf("nonholder release: %v", err)
	}
	requireKernelBusy(t, probe)
	requireKernelBusy(t, borrowed)
	requireFixtureBytes(t, borrowed)
}

// TestNativeReleaseLeavesOtherFilesHeld changes only its selected inode.
func TestNativeReleaseLeavesOtherFilesHeld(t *testing.T) {
	examples := lockExamples(t)
	held := lockFixture(t)
	heldProbe := openLockHandle(t, held)
	file := lockFixture(t)
	other := openLockHandle(t, file)
	probe := openLockHandle(t, file)
	holdFixture(t, held)
	holdFixture(t, file)
	if err := examples.releaseWithin(t, localLock{file: file}); err != nil {
		t.Fatalf("independent release: %v", err)
	}
	requireKernelBusy(t, heldProbe)
	requireKernelAcquisition(t, other, probe)
	requireFixtureBytes(t, held)
	requireFixtureBytes(t, file)
}

// releaseOwnershipProperty retains borrowed, competing and probing handles.
type releaseOwnershipProperty struct {
	file, holder, probe *os.File
	examples            lockTestInputs
}

// check keeps the three raw unlocks attached to the trial in their old order.
func (property releaseOwnershipProperty) check(trial *rapid.T) {
	file := property.file
	holder := property.holder
	probe := property.probe
	competing := rapid.Bool().Draw(trial, "competing owner")
	repeats := rapid.IntRange(property.examples.MinimumAttempts,
		property.examples.MaximumAttempts).Draw(trial, "repeated releases")
	defer unlockFixture(trial, file)
	defer unlockFixture(trial, holder)
	defer unlockFixture(trial, probe)
	owner := file
	if competing {
		owner = holder
	}
	if err := rawLock(owner); err != nil {
		trial.Fatal(err)
	}
	for range repeats {
		if err := property.examples.releaseWithin(trial,
			localLock{file: file}); err != nil {
			trial.Fatalf("release: %v", err)
		}
		if competing {
			requireKernelBusy(trial, probe)
		} else {
			requireKernelAcquisition(trial, probe, holder)
		}
	}
}

// TestNativeReleaseGeneratedOwnership compares releases with kernel state.
func TestNativeReleaseGeneratedOwnership(t *testing.T) {
	examples := lockExamples(t)
	file := lockFixture(t)
	holder := openLockHandle(t, file)
	probe := openLockHandle(t, file)
	property := releaseOwnershipProperty{
		file: file, holder: holder, probe: probe,
		examples: examples,
	}
	rapid.Check(t, property.check)
	requireFixtureBytes(t, file)
}
