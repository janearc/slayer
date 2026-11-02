package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// schlick's fresnel is f0 + (1-f0)(1-cos)^5, with f0 = ((n-1)/(n+1))^2,
// scaled by Gloss. worked by hand: at n = 1.5, f0 = (0.5/2.5)^2 = 0.04.
//
//	cos 1:   0.04
//	cos 0.5: 0.04 + 0.96 * (0.5)^5 = 0.04 + 0.03 = 0.07
//	cos 0.2: 0.04 + 0.96 * (0.8)^5 = 0.04 + 0.96 * 0.32768 = 0.3545728
//	cos 0:   0.04 + 0.96 = 1
func TestMirrorsIsSchlicksFresnel(t *testing.T) {
	glass := &Material{Gloss: 1, Index: 1.5}
	for _, c := range []struct{ cos, want float64 }{
		{1, 0.04}, {0.5, 0.07}, {0.2, 0.3545728}, {0, 1},
	} {
		if got := glass.Mirrors(c.cos); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("glass at cos %v mirrors %v, want %v", c.cos, got, c.want)
		}
	}
	// a gloss of a half is half of it
	half := &Material{Gloss: 0.5, Index: 1.5}
	if got := half.Mirrors(0.5); math.Abs(got-0.035) > 1e-12 {
		t.Errorf("a gloss of a half at cos 0.5 mirrors %v, want 0.035", got)
	}
}

// f0 is ((n-1)/(n+1))^2, worked by hand face on at a gloss of one:
// n = 2 is (1/3)^2 = 1/9, n = 3 is (2/4)^2 = 0.25, and n = 1, no
// change from the air, mirrors nothing face on.
func TestMirrorsFaceOnIsTheIndexsFresnel(t *testing.T) {
	for _, c := range []struct{ n, want float64 }{
		{2, 1.0 / 9}, {3, 0.25}, {1, 0},
	} {
		m := &Material{Gloss: 1, Index: c.n}
		if got := m.Mirrors(1); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("an index of %v face on mirrors %v, want %v", c.n, got, c.want)
		}
	}
	// n = 2 at cos 0.5 is 1/9 + (8/9)/32 = 1/9 + 1/36 = 5/36
	two := &Material{Gloss: 1, Index: 2}
	if got := two.Mirrors(0.5); math.Abs(got-5.0/36) > 1e-12 {
		t.Errorf("an index of 2 at cos 0.5 mirrors %v, want 5/36", got)
	}
}

// no index, or one that is not greater than zero, is glass's 1.5: 0.04 face on,
// where an index of 4 would be 0.36 and of zero would be 1.
func TestNoIndexIsGlass(t *testing.T) {
	for _, n := range []float64{0, -1, -4} {
		m := &Material{Gloss: 1, Index: n}
		if got := m.Mirrors(1); math.Abs(got-0.04) > 1e-12 {
			t.Errorf("an index of %v face on mirrors %v, want 0.04", n, got)
		}
	}
}

// a cos past one or less than zero is the nearest of face on and grazing: 0.04
// and 1 at glass.
func TestMirrorsClampsTheCosine(t *testing.T) {
	glass := &Material{Gloss: 1, Index: 1.5}
	if got := glass.Mirrors(1.5); math.Abs(got-0.04) > 1e-12 {
		t.Errorf("a cos of 1.5 mirrors %v, want 0.04", got)
	}
	if got := glass.Mirrors(-0.5); math.Abs(got-1) > 1e-12 {
		t.Errorf("a cos of -0.5 mirrors %v, want 1", got)
	}
}

// a surface with no gloss, or a gloss less than zero, mirrors nothing, at any
// look: zero is zero even for a look of no number, which the formula alone
// would carry through as no number.
func TestADullSurfaceMirrorsNothing(t *testing.T) {
	for _, g := range []float64{0, -0.5, -3} {
		m := &Material{Gloss: g, Index: 1.5}
		for _, cos := range []float64{1, 0.5, 0, math.NaN()} {
			if got := m.Mirrors(cos); got != 0 {
				t.Errorf("a gloss of %v at cos %v mirrors %v", g, cos, got)
			}
		}
	}
}

// roughness is one without a texture, and with one is the texture's
// luminance at the point asked, in the material's own space. a grey of
// v has luminance v: the weights 0.2126 + 0.7152 + 0.0722 come to one.
func TestRoughnessIsTheTexturesLuminance(t *testing.T) {
	if got := (&Material{}).Roughness(maths.V(3, 4, 5)); got != 1 {
		t.Errorf("a surface rough everywhere alike is %v, want 1", got)
	}
	m := &Material{Rough: slopeTexture{}}
	for _, c := range []struct {
		at   maths.Vec
		want float64
	}{
		{maths.V(0.25, 9, 9), 0.25}, {maths.V(0, 1, 1), 0}, {maths.V(0.5, 0, 0), 0.5},
	} {
		if got := m.Roughness(c.at); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("roughness at %v is %v, want %v", c.at, got, c.want)
		}
	}
	// a colour that is not grey: red alone is 0.2126
	red := &Material{Rough: flatTexture{maths.RGB(1, 0, 0)}}
	if got := red.Roughness(maths.Vec{}); math.Abs(got-0.2126) > 1e-12 {
		t.Errorf("roughness of red is %v, want 0.2126", got)
	}
}

// slopeTexture is a grey as light as the point is far along x.
type slopeTexture struct{}

func (slopeTexture) At(p maths.Vec) maths.Light { return maths.Grey(p.X) }

// flatTexture is the same light everywhere.
type flatTexture struct{ light maths.Light }

func (f flatTexture) At(maths.Vec) maths.Light { return f.light }
