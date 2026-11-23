package terminal

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestAFrameIsSynchronisedAndResetsEachRow(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 60), uint8(y * 40), 90, 255})
		}
	}
	var builder strings.Builder
	if err := Draw(&builder, Chop(img)); err != nil {
		t.Fatal(err)
	}
	s := builder.String()
	if !strings.HasPrefix(s, "\x1b[?2026h") || !strings.HasSuffix(s, "\x1b[?2026l") {
		t.Fatal("the frame is not wrapped in synchronised output")
	}
	if count := strings.Count(s, "\x1b[0m"); count != 2 {
		t.Fatalf("%d resets, want one per row, 2", count)
	}
}
