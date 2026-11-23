//go:build darwin && cgo

package session

import "github.com/janearc/slayer/metal"

// metalMemorySnapshot reads the Metal memory counters.
func metalMemorySnapshot() (uint64, uint64, bool) { return metal.MemorySnapshot() }
