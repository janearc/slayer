package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// whites is a pixel's luminance in whites of the print, gain whites to a
// unit of light.
func whites(p *picture, pixel int, gain float64) float64 {
	at := pixel * 3
	return (0.2126*p.light[at] + 0.7152*p.light[at+1] +
		0.0722*p.light[at+2]) * gain
}

// The swell pass's flame core, sampled from its first frame, reaches the
// print at about two hundred whites at the moon's 23 stops; its edge is
// taken as ten thousand times fainter. Through a ceiling of 0.6 whites
// the core lands just under the ceiling, its red, green and blue in the
// proportions it had, and the edge keeps nearly all its light.
func TestTheFireShoulderKeepsTheFlamesColourUnderItsCeiling(t *testing.T) {
	const top = 0.6
	gain := math.Exp2(23)
	core := [3]float64{7.371519e-05, 1.960431e-05, 3.828279e-07}
	edge := [3]float64{core[0] * 1e-4, core[1] * 1e-4, core[2] * 1e-4}
	fire := &picture{2, 1, []float64{core[0], core[1], core[2],
		edge[0], edge[1], edge[2]}}
	before := whites(fire, 1, gain)
	shoulder(fire, top, gain)
	if got := whites(fire, 0, gain); !(got >= 0.99*top && got <= top) {
		t.Errorf("the core is %v whites through a ceiling of %v", got, top)
	}
	for c, want := range []float64{core[1] / core[0], core[2] / core[0]} {
		got := fire.light[1+c] / fire.light[0]
		if !(math.Abs(got-want) <= 1e-12*want) {
			t.Errorf("the core's channel %d over its red is %v, was %v",
				1+c, got, want)
		}
	}
	if after := whites(fire, 1, gain); !(after <= before &&
		after >= 0.97*before) {
		t.Errorf("the edge is %v whites, was %v", after, before)
	}
}

// A pixel of endless light has no colour to keep: it is the ceiling, in
// white, and not the not-a-number that endless over endless would make.
func TestTheFireShoulderPrintsEndlessLightAsTheCeilingInWhite(t *testing.T) {
	const top, gain = 0.6, 1e6
	fire := &picture{1, 1, []float64{math.Inf(1), 1e-6, 0}}
	shoulder(fire, top, gain)
	for c, got := range fire.light {
		if !(math.Abs(got*gain-top) <= 1e-12) {
			t.Errorf("channel %d is %v whites, not the ceiling %v", c,
				got*gain, top)
		}
	}
}

// Dark, negative and not-a-number pixels are left as they were: the
// shoulder only rolls off light.
func TestTheFireShoulderLeavesDarkPixelsAlone(t *testing.T) {
	fire := &picture{3, 1, []float64{0, 0, 0, -1, 0, 0, math.NaN(), 0, 0}}
	shoulder(fire, 0.6, 1e6)
	if fire.light[0] != 0 || fire.light[3] != -1 ||
		!math.IsNaN(fire.light[6]) {
		t.Errorf("the dark pixels moved: %v", fire.light)
	}
}

// guard saves the pass's package state and puts it back when the test
// ends, so a test may set the cues, the tree masks and the fire's
// settings freely.
func guard(t *testing.T) {
	t.Helper()
	savedShow, savedNear, savedClose := show, near, closeTrees
	savedSwell, savedBurst := fireSwellDir, fireBurstDir
	t.Cleanup(func() {
		show, near, closeTrees = savedShow, savedNear, savedClose
		fireSwellDir, fireBurstDir = savedSwell, savedBurst
	})
}

// flat is a picture of a size with every number the same.
func flat(width, height int, value float64) *picture {
	p := &picture{width, height, make([]float64, width*height*3)}
	for i := range p.light {
		p.light[i] = value
	}
	return p
}

// total is the sum of every number in a picture.
func total(p *picture) float64 {
	sum := 0.0
	for _, v := range p.light {
		sum += v
	}
	return sum
}

// allOpen is a sky mask with every pixel open.
func allOpen(width, height int) []bool {
	open := make([]bool, width*height)
	for i := range open {
		open[i] = true
	}
	return open
}

// keep writes a picture into a temporary directory and says where.
func keep(t *testing.T, dir, name string, p *picture, scale float64) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := write(path, p, scale); err != nil {
		t.Fatal(err)
	}
	return path
}

// within reports whether two numbers are within a tolerance.
func within(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// TestVecArithmetic checks the vector helpers against worked values.
func TestVecArithmetic(t *testing.T) {
	a, b := vec{1, 2, 3}, vec{4, 5, 6}
	if got := a.add(b); got != (vec{5, 7, 9}) {
		t.Errorf("add is %v", got)
	}
	if got := a.scale(2); got != (vec{2, 4, 6}) {
		t.Errorf("scale is %v", got)
	}
	if got := a.dot(b); got != 32 {
		t.Errorf("dot is %v", got)
	}
	if got := (vec{3, 0, 4}).unit(); !within(got.x, 0.6, 1e-12) ||
		got.y != 0 || !within(got.z, 0.8, 1e-12) {
		t.Errorf("unit is %v", got)
	}
	if got := (vec{1, 0, 0}).cross(vec{0, 1, 0}); got != (vec{0, 0, 1}) {
		t.Errorf("cross of east and north is %v", got)
	}
}

// TestTowardCompass checks that azimuth runs clockwise from north with
// altitude up.
func TestTowardCompass(t *testing.T) {
	for _, c := range []struct {
		az, alt float64
		want    vec
	}{
		{0, 0, vec{0, 1, 0}}, {90, 0, vec{1, 0, 0}}, {0, 90, vec{0, 0, 1}},
		{180, 0, vec{0, -1, 0}},
	} {
		got := toward(c.az, c.alt)
		if !within(got.x, c.want.x, 1e-9) || !within(got.y, c.want.y, 1e-9) ||
			!within(got.z, c.want.z, 1e-9) {
			t.Errorf("toward %v %v is %v, want %v", c.az, c.alt, got, c.want)
		}
	}
}

// TestNumberFrom reads a number from the environment and falls back to
// the default when the variable is unset.
func TestNumberFrom(t *testing.T) {
	if got := numberFrom("LIGHTSHOW_TEST_UNSET", 7.5); got != 7.5 {
		t.Errorf("unset gave %v", got)
	}
	t.Setenv("LIGHTSHOW_TEST_SET", "2.25")
	if got := numberFrom("LIGHTSHOW_TEST_SET", 7.5); got != 2.25 {
		t.Errorf("set gave %v", got)
	}
	t.Setenv("LIGHTSHOW_TEST_WORD", "")
	if got := textFrom("LIGHTSHOW_TEST_WORD", "otherwise"); got != "otherwise" {
		t.Errorf("empty text gave %q", got)
	}
	t.Setenv("LIGHTSHOW_TEST_WORD", "chosen")
	if got := textFrom("LIGHTSHOW_TEST_WORD", "otherwise"); got != "chosen" {
		t.Errorf("set text gave %q", got)
	}
}

// TestSmoothEasesBetweenItsEnds checks the ends, the middle and a
// reversed range.
func TestSmoothEasesBetweenItsEnds(t *testing.T) {
	for _, c := range []struct{ a, b, x, want float64 }{
		{0, 1, -1, 0}, {0, 1, 2, 1}, {0, 1, 0.5, 0.5}, {0, 1, 0.25, 0.15625},
		{10, 0, 5, 0.5}, {10, 0, 20, 0},
	} {
		if got := smooth(c.a, c.b, c.x); !within(got, c.want, 1e-12) {
			t.Errorf("smooth(%v, %v, %v) is %v, want %v", c.a, c.b, c.x, got,
				c.want)
		}
	}
}

// TestSmallHelpers checks clampInt, moonLum and the ordering of bytes.
func TestSmallHelpers(t *testing.T) {
	for _, c := range []struct{ v, n, want int }{
		{-3, 10, 0}, {4, 10, 4}, {10, 10, 9}, {99, 10, 9},
	} {
		if got := clampInt(c.v, c.n); got != c.want {
			t.Errorf("clampInt(%d, %d) is %d, want %d", c.v, c.n, got, c.want)
		}
	}
	if got := moonLum([3]float64{3, 6, 9}); got != 6 {
		t.Errorf("moonLum is %v", got)
	}
	if order(1) != binary.BigEndian || order(-1) != binary.LittleEndian {
		t.Error("a positive scale is big endian and a negative one little")
	}
}

// TestBrightestTakesTheMeanOfTheBrightHalf: pixels under half the peak
// are left out of the mean.
func TestBrightestTakesTheMeanOfTheBrightHalf(t *testing.T) {
	p := &picture{4, 1, []float64{
		10, 0, 0, // peak, sum 10
		6, 2, 0, // sum 8, kept
		1, 1, 1, // sum 3, under half of 10
		0, 0, 0}}
	got := brightest(p)
	if got != [3]float64{8, 1, 0} {
		t.Errorf("brightest is %v, want [8 1 0]", got)
	}
}

// TestLatticeAndNoise: the lattice is in range and repeatable, and value
// noise at a lattice point is the lattice's number there.
func TestLatticeAndNoise(t *testing.T) {
	for x := int64(-3); x < 4; x++ {
		v := lattice(x, 2*x, 5)
		if v < 0 || v >= 1 || v != lattice(x, 2*x, 5) {
			t.Errorf("lattice(%d) is %v", x, v)
		}
	}
	if lattice(1, 2, 3) == lattice(1, 2, 4) {
		t.Error("two seeds gave the same lattice number")
	}
	if got, want := noise(3, 4, 9), lattice(3, 4, 9); got != want {
		t.Errorf("noise at a lattice point is %v, the lattice has %v", got, want)
	}
	if got, want := noise3(3, 4, 5, 9), lattice(3, 4+5*104729, 9); got != want {
		t.Errorf("noise3 at a lattice point is %v, the lattice has %v", got,
			want)
	}
	// half way along an edge it is the mean of its two ends
	mid := noise(3.5, 4, 9)
	if want := (lattice(3, 4, 9) + lattice(4, 4, 9)) / 2; !within(mid, want, 1e-12) {
		t.Errorf("noise half way is %v, want %v", mid, want)
	}
	for i := 0; i < 50; i++ {
		u, v := float64(i)*0.37, float64(i)*0.91
		for name, got := range map[string]float64{
			"fbm": fbm(u, v, 3), "fbm3": fbm3(u, v, u+v, 3),
			"curled": curled(u, v, 1.5, 3)} {
			if got < 0 || got > 1 {
				t.Errorf("%s(%v, %v) is %v, outside 0 to 1", name, u, v, got)
			}
		}
	}
}

// TestFogThinsWithHeightAndIsNoneBelowTheWater checks the layer's edges.
func TestFogThinsWithHeightAndIsNoneBelowTheWater(t *testing.T) {
	if got := fog(vec{0, 0, waterZ - 1}, 0); got != 0 {
		t.Errorf("fog under the water is %v", got)
	}
	low, high := 0.0, 0.0
	for i := 0; i < 40; i++ {
		x := float64(i) * 13
		low += fog(vec{x, x, waterZ + 1}, 2)
		high += fog(vec{x, x, waterZ + 400}, 2)
	}
	if !(low > 10*high) {
		t.Errorf("fog near the water %v is not far over fog at 400 m %v", low,
			high)
	}
}

// TestPfmRoundTrip writes a picture of each byte order and reads it back,
// flipping rows as the format does.
func TestPfmRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := &picture{2, 2, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	for _, scale := range []float64{1, -1} {
		path := keep(t, dir, "a.pfm", p, scale)
		got, gotScale, err := read(path)
		if err != nil {
			t.Fatal(err)
		}
		if gotScale != scale || got.width != 2 || got.height != 2 {
			t.Fatalf("read back %dx%d scale %v", got.width, got.height,
				gotScale)
		}
		for i, v := range p.light {
			if got.light[i] != v {
				t.Errorf("number %d is %v, want %v", i, got.light[i], v)
			}
		}
	}
}

// TestPfmBytes reads a hand written file: rows are stored bottom up and a
// negative scale is little endian.
func TestPfmBytes(t *testing.T) {
	var body bytes.Buffer
	body.WriteString("PF\n1 2\n-1\n")
	// the bottom row is stored first
	binary.Write(&body, binary.LittleEndian, []float32{7, 8, 9, 1, 2, 3})
	path := filepath.Join(t.TempDir(), "h.pfm")
	if err := os.WriteFile(path, body.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := read(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{1, 2, 3, 7, 8, 9}
	for i := range want {
		if got.light[i] != want[i] {
			t.Fatalf("light is %v, want %v", got.light, want)
		}
	}
	var out bytes.Buffer
	binary.Write(&out, binary.BigEndian, []float32{7, 8, 9, 1, 2, 3})
	written := keep(t, t.TempDir(), "w.pfm", got, 1)
	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), "PF\n1 2\n1\n") ||
		!bytes.HasSuffix(raw, out.Bytes()) {
		t.Errorf("written file is % x", raw)
	}
}

// TestPfmRefusals: a missing file, a grey map and a short body are each
// an error, and a write to a missing directory is one too.
func TestPfmRefusals(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := read(filepath.Join(dir, "none.pfm")); err == nil {
		t.Error("a missing file was read")
	}
	grey := filepath.Join(dir, "grey.pfm")
	os.WriteFile(grey, []byte("Pf\n1 1\n-1\n\x00\x00\x00\x00"), 0o644)
	if _, _, err := read(grey); err == nil ||
		!strings.Contains(err.Error(), "not a colour pfm") {
		t.Errorf("a grey map gave %v", err)
	}
	short := filepath.Join(dir, "short.pfm")
	os.WriteFile(short, []byte("PF\n2 2\n-1\n\x00\x00"), 0o644)
	if _, _, err := read(short); err == nil {
		t.Error("a short body was read")
	}
	junk := filepath.Join(dir, "junk.pfm")
	os.WriteFile(junk, []byte("PF"), 0o644)
	if _, _, err := read(junk); err == nil {
		t.Error("a file with no header was read")
	}
	if err := write(filepath.Join(dir, "no", "such", "x.pfm"), flat(1, 1, 0),
		1); err == nil {
		t.Error("a write into a missing directory was taken")
	}
}
