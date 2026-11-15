//go:build !darwin

package weather

// RunOnGPU is refused off a mac: the weather's kernels are metal's.
func (made *Weather) RunOnGPU(until float64) (int, error) {
	return 0, ErrNoGPU
}

// HasGPU is false off a mac: the weather's kernels are metal's.
func HasGPU() bool { return false }
