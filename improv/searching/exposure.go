package searching

import (
	"image"
	"image/color"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Exposure owns a completed trace and its display settings. It holds no
// mutable scene or GPU state, so developing it may overlap the next trace.
// Transfer it to one consumer; concurrent Develop calls are not supported.
type Exposure struct {
	linear     *frame.Frame
	tone       colour.Tone
	picture    *image.RGBA
	graded     bool
	saturation float64
	labels     []string
}

// Develop finishes this exposure once, retaining only the resulting image.
// A subsequent call returns that same image without grading it again.
func (exposure *Exposure) Develop() *image.RGBA {
	if exposure.picture != nil {
		return exposure.picture
	}
	picture := exposure.tone.Develop(exposure.linear)
	exposure.linear = nil
	if exposure.graded {
		grade(picture, exposure.saturation)
	}
	for index, text := range exposure.labels {
		label := font.Drawer{Dst: picture,
			Src:  image.NewUniform(color.RGBA{180, 205, 220, 255}),
			Face: basicfont.Face7x13,
			Dot:  fixed.P(20, picture.Rect.Dy()-18*(len(exposure.labels)-index))}
		label.DrawString(text)
	}
	exposure.picture = picture
	return picture
}
