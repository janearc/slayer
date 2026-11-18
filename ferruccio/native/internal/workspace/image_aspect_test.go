package workspace

import (
	"image"
	"testing"

	"github.com/gogpu/ui/geometry"
)

// TestWorkingReadbackKeepsAuthoredImageBounds prevents integer GPU target sizes
// from becoming a new aspect ratio and shrinking the next rendering target.
func TestWorkingReadbackKeepsAuthoredImageBounds(t *testing.T) {
	cases := []struct {
		name          string
		width, height float32
	}{
		{name: "whole pixels", width: 1600, height: 747},
		{name: "fractional viewport", width: 1600.375, height: 747.625},
	}
	photo := image.NewRGBA(image.Rect(0, 0, 640, 400))
	readback := image.NewRGBA(image.Rect(0, 0, 1195, 747))
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			view := newImageView()
			viewport := geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(test.width, test.height)}
			view.SetBounds(viewport)
			view.set(photo)
			initial := view.imageBounds()
			if initial.IsEmpty() || initial.Min.X < viewport.Min.X || initial.Min.Y < viewport.Min.Y || initial.Max.X > viewport.Max.X || initial.Max.Y > viewport.Max.Y {
				t.Fatalf("initial photograph bounds %v escape viewport %v", initial, viewport)
			}
			for update := 0; update < 100; update++ {
				view.setWorking(readback, 1.6)
				if got := view.imageBounds(); got != initial {
					t.Fatalf("readback update %d changed image bounds from %v to %v", update+1, initial, got)
				}
			}
		})
	}
}
