package lights

import "fmt"

type Image struct {
	Width, Height int
	Light         []float64
}

// At is the index in Light of the first of the three values (red, green,
// blue) of pixel x, y. Rows run from the top down.
func (p Image) At(x, y int) int { return (y*p.Width + x) * 3 }

// dimensions reports whether width and height are positive and small enough
// that width times height times 24 fits in an int.
func dimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= int(^uint(0)>>1)/24/height
}

// validate refuses an image whose size is not valid or whose Light does not
// hold exactly three values for each pixel.
func (p Image) validate() error {
	if !dimensions(p.Width, p.Height) || len(p.Light) != p.Width*p.Height*3 {
		return fmt.Errorf("invalid lights image dimensions or channel count")
	}
	return nil
}
