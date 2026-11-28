package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// mostVolumes bounds how many volumes a scene may carry. The kernel's
// crossing arrays are sized to the scene when its source is built, so
// a scene pays thread memory only for the volumes it has: about 40
// bytes a volume a thread, which at this ceiling is ten kilobytes,
// where a dispatch still runs but occupancy is paying for it. The
// ceiling refuses a pathological pack with a plain error instead.
const mostVolumes = 256

// the kinds of volume the kernel reads: a puff, a wispy puff, a shell, a
// ring and a wake, which it asks itself, and a boxed medium, which the cpu
// asks into a grid
const (
	puffKind = iota
	wispyKind
	boxedKind
	shellKind
	ringKind
	wakeKind
	propellerKind
	fizzKind
)

// the shapes a boxed medium's density is asked in by the kernel itself,
// in its info's x, in place of its grid: a giant's clouds are 1 and a
// plain disc 2, written where they are made; a body's air and its clouds
// are these
const (
	boxedAir    = 3
	boxedClouds = 4
)

// shelled is whether a medium is a shell round a body: a giant's clouds,
// a body's air or its clouds, each far thinner than its box.
func shelled(med medium.Medium) bool {
	switch med.(type) {
	case medium.Giant, medium.Air, medium.Clouds:
		return true
	}
	return false
}

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
// asks into a grid over its box. a gas takes the same paths by its shape, the
// medium a gas.Of wraps, and carries its optics beside.
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
		if err := encoded.volumeAt(volume, at); err != nil {
			return err
		}
	}
	encoded.boxVolumes()
	return nil
}

// volumeAt appends one seen volume, placed at time t, as volumesOf does: a
// ring or a wake the kernel asks itself, or the shapes of plainVolume.
func (encoded *scene) volumeAt(volume *thing.Volume, at time.Duration) error {
	if volume.Beam != nil && !encoded.gravity {
		return unsupported{"a volume with a beam"}
	}
	if volume.Beam != nil && !volume.Beam.Round && volume.Beam.Speed > 0 && volume.Beam.Speed < 1 {
		return unsupported{"an axial beam in a field"}
	}
	first := len(encoded.volumes.bytes)
	var cloudPhase float64
	var explicitCloudPhase bool
	defer func() {
		if len(encoded.volumes.bytes) > first {
			encoded.volumeBeam(first, volume.Beam)
			if explicitCloudPhase {
				encoded.cloudPhaseLast(cloudPhase)
			}
			encoded.glowLast(volume)
		}
	}()
	med := volume.Medium
	clearCloud := false
	if shell, ok := med.(*medium.ShellWeather); ok && shell == nil {
		return unsupported{"a nil spherical weather shell"}
	}
	if moving, ok := med.(medium.Timed); ok {
		med = moving.Tick(at)
	}
	if cloud, ok := med.(medium.ClearCloud); ok {
		if encoded.oceanMoon == nil || !encoded.oceanMoon.MixedCloud {
			return unsupported{"clear-cloud transport outside the opted-in finite lunar route"}
		}
		cloudPhase, explicitCloudPhase = cloud.PhaseAsymmetry()
		if explicitCloudPhase {
			if err := metalCloudAsymmetry(cloudPhase); err != nil {
				return err
			}
		}
		med = cloud.DensityField()
		clearCloud = true
	}
	material := encoded.material(volume.Material)
	textures := encoded.paints[material]
	if fizz, ok := med.(medium.Fizz); ok && clearCloud {
		if textures != (painting{}) {
			return unsupported{"a spatially painted clear-cloud Fizz"}
		}
		return encoded.fizzVolume(volume.At(at), fizz, material)
	}
	air, gassy := med.(gas.Optical)
	var optics airing
	if gassy {
		if encoded.gravity {
			return unsupported{"a gas volume in a gravity field"}
		}
		found, err := airingOf(air)
		if err != nil {
			return err
		}
		optics, med, textures = *found, shapeOf(air), painting{}
	}
	if asked, err := encoded.ringIf(volume.At(at), med, textures, material,
		gassy, optics); err != nil || asked {
		return err
	}
	if asked, err := encoded.wakeIf(volume.At(at), med, textures, material,
		gassy, optics); err != nil || asked {
		return err
	}
	return encoded.plainVolume(volume.At(at), med, textures, material, gassy,
		optics)
}

// plainVolume appends a volume that is neither a ring nor a wake: a spherical
// weather shell, a puff or wispy medium the kernel asks itself, or any other
// medium as a grid over its box.
func (encoded *scene) plainVolume(space maths.Subspace, med medium.Medium,
	textures painting, material int, gassy bool, optics airing) error {
	if field, ok := med.(medium.PropellerCavitation); ok && textures == (painting{}) {
		if err := encoded.propellerVolume(space, field, material); err != nil {
			return err
		}
		encoded.airIf(gassy, optics)
		return nil
	}
	if shell, ok := med.(*medium.ShellWeather); ok {
		if textures != (painting{}) || shell.Parameters().Radius <= 0 {
			return unsupported{"an invalid or textured spherical weather shell"}
		}
		encoded.shellVolume(space, shell.Parameters(), material)
		encoded.airIf(gassy, optics)
		return nil
	}
	found, err := hazeOf(med)
	if err == nil && textures == (painting{}) {
		encoded.volume(space, found, material)
		encoded.airIf(gassy, optics)
		return nil
	}
	grid := boxOf(med, textures.paint, textures.shine, encoded.volumeCells)
	smoking, reached := len(encoded.smokes), len(encoded.reached)
	encoded.boxedVolume(space, grid, material)
	if gassy {
		// a gas is lit by its lamps at every step, never from a
		// cached source grid or a lamp store
		encoded.smokes = encoded.smokes[:smoking]
		encoded.reached = encoded.reached[:reached]
	}
	if shelled(med) {
		// a giant's lit clouds, a body's air and its clouds are shells
		// far thinner than a cell of a grid over their whole box, so the
		// store's cells would show as fine ripples; each is lit at every
		// step, as a gas is
		encoded.reached = encoded.reached[:reached]
	}
	encoded.airIf(gassy, optics)
	return nil
}

// airIf makes the volume appended last a gas, when it is one.
func (encoded *scene) airIf(gassy bool, optics airing) {
	if gassy {
		encoded.airLast(optics)
	}
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
	intField("lighting", "cached source RGB dimensions and first number; -1 for direct"),
	intField("grid", "a boxed medium's cells across x, y, z; first cell"),
	intField("shone", "its shine's first number, its paint's; or -1"),
	floatField("tint", "how many times its density each channel meets; "+
		"a gas's extinction at the eye's three bands"),
	floatField("scatter", "a gas's scatter at the eye's three bands, "+
		"its phase asymmetry in w; a lopsided mist's asymmetry in w"),
	intField("air", "one in x for a gas, nought for the rest; "+
		"one in z for the air the world's glow lights"),
	floatField("beam", "axis and speed"),
	floatField("flow", "round and reference radius"),
	intField("thermal", "radial light table, count, seed low, seed high"),
	floatField("thin", "inner, outer, streaks, shear"),
	floatField("phase", "orbital coefficient times seconds"),
	floatField("world_low", "the low corner of a box in the world "+
		"that the volume lies in, and one in w where it has a box; "+
		"nought for none (volume_reach.go)"),
	floatField("world_high", "the high corner of that box"),
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
	given["lighting"] = wholes(0, 0, 0, -1)
	given["tint"] = []float64{1, 1, 1}
	noAir(given)
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
}

// boxedVolume appends one boxed medium, with a reserved or CPU-sampled grid,
// its cells after every other boxed medium's.
func (encoded *scene) boxedVolume(space maths.Subspace, grid boxed,
	material int) {
	thin, radial := grid.shine.(texture.Thin)
	if radial {
		grid.shine = nil
	}
	first, shone, deferred := encoded.deferFlame(grid)
	painted := -1
	if !deferred {
		started := time.Now()
		density, shine, paint := grid.asked()
		encoded.boxedCPU.Wall += time.Since(started)
		encoded.boxedCPU.Cells += int64(grid.count())
		encoded.boxedCPU.Volumes++
		first = encoded.boxes.floats(density)
		shone, painted = encoded.boxes.floats(shine), encoded.boxes.floats(paint)
	}
	tint := maths.Grey(1)
	if thick, ok := grid.med.(tinted); ok {
		tint = thick.Tint()
	}
	given := placedAt(space)
	given["puff"], given["baked"] = nil, nil
	given["info"] = wholes(0, 0, boxedKind, material)
	// A giant's thin cloud boundary stays analytic at every grid size.
	// Its painted bands still use the shared texture grid.
	if giant, ok := grid.med.(medium.Giant); ok {
		given["puff"] = []float64{giant.Radius, giant.Depth, giant.Thick, 0}
		given["info"] = wholes(1, 0, boxedKind, material)
	}
	if disc, ok := grid.med.(medium.Disc); ok && len(disc.Profile) == 0 && disc.Grain == 0 {
		given["puff"] = []float64{disc.Inner, disc.Outer, disc.Thick, disc.Dense}
		given["info"] = wholes(2, 0, boxedKind, material)
	}
	// A body's air and clouds are shells tens of kilometres thick in a box
	// thousands across: a grid's cells showed in them as rings and cubes,
	// so the kernel asks their density as the medium does.
	if air, ok := grid.med.(medium.Air); ok {
		given["puff"] = []float64{air.Radius, air.Height, air.Thick,
			air.Bounds().Max.X}
		given["info"] = wholes(boxedAir, 0, boxedKind, material)
	}
	clouds, clouded := grid.med.(medium.Clouds)
	if clouded {
		given["puff"] = []float64{clouds.Radius, clouds.Low, clouds.High,
			clouds.Thick}
		given["info"] = wholes(boxedClouds, clouds.Patches().Octaves,
			boxedKind, material)
		// the weather's table, its belts, winds and storms, is read from
		// the boxes where a grid's cells would be
		table := clouds.Weather().Table()
		narrow := make([]float32, len(table))
		for index, number := range table {
			narrow[index] = float32(number)
		}
		first = encoded.boxes.floats(narrow)
	}
	given["low"], given["high"] = point(grid.low), point(grid.high)
	given["grid"] = append(wholes(grid.cells[:]...), float64(first))
	given["shone"] = wholes(shone, painted)
	given["lighting"] = wholes(0, 0, 0, -1)
	given["tint"] = []float64{tint.R, tint.G, tint.B}
	noAir(given)
	if clouded {
		// clouds have no thermal table and no thin texture: their seed
		// is where a table's seed would be, behind a count of none, their
		// cover where thin's inner would be, and their patches' numbers
		// in phase
		patches := clouds.Patches()
		// the billow's octaves and soft-pattern flag are in thermal;
		// its amount, frequency and the patches' reach are in thin
		soft := 0
		if clouds.Soft {
			soft = 1
		}
		given["thermal"] = wholes(patches.DetailOctaves, soft,
			int(uint32(clouds.Seed)), int(uint32(clouds.Seed>>32)))
		given["thin"] = []float64{clouds.Cover, patches.Billow,
			patches.Detail, patches.Reach}
		given["phase"] = []float64{patches.Frequency, patches.Edge,
			patches.Whole, patches.Top}
	}
	if radial {
		const samples = 1024
		base := thin
		base.Streaks = 0
		table := make([]float32, 0, samples*3)
		for i := range samples {
			r := thin.Inner + (thin.Outer-thin.Inner)*float64(i)/float64(samples-1)
			light := base.At(maths.V(r, 0, 0))
			table = append(table, float32(light.R), float32(light.G), float32(light.B))
		}
		first := encoded.boxes.floats(table)
		given["thermal"] = wholes(first, samples, int(uint32(thin.Seed)), int(uint32(thin.Seed>>32)))
		given["thin"] = []float64{thin.Inner, thin.Outer, thin.Streaks, thin.Shear}
		given["phase"] = []float64{thin.Omega * thin.Time}
	}
	if _, ok := grid.med.(*plume.Column); ok && grid.paint == nil && grid.shine == nil {
		encoded.smokes = append(encoded.smokes, smokeGrid{index: encoded.counts.volumes, grid: grid})
	}
	if !deferred {
		// a flame's deferred box sends its own light and scatters none
		encoded.reached = append(encoded.reached,
			reachGrid{index: encoded.counts.volumes, grid: grid})
	}

	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
	encoded.lobeLast(grid.med)
}

// lobedMedium is a medium that scatters more one way than another, by a
// Henyey-Greenstein asymmetry: a mist that says one.
type lobedMedium interface{ Asymmetry() float64 }

// lobeLast writes a lopsided medium's asymmetry into the scatter's w of
// the volume appended last, where a gas keeps its own; the kernel's
// scattered_direct weighs each lamp by gas_phase at it. A medium that
// scatters evenly, or does not say, leaves the row as it was.
func (encoded *scene) lobeLast(made medium.Medium) {
	said, ok := made.(lobedMedium)
	if !ok || said.Asymmetry() == 0 {
		return
	}
	encoded.putLast("scatter", floatWords(0, 0, 0, said.Asymmetry()))
}

// volumeBeam writes flow metadata into the volume just appended.
func (encoded *scene) volumeBeam(first int, beam *thing.Beam) {
	if beam == nil {
		return
	}
	axis := beam.Axis
	if axis == (maths.Vec{}) {
		axis = maths.Up
	}
	axis = axis.Unit()
	for i, member := range volumeStruct.fields {
		var parts []float64
		switch member.name {
		case "beam":
			parts = []float64{axis.X, axis.Y, axis.Z, beam.Speed}
		case "flow":
			parts = []float64{float64(flag(beam.Round)), beam.At}
		default:
			continue
		}
		for j, value := range parts {
			binary.LittleEndian.PutUint32(encoded.volumes.bytes[first+i*16+j*4:], math.Float32bits(float32(value)))
		}
	}
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

// VolumeLimit is the number of participating media a GPU ray can cross.
func VolumeLimit() int { return mostVolumes }
