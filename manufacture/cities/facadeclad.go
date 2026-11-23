package cities

import (
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// lap is boards laid in courses, each lapping the one below: across a
// wall, a board's face, and at its foot a shadow line where the next
// course over it throws one. It is a function of position in the part's
// own space, as every texture is: the courses run level, and a point on
// a wall facing along x or along z is read by its height alone.
type lap struct {
	// Board is the colour of a board's face, Height how tall a course is
	// in metres, and Shadow the share of its light the shadow line gives
	// back, which takes up Share of a course's height.
	Board         maths.Light
	Height        float64
	Shadow, Share float64
	// Tone is how much a course differs from the next, as a share of its
	// light, by a hash of which course it is.
	Tone float64
}

// At is the board at p: the face, or the shadow line at the foot of its
// course, a little lighter or darker than its neighbour.
func (lap lap) At(p maths.Vec) maths.Light {
	course := math.Floor(p.Y / lap.Height)
	within := p.Y/lap.Height - course
	paint := lap.Board.Scale(1 + lap.Tone*(hashed(course, 0)-evenTone))
	if within < lap.Share {
		return paint.Scale(lap.Shadow)
	}
	return paint
}

// brick is courses of bricks in stretcher bond: each course a half brick
// along from the one below, a mortar joint between every two bricks and
// between courses, and each brick a little lighter or darker than the
// next. A wall facing along x is read by its place along z, and one facing
// along z by its place along x, so a point is read by their sum.
type brick struct {
	Brick, Mortar maths.Light
	// Length and Height are a brick's, Joint the mortar's thickness, all in
	// metres, and Tone how much a brick differs from the next, as a share
	// of its light.
	Length, Height, Joint, Tone float64
}

// At is the brick or the mortar at p.
func (brick brick) At(p maths.Vec) maths.Light {
	row, course, joint := brick.course(p)
	if joint {
		return brick.Mortar
	}
	return brick.Brick.Scale(1 + brick.Tone*(hashed(row, course)-evenTone))
}

// InJoint is whether p is in a joint, which is what a weathering agent
// asks of brick to lodge more of itself there.
func (brick brick) InJoint(p maths.Vec) bool {
	_, _, joint := brick.course(p)
	return joint
}

// course is which row and which brick of it p is in, and whether p is in
// the mortar round it.
func (brick brick) course(p maths.Vec) (row, course float64, joint bool) {
	row = math.Floor(p.Y / (brick.Height + brick.Joint))
	up := p.Y - row*(brick.Height+brick.Joint)
	along := p.X + p.Z
	if math.Mod(row, half) == 1 {
		along += brick.Length / half
	}
	course = math.Floor(along / (brick.Length + brick.Joint))
	across := along - course*(brick.Length+brick.Joint)
	return row, course, up > brick.Height || across > brick.Length
}

// evenTone is the middle of the range hashed gives, so that a brick or a
// board is as often darker than its colour as lighter.
const evenTone = 0.5

// hashed is a number from nought to one that is the same for the same
// pair of whole numbers, which is enough to make bricks and boards differ
// from one another without a seed.
func hashed(first, second float64) float64 {
	mixed := math.Sin(first*hashFirst+second*hashSecond) * hashScale
	return mixed - math.Floor(mixed)
}

// The numbers hashed mixes by: the usual few of a shader's hash, picked
// for their lack of pattern and no more than that.
const (
	hashFirst  = 12.9898
	hashSecond = 78.233
	hashScale  = 43758.5453
)

// claddingOf is the name of the cladding a wall's material was made of: the
// last word of its name, as wall names it.
func claddingOf(base *thing.Material) string {
	words := strings.Fields(base.Name)
	return words[len(words)-1]
}

// clad is a wall's material in the surface of its cladding, if it is one
// the demo level draws other than plaster: timber lap and vinyl siding in
// courses, and brick with its mortar. It is nil for plaster.
func (drawing *drawer) clad(base *thing.Material) *thing.Material {
	surfaced := *base
	switch claddingOf(base) {
	case "timber-lap":
		surfaced.Texture = drawing.lap(base.Colour, "lap")
	case "vinyl-siding":
		surfaced.Texture = drawing.lap(base.Colour, "vinyl")
	case "brick-slips":
		surfaced.Texture = brick{Brick: base.Colour,
			Mortar: maths.RGB(drawing.fixed("mortar-tone"),
				drawing.fixed("mortar-tone"), drawing.fixed("mortar-tone")),
			Length: drawing.fixed("brick-length"),
			Height: drawing.fixed("brick-height"),
			Joint:  drawing.fixed("mortar-joint"),
			Tone:   drawing.fixed("brick-tone")}
	default:
		return nil
	}
	surfaced.Name += " courses"
	return &surfaced
}

// lap is boards of a colour in courses, measured by the culture's data
// under a prefix: how tall a course is, how dark its shadow line and how
// much of it, and how much a course differs from the next.
func (drawing *drawer) lap(colour maths.Light, prefix string) lap {
	return lap{Board: colour, Height: drawing.fixed(prefix + "-height"),
		Shadow: drawing.fixed(prefix + "-shadow"),
		Share:  drawing.fixed(prefix + "-share"),
		Tone:   drawing.fixed(prefix + "-tone")}
}
