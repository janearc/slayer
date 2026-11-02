//ninja:vow dojo rangeOf's refusal is reached only by ends libtheme never gives, so it is held from inside
package observer

import "testing"

// the ends the inside tests give a range: a short and a long, defined,
// and the two the other way round
const (
	short = 1 // defined: a short end
	long  = 2 // defined: a long end past it
)

// two ends the wrong way round are no range, and the zero range they
// give holds nothing, not even what lies between them.
func TestBackwardEndsHoldNothing(t *testing.T) {
	if backwards := rangeOf(long, short); backwards.Holds(short) ||
		backwards.Holds(long) {
		t.Errorf("a range from %v back to %v holds a wavelength", long,
			short)
	}
}

// anti-tests: two ends the right way round are a range, holding its
// short end.
func TestForwardEndsHoldTheirShortEnd(t *testing.T) {
	if !rangeOf(short, long).Holds(short) {
		t.Errorf("a range from %v to %v does not hold %v", short, long,
			short)
	}
}
