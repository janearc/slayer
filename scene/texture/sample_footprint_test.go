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
