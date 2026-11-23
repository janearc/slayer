//go:build !darwin

package gpu

import "context"

// stopDevice has nothing to stop where there is no Metal device.
func stopDevice(context.Context) func() {
	return func() {}
}
