package walk

import (
	"fmt"
	"image"
	"image/draw"
	"math"
)

// Layout is a frame cut into panes: its size, and each pane's place on
// it, by name. The panes tile the frame, no gap and no overlap.
type Layout struct {
	size  image.Point
	panes map[string]image.Rectangle
}

// The two layouts of four panes. Picked for slayer: untraced,
// 2026-11-12.
const (
	// PaneViewer is the audience's camera; PaneExplain the explaining
	// panel; PaneAbove and PaneMoon the two other cameras.
	PaneViewer  = "viewer"
	PaneExplain = "explain"
	PaneAbove   = "above"
	PaneMoon    = "moon"
)

// Wide is 1920 by 1080, two by two: the viewer top left, the panel top
// right, above foot left, the moon foot right.
func Wide() Layout {
	width, height := wideSize.X/2, wideSize.Y/2
	return Layout{size: wideSize, panes: map[string]image.Rectangle{
		PaneViewer:  image.Rect(0, 0, width, height),
		PaneExplain: image.Rect(width, 0, wideSize.X, height),
		PaneAbove:   image.Rect(0, height, width, wideSize.Y),
		PaneMoon:    image.Rect(width, height, wideSize.X, wideSize.Y),
	}}
}

// Tall is 1080 by 1920: the viewer across the top, above and the moon
// side by side under it, square, and the panel at the foot.
func Tall() Layout {
	across := tallSize.X / 2
	under := tallViewer + across
	return Layout{size: tallSize, panes: map[string]image.Rectangle{
		PaneViewer:  image.Rect(0, 0, tallSize.X, tallViewer),
		PaneAbove:   image.Rect(0, tallViewer, across, under),
		PaneMoon:    image.Rect(across, tallViewer, tallSize.X, under),
		PaneExplain: image.Rect(0, under, tallSize.X, tallSize.Y),
	}}
}

// The layouts' sizes, and the tall one's viewer pane height: the 15/70
// film's whole aperture, 70.41 by 51.61, at 1080 across. Defined by the
// brief and the film.
var (
	wideSize   = image.Pt(1920, 1080)
	tallSize   = image.Pt(1080, 1920)
	tallViewer = 792
)

// Size is the layout's frame.
func (layout Layout) Size() image.Point { return layout.size }

// Pane is a pane's place on the frame, refused by name when the layout
// has none of that name.
func (layout Layout) Pane(name string) (image.Rectangle, error) {
	place, found := layout.panes[name]
	if !found {
		return image.Rectangle{}, fmt.Errorf("%w: the layout has no pane "+
			"%q", errWalk, name)
	}
	return place, nil
}

// Fit is how a picture of another shape goes into a pane: Crop fills it
// and cuts the picture's middle to the pane's shape; letterbox shows the
// whole picture as large as fits, the rest left as it was. Neither ever
// stretches it.
type Fit int

// The two fits.
const (
	Crop Fit = iota
	letterbox
)

// Laid lays a picture into a pane of a frame by a fit, scaled with its
// shape kept. The picture's top left is its pixel nought, nought.
func Laid(frame, picture *image.RGBA, pane image.Rectangle, fit Fit) {
	source := picture.Rect.Size()
	across := float64(pane.Dx()) / float64(source.X)
	down := float64(pane.Dy()) / float64(source.Y)
	scale := math.Max(across, down)
	if fit == letterbox {
		scale = math.Min(across, down)
	}
	width := int(math.Round(float64(source.X) * scale))
	height := int(math.Round(float64(source.Y) * scale))
	middle := pane.Min.Add(pane.Size().Div(2))
	target := image.Rect(middle.X-width/2, middle.Y-height/2,
		middle.X-width/2+width, middle.Y-height/2+height).Intersect(pane)
	for y := target.Min.Y; y < target.Max.Y; y++ {
		for x := target.Min.X; x < target.Max.X; x++ {
			sx := (float64(x-(middle.X-width/2))+half)/scale - half
			sy := (float64(y-(middle.Y-height/2))+half)/scale - half
			frame.SetRGBA(x, y, bilinear(picture, sx, sy))
		}
	}
}

// Ground is a frame of a layout filled with the hud's ground.
func (layout Layout) Ground() (*image.RGBA, error) {
	tones, err := colours()
	if err != nil {
		return nil, err
	}
	frame := image.NewRGBA(image.Rectangle{Max: layout.size})
	draw.Draw(frame, frame.Bounds(), image.NewUniform(tones["ground"]),
		image.Point{}, draw.Src)
	return frame, nil
}
