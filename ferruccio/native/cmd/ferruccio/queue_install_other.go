//go:build !darwin

package main

import "errors"

// installPrivateQueue is available in the packaged macOS app.
func installPrivateQueue(_, _ string) error { return errors.ErrUnsupported }
