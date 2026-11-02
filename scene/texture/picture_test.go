package texture

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A picture wraps the sphere: its left edge at the back, its middle
// column at the front, its top row at the north pole, red on the left
// half and blue on the right, blended at the seam between.
func TestPictureWrapsTheSphere(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if x >= 20 {
				c = color.RGBA{0, 0, 255, 255}
			}
			if y < 2 {
				c = color.RGBA{255, 255, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	pic := Picture{Image: img}
	if l := pic.At(maths.V(0, 1, 0)); l.R < 0.99 || l.G < 0.99 {
		t.Errorf("the north pole is %v, not white", l)
	}
	// the middle column of the picture, u of a half, is plus x. the
	// picture's left, its west, is red and lands toward minus z, which
	// by the package's own spin (TestAThinDiscWindsWithTime) is east of
	// plus x: the map is drawn mirrored, as it was first drawn, and held
	// so. this holds what is, not what is right.
	minusZ := pic.At(maths.V(math.Cos(-0.3), 0, math.Sin(-0.3)))
	plusZ := pic.At(maths.V(math.Cos(0.3), 0, math.Sin(0.3)))
	if !(minusZ.R > 0.9 && minusZ.B < 0.1) || !(plusZ.B > 0.9 && plusZ.R < 0.1) {
		t.Errorf("toward minus z is %v and toward plus z is %v", minusZ, plusZ)
	}
	if l := pic.At(maths.V(1, 0, 0)); !(l.R > 0.2 && l.B > 0.2) {
		t.Errorf("the seam is %v, not blended", l)
	}
	if (Picture{}).At(maths.XAxis) != (maths.Light{}) {
		t.Error("no picture gave light")
	}
	// the right half is blue: water; the left is red: land
	if !pic.Water(maths.V(math.Cos(0.3), 0, math.Sin(0.3))) || pic.Water(maths.V(math.Cos(-0.3), 0, math.Sin(-0.3))) {
		t.Error("the map's water is not where its blue is")
	}
}
