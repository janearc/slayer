package walk

import (
	"errors"
	"image"
	"image/color"
	"testing"
)

// TestLayoutsTile: each layout's panes cover its frame with no gap and
// no overlap; a pane it lacks is refused.
func TestLayoutsTile(t *testing.T) {
	for name, layout := range map[string]Layout{"wide": Wide(),
		"tall": Tall()} {
		covered := 0
		names := []string{PaneViewer, PaneExplain, PaneAbove, PaneMoon}
		if len(layout.panes) != len(names) {
			t.Fatalf("%s has panes %v", name, layout.panes)
		}
		for i, one := range names {
			place, err := layout.Pane(one)
			if err != nil {
				t.Fatal(err)
			}
			covered += place.Dx() * place.Dy()
			for _, other := range names[i+1:] {
				second, _ := layout.Pane(other)
				if place.Overlaps(second) {
					t.Errorf("%s: %s and %s overlap", name, one, other)
				}
			}
			if !place.In(image.Rectangle{Max: layout.Size()}) {
				t.Errorf("%s: %s is off the frame", name, one)
			}
		}
		if size := layout.Size(); covered != size.X*size.Y {
			t.Errorf("%s: the panes cover %d of %d pixels", name, covered,
				size.X*size.Y)
		}
		if _, err := layout.Pane("side"); !errors.Is(err, errWalk) {
			t.Errorf("%s: a pane it lacks: %v", name, err)
		}
	}
}

// TestLaidFits: a picture twice as wide as tall, letterboxed into a
// square, fills its middle half and leaves the rest; cropped, it fills
// the square and shows its middle. Neither stretches it.
func TestLaidFits(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for x := range 200 {
		for y := range 100 {
			shade := uint8(0)
			if x >= 100 {
				shade = 255
			}
			picture.SetRGBA(x, y, color.RGBA{shade, shade, shade, 255})
		}
	}
	ground := color.RGBA{1, 2, 3, 255}
	pane := image.Rect(0, 0, 100, 100)
	boxed := image.NewRGBA(pane)
	fill(boxed, pane, ground)
	Laid(boxed, picture, pane, letterbox)
	if boxed.RGBAAt(50, 10) != ground || boxed.RGBAAt(50, 90) != ground ||
		boxed.RGBAAt(10, 50).R != 0 || boxed.RGBAAt(90, 50).R != 255 {
		t.Error("the letterbox is not the picture's whole, banded")
	}
	cropped := image.NewRGBA(pane)
	Laid(cropped, picture, pane, Crop)
	if cropped.RGBAAt(50, 5) == ground || cropped.RGBAAt(10, 50).R != 0 ||
		cropped.RGBAAt(90, 50).R != 255 {
		t.Error("the crop does not fill with the picture's middle")
	}
	frame, err := Wide().Ground()
	if err != nil || frame.Bounds().Size() != Wide().Size() {
		t.Errorf("the ground is %v: %v", frame.Bounds(), err)
	}
	if err := Caption(frame, "a pane", image.Pt(10, 10), 30); err != nil {
		t.Error(err)
	}
}
