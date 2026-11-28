package workspace

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/session/placement"
)

type volumeOutlineRecorder struct {
	widget.Canvas
	clip geometry.Rect
	line [2]geometry.Point
}

func (r *volumeOutlineRecorder) PushClip(clip geometry.Rect) { r.clip = clip }
func (*volumeOutlineRecorder) PopClip()                      {}
func (*volumeOutlineRecorder) MeasureText(string, float32, bool) float32 {
	return 200
}
func (r *volumeOutlineRecorder) DrawLine(from, to geometry.Point, _ widget.Color, _ float32) {
	r.line = [2]geometry.Point{from, to}
}

func TestVolumeOutlineDrawUsesDistinctImageClip(t *testing.T) {
	fitted := geometry.Rect{Min: geometry.Pt(10, 20), Max: geometry.Pt(110, 120)}
	view := imageView{volumeBounds: []volumeOutline{{
		edges: []placement.VolumeEdge{{{0.25, 0.25}, {0.75, 0.75}}},
		label: "Fizz bounds",
	}}}
	canvas := &volumeOutlineRecorder{}
	view.drawVolumeBounds(canvas, fitted)
	if want := fitted.Expand(-1); canvas.clip != want {
		t.Errorf("outline clip = %v, want %v distinct from the image clip", canvas.clip, want)
	}
	if want := [2]geometry.Point{geometry.Pt(35, 45), geometry.Pt(85, 95)}; canvas.line != want {
		t.Errorf("outline endpoints = %v, want %v in the fitted image", canvas.line, want)
	}
}
