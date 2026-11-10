// Package learn renders terminal cells as raster images for model inspection.
// Each cell becomes six subpixels in two colours. The result approximates a
// terminal: actual glyphs depend on the terminal's font and sextant support.
// Contact sheets are provided by interfaces/sheet.
package learn

import (
	"image"

	"github.com/janearc/slayer/interfaces/terminal"
)

// Chopped is what the terminal will show: each cell as its six squares, in
// the two colours it actually got, its glyph unread. It is
// terminal.Picture, kept here for the callers that learned it by this name.
//
// a picture Chop made is whole cells, so a 5 by 4 comes back 6 by 6, its
// edge repeated; cells of nil are an empty picture.
func Chopped(c *terminal.Cells) *image.RGBA {
	if c == nil {
		return image.NewRGBA(image.Rectangle{})
	}
	return terminal.Picture(c)
}
