package colour

import (
	"image"
	"image/color"
	"testing"
	"time"
)

// ends runs call and fails the test if it has not come back in a few
// seconds. a loop that counts the wrong way never ends, and would hold
// up the whole run instead of failing in it.
func ends(t *testing.T, what string, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s did not finish in three seconds", what)
	}
}

// a small picture is masked and grained in no time, whichever way the
// bars fall. the file name puts this before the tests that use the
// same calls, see above_widest_test.go.
func TestMaskingAndGrainingEndInNoTime(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:], []byte{90, 90, 90, 255})
	}
	ends(t, "a mask with bars above and below", func() { Masked(img, 6, 1) })
	ends(t, "a mask with bars either side", func() { Masked(img, 1, 1) })
	ends(t, "a grain", func() { Grained(img, 2, 0.3, 1) })
	if got := Masked(img, 1.5, 1).RGBAAt(0, 0); got != (color.RGBA{90, 90, 90, 255}) {
		t.Errorf("a mask of the same shape cut %v", got)
	}
}
