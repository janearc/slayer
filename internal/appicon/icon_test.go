package appicon

import (
	"bytes"
	"image/png"
	"testing"
)

// TestPNGDecodesAt1024 checks that both icons are 1024 pixel square pngs and
// that the light and dark icons are different pictures.
func TestPNGDecodesAt1024(t *testing.T) {
	var data [2][]byte
	for i, dark := range []bool{false, true} {
		raw, err := PNG(dark)
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(raw))
		if err != nil || config.Width != 1024 || config.Height != 1024 {
			t.Errorf("dark %t: %dx%d, %v", dark, config.Width, config.Height, err)
		}
		data[i] = raw
	}
	if bytes.Equal(data[0], data[1]) {
		t.Error("the light and dark icons are the same file")
	}
}

// TestPNGReturnsAnOwnedCopy checks that changing the bytes returned does not
// change what the next call returns.
func TestPNGReturnsAnOwnedCopy(t *testing.T) {
	first, err := PNG(false)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte(nil), first...)
	for i := range first {
		first[i] = 0
	}
	second, err := PNG(false)
	if err != nil || !bytes.Equal(second, want) {
		t.Errorf("the second copy was changed: %v", err)
	}
}
