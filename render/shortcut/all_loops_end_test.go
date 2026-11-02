package shortcut

import (
	"image"
	"testing"
	"time"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/medium"
)

// ends runs call and fails the test if it has not come back in a few
// seconds. a loop that counts the wrong way never ends, and would hold
// up the whole run instead of failing in it.
//
// the tests that call Bake, Threshold or Apart go through the helpers
// below, so none of them can hang on a loop a mutant has broken.
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

// bakeIn is Bake, which must finish.
func bakeIn(t *testing.T, m medium.Medium, resolution int) *Baked {
	t.Helper()
	var b *Baked
	ends(t, "a bake", func() { b = Bake(m, resolution) })
	return b
}

// thresholdIn is Threshold, which must finish.
func thresholdIn(t *testing.T, tone colour.Tone, top float64) float64 {
	t.Helper()
	var got float64
	ends(t, "a threshold", func() { got = Threshold(tone, top) })
	return got
}

// apartIn is Apart, which must finish.
func apartIn(t *testing.T, a, b *image.RGBA) (worst, mean float64) {
	t.Helper()
	ends(t, "a comparison of two pictures", func() { worst, mean = Apart(a, b) })
	return worst, mean
}

// a small bake, a threshold and a comparison all come back in no time.
func TestBakingAndComparingEndInNoTime(t *testing.T) {
	if b := bakeIn(t, ramp{}, 3); b.Resolution != 3 {
		t.Errorf("a bake of 3 has resolution %d", b.Resolution)
	}
	if got := thresholdIn(t, colour.Tone{}, 1); !(got > 0 && got < 1) {
		t.Errorf("the clip threshold is %v", got)
	}
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	if worst, mean := apartIn(t, img, img); worst != 0 || mean != 0 {
		t.Errorf("a picture apart from itself is %v, %v", worst, mean)
	}
}
