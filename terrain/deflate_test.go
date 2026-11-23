package terrain

import (
	"bytes"
	"compress/zlib"
	"strings"
	"testing"
)

// zlibOf is data as a zlib stream, which is what TIFF's deflate holds.
func zlibOf(t *testing.T, data []byte) []byte {
	t.Helper()
	var packed bytes.Buffer
	writer := zlib.NewWriter(&packed)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return packed.Bytes()
}

func TestDeflatedTilesAreReadUnderBothNumbers(t *testing.T) {
	values := make([]float32, 16)
	for index := range values {
		values[index] = float32(index) + 0.5
	}
	for _, number := range []uint16{deflated, oldDeflated} {
		spec := newTiffSpec(tiffLittle, false, 4, 4, 4, 4, rampAt)
		spec.shorts(tagCompression, number)
		spec.tiles = [][]byte{zlibOf(t, floatBytes(values...))}
		got, _, err := openSpec(t, spec).sample(3, 3)
		if err != nil || got != 15.5 {
			t.Errorf("compression %d: %v %v, want 15.5", number, got, err)
		}
	}
}

func TestADeflatedTileThatEndsWrongIsRefused(t *testing.T) {
	values := make([]float32, 16)
	overlong := append(floatBytes(values...), 1)
	cases := map[string]struct {
		payload []byte
		want    string
	}{
		"extra output": {zlibOf(t, overlong),
			"exceeds the declared tile grid"},
		"short output": {zlibOf(t, floatBytes(values[:3]...)), "EOF"},
		"not a stream": {[]byte("not deflate at all"), "zlib"},
	}
	for name, each := range cases {
		spec := newTiffSpec(tiffLittle, false, 4, 4, 4, 4, rampAt)
		spec.shorts(tagCompression, deflated)
		spec.tiles = [][]byte{each.payload}
		_, _, err := openSpec(t, spec).sample(3, 3)
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", name, err, each.want)
		}
	}
}
