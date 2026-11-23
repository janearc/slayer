package moonlight

import (
	"bytes"
	"math"
	"sync"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// workedOut is a sky worked out once for the table's tests, since one
// takes seconds: the moon low in the east-south-east, from 1222 m, as
// the sky's own test has it.
var workedOut = sync.OnceValues(func() (*Sky, error) {
	moon := maths.V(math.Sin(1.8), math.Sin(0.1), -math.Cos(1.8)).Unit()
	return NewSky(measure.Metres(1222), moon, Full(), 1e5)
})

// tableSky is a copy of the sky worked out for the table's tests.
func tableSky(t *testing.T) *Sky {
	t.Helper()
	sky, err := workedOut()
	if err != nil {
		t.Fatal(err)
	}
	copied := *sky
	return &copied
}

func TestASkyOfOtherNumbersIsOtherBytes(t *testing.T) {
	sky := tableSky(t)
	other := *sky
	other.table[3][4].G = math.Nextafter(other.table[3][4].G, 1)
	if bytes.Equal(other.Bytes(), sky.Bytes()) {
		t.Fatal("a sky one bit apart gave the same bytes")
	}
}

func TestSkyFromBytesRefusesBytesOfAnotherLength(t *testing.T) {
	raw := tableSky(t).Bytes()
	for _, cut := range [][]byte{nil, raw[:len(raw)-1],
		append(raw[:len(raw):len(raw)], 0)} {
		if _, err := SkyFromBytes(cut); err == nil {
			t.Errorf("%d bytes were read as a sky", len(cut))
		}
	}
}

func TestASpectrumsBytesAreItsNumbersInOrder(t *testing.T) {
	one := Spectrum{500: 1, 400: 2}
	if !bytes.Equal(one.Bytes(), Spectrum{400: 2, 500: 1}.Bytes()) {
		t.Fatal("one spectrum said two ways gave two byte strings")
	}
	if len(one.Bytes()) != 2*2*float64Bytes {
		have := len(one.Bytes())
		t.Fatalf("%d bytes for two nanometres", have)
	}
}

func TestSpectraOneBitApartAreOtherBytes(t *testing.T) {
	one := Spectrum{500: 1}
	for _, other := range []Spectrum{{500: math.Nextafter(1, 2)},
		{501: 1}, {500: 1, 600: 0}} {
		if bytes.Equal(one.Bytes(), other.Bytes()) {
			t.Errorf("%v gave the bytes of %v", other, one)
		}
	}
}
