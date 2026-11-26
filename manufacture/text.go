package manufacture

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
)

// textAbove is how many ems a line's letters are allowed to stand over
// their baseline, and textBelow how many under it, for the box a line of
// text is asked for before its letters are made: a capital is about seven
// tenths of an em tall and a descender about a quarter under.
const (
	textAbove = 1.0 // picked
	textBelow = 0.5 // picked
)

// errText is the refusal of a line of text with no size, or of letters
// that stand nowhere.
var errText = errors.New("manufacture: a text needs a size and a depth " +
	"above nothing")

// Text is a line of words stood in a world: each letter a thing in the
// text's colour, the line facing a bearing and turning as it says. It is an
// object of the manufactory whose letters are made when it is staged, and
// whose box is told from the font's advance alone.
type Text struct {
	// Name is what the letters' material is called.
	Name string
	// Words is the line, Font its face and Dir the directory a font that
	// is a file is found from.
	Words, Font, Dir string
	// Size is an em, Depth how deep the letters are, Bevel how far their
	// faces are stepped in, all in scene units, Smooth the angle in
	// radians below which two walls are shaded as one, and Tolerance how
	// far a step may stray from its curve, in the font's units.
	Size, Depth, Bevel, Smooth, Tolerance float64
	// Colour is the letters' colour and Gloss how glossy they are.
	Colour maths.Light
	Gloss  float64
	// At is where the line's baseline stands, in the world's frame.
	At maths.Vec
	// Bearing is the line's front's bearing, in degrees clockwise from
	// north, and Spin how many seconds it takes to turn once about up,
	// nought for standing still.
	Bearing, Spin float64
}

// TextTally is what a text's letters came to when staged: the things they
// are, their faces as drawn and, since a letter used twice is one mesh
// standing twice, each mesh's faces once.
type TextTally struct {
	Things, Faces, Kept int
}

// Text is an object.
var _ object = Text{}

// motion is how the line stands and moves: at its place, its front, which
// faces plus z in the letters' own space, turned to its bearing, and
// turning about up once in its spin's seconds.
//
// The scene's plus z is south and plus x east, so the bearing b, clockwise
// from north, is the front turned pi less b about up.
func (text Text) motion() thing.Motion {
	facing := math.Pi - text.Bearing*maths.Degree
	return thing.Spin{In: maths.Subspace{Origin: text.At,
		Orientation: maths.About(maths.Up, facing)}, Axis: maths.Up,
		Period: time.Duration(text.Spin * float64(time.Second))}
}

// extent is the box of the text, from the font's advance alone.
func (text Text) extent(from seed) (maths.Box, error) {
	return text.Extent()
}

// Extent is the box of the line in the world's frame, without making its
// letters: the width the font lays the words out to, the depth, and an em
// and a half of height about the baseline. A line that turns is held to the
// disc of its half width and depth, all the way round.
func (text Text) Extent() (maths.Box, error) {
	if !(text.Size > 0 && text.Depth > 0) {
		return maths.Box{}, errText
	}
	face, err := typography.Named(text.Font, text.Dir)
	if err != nil {
		return maths.Box{}, err
	}
	line, err := typography.Layout(face, text.Words)
	if err != nil {
		return maths.Box{}, err
	}
	half := line.Width / 2 * text.Size / face.Units()
	deep := text.Depth / 2
	local := maths.Box{Min: maths.V(-half, -textBelow*text.Size, -deep),
		Max: maths.V(half, textAbove*text.Size, deep)}
	if text.Spin != 0 {
		round := math.Hypot(half, deep)
		local.Min.X, local.Max.X = -round, round
		local.Min.Z, local.Max.Z = -round, round
	}
	space := maths.Subspace{Origin: text.At, Scale: 1,
		Orientation: maths.About(maths.Up,
			math.Pi-text.Bearing*maths.Degree)}
	return transformed(local, space), nil
}

// seen is how an eye sees the text's box.
func (text Text) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenObject(text, at, by)
}

// Staged makes the line's letters and adds each to a world as a thing in
// the text's colour, in the order of the words, and says what they came to.
func (text Text) Staged(world *thing.World) (TextTally, error) {
	face, err := typography.Named(text.Font, text.Dir)
	if err != nil {
		return TextTally{}, err
	}
	made, err := typography.Make(face, typography.Text{Words: text.Words,
		Size: text.Size, Depth: text.Depth, Bevel: text.Bevel,
		Smooth: text.Smooth, Tolerance: text.Tolerance})
	if err != nil {
		return TextTally{}, err
	}
	material := &thing.Material{Name: text.Name, Colour: text.Colour,
		Gloss: text.Gloss}
	for _, one := range typography.Things(made, material, text.motion()) {
		world.Add(one)
	}
	return tallied(made), nil
}

// tallied counts a text's letters as things, each one's faces as drawn,
// and each mesh's faces once as kept.
func tallied(made []typography.Letter) TextTally {
	var tally TextTally
	kept := map[*mesh.Mesh]bool{}
	for _, letter := range made {
		if letter.Mesh == nil {
			continue
		}
		tally.Things++
		tally.Faces += letter.Mesh.FaceCount()
		if !kept[letter.Mesh] {
			kept[letter.Mesh] = true
			tally.Kept += letter.Mesh.FaceCount()
		}
	}
	return tally
}
