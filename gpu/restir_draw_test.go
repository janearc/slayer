//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
)

// meanAndApart is the mean luminance of a, and the mean absolute
// difference of a from b over the mean of b.
func meanAndApart(a, b *frame.Frame, w, h int) (mean, apart float64) {
	var sa, sb, d float64
	for y := range h {
		for x := range w {
			pa, pb := a.At(x, y), b.At(x, y)
			la := float64(pa.R)*0.2126 + float64(pa.G)*0.7152 + float64(pa.B)*0.0722
			lb := float64(pb.R)*0.2126 + float64(pb.G)*0.7152 + float64(pb.B)*0.0722
			sa += la
			sb += lb
			d += math.Abs(la - lb)
		}
	}
	n := float64(w * h)
	return sa / n, d / sb
}

func drawRestir(t *testing.T, lamps int, frames int, set restirSettings) (*frame.Frame, Profile) {
	t.Helper()
	world, eye := crowdGain(t, lamps, 300)
	drawer := &Restir{Settings: set}
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	var picture *frame.Frame
	var profile Profile
	var err error
	for range frames {
		picture, profile, err = drawer.Draw(world, eye, lens, 0, 96, 54, 1)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return picture, profile
}

// restir is an approximation of the same light: against the exact
// native picture of twelve lamps, each lit exactly, the mean light is
// near and the pictures are near.
func TestRestirAgreesWithExactLightOfTwelveLamps(t *testing.T) {
	world, eye := crowdGain(t, 12, 300)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	truth, profile, err := SampledAccelerated(world, eye, lens, 0, 96, 54, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	if profile.Restir {
		t.Error("the exact picture says it was restir's")
	}
	got, spent := drawRestir(t, 12, 8, defaultRestir())
	if !spent.Restir || spent.RestirLamps != 12 || spent.RestirNodes != 23 {
		t.Errorf("profile does not say restir: %+v", spent)
	}
	mean, apart := meanAndApart(got, truth, 96, 54)
	t.Logf("restir mean %.4f, apart %.3f, restir gpu %v", mean, apart, spent.RestirGPU)
	tmean, _ := meanAndApart(truth, truth, 96, 54)
	if math.Abs(mean-tmean)/tmean > 0.15 {
		t.Errorf("mean light %.4f, exact %.4f", mean, tmean)
	}
	if apart > 0.6 {
		t.Errorf("pictures are %.2f of the light apart", apart)
	}
}

// anti: a restir with its lamps taken from it draws darkness; if the
// reservoir were not what lights the surface the picture would not
// follow the lamps.
func TestRestirFollowsTheLamps(t *testing.T) {
	with, _ := drawRestir(t, 12, 2, defaultRestir())
	world, eye := crowdGain(t, 12, 300)
	world.Lamps = world.Lamps[:1]
	drawer := NewRestir()
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	one, _, err := drawer.Draw(world, eye, lens, 0, 96, 54, 1)
	if err != nil {
		t.Fatal(err)
	}
	m12, _ := meanAndApart(with, with, 96, 54)
	m1, _ := meanAndApart(one, one, 96, 54)
	if !(m1 < 0.5*m12) {
		t.Errorf("one lamp drew %.4f, twelve %.4f", m1, m12)
	}
}

func TestRestirRefusesALampWithNoPlace(t *testing.T) {
	world, eye := crowdGain(t, 3, 300)
	world.Lamp(film.Sun{Toward: maths.V(0, 1, 0), Strength: maths.Grey(1)})
	_, _, err := NewRestir().Draw(world, eye, film.Pinhole{Wide: 50 * maths.Degree}, 0, 16, 9, 1)
	var refused unsupported
	if !errors.As(err, &refused) {
		t.Errorf("a sun among restir's lamps was not refused: %v", err)
	}
}
