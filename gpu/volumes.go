package gpu

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// mostVolumes is how many volumes a ray may cross in the kernel, which
// keeps its crossings in a fixed array: a fire's smoke and flame for
// each of a clearing's fires, picked
const mostVolumes = 32

// the kinds of volume the kernel reads: a puff, a wispy puff, which it
// asks itself, and a boxed medium, which the cpu asks into a grid
const (
	puffKind = iota
	wispyKind
	boxedKind
)

// tinted is a medium thicker to some light than to other, as soot is to
// blue: trace's Tinted, how many times its density each channel meets.
type tinted interface {
	Tint() maths.Light
}

// haze is a medium the kernel knows: a puff, its noise if it is wispy,
// and how much of the puff the noise may take away.
type haze struct {
	puff          medium.Puff
	wispy         bool
	seed          uint64
	scale, amount float64
}

// hazeOf is a medium as the kernel knows it, or why it cannot be.
func hazeOf(made medium.Medium) (haze, error) {
	switch kind := made.(type) {
	case medium.Puff:
		return haze{puff: kind}, nil
	case medium.Wispy:
		puff, ok := kind.Of.(medium.Puff)
		if !ok {
			return haze{}, unsupported{fmt.Sprintf(
				"a wispy medium over %T", kind.Of)}
		}
		return haze{puff: puff, wispy: true, seed: kind.Seed,
			scale: kind.Scale, amount: kind.Amount}, nil
	}
	return haze{}, unsupported{fmt.Sprintf("a medium of kind %T", made)}
}

// volumesOf appends every seen volume, placed at time t: where its
// space is, the axes that take the world into it, its medium, and what
// it is made of.
//
// a puff, plain or wispy, with no paint or shine, the kernel asks
// itself; any other medium, and any volume painted or shining, the cpu
// asks into a grid over its box. a gas is refused.
func (encoded *scene) volumesOf(world *thing.World,
	at time.Duration) error {
	if len(world.Volumes) > mostVolumes {
		return unsupported{fmt.Sprintf("more than %d volumes",
			mostVolumes)}
	}
	for _, volume := range world.Volumes {
		if volume.Hidden() || volume.Medium == nil {
			continue
		}
		if volume.Beam != nil {
			return unsupported{"a volume with a beam"}
		}
		med := volume.Medium
		if moving, ok := med.(medium.Timed); ok {
			med = moving.Tick(at)
		}
		if _, gassy := med.(gas.Optical); gassy {
			return unsupported{fmt.Sprintf("a gas volume, %T", med)}
		}
		material := encoded.material(volume.Material)
		textures := encoded.paints[material]
		found, err := hazeOf(med)
		if err == nil && textures == (painting{}) {
			encoded.volume(volume.At(at), found, material)
			continue
		}
		grid := boxOf(med, textures.paint, textures.shine)
		encoded.boxedVolume(volume.At(at), grid, material)
	}
	return nil
}

// volumeStruct is a volume as the kernel reads it: where it is, its
// kind, a puff's own numbers, and a boxed medium's box, grid and tint.
var volumeStruct = structure{"Volume", []field{
	floatField("origin", "scale in w"),
	floatField("x", "the volume's axes in the world"),
	floatField("y", ""),
	floatField("z", ""),
	floatField("puff", "radius, thick, noise scale, amount"),
	intField("info", "seed low, seed high, kind, material"),
	intField("baked", "first cell, cells a side; none at nought"),
	floatField("low", "a boxed medium's box, in its own space"),
	floatField("high", ""),
	intField("grid", "a boxed medium's cells across x, y, z; first cell"),
	intField("shone", "its shine's first number, its paint's; or -1"),
	floatField("tint", "how many times its density each channel meets"),
}}

// volume appends one puff. The axes are the rows that take a world
// direction into the volume's own: a direction's part along each of
// the volume's axes.
func (encoded *scene) volume(space maths.Subspace, found haze,
	material int) {
	seed := wholes(int(uint32(found.seed)), int(uint32(found.seed>>32)))
	first, side := encoded.bakeIfWispy(found)
	kind := puffKind
	if found.wispy {
		kind = wispyKind
	}
	given := placedAt(space)
	given["puff"] = []float64{found.puff.Radius, found.puff.Thick,
		found.scale, found.amount}
	given["info"] = append(seed, wholes(kind, material)...)
	given["baked"] = wholes(first, side)
	given["low"], given["high"] = nil, nil
	given["grid"], given["shone"] = nil, wholes(-1, -1)
	given["tint"] = []float64{1, 1, 1}
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
}

// boxedVolume appends one boxed medium, its grid asked on the cpu and
// its cells after every other boxed medium's.
func (encoded *scene) boxedVolume(space maths.Subspace, grid boxed,
	material int) {
	density, shine, paint := grid.asked()
	first := encoded.boxes.floats(density)
	shone, painted := encoded.boxes.floats(shine),
		encoded.boxes.floats(paint)
	tint := maths.Grey(1)
	if thick, ok := grid.med.(tinted); ok {
		tint = thick.Tint()
	}
	given := placedAt(space)
	given["puff"], given["baked"] = nil, nil
	given["info"] = wholes(0, 0, boxedKind, material)
	given["low"], given["high"] = point(grid.low), point(grid.high)
	given["grid"] = append(wholes(grid.cells[:]...), float64(first))
	given["shone"] = wholes(shone, painted)
	given["tint"] = []float64{tint.R, tint.G, tint.B}
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
}

// placedAt is where a volume is: its space's origin and scale, and the
// axes that take a world direction into it.
func placedAt(space maths.Subspace) values {
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	axes := space.Orientation
	return values{"origin": append(point(space.Origin), scale),
		"x": point(axes.Out(maths.XAxis)),
		"y": point(axes.Out(maths.YAxis)),
		"z": point(axes.Out(maths.ZAxis))}
}

// floats appends numbers to a buffer of floats and says where the first
// went, by its place among the floats; minus one for none.
func (buffer *packer) floats(numbers []float32) int {
	if len(numbers) == 0 {
		return -1
	}
	first := len(buffer.bytes) / 4
	for _, number := range numbers {
		buffer.word(math.Float32bits(number))
	}
	return first
}

// bakeSide is how many cells a baked grid has along each side of the cube round
// its puff: 160, so a puff of radius 14, the swarm's, has cells 0.175 across,
// and its finest wisp, about 0.3, spans two.
//
// zero bakes nothing, and asks the noise as the cpu does: the exact path, which
// the tests hold to the tight tolerance.
var bakeSide = 160

// bake is one volume to be baked: which, and where its cells begin in
// the one buffer of every baked cell
type bake struct{ index, first int }

// bakeIfWispy marks a wispy volume to be baked, and says where its
// cells will be and how many a side; a plain puff is cheap to ask, and
// is not baked.
func (encoded *scene) bakeIfWispy(found haze) (int, int) {
	if !found.wispy || bakeSide == 0 {
		return 0, 0
	}
	first := encoded.cells
	encoded.bakes = append(encoded.bakes,
		bake{index: encoded.counts.volumes, first: first})
	encoded.cells += bakeSide * bakeSide * bakeSide
	return first, bakeSide
}
