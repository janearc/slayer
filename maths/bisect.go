package maths

// Bisect narrows a bracket, moving its low bound when below is true.
// It makes exactly steps midpoint evaluations and leaves endpoint checks to
// the caller. The returned bounds preserve the caller's final rounding.
func Bisect(low, high float64, steps int, below func(float64) bool) (float64, float64) {
	for range steps {
		middle := (low + high) / 2
		if below(middle) {
			low = middle
		} else {
			high = middle
		}
	}
	return low, high
}
