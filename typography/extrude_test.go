package typography

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

// made is a letter made solid at an em of one, or the test fails.
func made(t *testing.T, face *Font, letter rune, text Text) *mesh.Mesh {
	t.Helper()
	glyph, err := face.Index(letter)
	if err != nil {
		t.Fatal(err)
	}
	solid, err := face.letter(glyph, text)
	if err != nil {
		t.Fatalf("%q: %v", letter, err)
	}
	return solid
}

// plain is a text with no bevel, an em of one and a depth of a fifth.
var plain = Text{Words: "x", Size: 1, Depth: 0.2, Smooth: 0.5}

// shared is how many faces run each edge of a mesh, either way, its
// corners taken by place as the mesh takes them.
func shared(solid *mesh.Mesh) map[[2]maths.Vec]int {
	points, counts := solid.Points(), map[[2]maths.Vec]int{}
	for _, f := range solid.Faces() {
		for k := range 3 {
			a, b := points[f[k]], points[f[(k+1)%3]]
			if b.X < a.X || b.X == a.X && (b.Y < a.Y ||
				b.Y == a.Y && b.Z < a.Z) {
				a, b = b, a
			}
			counts[[2]maths.Vec{a, b}]++
		}
	}
	return counts
}

// volume is the volume a mesh encloses, by the cones from zero to its faces:
// positive when they turn counterclockwise from outside.
func volume(solid *mesh.Mesh) float64 {
	sum := 0.0
	for i := range solid.FaceCount() {
		a, b, c := solid.Corners(i)
		sum += a.Dot(b.Cross(c)) / 6
	}
	return sum
}

// every printable letter of go's regular, bold and italic, made solid
// with and without a bevel, is a closed mesh as slayer's mesh checks
// it: mesh.New refuses an edge two faces run the same way and a closed
// piece inside out, and Closed is every edge met by a face running it
// back. counted here as well, every edge is run by exactly two faces.
//
// "k" and "K" take no bevel: go's join their arms to their stems by a
// neck two units thick, which a bevel of more than one cuts in two, and
// a ring moved in here cannot become two. they are refused as errBevel.
func TestEveryLetterIsClosed(t *testing.T) {
	bevelled := plain
	bevelled.Bevel = 0.01
	for _, raw := range [][]byte{goregular.TTF, gobold.TTF, goitalic.TTF} {
		face := mustFont(t, raw)
		for _, letter := range printable {
			closed(t, face, letter, plain)
			if strings.ContainsRune("kK", letter) {
				refused(t, face, letter, bevelled)
				continue
			}
			closed(t, face, letter, bevelled)
		}
	}
}

// closed fails a letter whose mesh is open, or has an edge run by other
// than two faces.
func closed(t *testing.T, face *Font, letter rune, text Text) {
	t.Helper()
	solid := made(t, face, letter, text)
	if !solid.Closed() {
		t.Errorf("%q, bevel %v: not closed", letter, text.Bevel)
	}
	for edge, n := range shared(solid) {
		if n != 2 {
			t.Errorf("%q: edge %v run by %d faces", letter, edge, n)
			return
		}
	}
}

// refused is a letter whose making must fail as errBevel.
func refused(t *testing.T, face *Font, letter rune, text Text) {
	t.Helper()
	glyph, _ := face.Index(letter)
	if _, err := face.letter(glyph, text); !errors.Is(err, errBevel) {
		t.Errorf("%q bevelled %v: %v", letter, text.Bevel, err)
	}
}

// the edge count can fail: a letter with one face taken away has edges
// run by one face only, and mesh.New says it is not closed.
func TestALetterWithAFaceMissingIsOpen(t *testing.T) {
	solid := made(t, mustFont(t, goregular.TTF), 'o', plain)
	faces := solid.Faces()[1:]
	open, err := mesh.New(solid.Points(), faces, nil, false)
	if err != nil || open.Closed() {
		t.Fatalf("one face less: %v, closed %v", err, open.Closed())
	}
	ones := 0
	for _, n := range shared(open) {
		if n == 1 {
			ones++
		}
	}
	if ones != 3 {
		t.Errorf("%d edges run by one face, want the missing face's 3",
			ones)
	}
}

// every letter's faces turn outward: its signed volume is positive, and
// without a bevel it is its face's area times its depth, the area the
// faces of its rings cover, scaled to the em.
func TestALettersVolumeIsItsAreaTimesItsDepth(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	scale := 1 / face.Units()
	for _, letter := range printable {
		tris, _, _ := cut(t, face, letter, face.Units()/1000)
		area, _ := covered(tris)
		want := area * scale * scale * plain.Depth
		got := volume(made(t, face, letter, plain))
		if !(got > 0) || math.Abs(got-want) > 1e-9*want {
			t.Errorf("%q: volume %v, want %v", letter, got, want)
		}
	}
}

// a letter turned inside out, every face's turn reversed, is refused by
// mesh.New: the volume check above is not the only one that would see
// faces facing in.
func TestALetterInsideOutIsRefused(t *testing.T) {
	solid := made(t, mustFont(t, goregular.TTF), 'e', plain)
	var turned [][3]int
	for _, f := range solid.Faces() {
		turned = append(turned, [3]int{f[0], f[2], f[1]})
	}
	if _, err := mesh.New(solid.Points(), turned, nil, false); !errors.Is(
		err, mesh.ErrMesh) {
		t.Errorf("an inside out letter: %v", err)
	}
}

// at every corner the normal given faces the same side as the face's
// own turn: no corner of any face is shaded as though it faced in.
func TestNormalsFaceOut(t *testing.T) {
	face := mustFont(t, gobold.TTF)
	bevelled := plain
	bevelled.Bevel = 0.01
	for _, letter := range "carrie@S&" {
		for _, text := range []Text{plain, bevelled} {
			solid := made(t, face, letter, text)
			for i := range solid.FaceCount() {
				facesOut(t, letter, solid, i)
			}
		}
	}
}

// facesOut fails face i of a letter if a normal at a corner of it is
// more than sixty degrees from the way the face turns.
func facesOut(t *testing.T, letter rune, solid *mesh.Mesh, i int) {
	t.Helper()
	a, b, c := solid.Corners(i)
	turn := b.Sub(a).Cross(c.Sub(a)).Unit()
	normals, _ := solid.CornerNormals(i)
	for _, n := range normals {
		if !(n.Dot(turn) > 0.5) {
			t.Errorf("%q face %d: normal %v, turn %v", letter, i, n,
				turn)
		}
	}
}

// a bevel takes from the volume: a bevelled letter holds less than its
// area times its depth, and more than its inset's area times its depth,
// and its front is at half the depth, as the plain one's is.
func TestABevelTakesFromTheVolume(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	bevelled := plain
	bevelled.Bevel = 0.01
	for _, letter := range "carieoB" {
		whole := volume(made(t, face, letter, plain))
		glyph, _ := face.Index(letter)
		solids, _ := face.solids(glyph, face.Units()/1000)
		inner := 0.0
		for _, one := range solids {
			in, _, err := inset(one, bevelled.Bevel*face.Units())
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range in.rings() {
				em := face.Units()
				inner += r.area() / em / em * plain.Depth
			}
		}
		solid := made(t, face, letter, bevelled)
		got := volume(solid)
		if !(got < whole && got > inner) {
			t.Errorf("%q: bevelled %v, plain %v, inset %v",
				letter, got, whole, inner)
		}
		if box := solid.Bounds(); box.Max.Z != plain.Depth/2 {
			t.Errorf("%q: the front at %v", letter, box.Max.Z)
		}
	}
}

// a bevel wider than a stroke is refused by name, and so is a text whose
// measures cannot be made: no words, no size or depth, a bevel of half the
// depth, a smoothing angle past pi, a tolerance less than zero.
func TestWhatCannotBeMadeIsRefused(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	wide := Text{Words: "il", Size: 1, Depth: 1, Bevel: 0.2}
	if _, err := Make(face, wide); !errors.Is(err, errBevel) ||
		!errors.Is(err, ErrType) {
		t.Errorf("a bevel past the stroke: %v", err)
	}
	for _, text := range []Text{{Size: 1, Depth: 1},
		{Words: "a", Depth: 1}, {Words: "a", Size: 1},
		{Words: "a", Size: math.NaN(), Depth: 1},
		{Words: "a", Size: 1, Depth: 1, Bevel: 0.5},
		{Words: "a", Size: 1, Depth: 1, Bevel: -1},
		{Words: "a", Size: 1, Depth: 1, Smooth: 4},
		{Words: "a", Size: 1, Depth: 1, Tolerance: -1},
		{Words: "a͸", Size: 1, Depth: 1}} {
		if _, err := Make(face, text); !errors.Is(err, ErrType) {
			t.Errorf("%+v: %v", text, err)
		}
	}
}

// where a curve is flattened its walls blend, so the corners between
// its steps each have one normal; at a sharp corner, the top of the
// stem of an "l", the two walls keep their own. with no smoothing every
// corner is sharp.
func TestWallsBlendAcrossGentleCornersOnly(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	round := map[float64]int{}
	for _, smooth := range []float64{0, 0.5} {
		text := plain
		text.Smooth = smooth
		round[smooth] = normalsAt(made(t, face, 'o', text))
		if got := normalsAt(made(t, face, 'l', text)); got != 2 {
			t.Errorf("smooth %v: the l's corners have %d normals",
				smooth, got)
		}
	}
	if round[0] != 2 || round[0.5] != 1 {
		t.Errorf("the o's walls have %d normals a corner with no "+
			"smoothing and %d with", round[0], round[0.5])
	}
}

// normalsAt is the most normals in the plane any one place on a mesh's
// walls has: one where the walls blend, two where they meet sharp.
func normalsAt(solid *mesh.Mesh) int {
	at := map[maths.Vec]map[maths.Vec]bool{}
	for i := range solid.FaceCount() {
		a, b, c := solid.Corners(i)
		normals, _ := solid.CornerNormals(i)
		for k, p := range [3]maths.Vec{a, b, c} {
			if n := normals[k]; n.Z == 0 {
				if at[p] == nil {
					at[p] = map[maths.Vec]bool{}
				}
				at[p][n] = true
			}
		}
	}
	most := 0
	for _, normals := range at {
		most = max(most, len(normals))
	}
	return most
}
