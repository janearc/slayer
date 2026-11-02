package typography

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

//go:generate sh mocks/generate.sh

// ErrType is a font, a glyph or a line of text that cannot be made into
// letters; the error says which and why.
var ErrType = errors.New("typography")

// Glyph is a glyph's number in its font.
type Glyph = sfnt.GlyphIndex

// Metrics is what laying out a line asks of a font: how many of its
// units make an em, which glyph draws a letter, how far a glyph moves
// the pen, and how much closer or further a pair of glyphs is set than
// their advances say. every length is in the font's own units.
//
// a *Font is one. a test can give another, for a font this one cannot
// read, such as one that kerns.
type Metrics interface {
	Units() float64
	Index(letter rune) (Glyph, error)
	Advance(glyph Glyph) (float64, error)
	Kern(left, right Glyph) (float64, error)
}

// Font is a truetype or opentype font, read. it keeps one buffer for
// sfnt to work in, so one Font is for one goroutine at a time.
type Font struct {
	face *sfnt.Font
	em   fixed.Int26_6 // an em, as the size sfnt reads at: one per unit
	work sfnt.Buffer
}

// Parse reads a font from the bytes of its file.
func Parse(raw []byte) (*Font, error) {
	face, err := sfnt.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrType, err)
	}
	return &Font{face: face, em: fixed.I(int(face.UnitsPerEm()))}, nil
}

// Load reads the font file at path.
func Load(path string) (*Font, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrType, err)
	}
	return Parse(raw)
}

// Units is how many of the font's units make an em.
func (made *Font) Units() float64 { return float64(made.face.UnitsPerEm()) }

// Index is the glyph that draws a letter; a letter the font has no
// glyph for is an error that names it.
func (made *Font) Index(letter rune) (Glyph, error) {
	glyph, err := made.face.GlyphIndex(&made.work, letter)
	if err == nil && glyph == 0 {
		err = errors.New("the font has no glyph for it")
	}
	if err != nil {
		return 0, fmt.Errorf("%w: %q: %w", ErrType, letter, err)
	}
	return glyph, nil
}

// Advance is how far a glyph moves the pen along the line.
func (made *Font) Advance(glyph Glyph) (float64, error) {
	got, err := made.face.GlyphAdvance(&made.work, glyph, made.em,
		font.HintingNone)
	return units(got), err
}

// Kern is how far the right glyph is set from where the left one's advance
// leaves the pen: less than zero is nearer. a glyph the font does not have is
// an error.
//
// it is zero for a font with no kerning, as go's own fonts have none, and for a
// pair a font's gpos table does not kern, which sfnt says is not found.
func (made *Font) Kern(left, right Glyph) (float64, error) {
	if n := made.face.NumGlyphs(); int(left) >= n || int(right) >= n {
		return 0, fmt.Errorf("%w: kerning glyphs %d and %d of %d",
			ErrType, left, right, n)
	}
	got, err := made.face.Kern(&made.work, left, right, made.em,
		font.HintingNone)
	if errors.Is(err, sfnt.ErrNotFound) {
		return 0, nil
	}
	return units(got), err
}

// units is a length sfnt gives at one unit a pixel, in the font's units:
// sfnt counts in 64ths.
func units(length fixed.Int26_6) float64 { return float64(length) / 64 }

// outline is a glyph's contours, each a closed run of pieces in the
// font's units with y up.
func (made *Font) outline(glyph Glyph) ([][]piece, error) {
	segments, err := made.face.LoadGlyph(&made.work, glyph, made.em, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: glyph %d: %w", ErrType, glyph, err)
	}
	return contours(segments), nil
}

// contours is sfnt's segments as pieces, a contour for each move.
func contours(segments sfnt.Segments) [][]piece {
	var made [][]piece
	var pen point
	for _, each := range segments {
		ends := pointsOf(each)
		if each.Op == sfnt.SegmentOpMoveTo || len(made) == 0 {
			made = append(made, nil)
		}
		if each.Op != sfnt.SegmentOpMoveTo {
			last := len(made) - 1
			step := append(piece{pen}, ends...)
			made[last] = append(made[last], step)
		}
		pen = ends[len(ends)-1]
	}
	return made
}

// pointsOf is a segment's points after the pen: one for a move or a
// line, two for a quadratic curve, three for a cubic. sfnt's y grows
// down the page, so it is turned to grow up.
func pointsOf(each sfnt.Segment) []point {
	count := map[sfnt.SegmentOp]int{sfnt.SegmentOpQuadTo: 2,
		sfnt.SegmentOpCubeTo: 3}[each.Op]
	count = max(count, 1)
	made := make([]point, count)
	for i := range made {
		made[i] = point{units(each.Args[i].X), -units(each.Args[i].Y)}
	}
	return made
}
