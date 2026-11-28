package workspace

import (
	"fmt"
	"math"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

const lightMarkerNotice = "Light markers are read-only and appear in frame · staged numbers identify lights within this view · lighting and shadows are omitted"

// localLightNotice describes the narrower light sources of the cached scene.
func localLightNotice(read set.Set, scene *placement.DisplayScene) string {
	if len(scene.Lights) == 0 && read.Light() != "sun" && read.Light() != "moon" {
		return ""
	}
	for _, light := range scene.Lights {
		switch light.Author {
		case thing.AuthorRef{Collection: "scene", Name: "sun"}:
			return lightMarkerNotice + " · Local markers cover the scene sun and supported world lights"
		case thing.AuthorRef{Collection: "scene", Name: "moon"}:
			return lightMarkerNotice + " · The Moon marker shows its direct-light direction · Full staging checks the lunar sky"
		}
	}
	switch read.Light() {
	case "moon":
		return lightMarkerNotice + " · Local markers cover supported world lights · Scene-level Moon marker is omitted in this working view"
	case "sun":
		return lightMarkerNotice + " · Local markers cover supported world lights · Scene-level sun is omitted"
	}
	return lightMarkerNotice + " · Local markers cover supported world lights"
}

type lightMarker struct {
	at    placement.LightProjection
	label string
}

func visibleLightMarkers(scene *placement.DisplayScene) []lightMarker {
	var visible []lightMarker
	for _, light := range scene.Lights {
		at, inFrame := scene.ProjectLight(light)
		if !inFrame {
			continue
		}
		name := light.Author.Name
		if light.Author.Collection == "scene" && (name == "sun" || name == "moon") {
			name = "scene"
		} else if name == "" {
			name = fmt.Sprintf("staged %d", light.Ordinal)
		}
		visible = append(visible, lightMarker{at: at, label: light.Kind + " · " + name})
	}
	return visible
}

// drawLightMarkers is an image overlay with no pointer or document handlers.
func (view *imageView) drawLightMarkers(canvas widget.Canvas, fitted geometry.Rect) {
	if len(view.lightMarkers) == 0 || fitted.Width() <= 2 || fitted.Height() <= 2 {
		return
	}
	// A distinct scissor group keeps the geometry above the working image.
	clip := fitted.Expand(-1)
	canvas.PushClip(clip)
	defer canvas.PopClip()
	var labels []geometry.Rect
	for _, marker := range view.lightMarkers {
		at := geometry.Pt(fitted.Min.X+float32(marker.at.X)*fitted.Width(),
			fitted.Min.Y+float32(marker.at.Y)*fitted.Height())
		drawLightSymbol(canvas, at, marker.at)
		label := lightLabelBounds(canvas, clip, at, marker.label, view.fontSize)
		if label.IsEmpty() || intersectsLightLabel(label, labels) {
			continue
		}
		labels = append(labels, label)
		canvas.DrawRoundRect(label, appearance.Panel, 4)
		canvas.DrawText(marker.label, label, view.fontSize, appearance.Ink, false, widget.TextAlignCenter)
	}
}

func drawLightSymbol(canvas widget.Canvas, at geometry.Point, projection placement.LightProjection) {
	canvas.DrawCircle(at, 6, appearance.Panel)
	canvas.StrokeCircle(at, 4, appearance.Cyan, 2)
	length := math.Hypot(projection.AimX, projection.AimY)
	if !(length > 1e-8) || math.IsInf(length, 0) {
		return
	}
	x, y := float32(projection.AimX/length), float32(projection.AimY/length)
	end := geometry.Pt(at.X+x*19, at.Y+y*19)
	canvas.DrawLine(at, end, appearance.Cyan, 2)
	canvas.DrawLine(end, geometry.Pt(end.X-x*6+y*4, end.Y-y*6-x*4), appearance.Cyan, 2)
	canvas.DrawLine(end, geometry.Pt(end.X-x*6-y*4, end.Y-y*6+x*4), appearance.Cyan, 2)
}

func lightLabelBounds(canvas widget.Canvas, clip geometry.Rect, at geometry.Point, label string, size float32) geometry.Rect {
	width, height := canvas.MeasureText(label, size, false)+12, size+8
	if width > clip.Width() || height+8 > clip.Height() {
		return geometry.Rect{}
	}
	x := min(max(at.X-width/2, clip.Min.X), clip.Max.X-width)
	y := at.Y + 8
	if y+height > clip.Max.Y {
		y = max(clip.Min.Y, at.Y-height-8)
	}
	return geometry.Rect{Min: geometry.Pt(x, y), Max: geometry.Pt(x+width, y+height)}
}

func intersectsLightLabel(next geometry.Rect, labels []geometry.Rect) bool {
	for _, label := range labels {
		if next.Intersects(label) {
			return true
		}
	}
	return false
}
