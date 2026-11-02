package typography

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"golang.org/x/image/font/gofont/goregular"
)

// a line's letters stand at their pens, scaled to the em, the line's advance
// centred on zero; a letter used twice is one mesh in two places; a space
// stands where it is set and has no mesh.
func TestALineStandsCentredAndSharesItsLetters(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	text := Text{Words: "rr i", Size: 2, Depth: 0.1}
	letters, err := Make(face, text)
	if err != nil {
		t.Fatal(err)
	}
	line, _ := Layout(face, text.Words)
	scale := text.Size / face.Units()
	for i, each := range letters {
		want := (line.Letters[i].Pen - line.Width/2) * scale
		if math.Abs(each.At.X-want) > 1e-12 || each.At.Y != 0 ||
			each.At.Z != 0 {
			t.Errorf("%q at %v, want x %v", each.Letter, each.At,
				want)
		}
	}
	if letters[0].Mesh != letters[1].Mesh || letters[0].Mesh == nil {
		t.Error("the two r's are not one mesh")
	}
	if letters[2].Mesh != nil || letters[3].Mesh == nil {
		t.Error("the space has a mesh, or the i has none")
	}
	first, last := letters[0].At.X, letters[3].At.X
	if !(first < 0 && last > 0) {
		t.Errorf("the line runs from %v to %v, not across nought",
			first, last)
	}
}

// the letters as things are one a letter with an outline, each of the
// material given, each standing at its place in the line carried by
// the line's motion: turned with it, and moved with it.
func TestTheLettersStandInTheLinesMotion(t *testing.T) {
	letters, err := Make(mustFont(t, goregular.TTF),
		Text{Words: "a b", Size: 1, Depth: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	clay := &thing.Material{Colour: maths.Grey(0.5)}
	turn := thing.Spin{In: maths.Subspace{Origin: maths.V(5, 0, 0)},
		Axis: maths.Up, Period: 4 * time.Second}
	things := Things(letters, clay, turn)
	if len(things) != 2 {
		t.Fatalf("%d things for two letters and a space", len(things))
	}
	for i, each := range things {
		at := letters[2*i].At
		if each.Material != clay || each.Shape != letters[2*i].Mesh {
			t.Errorf("thing %d is not its letter in clay", i)
		}
		// a quarter turn about up takes the line's x to its minus z
		got := each.Motion.At(time.Second).Origin
		want := maths.V(5, 0, -at.X)
		if got.Sub(want).Len() > 1e-9 {
			t.Errorf("letter %d a second in at %v, want %v", i, got,
				want)
		}
	}
}
