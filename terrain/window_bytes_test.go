package terrain

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// testWindow is a small window with every kind of number in it.
func testWindow() Window {
	return Window{west: 263716.5, north: 2148844, step: 2, across: 3,
		down: 2, heights: []float64{1100.25, -3, 0, math.Nextafter(1, 2),
			1e-300, 4096}}
}

func TestAWindowReadBackIsTheSameToTheBit(t *testing.T) {
	read, err := WindowFromBytes(testWindow().Bytes())
	if err != nil || !reflect.DeepEqual(read, testWindow()) {
		t.Fatalf("read %+v, %v", read, err)
	}
}

func TestAWindowOneBitApartIsOtherBytes(t *testing.T) {
	other := testWindow()
	other.heights = append([]float64(nil), other.heights...)
	other.heights[2] = math.Copysign(0, -1)
	if bytes.Equal(other.Bytes(), testWindow().Bytes()) {
		t.Fatal("nought and minus nought gave the same bytes")
	}
}

func TestWindowFromBytesRefusesWhatIsNotAWindow(t *testing.T) {
	raw := testWindow().Bytes()
	copied := append([]byte(nil), raw...)
	for name, cut := range map[string][]byte{
		"nothing":        nil,
		"a head cut":     raw[:windowHead*numberBytes-1],
		"a height short": raw[:len(raw)-numberBytes],
		"a byte over":    append(copied, 0),
		"no heights":     raw[:windowHead*numberBytes],
	} {
		if _, err := WindowFromBytes(cut); err == nil {
			t.Errorf("%s was read as a window", name)
		}
	}
}
