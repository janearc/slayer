package colour

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// A band stands for a visible colour as bright as white at unit
// strength, different ends of a range for different hues, and a false
// colour picture adds its layers: a gamma layer at half strength over a
// dark visible layer is half that colour.
func TestFalseColour(t *testing.T) {
	short, long := maths.Band{Nanometres: 2e-5}, maths.Band{Nanometres: 8e-4}
	a, b := Stands(short), Stands(long)
	// the short end is the violet end, more blue than red, and the long
	// end the red, as Stands says: not merely two colours that differ
	if !(a.B > a.R && a.B > a.G) {
		t.Errorf("gamma's short end stands for %v, not violet", a)
	}
	if !(b.R > b.G && b.R > b.B) {
		t.Errorf("gamma's long end stands for %v, not red", b)
	}
	for _, l := range []maths.Light{a, b} {
		if y := l.Luminance(); y < 0.99 || y > 1.01 {
			t.Errorf("%v stands at luminance %v", l, y)
		}
	}
	vis := frame.New(2, 1)
	vis.Set(0, 0, frame.Pixel{R: 0.2, G: 0.2, B: 0.2, Cover: 1})
	g := frame.New(2, 1)
	g.Set(0, 0, frame.Pixel{R: 0.5, G: 0.5, B: 0.5, Cover: 1})
	img := False([]Layer{{Band: maths.Band{}, Frame: vis},
		{Band: long, Frame: g}})
	want := maths.Grey(0.2).Add(b.Scale(0.5))
	got := img.RGBAAt(0, 0)
	if got.R != Encode(want.R) || got.G != Encode(want.G) || got.B != Encode(want.B) {
		t.Errorf("the composite is %v, want %v", got, want)
	}
	if dark := img.RGBAAt(1, 0); dark.R != 0 || dark.G != 0 || dark.B != 0 {
		t.Errorf("an empty pixel is %v", dark)
	}
}

// A tinted layer stands in its tint, not libtheme's colour for its
// range.
func TestATintedLayerStandsInItsTint(t *testing.T) {
	f := frame.New(1, 1)
	f.Set(0, 0, frame.Pixel{R: 1, Cover: 1})
	img := False([]Layer{{Band: maths.Band{Nanometres: 656}, Frame: f,
		Tint: maths.RGB(0, 1, 0)}})
	c := img.RGBAAt(0, 0)
	if c.G < 250 || c.R > 5 || c.B > 5 {
		t.Errorf("h-alpha tinted green came out %v", c)
	}
}
