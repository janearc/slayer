//go:build darwin

// Package locking contains private native transaction-lock operations.
package locking

import (
	"errors"
	"os"
	"runtime"
	"syscall"
)

// acquisitionState distinguishes ownership, contention and an unusable attempt.
type acquisitionState uint8

const (
	acquisitionUnknown acquisitionState = iota
	acquisitionHeld
	acquisitionBusy
	acquisitionFailed
)

// acquisition reports kernel ownership or the original refusal reason.
// Its zero value means no ownership.
type acquisition struct {
	state acquisitionState
	err   error
}

// localLock borrows a selected open descriptor without choosing its lifetime.
type localLock struct{ file *os.File }

// tryAcquire attempts exclusive ownership without waiting or changing the file.
// The caller keeps the handle private and serializes its lifetime.
func (lock localLock) tryAcquire() acquisition {
	if lock.file == nil {
		return acquisition{state: acquisitionFailed, err: os.ErrInvalid}
	}
	descriptor := int(lock.file.Fd())
	err := syscall.Flock(descriptor,
		syscall.LOCK_EX|syscall.LOCK_NB)
	runtime.KeepAlive(lock.file)
	if err == nil {
		return acquisition{state: acquisitionHeld}
	}
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return acquisition{state: acquisitionBusy, err: err}
	}
	return acquisition{state: acquisitionFailed, err: err}
}
