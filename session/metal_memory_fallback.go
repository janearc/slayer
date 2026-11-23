//go:build !darwin || !cgo

package session

// metalMemorySnapshot reports no Metal memory where Metal is not built in.
func metalMemorySnapshot() (uint64, uint64, bool) { return 0, 0, false }
