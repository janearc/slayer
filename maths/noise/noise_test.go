package noise

import (
	"math"
	"testing"
)

// The same inputs give the same number, and different seeds give different
// ones.
func TestHashIsRepeatableAndSeeded(t *testing.T) {
	if Hash(1, 42) != Hash(1, 42) ||
		Hash3(9, 1, 2, 3) != Hash3(9, 1, 2, 3) {
		t.Fatal("a hash changed between two asks")
	}
	if Hash(1, 42) == Hash(2, 42) {
		t.Fatal("two seeds gave one number")
	}
}

// Neighbouring indices give numbers spread evenly over zero to one: each of ten
// bins gets its tenth of a hundred thousand, within a few percent.
func TestHashSpreadsNeighbours(t *testing.T) {
	var bins [10]int
	const n = 100000
	for i := range int64(n) {
		h := Hash2(3, i, i+1)
		if h < 0 || h >= 1 {
			t.Fatalf("hash %v out of range", h)
		}
		bins[int(h*10)]++
	}
	for b, c := range bins {
		if math.Abs(float64(c)-n/10) > n/10*0.04 {
			t.Errorf("bin %d holds %d of %d", b, c, n)
		}
	}
}

// Gradient noise is zero on the lattice, bounded, and continuous: a step of a
// thousandth moves it by a small amount everywhere it is tried. the bound is
// one and a little, since it can pass one.
func TestGradientIsSmoothAndBounded(t *testing.T) {
	for i := range int64(2000) {
		x := Hash(1, i)*40 - 20
		y := Hash(2, i)*40 - 20
		z := Hash(3, i)*40 - 20
		v := Gradient(5, x, y, z)
		if math.Abs(v) > 1.05 {
			t.Fatalf("noise %v at %v,%v,%v", v, x, y, z)
		}
		if d := math.Abs(Gradient(5, x+1e-3, y, z) - v); d > 0.01 {
			t.Fatalf("a step of 1e-3 moved it by %v", d)
		}
		ix, iy, iz := math.Floor(x), math.Floor(y), math.Floor(z)
		if g := Gradient(5, ix, iy, iz); g != 0 {
			t.Fatalf("not nought on the lattice: %v", g)
		}
	}
}

// Fractal noise stays in about the range of one octave, and with no octaves it
// is zero.
func TestFractalIsBounded(t *testing.T) {
	for i := range int64(500) {
		x, y, z := Hash(1, i)*9, Hash(2, i)*9, Hash(3, i)*9
		if v := Fractal(7, x, y, z, 5); math.Abs(v) > 1.05 {
			t.Fatalf("fractal %v at %v,%v,%v", v, x, y, z)
		}
	}
	if Fractal(7, 0.3, 0.3, 0.3, 0) != 0 {
		t.Fatal("no octaves gave a value")
	}
}

// the numbers themselves, bit for bit, as they were recorded at ad2fff9: what
// slayer draws with. a change to mix, a constant, the fade or the octave sum
// shows here before it shows in a picture.
func TestTheNumbersAreTheRecorded(t *testing.T) {
	for _, want := range []struct {
		name string
		got  float64
		bits uint64
	}{
		{"Hash(1, 42)", Hash(1, 42), 0x3fe8d3e6ab10335e},
		{"Hash(0, 0)", Hash(0, 0), 0x3fd2086089bfcf34},
		{"Hash(7, -3)", Hash(7, -3), 0x3fe97f06ba09fb20},
		{"Hash2(3, 5, 6)", Hash2(3, 5, 6), 0x3fe8022e13bc0730},
		{"Hash3(9, 1, 2, 3)", Hash3(9, 1, 2, 3), 0x3fc31bb07ec445a8},
		{"Gradient(5, 0.3, 1.7, -2.2)", Gradient(5, 0.3, 1.7, -2.2),
			0x3fb04513f522a3e0},
		{"Gradient(0, -7.25, 0.5, 3.125)", Gradient(0, -7.25, 0.5, 3.125),
			0xbfde5ae2a8000000},
		{"Fractal(7, 0.3, 1.7, -2.2, 5)", Fractal(7, 0.3, 1.7, -2.2, 5),
			0xbfb9eeb5ad2a58b4},
		{"Fractal(11, 4.5, -1.25, 0.75, 8)",
			Fractal(11, 4.5, -1.25, 0.75, 8), 0x3fc684719368d06f},
	} {
		if bits := math.Float64bits(want.got); bits != want.bits {
			t.Errorf("%s is %#016x, recorded is %#016x", want.name, bits,
				want.bits)
		}
	}
}

// each of the twelve edges of a cube is its own direction: the offset
// 1, 2, 4 against each, worked by hand, and the thirteenth is the first.
func TestEveryEdgeOfTheCubeIsAGradient(t *testing.T) {
	for h, want := range []float64{3, 1, -1, -3, 5, 3, -3, -5, 6, 2, -2,
		-6, 3} {
		if got := grad(uint64(h), 1, 2, 4); got != want {
			t.Errorf("edge %d: %v, want %v", h, got, want)
		}
	}
}

// the fade is zero at zero, one at one and a half at a half, and flat at both
// ends.
func TestTheFadeIsQuintic(t *testing.T) {
	for _, want := range [][2]float64{{0, 0}, {1, 1}, {0.5, 0.5}} {
		if got := fade(want[0]); got != want[1] {
			t.Errorf("fade(%v) is %v", want[0], got)
		}
	}
	if slope := (fade(1e-6) - fade(0)) / 1e-6; slope > 1e-9 {
		t.Errorf("the fade leaves nought at a slope of %v", slope)
	}
	if slope := (fade(1) - fade(1-1e-6)) / 1e-6; slope > 1e-9 {
		t.Errorf("the fade comes to one at a slope of %v", slope)
	}
}

// Gradient can pass one, a little; the doc says where, and so it is.
func TestGradientPassesOneALittle(t *testing.T) {
	if v := Gradient(41885775, 0.35, 0.5, 0.475); v <= 1 || v > 1.05 {
		t.Errorf("Gradient there is %v", v)
	}
}

// across a lattice plane the noise does not jump: a billionth either
// side of it, anywhere, the two values are within a few billionths.
func TestGradientIsContinuousAcrossALatticePlane(t *testing.T) {
	for i := range int64(500) {
		x := math.Floor(Hash(1, i)*40-20) + 0
		y, z := Hash(2, i)*40-20, Hash(3, i)*40-20
		a, b := Gradient(5, x-1e-9, y, z), Gradient(5, x+1e-9, y, z)
		if math.Abs(a-b) > 1e-8 {
			t.Fatalf("across x=%v it jumps from %v to %v", x, a, b)
		}
	}
}

// zero on the lattice far from the middle, either side.
func TestGradientIsNoughtOnTheLatticeFarOut(t *testing.T) {
	for _, at := range []float64{-1e6, 1e6, -1e12, 1e12, -3, 0} {
		if g := Gradient(5, at, at+1, -at); g != 0 {
			t.Errorf("not nought at the lattice point %v: %v", at, g)
		}
	}
}

// a fractal of no octaves or fewer is zero, and of more octaves than a float
// can scale to is still a number within one: the octaves past that are not
// added.
func TestFractalOfAnyOctavesIsANumber(t *testing.T) {
	for _, octaves := range []int{-3, 0} {
		if v := Fractal(7, 0.3, 0.3, 0.3, octaves); v != 0 {
			t.Errorf("%d octaves gave %v", octaves, v)
		}
	}
	for _, octaves := range []int{1100, 2000} {
		// each coordinate overflowing alone, either way, and all three
		for _, at := range [][3]float64{{0.3, 1.7, -2.2}, {0, 0, -1.7},
			{0, 1.3, 0}, {-0.9, 0, 0}} {
			v := Fractal(7, at[0], at[1], at[2], octaves)
			if math.IsNaN(v) || math.Abs(v) > 1 {
				t.Errorf("%d octaves at %v gave %v", octaves, at, v)
			}
		}
	}
}

// a number is finite when it is neither NaN nor infinite either way.
func TestFiniteIsANumber(t *testing.T) {
	for v, want := range map[float64]bool{0: true, -1e308: true,
		math.Inf(1): false, math.Inf(-1): false} {
		if finite(v) != want {
			t.Errorf("finite(%v) is %v", v, !want)
		}
	}
	if finite(math.NaN()) {
		t.Error("NaN is finite")
	}
}
