package workspace

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

const volumeBoundsNotice = "Wisp and fizz outlines show read-only integration bounds clipped to the working frame · Volumetric density, scattering and glow are omitted"
const giantBoundaryNotice = "When visible, gas giant overlays mark the read-only outer density boundary and may cross nearer objects · Cloud bands, opacity and illumination are omitted"
const globeOutlineNotice = "When visible, distant globe outlines are read-only overlays and may cross nearer objects · Surface appearance and illumination are omitted"

func localWorkingNotice(read set.Set, scene *placement.DisplayScene) string {
	notices := scene.CachedMaterialNotices()
	if light := localLightNotice(read, scene); light != "" {
		notices = append(notices, light)
	}
	if len(scene.VolumeBounds) > 0 {
		notices = append(notices, volumeBoundsNotice)
	}
	if scene.GiantAtmospheres > 0 {
		notices = append(notices, omittedGiantNotice(scene.GiantAtmospheres))
	}
	if scene.OmittedVolumeBounds > 0 {
		notices = append(notices, fmt.Sprintf("Volume bounds omitted (%d)", scene.OmittedVolumeBounds))
	}
	if len(scene.GiantBoundaries) > 0 {
		notices = append(notices, giantBoundaryNotice)
	}
	if scene.SkyGradient != nil {
		notices = append(notices, workingSkyNotice)
	} else if scene.WorkingSkyOmitted {
		notices = append(notices, omittedWorkingSkyNotice)
	}
	if scene.FarClippedWorldSurfaces > 0 {
		notices = append(notices, farClipNotice(scene.FarClippedWorldSurfaces, "World surfaces"))
	}
	return strings.Join(notices, " · ")
}

type volumeOutline struct {
	edges []placement.VolumeEdge
	label string
}

func visibleVolumeBounds(scene *placement.DisplayScene) []volumeOutline {
	var outlines []volumeOutline
	for _, bound := range scene.VolumeBounds {
		if edges := scene.ProjectVolumeBound(bound); len(edges) > 0 {
			outlines = append(outlines, volumeOutline{edges, bound.Kind + " bounds"})
		}
	}
	for _, boundary := range scene.GiantBoundaries {
		if edges := scene.ProjectGiantBoundary(boundary); len(edges) > 0 {
			outlines = append(outlines, volumeOutline{edges, "Gas giant boundary overlay"})
		}
	}
	for _, outline := range scene.GlobeOutlines {
		if edges := scene.ProjectDistantSphere(outline); len(edges) > 0 {
			outlines = append(outlines, volumeOutline{edges, "Globe outline overlay"})
		}
	}
	return outlines
}

// drawVolumeBounds overlays edge lines without adding geometry or hit targets.
func (view *imageView) drawVolumeBounds(canvas widget.Canvas, fitted geometry.Rect) {
	if len(view.volumeBounds) == 0 || fitted.Width() <= 2 || fitted.Height() <= 2 {
		return
	}
	// A distinct scissor group keeps the outlines above the working image.
	clip := fitted.Expand(-1)
	canvas.PushClip(clip)
	defer canvas.PopClip()
	var labels []geometry.Rect
	for _, outline := range view.volumeBounds {
		top := drawVolumeOutline(canvas, fitted, outline)
		label := lightLabelBounds(canvas, clip, top, outline.label, view.fontSize)
		if label.IsEmpty() || intersectsLightLabel(label, labels) {
			continue
		}
		labels = append(labels, label)
		canvas.DrawRoundRect(label, appearance.Panel, 4)
		canvas.DrawText(outline.label, label, view.fontSize, appearance.Ink, false, widget.TextAlignCenter)
	}
}

func drawVolumeOutline(canvas widget.Canvas, fitted geometry.Rect, outline volumeOutline) geometry.Point {
	top := geometry.Pt(fitted.Min.X, fitted.Max.Y)
	for _, edge := range outline.edges {
		var points [2]geometry.Point
		for i, point := range edge {
			points[i] = geometry.Pt(fitted.Min.X+float32(point[0])*fitted.Width(),
				fitted.Min.Y+float32(point[1])*fitted.Height())
			if points[i].Y <= top.Y {
				top = points[i]
			}
		}
		canvas.DrawLine(points[0], points[1], appearance.Ink, 1)
	}
	return top
}
