package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// aware is a texture that paints one light by itself and another when it
// is told the pixel's footprint, so a test can see which it was asked.
type aware struct{}

var awareAlone, awareWithin = maths.Grey(0.25), maths.Grey(0.75)

func (aware) At(maths.Vec) maths.Light { return awareAlone }

func (aware) Within(maths.Vec, float64) maths.Light { return awareWithin }

// Sample asks a footprinted texture by footprint only when the footprint is
// more than zero: zero is a lone ray, and a footprint below zero is none
// either.
func TestSampleUsesTheFootprintOnlyWhenThereIsOne(t *testing.T) {
	p := maths.V(1, 2, 3)
	for _, c := range []struct {
		footprint float64
		want      maths.Light
	}{
		{0.5, awareWithin},
		{1, awareWithin},
		{3, awareWithin},
		{0, awareAlone},
		{-0.5, awareAlone},
	} {
		if got := Sample(aware{}, p, c.footprint); got != c.want {
			t.Errorf("a footprint of %v gave %v, want %v", c.footprint, got, c.want)
		}
	}
}

// A texture that cannot paint by footprint is painted as it always is,
// whatever the footprint.
func TestSampleOfAPlainTextureIgnoresTheFootprint(t *testing.T) {
	c := Checker(1, a, b)
	p := maths.V(1, 0, 0)
	for _, footprint := range []float64{-1, 0, 0.5, 4} {
		if got := Sample(c, p, footprint); got != c.At(p) {
			t.Errorf("a footprint of %v moved a plain texture: %v", footprint, got)
		}
	}
}

// A checker says what it is: its size and its two colours, as it was
// made, in the order a then b.
func TestCheckerSaysWhatItIs(t *testing.T) {
	c, ok := Checker(2.5, a, b).(Checks)
	if !ok {
		t.Fatal("a checker does not say it is a checker")
	}
	size, first, second := c.Checks()
	if size != 2.5 || first != a || second != b {
		t.Errorf("a checker of 2.5 in %v then %v says %v, %v, %v", a, b, size, first, second)
	}
	if _, ok := Texture(Wire{}).(Checks); ok {
		t.Error("a wire says it is a checker")
	}
}
