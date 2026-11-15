//go:build draw

package trace

import (
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// samePixels holds one picture to another, pixel for pixel, and says
// the first pixel that differs.
func samePixels(t *testing.T, got, want *frame.Frame, what string) {
	t.Helper()
	if len(got.Pix) != len(want.Pix) {
		t.Fatalf("%s the picture has %d pixels, not %d", what,
			len(got.Pix), len(want.Pix))
	}
	for i := range want.Pix {
		if got.Pix[i] == want.Pix[i] {
			continue
		}
		t.Fatalf("%s pixel %d is %v, not %v", what, i, got.Pix[i],
			want.Pix[i])
	}
}
