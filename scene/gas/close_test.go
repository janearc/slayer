package gas

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// closeTo is whether got is within tol of want, relative to want, or within tol
// of zero when want is zero. a got that is not a number is never close to
// anything.
func closeTo(got, want, tol float64) bool {
	if got == want {
		return true
	}
	if want == 0 {
		return math.Abs(got) <= tol
	}
	return math.Abs(got-want) <= tol*math.Abs(want)
}

// wantClose fails the test when got is not within a trillionth of want,
// which is rounding and no more.
func wantClose(t *testing.T, what string, got, want float64) {
	t.Helper()
	if !closeTo(got, want, 1e-12) {
		t.Errorf("%s is %v, want %v", what, got, want)
	}
}

// wantLight is wantClose for the three parts of a light.
func wantLight(t *testing.T, what string, got, want maths.Light) {
	t.Helper()
	wantClose(t, what+" red", got.R, want.R)
	wantClose(t, what+" green", got.G, want.G)
	wantClose(t, what+" blue", got.B, want.B)
}
