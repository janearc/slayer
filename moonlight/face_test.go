package moonlight

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/maths"
	"golang.org/x/image/tiff"
)

// mosaic writes a small mosaic: a dark western half and a bright
// eastern one, as a tiff, and says where.
func mosaic(t *testing.T) string {
	t.Helper()
	picture := image.NewRGBA64(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			level := uint16(16000)
			if x >= 32 {
				level = 48000
			}
			picture.SetRGBA64(x, y, color.RGBA64{level, level, level, 65535})
		}
	}
	path := filepath.Join(t.TempDir(), "moon.tif")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := tiff.Encode(file, picture, nil); err != nil {
		t.Fatal(err)
	}
	return path
}

// the face's mean over the whole moon is the glow; its east, which the eye sees
// on its right, is the mosaic's eastern half, brighter; and a point straight
// toward the eye is longitude zero, between them.
func TestTheFaceIsTheMosaicsPatternAtTheMoonsGlow(t *testing.T) {
	way := maths.V(0, 0, -1)
	glow := maths.RGB(2, 1, 0.5)
	face, err := NewFace(mosaic(t), way, glow)
	if err != nil {
		t.Fatal(err)
	}
	sum, count := 0.0, 0.0
	for _, b := range face.bright {
		sum += float64(b)
		count++
	}
	if mean := sum / count; math.Abs(mean-1) > 1e-6 {
		t.Errorf("the mosaic's mean is %v, not one", mean)
	}
	right := face.At(maths.V(1, 0, 0.2))
	left := face.At(maths.V(-1, 0, 0.2))
	if !(right.R > left.R) || right.R/right.G != 2 {
		t.Errorf("the eye's right %v, its left %v", right, left)
	}
}
