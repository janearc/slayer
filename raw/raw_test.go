package raw

import (
	"math"
	"os"
	"path/filepath"
	"strings"
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

// TestWriteRefusesAPathItCannotCreate checks that a path in a folder that
// is not there is an error, and that no file appears.
func TestWriteRefusesAPathItCannotCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "none", "sample.pfm")
	if err := Write(path, sample()); err == nil {
		t.Error("a raw was written into a folder that is not there")
	}
}

// TestWriteMakesTheHeaderAndBottomRowFirst checks the bytes of a one by
// two raw: the header, then the bottom row's pixel, then the top row's.
func TestWriteMakesTheHeaderAndBottomRowFirst(t *testing.T) {
	made := frame.New(1, 2)
	made.Set(0, 0, frame.Pixel{R: 1, G: 2, B: 3})
	made.Set(0, 1, frame.Pixel{R: 4, G: 5, B: 6})
	path := filepath.Join(t.TempDir(), "two.pfm")
	if err := Write(path, made); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := "PF\n1 2\n-1\n"
	if len(data) != len(header)+2*12 || string(data[:len(header)]) != header {
		t.Fatalf("file is %q", data)
	}
	// 4.0 as a little-endian float32 is 00 00 80 40; the bottom row is
	// first, so the first float is the bottom pixel's red, 4.
	if got := data[len(header) : len(header)+4]; got[0] != 0 || got[1] != 0 || got[2] != 0x80 || got[3] != 0x40 {
		t.Errorf("first float is % x, want the bottom pixel's red, 4.0", got)
	}
}

// TestReadRefusals checks each way a file is turned away, by name.
func TestReadRefusals(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name, content, want string
	}{
		{"no size", "PF\nwide tall\n-1\n", "has no size"},
		{"zero width", "PF\n0 1\n-1\n", "has no size"},
		{"zero height", "PF\n1 0\n-1\n", "has no size"},
		{"no scale", "PF\n1 1\nbig\n", "has no scale"},
		{"zero scale", "PF\n1 1\n0\n", "has no scale"},
		{"truncated pixels", "PF\n1 1\n-1\n\x00\x00\x80\x3f", "ends before its pixels do"},
		{"not a colour pfm", "Pf\n1 1\n-1\n", "not a colour PFM"},
	}
	for _, c := range cases {
		path := filepath.Join(dir, "x.pfm")
		if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Read(path)
		if err == nil || got != nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v %v, want an error containing %q", c.name, got, err, c.want)
		}
	}
	if _, err := Read(filepath.Join(dir, "missing.pfm")); err == nil {
		t.Error("a file that is not there was read")
	}
}

// TestReadBigEndianScalesBySize checks that a positive scale means big
// endian samples and multiplies each by the size of the scale. A one by
// two raw stores the bottom row first.
func TestReadBigEndianScalesBySize(t *testing.T) {
	var body []byte
	body = append(body, "PF\n1 2\n2\n"...)
	// big-endian 1.0, 2.0, 3.0 then 0.5, 0.25, 0.125
	for _, bits := range []uint32{0x3f800000, 0x40000000, 0x40400000, 0x3f000000, 0x3e800000, 0x3e000000} {
		body = append(body, byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
	}
	path := filepath.Join(t.TempDir(), "big.pfm")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.W != 1 || got.H != 2 {
		t.Fatalf("size %dx%d", got.W, got.H)
	}
	if p := got.At(0, 1); p.R != 2 || p.G != 4 || p.B != 6 {
		t.Errorf("bottom pixel %v, want 2 4 6", p)
	}
	if p := got.At(0, 0); p.R != 1 || p.G != 0.5 || p.B != 0.25 {
		t.Errorf("top pixel %v, want 1 0.5 0.25", p)
	}
}
