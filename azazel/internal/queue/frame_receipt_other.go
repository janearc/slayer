//go:build !darwin && !linux

package queue

import (
	"errors"
	"os"
)

type frameReceipt struct{ child *os.File }

// newFrameReceipt reports that frame receipts are not available here.
func newFrameReceipt(string, Job) (*frameReceipt, error) {
	return nil, errors.New("frame observations unavailable")
}

// environment returns env unchanged, since no receipt descriptor exists.
func (*frameReceipt) environment(env []string) []string { return env }

// start does nothing on this platform.
func (*frameReceipt) start() {}

// close does nothing on this platform.
func (*frameReceipt) close() {}
