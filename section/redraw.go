package section

import (
	"image"

	"github.com/janearc/slayer/render/frame"
)

// Redrawn is the frame after a change, made from the frame before it:
// each section the change keeps is copied from the earlier frame, and the
// rest are drawn, all in one call of draw, which gives a frame for each
// box it is handed, in order. It also says how many sections were kept.
//
// boxes and sights are a frame's sections and the sight of each. The
// earlier frame must be the picture of the change's first world at its
// first moment, from the same eye through the same lens, or the result is
// not the later frame.
func (change Change) Redrawn(earlier *frame.Frame, boxes []image.Rectangle,
	sights []Sight, draw func([]image.Rectangle) ([]*frame.Frame,
		error)) (*frame.Frame, int, error) {
	var again []image.Rectangle
	for at, box := range boxes {
		if !change.keeps(sights[at]) {
			again = append(again, box)
		}
	}
	drawn, err := draw(again)
	if err != nil {
		return nil, nothing, err
	}
	later := frame.New(earlier.W, earlier.H)
	whole := image.Rect(nothing, nothing, earlier.W, earlier.H)
	copyOver(later, earlier, whole, image.Point{})
	for at, box := range again {
		copyOver(later, drawn[at], box, box.Min)
	}
	return later, len(boxes) - len(again), nil
}

// copyOver sets the pixels of a box of one frame from another frame,
// whose own pixel at nothing is the box's pixel at shift.
func copyOver(into, from *frame.Frame, box image.Rectangle,
	shift image.Point) {
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			into.Set(x, y, from.At(x-shift.X, y-shift.Y))
		}
	}
}
