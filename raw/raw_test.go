package raw

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// sample is a small frame whose light reaches past white, as a render's
// does, and differs in every pixel and channel
func sample() *frame.Frame {
	made := frame.New(3, 2)
	for i := range made.Pix {
		light := float32(i)
		made.Pix[i].R, made.Pix[i].G, made.Pix[i].B =
			light*7.5, light/3, light*light+0.25
	}
	return made
}

func TestARawComesBackAsItWent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.pfm")
	kept := sample()
	if err := Write(path, kept); err != nil {
		t.Fatal(err)
	}
	back, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.W != kept.W || back.H != kept.H {
		t.Fatalf("a %dx%d raw came back %dx%d", kept.W, kept.H, back.W,
			back.H)
	}
	for i, pixel := range kept.Pix {
		got := back.Pix[i]
		if got.R != pixel.R || got.G != pixel.G || got.B != pixel.B {
			t.Errorf("pixel %d went in %v and came back %v", i,
				pixel, got)
		}
	}
}

func TestARawRefusesBrokenLight(t *testing.T) {
	broken := sample()
	broken.Pix[2].G = float32(math.NaN())
	path := filepath.Join(t.TempDir(), "broken.pfm")
	if err := Write(path, broken); err == nil {
		t.Error("a raw with light that is not a number was written")
	}
}

func TestARawRefusesWhatIsNotARaw(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not.pfm")
	ppm := []byte("P6\n1 1\n255\nabc")
	if err := os.WriteFile(path, ppm, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("a ppm was read as a raw")
	}
}

func TestMomentsAddUpToALongExposure(t *testing.T) {
	one, two := sample(), sample()
	total, err := Sum([]*frame.Frame{one, two})
	if err != nil {
		t.Fatal(err)
	}
	for i, pixel := range one.Pix {
		if total.Pix[i].R != 2*pixel.R || total.Pix[i].B != 2*pixel.B {
			t.Errorf("pixel %d summed to %v from %v twice", i,
				total.Pix[i], pixel)
		}
	}
	if _, err := Sum([]*frame.Frame{one, frame.New(4, 2)}); err == nil {
		t.Error("frames of two sizes were added")
	}
	if _, err := Sum(nil); err == nil {
		t.Error("nothing at all was added up")
	}
}
