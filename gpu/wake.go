package gpu

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// wakeFields are the words of a wake's card, in order: everything the kernel
// needs to find the gas of medium.Wake at a point, as its Density does. The
// seed is two words, low then high, and the octaves and the path count are
// whole numbers; every other word is a float. Behind these come the paths,
// wakePathFields words each, from the word path_first names.
var wakeFields = []string{
	"reach", "clear", "thick", "seed_low", "seed_high", "scale", "octaves",
	"fade", "swept", "ridge", "bank", "lift", "path_count", "path_first",
}

// wakePathFields are the words of one path in a wake's card: its space's
// origin, the three axes that take a point of the wake into the space, one
// to a row of the rotation as Orientation.In reads them, its scale, and the
// radius of its circle.
var wakePathFields = []string{
	"origin_x", "origin_y", "origin_z", "axis_x_x", "axis_x_y", "axis_x_z",
	"axis_y_x", "axis_y_y", "axis_y_z", "axis_z_x", "axis_z_y", "axis_z_z",
	"scale", "radius",
}

// mostWakePaths is how many paths a wake's card holds: eight, picked, which
// is four times the bands orbital has. A wake of more is refused by name.
const mostWakePaths = 8

// unscaled is the scale of a space that has none, which is one.
const (
	unscaled = 0
	scaleOne = 1
)

// smoothThree and smoothTwo are the numbers of smoothstep's cubic, three
// along squared less two along cubed, true by definition.
const (
	smoothThree = 3
	smoothTwo   = 2
)

// wakeNumbers are the kernel's declarations of a wake's kind, its card's words
// and its paths', and the numbers it is told.
func wakeNumbers() string {
	var text strings.Builder
	fmt.Fprintf(&text, "constexpr constant int wake_kind = %d;\n", wakeKind)
	for at, name := range wakeFields {
		fmt.Fprintf(&text, "constexpr constant int wake_%s = %d;\n", name, at)
	}
	for at, name := range wakePathFields {
		fmt.Fprintf(&text, "constexpr constant int path_%s = %d;\n", name, at)
	}
	fmt.Fprintf(&text, "constexpr constant int path_words = %d;\n",
		len(wakePathFields))
	fmt.Fprintf(&text, "constexpr constant float smooth_three = %d;\n",
		smoothThree)
	fmt.Fprintf(&text, "constexpr constant float smooth_two = %d;\n",
		smoothTwo)
	fmt.Fprintf(&text, "constexpr constant float wake_farthest = %.9g;\n",
		math.MaxFloat32)
	return text.String()
}

// pathsOn writes a wake's paths behind its card's words, each as the
// kernel reads it.
func (card *wordCard) pathsOn(made *medium.Wake) {
	paths := made.Paths()
	card.whole("path_count", uint32(len(paths)))
	card.whole("path_first", uint32(len(card.words)))
	for _, path := range paths {
		space := path.Space()
		scale := space.Scale
		if scale == unscaled {
			scale = scaleOne
		}
		axes := space.Orientation
		card.words = append(card.words, floatWord(space.Origin.X),
			floatWord(space.Origin.Y), floatWord(space.Origin.Z))
		for _, direction := range []maths.Vec{maths.XAxis, maths.YAxis,
			maths.ZAxis} {
			way := axes.Out(direction)
			card.words = append(card.words, floatWord(way.X),
				floatWord(way.Y), floatWord(way.Z))
		}
		card.words = append(card.words, floatWord(scale),
			floatWord(path.Radius()))
	}
}

// floatWord is a number as the float32 word that holds it.
func floatWord(number float64) uint32 { return math.Float32bits(float32(number)) }

// wakeCardOf is a wake's card: every number its density is made from, or
// why the kernel cannot hold it.
func wakeCardOf(made *medium.Wake) ([]uint32, error) {
	if count := len(made.Paths()); count > mostWakePaths {
		return nil, unsupported{fmt.Sprintf("a wake of %d paths, more than "+
			"the %d a card holds", count, mostWakePaths)}
	}
	card := newCard(wakeFields)
	card.set("reach", made.Reach())
	card.set("clear", made.Clear())
	card.set("thick", made.Thick())
	grain := made.Grain()
	card.seed("seed", grain.Seed())
	card.set("scale", grain.Scale())
	card.whole("octaves", uint32(grain.Octaves()))
	card.set("fade", made.Fade())
	card.set("swept", made.Swept())
	width, bank := made.Ridge()
	card.set("ridge", width)
	card.set("bank", bank)
	card.set("lift", made.Lift())
	card.pathsOn(made)
	return card.words, nil
}

// wakeVolume appends one wake: its ball's radius where a puff's is, which the
// kernel's span reads, and where its card begins where a boxed medium's
// cells begin.
func (encoded *scene) wakeVolume(space maths.Subspace, made *medium.Wake,
	words []uint32, material int) {
	first := encoded.boxes.words(words)
	given := placedAt(space)
	given["puff"] = []float64{made.Reach()}
	given["baked"], given["low"], given["high"] = nil, nil, nil
	given["info"] = wholes(0, 0, wakeKind, material)
	given["grid"], given["shone"] = wholes(0, 0, 0, first), wholes(-1, -1)
	given["lighting"] = wholes(0, 0, 0, -1)
	given["tint"] = []float64{1, 1, 1}
	noAir(given)
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
}

// wakeIf appends the volume if its medium is a wake the kernel asks itself,
// and says so; a gas wake is a gas, with its optics beside. A wake whose
// gas is painted or shines by a texture is not asked, for the kernel's
// density has no paint, and is boxed with its paint like any medium; one
// of more paths than the card holds is refused.
func (encoded *scene) wakeIf(space maths.Subspace, med any,
	textures painting, material int, gassy bool, optics airing) (bool,
	error) {
	made, ok := med.(*medium.Wake)
	if !ok || made == nil || textures != (painting{}) {
		return false, nil
	}
	words, err := wakeCardOf(made)
	if err != nil {
		return true, err
	}
	encoded.wakeVolume(space, made, words, material)
	encoded.airIf(gassy, optics)
	return true, nil
}
