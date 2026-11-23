package typography

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"golang.org/x/image/font/gofont/gobold"
)

// areaOf is the signed area of a flat ring of points by the shoelace:
// positive for one that turns counterclockwise.
func areaOf(ring []maths.Vec) float64 {
	sum := 0.0
	for index, at := range ring {
		next := ring[(index+1)%len(ring)]
		sum += at.X*next.Y - next.X*at.Y
	}
	return sum / 2
}

// the letter O has an outer ring turning counterclockwise and a hole inside
// it turning clockwise, a space has no rings, and the letters stand where
// Make stands them.
func TestOutlinesGiveEachLetterItsRingsAndPlace(t *testing.T) {
	face := mustFont(t, gobold.TTF)
	text := Text{Words: "O O", Size: 2}
	outlines, err := Outlines(face, text)
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(face, Text{Words: text.Words, Size: 2, Depth: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if len(outlines) != 3 || len(outlines[1].Rings) != 0 ||
		len(outlines[0].Rings) != 2 {
		t.Fatalf("O O has %d letters, rings %d, %d", len(outlines),
			len(outlines[0].Rings), len(outlines[1].Rings))
	}
	if outer, hole := areaOf(outlines[0].Rings[0]),
		areaOf(outlines[0].Rings[1]); !(outer > 0) || !(hole < 0) ||
		math.Abs(hole) >= outer {
		t.Errorf("O has an outer area %v and a hole of %v", outer, hole)
	}
	for index, one := range outlines {
		if one.At != made[index].At {
			t.Errorf("letter %d stands at %v, Make has it at %v", index,
				one.At, made[index].At)
		}
		for _, ring := range one.Rings {
			for _, at := range ring {
				if at.Z != 0 {
					t.Errorf("a point of letter %d is out of the plane: %v",
						index, at)
				}
			}
		}
	}
}

// an outline has no depth to ask for, and is refused what Make refuses
// that is not about depth.
func TestOutlinesNeedNoDepthAndRefuseWhatIsNoText(t *testing.T) {
	face := mustFont(t, gobold.TTF)
	if _, err := Outlines(face, Text{Words: "A", Size: 1}); err != nil {
		t.Errorf("an outline with no depth was refused: %v", err)
	}
	for _, text := range []Text{{Words: "", Size: 1}, {Words: "A", Size: 0},
		{Words: "A", Size: math.NaN()}, {Words: "拉", Size: 1}} {
		if _, err := Outlines(face, text); err == nil {
			t.Errorf("an outline of %+v was made", text)
		}
	}
}
