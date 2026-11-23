//go:build darwin

package gpu

import (
	"context"

	"github.com/janearc/slayer/metal"
)

// stopDevice has every wait for the device end when stop is done, and
// gives back the function that lets that go. With no context there is
// nothing to stop on.
func stopDevice(stop context.Context) func() {
	return metal.StopOn(stop)
}
