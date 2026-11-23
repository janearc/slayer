package session

import (
	"testing"
)

// a star's light falls as its distance squared, in every band.
func TestStarlightFallsAsTheSquare(t *testing.T) {
	near, far := starlight(1), starlight(2)
	first, last := near.Span()
	for band := first; band <= last; band++ {
		if want := near.Power(band) / 4; far.Power(band) != want &&
			!(far.Power(band)-want < want*1e-12) {
			t.Errorf("band %d: %g, want %g", band, far.Power(band), want)
		}
	}
	if !(near.Power((first+last)/2) > 0) {
		t.Error("the star gives no light")
	}
}
