//go:build !darwin || !cgo

package hillside

// metalMemorySnapshot reports no Metal memory where Metal is not built in.
func metalMemorySnapshot() (uint64, uint64, bool) { return 0, 0, false }
