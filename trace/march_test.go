package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// slab is a medium of one density everywhere inside a box and zero outside it,
// so the density added up along any ray is exact however the steps fall.
type slab struct {
	density float64
	box     maths.Box
}

// Density is the slab's one density, inside its box.
func (block slab) Density(point maths.Vec) float64 {
	inside := point.X >= block.box.Min.X && point.X <= block.box.Max.X &&
		point.Y >= block.box.Min.Y && point.Y <= block.box.Max.Y &&
		point.Z >= block.box.Min.Z && point.Z <= block.box.Max.Z
	if !inside {
		return 0
	}
	return block.density
}

// Bounds is the slab's box.
func (block slab) Bounds() maths.Box { return block.box }

// plate is a slab wide and tall, and deep along z from near to far.
func plate(density, near, far float64) slab {
	return slab{density: density, box: maths.Box{
		Min: maths.V(-10, -10, near), Max: maths.V(10, 10, far)}}
}

// alongZ is a ray from well in front of every plate, down plus z.
var alongZ = maths.Ray{At: maths.V(0, 0, -50), Dir: maths.V(0, 0, 1)}

// staging is a world of volumes with no lamp, staged at zero.
func staging(volumes ...thing.Volume) staged {
	world := thing.NewWorld()
	for _, each := range volumes {
		world.AddVolume(each)
	}
	return stage(world, 0)
}

// Toward a lamp, a painted volume lets through e to the minus its
// density times the way through it: beer's law, to twelve places. A
// volume three times the size has the same depth through its middle,
// since density is per unit of the medium's own space.
func TestClearIsBeersLaw(t *testing.T) {
	const density, deep = 0.7, 2.0
	want := math.Exp(-density * deep)
	painted := thing.Volume{Medium: plate(density, -1, 1)}
	volumes := staging(painted).volumes
	got := paintedClear(volumes, alongZ.At, alongZ.Dir, 100).R
	if !within(got, want, 1e-12) {
		t.Errorf("through the plate %v, want %v", got, want)
	}
	painted.Motion = thing.Still(maths.Subspace{Scale: 3})
	got = paintedClear(staging(painted).volumes, alongZ.At, alongZ.Dir,
		100).R
	if !within(got, want, 1e-12) {
		t.Errorf("at scale three it passes %v, want %v", got, want)
	}
}

// Toward a lamp, a gas lets through of each channel e to the minus the
// gas's own extinction at that channel's band, times the density, times
// the way through.
func TestGasClearIsBeersLawByChannel(t *testing.T) {
	const density, deep = 2e4, 2.0
	air := gas.Air()
	gassy := thing.Volume{Medium: gas.Of{Shape: plate(density, -1, 1),
		In: air, Unit: 1}}
	got := gasClear(staging(gassy).volumes, alongZ.At, alongZ.Dir, 100)
	want := byEye(func(band maths.Band) float64 {
		return math.Exp(-air.Extinction(band) * density * deep)
	})
	if !(want.B < want.R && want.R < 1) {
		t.Fatalf("the test's own gas does nothing to see: %v", want)
	}
	if !within(got.R, want.R, 1e-12) || !within(got.G, want.G, 1e-12) ||
		!within(got.B, want.B, 1e-12) {
		t.Errorf("through the gas %v, want %v", got, want)
	}
}

// Along the eye's ray a painted volume keeps e to the minus density
// times depth of what is behind it and adds its glow times what it
// took, however many steps the crossing is cut into.
func TestThroughIsBeersLawAtAnyStride(t *testing.T) {
	const density, deep, glow = 0.7, 2.0, 0.6
	glowing := thing.Volume{Medium: plate(density, -1, 1),
		Material: &thing.Material{Glow: maths.Grey(glow)}}
	big := glowing
	big.Motion = thing.Still(maths.Subspace{Scale: 3})
	kept := math.Exp(-density * deep)
	for _, cuts := range []int{4, 48, 200} {
		check(t, staging(glowing), cuts, kept, glow)
		check(t, staging(big), cuts, kept, glow)
	}
}

// check holds one painted scene's keep and glow along alongZ to their
// exact answers, with each crossing cut into so many steps.
func check(t *testing.T, scene staged, cuts int, kept, glow float64) {
	t.Helper()
	got := throughGain(scene, alongZ, math.Inf(1), nil, cuts)
	keep, added := got.keep, got.added
	if !within(keep.R, kept, 1e-9) || keep.G != keep.R ||
		keep.B != keep.R {
		t.Errorf("in %d steps the plate keeps %v, want %v", cuts, keep,
			kept)
	}
	if want := glow * (1 - kept); !within(added.R, want, 1e-9) {
		t.Errorf("in %d steps the plate adds %v, want %v", cuts,
			added.R, want)
	}
}

// Two volumes that overlap keep the product of what each keeps, in
// whichever order they were added to the world.
func TestOverlappingVolumesMultiply(t *testing.T) {
	thin := thing.Volume{Medium: plate(0.3, -2, 1)}
	thick := thing.Volume{Medium: plate(0.9, -1, 3)}
	want := math.Exp(-(0.3*3 + 0.9*4))
	for _, scene := range []staged{staging(thin, thick),
		staging(thick, thin)} {
		keep := through(scene, alongZ, math.Inf(1)).keep
		if !within(keep.R, want, 1e-9) {
			t.Errorf("two plates keep %v, want %v", keep.R, want)
		}
	}
}

// A march stops early once less than faint of what is behind still
// shows. Behind a slab ten deep a glow of one is e to the minus ten,
// and the early stop moves the answer by less than two thousandths.
func TestTheEarlyStopCostsUnderTwoThousandths(t *testing.T) {
	world := glowing()
	over := slab{density: 5, box: maths.Box{
		Min: maths.V(-10, 1, -10), Max: maths.V(10, 3, 10)}}
	world.AddVolume(thing.Volume{Medium: over, Material: unlit})
	got := float64(lone(world, straightDown(50), 0).R)
	if want := math.Exp(-10); !within(got, want, 2e-3) || !(got > 0) {
		t.Errorf("behind the slab the glow is %v, want %v", got, want)
	}
}

// Crossings come nearest first, and those that begin at the same
// distance stay in the order their volumes were added, which is what a
// tie between steps is settled by. A hundred plates beginning at four
// distances in turn: a sort that does not keep order shuffles a list
// that long, where a short one it leaves alone.
func TestCrossingsThatTieKeepTheirOrder(t *testing.T) {
	const count, distances = 100, 4
	var plates []thing.Volume
	for i := range count {
		near := float64((count - i) % distances)
		plates = append(plates, thing.Volume{
			Medium: plate(0.1, near, near+10)})
	}
	scene := staging(plates...)
	added := map[*thing.Volume]int{}
	for i, each := range scene.volumes {
		added[each.source] = i
	}
	found := crossings(scene.volumes, alongZ, math.Inf(1))
	if len(found) != count {
		t.Fatalf("%d crossings of %d plates", len(found), count)
	}
	for i := 1; i < len(found); i++ {
		before, here := found[i-1], found[i]
		earlier := added[before.volume.source]
		second := added[here.volume.source]
		tied := before.enter == here.enter
		if before.enter < here.enter || (tied && earlier < second) {
			continue
		}
		t.Fatalf("crossing %d, of the volume added %dth, comes "+
			"before the one added %dth, at distances %v and %v",
			i-1, earlier, second, before.enter, here.enter)
	}
}

// boxOnly is a medium with its own Span hidden, so only its box says
// where a ray crosses it.
type boxOnly struct{ medium.Medium }

// A ray's extent in a volume is the medium's own span where it has
// one and its box's where it has none. Through the middle of a ball
// along an axis the two agree: in at one radius short of the centre,
// out at one past it. A ray that misses has no extent, and neither has
// one stopped short of the volume.
func TestAVolumesSpan(t *testing.T) {
	ball := medium.Puff{Radius: 1, Thick: 1}
	spanner := staging(thing.Volume{Medium: ball}).volumes[0]
	boxed := staging(thing.Volume{Medium: boxOnly{ball}}).volumes[0]
	thinned := maths.Ray{At: maths.V(0, 0, -5), Dir: maths.V(0, 0, 1)}
	for name, each := range map[string]placedVolume{"its own span": spanner,
		"its box": boxed} {
		inside, crossed := each.span(thinned, 100)
		if !crossed || !within(inside.enter, 4, 1e-12) ||
			!within(inside.leave, 6, 1e-12) {
			t.Errorf("by %s the ray is inside from %v to %v, %v",
				name, inside.enter, inside.leave, crossed)
		}
		if _, crossed := each.span(thinned, 3); crossed {
			t.Errorf("by %s a ray stopped short crossed", name)
		}
		wide := maths.Ray{At: maths.V(3, 0, -5), Dir: maths.V(0, 0, 1)}
		if _, crossed := each.span(wide, 100); crossed {
			t.Errorf("by %s a ray three out still crossed", name)
		}
	}
	// a ray that only touches has no extent either: one along the
	// ball's side, and one across the box's edge, in at the same
	// distance it is out
	grazing := maths.Ray{At: maths.V(1, 0, -5), Dir: maths.V(0, 0, 1)}
	if inside, crossed := spanner.span(grazing, 100); crossed {
		t.Errorf("a ray touching the ball has the extent %+v", inside)
	}
	edge := maths.Ray{At: maths.V(2, 0, 0),
		Dir: maths.V(-1, 0, 1).Unit()}
	if inside, crossed := boxed.span(edge, 100); crossed {
		t.Errorf("a ray touching the box's edge has the extent %+v",
			inside)
	}
	// the ball's own span is closer than its box: a ray through the
	// box's corner misses the ball
	corner := maths.Ray{At: maths.V(0.9, 0.9, -5), Dir: maths.V(0, 0, 1)}
	_, byBall := spanner.span(corner, 100)
	_, byBox := boxed.span(corner, 100)
	if byBall || !byBox {
		t.Errorf("through the box's corner: ball %v, box %v", byBall,
			byBox)
	}
}

// A march toward a lamp is staggered by a noise that turns over twice
// in a step, and a step is measured in the medium's own units. So a
// volume three times the size has the same stagger at a point three
// times as far out: the pattern grows with the volume, and is not laid
// over the world at one size whatever the volume's.
func TestTheStaggerGrowsWithTheVolume(t *testing.T) {
	ball := thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 1}}
	plain := staging(ball).volumes[0]
	ball.Motion = thing.Still(maths.Subspace{Scale: 3})
	tripled := staging(ball).volumes[0]
	const step = 0.2
	differs := false
	for i := range 20 {
		spot := maths.V(0.31*float64(i), 0.17, -0.4*float64(i))
		small, big := plain.stagger(spot, step),
			tripled.stagger(spot.Scale(3), step)
		if !within(big, small, 1e-12) {
			t.Fatalf("at %v the stagger is %v, and %v at three "+
				"times the size and distance", spot, small, big)
		}
		differs = differs || small != plain.stagger(maths.Vec{}, step)
	}
	if !differs {
		t.Error("the stagger is the same everywhere: nothing was asked")
	}
}

// What gets from a lamp to a point is nothing past a wall, beer's law
// through a painted slab, and beer's law by channel through a gas.
func TestTransmitIsAWallASlabOrAGas(t *testing.T) {
	const density, deep = 0.7, 2.0
	painted := thing.Volume{Medium: plate(density, -1, 1)}
	air := gas.Air()
	gassy := thing.Volume{Medium: gas.Of{Shape: plate(2e4, -1, 1),
		In: air, Unit: 1}}
	from, toward := alongZ.At, alongZ.Dir

	walled := thing.NewWorld()
	walled.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	if got := stage(walled, 0).transmit(from, toward, 100); got !=
		(maths.Light{}) {
		t.Errorf("past a wall %v gets through", got)
	}
	want := math.Exp(-density * deep)
	got := staging(painted).transmit(from, toward, 100)
	if !within(got.R, want, 1e-12) || got.G != got.R || got.B != got.R {
		t.Errorf("through the slab %v, want %v", got, want)
	}
	got = staging(painted, gassy).transmit(from, toward, 100)
	for k, band := range eyeBands {
		thinned := want * math.Exp(-air.Extinction(band)*2e4*deep)
		channel := [3]float64{got.R, got.G, got.B}[k]
		if !within(channel, thinned, 1e-12) {
			t.Errorf("through the slab and the gas at %v nm: %v, "+
				"want %v", band.Nanometres, channel, thinned)
		}
	}
}

// The lamplight reaching a point, as one number, is the luminance of
// each lamp's light after what is in the way has taken its share of
// each channel. Under a red sun behind a gas that takes more blue
// than red, that is more than the sun's luminance times the gas's:
// the gas takes little of what a red sun has.
func TestShoneIsTheLuminanceOfWhatGetsThrough(t *testing.T) {
	air := gas.Air()
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: gas.Of{
		Shape: plate(2e4, -1, 1), In: air, Unit: 1}})
	red := maths.RGB(1, 0.1, 0.1)
	world.Lamp(thing.Sun(maths.V(0, 0, 1), red))
	scene := stage(world, 0)
	from := alongZ.At
	passed := scene.transmit(from, maths.V(0, 0, 1), math.Inf(1))
	want := red.Times(passed).Luminance()
	apart := red.Luminance() * passed.Luminance()
	if within(want, apart, 1e-3*want) {
		t.Fatalf("the test's scene cannot tell the two apart: %v, %v",
			want, apart)
	}
	if got := shone(scene, from, maths.V(0, 0, -1)); !within(got, want,
		1e-12) {
		t.Errorf("the lamplight reaching the point is %v, want %v; "+
			"luminance times luminance is %v", got, want, apart)
	}
}

// stridesAcross is every stride a ray takes across some volumes, each
// crossing cut into so many steps.
func stridesAcross(cuts int, volumes ...thing.Volume) []stride {
	found := crossings(staging(volumes...).volumes, alongZ, math.Inf(1))
	walk := stepping(found, cuts)
	var all []stride
	for {
		step, more := walk.next()
		if !more {
			return all
		}
		all = append(all, step)
	}
}

// A ray's strides across two volumes that overlap come nearest middle
// first, the two volumes' steps taken in turn, and where two middles
// tie the stride of the volume entered first comes first. Each says
// how thick it is: the density there times its length.
func TestStridesComeNearestFirst(t *testing.T) {
	near := thing.Volume{Medium: plate(0.5, 0, 2)}
	far := thing.Volume{Medium: plate(0.25, 1, 3)}
	got := stridesAcross(4, near, far)
	if len(got) != 8 {
		t.Fatalf("%d strides of two crossings cut in four", len(got))
	}
	// each step is half a unit long. The middles are 0.25, 0.75, 1.25
	// and 1.75 in the near plate and 1.25, 1.75, 2.25 and 2.75 in the
	// far one, past the eye's fifty
	order := []int{0, 0, 0, 1, 0, 1, 1, 1}
	for i, step := range got {
		thick := []float64{0.5, 0.25}[step.crossing] * 0.5
		right := step.crossing == order[i] &&
			within(step.length, 0.5, 1e-12) &&
			within(step.thick, thick, 1e-12)
		if !right {
			t.Errorf("stride %d: crossing %d, %v long, %v thick", i,
				step.crossing, step.length, step.thick)
		}
		if i > 0 && !(step.along >= got[i-1].along) {
			t.Errorf("stride %d is nearer than the one before", i)
		}
	}
}

// A step with nothing in it is passed over, and nine volumes on a ray
// are all stepped through and all counted in what the ray keeps.
func TestStridesSkipNothingAndCountNine(t *testing.T) {
	empty := thing.Volume{Medium: plate(0, 0, 4)}
	full := thing.Volume{Medium: plate(0.5, 10, 14)}
	if got := stridesAcross(4, empty, full); len(got) != 4 ||
		got[0].crossing != 1 {
		t.Errorf("of an empty plate and a full one: %d strides, the "+
			"first %+v", len(got), got)
	}
	var nine []thing.Volume
	for i := range 9 {
		nine = append(nine, thing.Volume{
			Medium: plate(0.1, float64(3*i), float64(3*i+2))})
	}
	if got := stridesAcross(2, nine...); len(got) != 18 {
		t.Errorf("%d strides of nine crossings cut in two", len(got))
	}
	keep := through(staging(nine...), alongZ, math.Inf(1)).keep
	if want := math.Exp(-0.1 * 2 * 9); !within(keep.R, want, 1e-9) {
		t.Errorf("nine plates keep %v, want %v", keep.R, want)
	}
}

// One step through a gas keeps of each channel what the gas itself
// says it transmits at that channel's band over the step's thickness:
// all of it over no thickness, exactly, and gas.Transmit's own number
// over twenty kilometres of sea-level air.
func TestAGasStepKeepsWhatTheGasTransmits(t *testing.T) {
	air := gas.Air()
	scene := staging(thing.Volume{Medium: gas.Of{
		Shape: plate(1, -1, 1), In: air, Unit: 1}})
	inside := crossings(scene.volumes, alongZ, math.Inf(1))[0]
	start := maths.RGB(0.9, 0.8, 0.7)
	step := func(thick float64) maths.Light {
		return gasStep(scene, &inside, stride{thick: thick}, alongZ, 1,
			passage{keep: start}).keep
	}
	if got := step(0); got != start {
		t.Errorf("over no thickness the step keeps %v of %v", got,
			start)
	}
	const thick = 2e4
	got := step(thick)
	want := start.Times(byEye(func(band maths.Band) float64 {
		return air.Transmit(band, thick)
	}))
	if !(want.B < 0.9*start.B) {
		t.Fatalf("the test's air takes too little to tell: %v", want)
	}
	if !within(got.R, want.R, 1e-15) || !within(got.G, want.G, 1e-15) ||
		!within(got.B, want.B, 1e-15) {
		t.Errorf("the step keeps %v, want %v", got, want)
	}
}
