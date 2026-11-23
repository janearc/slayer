package workspace

import (
	"fmt"
	"image"
	"image/color"
)

// viewerPixelLimit is the most pixels the viewer holds for display, 64
// megapixels. A larger picture is shown reduced; its file is never changed.
const viewerPixelLimit = 64 * 1024 * 1024

// reduceForViewer returns the picture itself when it fits the limit. Above
// it, the picture is averaged down by the smallest whole factor that fits,
// and the note states the true size. The note is empty when nothing changed.
func reduceForViewer(source image.Image, limit int64) (image.Image, string) {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if int64(width)*int64(height) <= limit {
		return source, ""
	}
	factor := 2
	for int64(width/factor)*int64(height/factor) > limit {
		factor++
	}
	reduced := image.NewNRGBA(image.Rect(0, 0, max(width/factor, 1), max(height/factor, 1)))
	for y := 0; y < reduced.Rect.Dy(); y++ {
		for x := 0; x < reduced.Rect.Dx(); x++ {
			reduced.SetNRGBA(x, y, averageBlock(source, bounds.Min.X+x*factor, bounds.Min.Y+y*factor, factor))
		}
	}
	return reduced, fmt.Sprintf(" · shown reduced %d to 1 from %d × %d", factor, width, height)
}

// averageBlock is the mean colour of a square of pixels, averaged in the
// alpha-premultiplied form the image package reports.
func averageBlock(source image.Image, left, top, size int) color.NRGBA {
	var red, green, blue, alpha uint64
	for y := top; y < top+size; y++ {
		for x := left; x < left+size; x++ {
			r, g, b, a := source.At(x, y).RGBA()
			red, green, blue, alpha = red+uint64(r), green+uint64(g), blue+uint64(b), alpha+uint64(a)
		}
	}
	count := uint64(size * size)
	mean := color.RGBA64{R: uint16(red / count), G: uint16(green / count), B: uint16(blue / count), A: uint16(alpha / count)}
	return color.NRGBAModel.Convert(mean).(color.NRGBA)
}
