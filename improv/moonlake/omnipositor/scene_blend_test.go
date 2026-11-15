package omnipositor

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// TestCompositeTransportLinear checks coloured extinction and source light
// without tone mapping, including unbounded HDR plate values.
func TestCompositeTransportLinear(t *testing.T) {
	p, s, tr := frame.New(1, 1), frame.New(1, 1), frame.New(1, 1)
	p.Pix[0] = frame.Pixel{R: 10, G: 4, B: 2, Depth: 7, Cover: 1}
	s.Pix[0] = frame.Pixel{R: 0.1, G: 0.2, B: 0.3, Depth: 7}
	tr.Pix[0] = frame.Pixel{R: 0.5, G: 0.25, B: 0, Depth: 7}
	got, err := CompositeTransport(p, s, tr)
	if err != nil {
		t.Fatal(err)
	}
	want := frame.Pixel{R: 5.1, G: 1.2, B: 0.3, Depth: 7, Cover: 1}
	if got.Pix[0] != want {
		t.Fatalf("%+v, want %+v", got.Pix[0], want)
	}
	tr.Pix[0].G = 1.01
	if _, err := CompositeTransport(p, s, tr); err == nil {
		t.Fatal("accepted amplifying transmission")
	}
	tr.Pix[0].G = float32(math.NaN())
	if _, err := CompositeTransport(p, s, tr); err == nil {
		t.Fatal("accepted NaN")
	}
}

// TestCompositeDepthEdge places a dark nearby silhouette next to distant
// glowing fog. Upsampling must not blur that fog across the silhouette.
func TestCompositeDepthEdge(t *testing.T) {
	p, s, tr := frame.New(4, 2), frame.New(2, 1), frame.New(2, 1)
	for i := range p.Pix {
		p.Pix[i] = frame.Pixel{R: 1, G: 1, B: 1, Depth: 100}
		if i%4 < 2 {
			p.Pix[i].Depth = 2
		}
	}
	s.Pix[0] = frame.Pixel{Depth: 2}
	s.Pix[1] = frame.Pixel{R: 8, G: 8, B: 8, Depth: 100}
	tr.Pix[0] = frame.Pixel{R: 1, G: 1, B: 1, Depth: 2}
	tr.Pix[1] = frame.Pixel{Depth: 100}
	got, err := CompositeTransport(p, s, tr)
	if err != nil {
		t.Fatal(err)
	}
	for i, pixel := range got.Pix {
		want := float32(8)
		if i%4 < 2 {
			want = 1
		}
		if pixel.R != want {
			t.Fatalf("pixel %d: %g, want %g", i, pixel.R, want)
		}
	}
	if p.Pix[0].R != 1 || s.Pix[1].R != 8 {
		t.Fatal("modified an input")
	}
}

// TestCompositeResolveKeepsNeedleGaps puts two opaque needle samples and
// two mist-lit gaps in one final pixel. The gap light must survive resolve.
func TestCompositeResolveKeepsNeedleGaps(t *testing.T) {
	samples := frame.New(2, 2)
	samples.Pix[0] = frame.Pixel{Depth: 2, Cover: 1}
	samples.Pix[1] = frame.Pixel{R: 4, G: 2, B: 1, Depth: 100, Cover: 1}
	samples.Pix[2] = samples.Pix[0]
	samples.Pix[3] = samples.Pix[1]
	resolved, err := ResolveTransport(samples, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := frame.Pixel{R: 2, G: 1, B: 0.5, Depth: 2, Cover: 1}
	if resolved.W != 1 || resolved.H != 1 || resolved.Pix[0] != want {
		t.Fatalf("needle gaps lost: %+v", resolved)
	}
	if _, err := ResolveTransport(samples, 3); err == nil {
		t.Fatal("accepted a partial pixel")
	}
}
