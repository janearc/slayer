package hud

import (
	"fmt"
	"image"
	"testing"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
)

func benchHud() Hud {
	return Hud{}
}

func BenchmarkPicture(b *testing.B) {
	img := grey(400, 200)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := benchHud().Picture(img, 10, 0, dusk, View{}); err != nil {
			b.Fatal(err)
		}
	}
}

// oldPicture is Picture as it was written, the font parsed at every call.
func oldPicture(made Hud, img *image.RGBA, rows int, at time.Duration,
	theme Theme, view View) error {
	if rows < 1 || img.Bounds().Dy()/rows < 2 {
		return fmt.Errorf("%w: %d rows", ErrHud, rows)
	}
	high := img.Bounds().Dy() / rows
	wide := high / 2
	cols := img.Bounds().Dx() / wide
	cover, err := made.Cover(image.Rect(0, 0, cols, rows), at, theme, view)
	if err != nil {
		return err
	}
	parsed, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size: float64(high) * 0.8, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	for _, place := range cover.Mask() {
		cell, _ := cover.At(place.X, place.Y)
		corner := image.Pt(place.X*wide, place.Y*high)
		min := img.Bounds().Min.Add(corner)
		paint(img, image.Rectangle{min, min.Add(image.Pt(wide, high))},
			cell, face)
	}
	return nil
}

// BenchmarkPictureOld is the old Picture, for the same minute.
func BenchmarkPictureOld(b *testing.B) {
	img := grey(400, 200)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := oldPicture(benchHud(), img, 10, 0, dusk, View{}); err != nil {
			b.Fatal(err)
		}
	}
}
