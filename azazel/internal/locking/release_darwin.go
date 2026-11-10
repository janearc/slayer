//go:build darwin

package locking

import (
	"os"
	"runtime"
	"syscall"
)

// release ends this handle's advisory ownership and keeps the file open.
// Its caller serializes lifetime and finishes the protected transaction first.
func (lock localLock) release() error {
	if lock.file == nil {
		return os.ErrInvalid
	}
	descriptor := int(lock.file.Fd())
	err := syscall.Flock(descriptor, syscall.LOCK_UN)
	runtime.KeepAlive(lock.file)
	return err
}
