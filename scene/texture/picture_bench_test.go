package texture

import (
	"image"
	"image/color"
	"math/rand"
	"testing"

	"github.com/janearc/slayer/maths"
)

func benchPicture() Picture {
	img := image.NewRGBA(image.Rect(0, 0, 256, 128))
	r := rand.New(rand.NewSource(1))
	for y := 0; y < 128; y++ {
		for x := 0; x < 256; x++ {
			red := uint8(r.Intn(256))
			green := uint8(r.Intn(256))
			blue := uint8(r.Intn(256))
			img.Set(x, y, color.RGBA{red, green, blue, 255})
		}
	}
	return Picture{Image: img, Turn: 0.3}
}

func BenchmarkPictureAt(b *testing.B) {
	pic := benchPicture()
	b.ReportAllocs()
	b.ResetTimer()
	var sink maths.Light
	for i := 0; i < b.N; i++ {
		f := float64(i%200) / 100
		sink = pic.At(maths.V(f-1, f/3-0.3, 1-f/2))
	}
	_ = sink
}

// BenchmarkPictureAtOld is the old At, for the same minute.
func BenchmarkPictureAtOld(b *testing.B) {
	pic := benchPicture()
	b.ReportAllocs()
	b.ResetTimer()
	var sink maths.Light
	for i := 0; i < b.N; i++ {
		f := float64(i%200) / 100
		sink = oldAt(pic, maths.V(f-1, f/3-0.3, 1-f/2))
	}
	_ = sink
}
