// Package typography turns font outlines into solid letters by triangulation,
// extrusion, and beveling. Repeated glyphs can share mesh geometry while using
// different placements.
package typography

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Text is a line to make into letters. lengths are in the scene's
// units except Tolerance, which is in the font's.
type Text struct {
	// Words is the line, one line: a letter the font has no glyph for
	// is an error that names it.
	Words string
	// Size is an em, the font's own measure of its letters, in scene
	// units: a capital is about seven tenths of it tall.
	Size float64
	// Depth is how deep the letters are, front to back.
	Depth float64
	// Bevel is how far the front and back faces are stepped in from the
	// walls, and how far toward the middle they start: zero for none. it is
	// less than half the depth.
	Bevel float64
	// Smooth is the angle, in radians, below which two walls meeting at a
	// corner are shaded as one curve; at or above it the corner is sharp.
	// from zero to pi.
	Smooth float64
	// Tolerance is how far a straight step may stray from the curve it
	// stands for, in the font's units: zero is a thousandth of an em.
	Tolerance float64
}

// Letter is one letter of a line made solid: the letter, its mesh, and where
// the mesh's zero stands in the line's own space. a letter with no outline, a
// space, has no mesh.
type Letter struct {
	Letter rune
	Mesh   *mesh.Mesh
	At     maths.Vec
}

// Make is the line's letters, made in face. the line's own space has the
// letters' baseline along x at y zero, the line's advance centred on x zero,
// and its depth centred on z zero: the front faces plus z, so an eye looking
// down minus z reads it left to right.
func Make(face *Font, text Text) ([]Letter, error) {
	if err := text.check(); err != nil {
		return nil, err
	}
	line, err := Layout(face, text.Words)
	if err != nil {
		return nil, err
	}
	scale := text.Size / face.Units()
	made, kept := []Letter{}, map[Glyph]*mesh.Mesh{}
	for _, each := range line.Letters {
		solid, err := face.once(kept, each, text)
		if err != nil {
			return nil, err
		}
		at := maths.V((each.Pen-line.Width/2)*scale, 0, 0)
		made = append(made,
			Letter{Letter: each.Letter, Mesh: solid, At: at})
	}
	return made, nil
}

// once is a placed letter's mesh, made the first time its glyph is
// asked for and kept for every time after.
func (face *Font) once(kept map[Glyph]*mesh.Mesh, each Placed,
	text Text) (*mesh.Mesh, error) {
	if solid, ok := kept[each.Glyph]; ok {
		return solid, nil
	}
	solid, err := face.letter(each.Glyph, text)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", ErrType, each.Letter, err)
	}
	kept[each.Glyph] = solid
	return solid, nil
}

// sane is whether x is a finite number.
func sane(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// check is what is wrong with a text, if anything.
func (text Text) check() error {
	switch {
	case text.Words == "":
		return fmt.Errorf("%w: a text needs words", ErrType)
	case !sane(text.Size) || text.Size <= 0:
		return fmt.Errorf("%w: a size above nought, not %v", ErrType,
			text.Size)
	case !sane(text.Depth) || text.Depth <= 0:
		return fmt.Errorf("%w: a depth above nought, not %v", ErrType,
			text.Depth)
	case !sane(text.Bevel) || text.Bevel < 0 || 2*text.Bevel >= text.Depth:
		return fmt.Errorf("%w: a bevel from nought to under half the "+
			"depth, not %v", ErrType, text.Bevel)
	case !(text.Smooth >= 0 && text.Smooth <= math.Pi):
		return fmt.Errorf("%w: a smoothing angle from nought to pi, "+
			"not %v", ErrType, text.Smooth)
	case !sane(text.Tolerance) || text.Tolerance < 0:
		return fmt.Errorf("%w: a tolerance of nought or more, not %v",
			ErrType, text.Tolerance)
	}
	return nil
}

// letter is one glyph made a mesh to the text's measures, or nil for a
// glyph with no outline.
func (face *Font) letter(glyph Glyph, text Text) (*mesh.Mesh, error) {
	tolerance := text.Tolerance
	if tolerance == 0 {
		tolerance = face.Units() / 1000
	}
	solids, err := face.solids(glyph, tolerance)
	if err != nil || len(solids) == 0 {
		return nil, err
	}
	scale := text.Size / face.Units()
	return shape{scale: scale, depth: text.Depth, bevel: text.Bevel,
		smooth: text.Smooth}.letter(solids)
}

// solids is a glyph's outline as solids, its curves flattened to the
// tolerance; a glyph whose contours cross is errCrossed.
func (face *Font) solids(glyph Glyph, tolerance float64) ([]solid, error) {
	outline, err := face.outline(glyph)
	if err != nil {
		return nil, err
	}
	return solidsOf(outline, tolerance)
}

// solidsOf is an outline's contours flattened, cleaned and nested into
// solids; contours that cross are errCrossed.
func solidsOf(outline [][]piece, tolerance float64) ([]solid, error) {
	var rings []ring
	for _, contour := range outline {
		if r := cleaned(flattened(contour, tolerance)); r != nil {
			rings = append(rings, r)
		}
	}
	if crossed(rings) {
		return nil, errCrossed
	}
	return nested(rings), nil
}

// Things is the letters as things in a world, all of one material, each
// standing at its place in the line, and the line carried by motion: nil is
// still, at the world's zero.
func Things(letters []Letter, material *thing.Material,
	motion thing.Motion) []thing.Thing {
	var made []thing.Thing
	for _, each := range letters {
		if each.Mesh == nil {
			continue
		}
		standing := thing.Nested{Of: motion,
			In: maths.Subspace{Origin: each.At}}
		made = append(made, thing.Thing{Shape: each.Mesh,
			Material: material, Motion: standing})
	}
	return made
}
