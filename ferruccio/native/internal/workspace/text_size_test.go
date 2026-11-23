package workspace

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// textRecorder keeps the size of every line of text drawn. Any other canvas
// call is a failure of the test, because it embeds no canvas.
type textRecorder struct {
	widget.Canvas
	sizes []float32
}

// DrawRoundRect ignores the background.
func (recorder *textRecorder) DrawRoundRect(geometry.Rect, widget.Color, float32) {}

// PushClipRoundRect ignores the clip.
func (recorder *textRecorder) PushClipRoundRect(geometry.Rect, float32) {}

// PopClip ignores the clip.
func (recorder *textRecorder) PopClip() {}

// DrawText records the size it was asked for.
func (recorder *textRecorder) DrawText(_ string, _ geometry.Rect, size float32, _ widget.Color, _ bool, _ widget.TextAlign) {
	recorder.sizes = append(recorder.sizes, size)
}

// TestEmptyViewerTextFollowsTheChosenSize checks that the placeholder text is
// drawn at the size the reader chose, at each size on offer.
func TestEmptyViewerTextFollowsTheChosenSize(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 24} {
		view := newImageView()
		view.SetFontSize(size)
		recorder := &textRecorder{}
		view.Draw(nil, recorder)
		if len(recorder.sizes) != 1 || recorder.sizes[0] != size {
			t.Fatalf("size %v: placeholder drawn at %v", size, recorder.sizes)
		}
	}
}

// TestHeadingRatiosKeepHeadingsAboveBodyText is the anti-test: a heading is
// larger than body text at every size, so following the size does not flatten
// the hierarchy.
func TestHeadingRatiosKeepHeadingsAboveBodyText(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 24} {
		if size*headingSizeRatio <= size || size*brandSizeRatio <= size || size*ballLabelLineRatio <= size {
			t.Fatalf("a ratio does not exceed body text at size %v", size)
		}
	}
}
