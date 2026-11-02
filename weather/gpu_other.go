//go:build !darwin

package weather

import "errors"

// RunOnGPU is refused off a mac: the weather's kernels are metal's.
func (made *Weather) RunOnGPU(until float64) (int, error) {
	return 0, errors.New("weather: the gpu runs on a mac only; run on the cpu")
}
