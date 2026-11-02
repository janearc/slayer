package typography

import (
	"math"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

// printable is every letter, digit and mark a line is likely to hold.
const printable = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_{|}~"

// gauss is the four-point gauss-legendre rule on zero to one: places and
// weights. it is exact for a polynomial of degree seven or less, and x times
// the slope of y along a cubic is of degree five.
var gauss = [4][2]float64{
	{0.0694318442029737, 0.1739274225687269},
	{0.3300094782075719, 0.3260725774312731},
	{0.6699905217924281, 0.3260725774312731},
	{0.9305681557970263, 0.1739274225687269},
}

// truly is the area a contour of curves encloses, exactly: by green's
// theorem, the sum along it of x times the change in y, each piece's
// integral taken by gauss's rule, positive counterclockwise.
func truly(contour []piece) float64 {
	sum := 0.0
	for _, p := range contour {
		for _, g := range gauss {
			sum += g[1] * on(p, g[0]).x * slope(p, g[0]).y
		}
	}
	return sum
}

// slope is how fast a piece moves at t: its derivative, a curve one
// degree lower of the differences of its points, times its degree.
func slope(p piece, t float64) point {
	n := float64(len(p) - 1)
	d := make(piece, len(p)-1)
	for i := range d {
		d[i] = p[i+1].sub(p[i]).scale(n)
	}
	return on(d, t)
}

// covered is the area of triangles, and whether every one turns
// counterclockwise.
func covered(tris [][3]point) (float64, bool) {
	sum, all := 0.0, true
	for _, t := range tris {
		a := t[1].sub(t[0]).cross(t[2].sub(t[0])) / 2
		sum += a
		all = all && a > 0
	}
	return sum, all
}

// cut is a letter's solids cut into faces, every face of every solid.
func cut(t *testing.T, face *Font, letter rune,
	tolerance float64) ([][3]point, []solid, [][]piece) {
	t.Helper()
	glyph, _ := face.Index(letter)
	solids, err := face.solids(glyph, tolerance)
	if err != nil {
		t.Fatalf("%q: %v", letter, err)
	}
	var tris [][3]point
	for _, one := range solids {
		got, err := faces(one)
		if err != nil {
			t.Fatalf("%q: %v", letter, err)
		}
		tris = append(tris, got...)
	}
	outline, _ := face.outline(glyph)
	return tris, solids, outline
}

// perimeter is how long a glyph's flattened rings are, all of them.
func perimeter(solids []solid) float64 {
	sum := 0.0
	for _, one := range solids {
		for _, r := range one.rings() {
			for i, p := range r {
				sum += r.at(i + 1).sub(p).length()
			}
		}
	}
	return sum
}

// the faces of every printable letter of go's regular, bold and italic
// cover exactly the area of their flattened rings, every triangle turned
// counterclockwise; and that area is the outline's true area, the
// curves' own, to within the tolerance times the rings' length, which is
// the most that steps within the tolerance of a curve can add or take.
func TestFacesCoverTheOutlinesArea(t *testing.T) {
	for _, raw := range [][]byte{goregular.TTF, gobold.TTF, goitalic.TTF} {
		face := mustFont(t, raw)
		for _, letter := range printable {
			areas(t, face, letter)
		}
	}
}

// areas holds one letter's faces to its rings' area and its curves'.
func areas(t *testing.T, face *Font, letter rune) {
	t.Helper()
	tris, solids, outline := cut(t, face, letter, 2)
	got, turned := covered(tris)
	rings, curved := 0.0, 0.0
	for _, one := range solids {
		for _, r := range one.rings() {
			rings += r.area()
		}
	}
	for _, contour := range outline {
		curved += truly(contour)
	}
	if !turned || math.Abs(got-rings) > 1e-9*abs(rings) {
		t.Errorf("%q: faces %v, rings %v, all turned %v", letter, got,
			rings, turned)
	}
	if math.Abs(abs(curved)-got) > 2*perimeter(solids) {
		t.Errorf("%q: faces %v, the curves %v", letter, got, curved)
	}
}

// the area check can fail: faces with one triangle left out are short
// of the rings' area by that triangle.
func TestFacesShortOfATriangleAreShortOfTheArea(t *testing.T) {
	tris, solids, _ := cut(t, mustFont(t, goregular.TTF), 'o', 2)
	whole, _ := covered(tris)
	less, _ := covered(tris[1:])
	rings := solids[0].outer.area() + solids[0].holes[0].area()
	if math.Abs(whole-rings) > 1e-9*rings || !(rings-less > 1e-6) {
		t.Errorf("rings %v, whole %v, less one %v", rings, whole, less)
	}
}

// a grid of points over each letter, set off the grid so none falls on
// an edge, is in exactly one face where the letter is, and in none
// where it is not: inside its outer ring and outside all its holes.
func TestFacesCoverTheLetterOnce(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	for _, letter := range "carieo8B%@&gQ" {
		tris, solids, _ := cut(t, face, letter, 2)
		for x := -200.0; x < 2200; x += 37.31 {
			for y := -600.0; y < 1700; y += 41.17 {
				p := point{x + 0.123, y + 0.456}
				in, want := over(tris, p), inLetter(solids, p)
				once(t, letter, p, in, want)
			}
		}
	}
}

// once fails a point in a letter's faces other than as often as the
// letter holds it.
func once(t *testing.T, letter rune, p point, in, want int) {
	t.Helper()
	if in != want {
		t.Errorf("%q at %v: in %d faces, want %d", letter, p, in, want)
	}
}

// over is how many of the triangles hold p.
func over(tris [][3]point, p point) int {
	n := 0
	for _, t := range tris {
		if inside(t[0], t[1], t[2], p) {
			n++
		}
	}
	return n
}

// inLetter is one where p is in a solid, inside its outer and in none of its
// holes, and zero elsewhere.
func inLetter(solids []solid, p point) int {
	for _, one := range solids {
		in := one.outer.holds(p)
		for _, hole := range one.holes {
			in = in && !hole.holds(p)
		}
		if in {
			return 1
		}
	}
	return 0
}

// a square with a square hole in it, the hole's corners level with the
// outer's so the bridge's ray meets a corner, is cut whole; so are two
// holes side by side, the one bridged second seeing the first's bridge;
// and so is a hole whose bridge to the right would cross a spike of the
// outer that reaches up past it, so the bridge goes to the spike.
func TestHolesAtAwkwardPlacesAreBridged(t *testing.T) {
	square := func(x, y, side float64) ring {
		return ring{{x, y}, {x + side, y}, {x + side, y + side},
			{x, y + side}}
	}
	for name, one := range map[string]solid{
		"level": {outer: square(0, 0, 10),
			holes: []ring{square(3, 0.5, 4).reversed()}},
		"side by side": {
			outer: ring{{0, 0}, {30, 0}, {30, 10}, {0, 10}},
			holes: []ring{square(2, 2, 6).reversed(),
				square(12, 2, 6).reversed(),
				square(22, 4, 2).reversed()}},
		"a corner in the way": {outer: ring{{0, 0}, {9, 0}, {10, 7},
			{11, 0}, {20, 0}, {20, 20}, {0, 20}},
			holes: []ring{square(2, 8, 4).reversed()}},
	} {
		tris, err := faces(one)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := 0.0
		for _, r := range one.rings() {
			want += r.area()
		}
		if got, turned := covered(tris); !turned ||
			math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: faces %v of %v, all turned %v", name, got,
				want, turned)
		}
	}
}

// a ring that turns the wrong way has no ear to cut and says so, rather
// than giving faces that face backward.
func TestARingTurnedBackwardHasNoEars(t *testing.T) {
	backward := ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}}
	if _, err := clipped(backward); err != errNoEar {
		t.Errorf("clipping a clockwise ring: %v", err)
	}
	line := ring{{0, 0}, {5, 0}, {10, 0}}
	if _, err := clipped(line); err != errNoEar {
		t.Errorf("clipping a ring with no area: %v", err)
	}
}
