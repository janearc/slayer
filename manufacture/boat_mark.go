package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/typography"
)

// The painted letters of a boat: a number on each bow and a name on the
// stern, each letter a solid made by package typography from a face of go's
// own, a real part standing a little out of the hull. The face and the names
// are data, and there is a place there for Chinese, not used yet: no font
// is read or kept here, and the Latin face is used while the Chinese one
// is named nowhere.
const (
	boatLetterDepth     = 0.012 // chosen: metres a painted letter stands out, front to back
	boatLetterTolerance = 10    // chosen: font units a letter's outline may stray, as the signs have it
	boatLetterTall      = 0.7   // defined: how tall a capital is as a share of its em, as typography says
	boatNameFill        = 0.85  // chosen: how much of the width it has a name takes
	boatNumberDigits    = 10000 // defined: a number of four digits
)

// boatGlyph is a letter's mesh as the form of a part: its faces.
type boatGlyph struct{ solid *mesh.Mesh }

// skin is the letter's faces as triangles, in the winding typography made
// them.
func (shape boatGlyph) skin(cut) surface {
	count := shape.solid.FaceCount()
	skin := make(surface, 0, count)
	for index := range count {
		first, second, third := shape.solid.Corners(index)
		skin = append(skin, face{first, second, third})
	}
	return skin
}

// The ways a line of letters faces, as the axes of its space: its x along
// the line, its y up, its z out of the hull. On the starboard side the line
// runs toward the bow, on the port toward the stern, and on the transom
// toward starboard, so that each reads left to right to one who stands
// off it.
var (
	boatStarboardFace = maths.Orientation{X: maths.V(0, 0, -1),
		Y: maths.YAxis, Z: maths.V(1, 0, 0)}
	boatPortFace = maths.Orientation{X: maths.V(0, 0, 1), Y: maths.YAxis,
		Z: maths.V(-1, 0, 0)}
	boatSternFace = maths.Orientation{X: maths.V(1, 0, 0), Y: maths.YAxis,
		Z: maths.V(0, 0, 1)}
)

// widthOf is how wide a line of words runs in a face, in ems.
func widthOf(face *typography.Font, words string) (float64, error) {
	line, err := typography.Layout(face, words)
	return line.Width / face.Units(), err
}

// paint is a line of words painted on the hull: set in a face, in letters
// that fit a width and a height, centred on a place, facing as the axes say.
// A face or a letter it cannot make is the assembly's trouble.
func (craft *boat) paint(face string, words string, wide, tall float64,
	at maths.Vec, facing maths.Orientation) {
	font, err := typography.Named(face, "")
	if err != nil {
		craft.assembly.trouble = err
		return
	}
	ems, err := widthOf(font, words)
	if err != nil {
		craft.assembly.trouble = err
		return
	}
	size := math.Min(tall/boatLetterTall, wide*boatNameFill/ems)
	letters, err := typography.Make(font, typography.Text{Words: words,
		Size: size, Depth: boatLetterDepth, Tolerance: boatLetterTolerance})
	if err != nil {
		craft.assembly.trouble = err
		return
	}
	lift := facing.Out(maths.V(0, 0, boatLetterDepth*spaceHalf))
	for _, letter := range letters {
		if letter.Mesh == nil {
			continue
		}
		origin := at.Add(facing.Out(maths.V(letter.At.X, letter.At.Y-
			size*boatLetterTall*spaceHalf, 0))).Add(lift)
		craft.assembly.add(boatGlyph{letter.Mesh}, maths.Subspace{
			Origin: origin, Orientation: facing, Scale: spaceOne}, boatLetter)
	}
}

// number is the boat's four digits, drawn from its seed.
func (craft *boat) number() string {
	return fmt.Sprintf("%04d", int(craft.from.draw(housing, boatDrawName, 0)*
		boatNumberDigits))
}

// name is the name the boat carries, and the face it is set in: one of the
// Chinese names in the Chinese face where the data names one, and else one
// of the Latin names in the Latin face.
func (craft *boat) name() (face, words string) {
	lettering := boatRead.Lettering
	names, face := lettering.Names, lettering.Face
	if lettering.Chinese.Face != "" {
		names, face = lettering.Chinese.Names, lettering.Chinese.Face
	}
	pick := int(craft.from.draw(housing, boatDrawName, 1) * float64(len(names)))
	return face, names[min(pick, len(names)-spaceOne)]
}

// bowNumbers paints the boat's number on each bow, on the outside of the
// bulwark or the topsides, a little aft of the stem.
func (craft *boat) bowNumbers(tall float64) {
	if !craft.detailed(boatQuick) {
		return
	}
	hull := craft.hull
	at := boatNumberAt
	along := hull.z(at) - hull.lean(at, hull.sheer(at))
	up := hull.sheer(at) + hull.rail*spaceHalf
	if hull.rail <= 0 {
		up = hull.sheer(at) * boatNumberLow
	}
	for index, facing := range []maths.Orientation{boatStarboardFace,
		boatPortFace} {
		turn := boatSide(index)
		out := turn * (hull.skin(at, up) + boatPlateThick)
		craft.paint(boatRead.Lettering.Face, craft.number(), boatNumberWide,
			tall, maths.V(out, up, along), facing)
	}
}

// How far aft of the stem the number stands, as a share of the length, and
// the width it has. Chosen.
const (
	boatNumberAt   = 0.2
	boatNumberWide = 1.6
	boatNumberLow  = 0.6 // chosen: how high up a hull with no bulwark its number stands, as a share of the deck's edge
)

// sternName paints the boat's name on its transom, across the middle of it.
func (craft *boat) sternName(tall float64) {
	if !craft.detailed(boatQuick) {
		return
	}
	hull := craft.hull
	face, words := craft.name()
	up := hull.sheer(spaceOne) * spaceHalf
	craft.paint(face, words, 2*hull.half(spaceOne)*boatSternFill, tall,
		maths.V(0, up, hull.length*spaceHalf), boatSternFace)
}

// boatSternFill is how much of the transom's width a name may span: chosen,
// so that it stays clear of the corners.
const boatSternFill = 0.9
