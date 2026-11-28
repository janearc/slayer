package diffusion

import (
	"testing"

	"github.com/janearc/slayer/render/frame"
)

func TestLightSpreadsAcrossSkinOnly(t *testing.T) {
	a, d, b := frame.New(19, 1), frame.New(19, 1), frame.New(19, 1)
	for i := range a.Pix {
		a.Pix[i] = frame.Pixel{R: 1, G: 1, B: 1, Depth: 1, Cover: 1}
		d.Pix[i].Depth = 1
	}
	d.Pix[9] = frame.Pixel{R: 1, G: 1, B: 1, Depth: 1, Cover: 1}
	a.Pix[10] = frame.Pixel{}
	d.Pix[8].Depth = 2
	Apply(b, d, a, .019, Profile{Radius: [3]float64{.004, .002, .0007}, Share: .65})
	if !(b.Pix[12].R > b.Pix[12].G && b.Pix[12].G > b.Pix[12].B) {
		t.Fatal(b.Pix[12])
	}
	if b.Pix[10].R != 0 || b.Pix[8].R != 0 {
		t.Fatal(b.Pix[10], b.Pix[8])
	}
}

func TestDiffusionRejectsBadInputs(t *testing.T) {
	f := frame.New(1, 1)
	p := Profile{Radius: [3]float64{.01, .01, .01}, Share: 1}
	bad := p
	bad.Radius[0] = 0
	if Apply(f, f, f, 1, bad) == nil || Apply(f, f, nil, 1, p) == nil || Apply(f, frame.New(1, 2), f, 1, p) == nil || Apply(f, f, f, 0, p) == nil {
		t.Fatal("accepted invalid input")
	}
	f.Pix[0].Cover = 1
	f.Pix[0].Depth = 0
	if Apply(f, f, f, 1, p) == nil {
		t.Fatal("accepted marked pixel at invalid depth")
	}
}
