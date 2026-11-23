//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
	thin "github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// the picture the exposure checks draw, picked 2026-11-12: small enough
// for the tracer to draw in a few seconds, three samples a side so a
// pixel's nine are not a power of two, and a frame's time that is not
// nought so the seed is not nought's.
const (
	openWidth  = 96
	openHeight = 64
	openSide   = 3
	openAt     = 250 * time.Millisecond
)

// what a probed sample gives back: its ray's start, its way, and its
// time in the shutter. Defined by probePrimary below.
const (
	probeWords = 7
	probeTime  = 6
)

// probePrimary asks the kernel's own exposed for every sample's ray, at
// the place its own placed gives the sample, and its shutter_time, so
// the rays are checked apart from any picture.
const probePrimary = `
kernel void probe_primary(constant Header *header [[buffer(0)]],
                          device float *out [[buffer(1)]],
                          constant uint2 &origin [[buffer(2)]],
                          uint2 at [[thread_position_in_grid]],
                          uint2 grid [[threads_per_grid]]) {
	uint2 sample = uint2(at.x, at.y + origin.y);
	Primary made = exposed(header, sample, placed(header, sample));
	device float *ray = out + (at.y * grid.x + at.x) * 7;
	ray[0] = made.from.x, ray[1] = made.from.y, ray[2] = made.from.z;
	ray[3] = made.way.x, ray[4] = made.way.y, ray[5] = made.way.z;
	ray[6] = shutter_time(header, sample);
}
`

// rayGiven is a lens whose rays the cpu makes, whatever lens it wraps:
// the kernel knows a pinhole and a thin lens by their types and no other.
type rayGiven struct{ film.Sensor }

// the two multipliers SplitMix64 defines, as the kernel's unit_of writes
// them: it steps its second and third indices by them before it stirs.
const (
	unitSecondStep = 0xbf58476d1ce4e5b9
	unitThirdStep  = 0x94d049bb133111eb
)

// unitOf is the kernel's unit_of: SplitMix64 over a seed and three
// indices, cut to a float's twenty-four bits.
func unitOf(seed uint64, a, b, c int) float64 {
	stirred := seed + noise.Golden*uint64(a+1) +
		unitSecondStep*uint64(b+1) + unitThirdStep*uint64(c+1)
	return float64(noise.Mix(stirred)>>hashDrop) * hashUnit
}

// shutterTime is the kernel's shutter_time, step for step, in a float's
// own arithmetic at its last step.
func shutterTime(seed uint32, x, y, side int) float64 {
	count := side * side
	index := (y%side)*side + x%side
	drawn := unitOf(uint64(seed), x/side, y/side, shutterTurn)
	share := (index + int(drawn/hashUnit)) % count
	along := unitOf(uint64(seed), x/side, y/side, shutterFirst+index)
	return float64((float32(share) + float32(along)) / float32(count))
}

// between is the kernel's eye_at: the eye at a time in the shutter, its
// place on the line between the two and its axes mixed and made square.
func between(open, closing maths.Subspace, when float64) maths.Subspace {
	mixed := func(one, other maths.Vec) maths.Vec {
		return one.Add(other.Sub(one).Scale(when))
	}
	from, to := open.Orientation, closing.Orientation
	back := mixed(from.Out(maths.ZAxis), to.Out(maths.ZAxis)).Unit()
	right := mixed(from.Out(maths.XAxis), to.Out(maths.XAxis))
	right = right.Sub(back.Scale(right.Dot(back))).Unit()
	return maths.Subspace{Origin: mixed(open.Origin, closing.Origin),
		Orientation: maths.Orientation{X: right, Y: back.Cross(right),
			Z: back},
		Scale: open.Scale}
}

// shuttered is the kernel's open shutter as a sensor the tracer looks
// through from the world's own space: each sample's ray is its lens's,
// from the eye at the kernel's time for that sample. The tracer has no
// shutter of its own; this is how it is given the same rays.
type shuttered struct {
	open, closing maths.Subspace
	lens          film.Sensor
	side          int
	seed          uint32
}

// Ray is the sample's ray, in the world.
func (through shuttered) Ray(x, y, w, h int, band maths.Band) maths.Ray {
	when := shutterTime(through.seed, x, y, through.side)
	eye := between(through.open, through.closing, when)
	return film.Look(eye, through.lens, x, y, w, h, band)
}

// slid is the hall's eye moved while the shutter is open: along and up
// a little, and turned to keep the hall's middle in view.
func slid(eye maths.Subspace) maths.Subspace {
	from := eye.Origin.Add(maths.V(0.6, 0.15, -0.2))
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0.3, 0, 0))}
}

// probedRays is every sample's ray and time as the kernel makes them for a
// view, probeWords numbers a sample.
func probedRays(t *testing.T, view values, width, height int) []float32 {
	t.Helper()
	header := make([]byte, headerStruct.size())
	headerStruct.patch(header, view)
	kernel, err := metal.Open(source()+probePrimary, "probe_primary")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	got, err := kernel.Grid(width, height, rows, probeWords,
		[][]byte{header})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// raysApart is how far the kernel's rays are from a sensor's seen from
// an eye: the furthest start, in metres, and the furthest way, as the
// length of the difference of two unit vectors.
func raysApart(got []float32, eye maths.Subspace, lens film.Sensor, width,
	height int) (start, way float64) {
	for y := range height {
		for x := range width {
			ray := got[(y*width+x)*probeWords:]
			want := film.Look(eye, lens, x, y, width, height, maths.Band{})
			from := maths.V(float64(ray[0]), float64(ray[1]),
				float64(ray[2]))
			toward := maths.V(float64(ray[3]), float64(ray[4]),
				float64(ray[5]))
			start = math.Max(start, from.Sub(want.At).Len())
			way = math.Max(way, toward.Sub(want.Dir).Len())
		}
	}
	return start, way
}

// how close the kernel's rays are held to the tracer's, measured on
// 2026-11-12 on an M-series device and given room: see the log line of
// each check for the day's own numbers. A float's last place at the
// hall's eye, thirteen metres from the origin, is a millionth of a metre.
const (
	startSlack = 4e-6
	waySlack   = 1e-6
	// timeSlack is a float's last place below one: the kernel's fast
	// arithmetic may make its one division another way
	timeSlack = floatLastPlace
)

// the kernel's rays through a thin lens are lens.Thin's own for the same
// samples, to a float's precision, across a picture of more than a
// million samples, where a float could not hold the sequence's index
func TestTheKernelsThinRaysAreTheLensesOwn(t *testing.T) {
	_, eye := hall(t)
	glass := thinLens(t)
	const width, height = 1536, 1024
	view := viewValues(eye, glass, shutter{}, width, height)
	got := probedRays(t, view, width, height)
	start, way := raysApart(got, eye, glass, width, height)
	t.Logf("starts %.3g m apart at most, ways %.3g", start, way)
	if !(start <= startSlack) || !(way <= waySlack) {
		t.Errorf("the kernel's rays are %g m and %g from the lens's; "+
			"want %g and %g or less", start, way, startSlack, waySlack)
	}
}

// and they are not a pinhole's: the starts are spread over the aperture,
// further from the eye than any float's rounding
func TestTheKernelsThinRaysDoNotStartAtTheEye(t *testing.T) {
	_, eye := hall(t)
	glass := thinLens(t)
	view := viewValues(eye, glass, shutter{}, openWidth, openHeight)
	got := probedRays(t, view, openWidth, openHeight)
	reach, _, _, _ := glass.Opening()
	var furthest float64
	for at := 0; at < len(got); at += probeWords {
		from := maths.V(float64(got[at]), float64(got[at+1]),
			float64(got[at+2]))
		furthest = math.Max(furthest, from.Sub(eye.Origin).Len())
	}
	if !(furthest > reach/wholeTwo) || !(furthest <= reach+startSlack) {
		t.Errorf("the furthest start is %g m from the eye; the aperture "+
			"reaches %g", furthest, reach)
	}
}

// the kernel's time for a sample is the one worked here, and its ray the
// lens's from the eye at that time: through a pinhole and through a thin
// lens
func TestTheKernelsShutterRaysAreTheEyesAtEachSamplesTime(t *testing.T) {
	_, eye := hall(t)
	closing := slid(eye)
	open := shutter{closing: &closing, side: openSide, at: openAt}
	width, height := openWidth*openSide, openHeight*openSide
	for name, lens := range map[string]film.Sensor{
		"a pinhole": shotLens, "a thin lens": thinLens(t)} {
		view := viewValues(eye, lens, open, width, height)
		got := probedRays(t, view, width, height)
		for y := range height {
			for x := range width {
				when := float64(got[(y*width+x)*probeWords+probeTime])
				want := shutterTime(seedOf(openAt), x, y, openSide)
				if math.Abs(when-want) > timeSlack {
					t.Fatalf("%s: sample %d,%d is at %v of the shutter; "+
						"want %v", name, x, y, when, want)
				}
			}
		}
		reference := shuttered{open: eye, closing: closing, lens: lens,
			side: openSide, seed: seedOf(openAt)}
		start, way := raysApart(got, maths.Subspace{}, reference, width,
			height)
		t.Logf("%s: starts %.3g m apart at most, ways %.3g", name, start,
			way)
		if !(start <= startSlack) || !(way <= waySlack) {
			t.Errorf("%s: the kernel's rays are %g m and %g from the "+
				"eye's at each time; want %g and %g or less", name, start,
				way, startSlack, waySlack)
		}
	}
}

// a pixel's samples share the shutter between them: each of its equal
// shares is taken once, in every pixel; and two neighbouring pixels do
// not take them in the same order everywhere
func TestAPixelsSamplesCoverTheShutterEvenly(t *testing.T) {
	seed := seedOf(openAt)
	count := openSide * openSide
	orders := map[int]bool{}
	for pixel := range openWidth {
		taken := make([]bool, count)
		for index := range count {
			when := shutterTime(seed, pixel*openSide+index%openSide,
				index/openSide, openSide)
			share := int(when * float64(count))
			if when < 0 || when >= 1 || taken[share] {
				t.Fatalf("pixel %d: sample %d at %v takes a share twice, "+
					"or none", pixel, index, when)
			}
			taken[share] = true
			if index == 0 {
				orders[share] = true
			}
		}
	}
	if len(orders) < count/wholeTwo {
		t.Errorf("the first samples of %d pixels take only %d of %d "+
			"shares", openWidth, len(orders), count)
	}
}

// a frame's seed is its time's: the same time the same seed, another
// time another
func TestAFramesSeedIsItsTimes(t *testing.T) {
	if seedOf(openAt) != seedOf(openAt) {
		t.Error("one time gave two seeds")
	}
	seen := map[uint32]bool{}
	const frames, rate = 240, 24
	for index := range frames {
		seen[seedOf(time.Duration(index)*time.Second/rate)] = true
	}
	if len(seen) != frames {
		t.Errorf("%d frames' times gave %d seeds", frames, len(seen))
	}
}

// a share of the square as a 64-bit whole is the share times two to the
// sixty-fourth: a half is the top bit, and the r2 steps keep a float64's
// figures; nought is nought
func TestAShareOfTheSquareIsAWholeNumber(t *testing.T) {
	if got := turned(beamHalf); got != 1<<63 {
		t.Errorf("a half is %#x", got)
	}
	if got := turned(0); got != 0 {
		t.Errorf("nought is %#x", got)
	}
	for _, step := range []float64{r2Across, r2Up} {
		back := float64(turned(step)) / math.Exp2(turnBits)
		if math.Abs(back-step) > 1e-15 {
			t.Errorf("step %v came back %v", step, back)
		}
	}
}

// a closed shutter over a pinhole writes nothing in the header's new
// rows but the samples along a pixel's side, which a wrinkle fades by:
// its exposure row says the shutter and the aperture are shut, so the
// kernel makes the ray it always made, and carries the side, one where
// none is given
func TestAClosedShutterAndNoApertureWriteNoExposure(t *testing.T) {
	_, eye := hall(t)
	view := viewValues(eye, shotLens, shutter{}, openWidth, openHeight)
	for _, name := range []string{"close_origin", "close_right",
		"close_back", "optics"} {
		for index, value := range view[name] {
			if value != 0 {
				t.Errorf("%s[%d] is %v with the shutter closed and no "+
					"aperture", name, index, value)
			}
		}
	}
	for side, want := range map[int][]float64{0: {0, 0, 1},
		openSide: {0, 0, openSide}} {
		view = viewValues(eye, shotLens, shutter{side: side}, openWidth,
			openHeight)
		if got := view["exposure"]; !slices.Equal(got, want) {
			t.Errorf("the closed exposure at a side of %d is %v; want %v",
				side, got, want)
		}
	}
}

// a shutter open over an eye that does not move is closed; over one that
// moves it is open, with the pixel's side and the frame's seed beside it
func TestAShutterIsOpenOnlyOverAnEyeThatMoves(t *testing.T) {
	_, eye := hall(t)
	still := eye
	held := shutter{closing: &still, side: openSide, at: openAt}
	if held.over(eye) {
		t.Error("a shutter over a still eye is open")
	}
	closed := viewValues(eye, shotLens, held, openWidth, openHeight)
	if closed["exposure"][0] != 0 || closed["close_origin"] != nil {
		t.Errorf("a still eye's header holds a shutter: %v",
			closed["exposure"])
	}
	closing := slid(eye)
	moving := shutter{closing: &closing, side: openSide, at: openAt}
	if !moving.over(eye) || (shutter{}).over(eye) {
		t.Error("a moving eye's shutter is closed, or no shutter is open")
	}
	open := viewValues(eye, shotLens, moving, openWidth, openHeight)
	want := []float64{1, 0, openSide, float64(int32(seedOf(openAt)))}
	for index, value := range want {
		if open["exposure"][index] != value {
			t.Errorf("exposure[%d] is %v, want %v", index,
				open["exposure"][index], value)
		}
	}
	if open["close_origin"][0] != closing.Origin.X {
		t.Errorf("the closing eye is at %v", open["close_origin"])
	}
}

// a thin lens is the kernel's to make, by value or by pointer, and its
// header holds its opening; a nil pointer and a lens the kernel does not
// know are not
func TestAThinLensIsTheKernelsToMake(t *testing.T) {
	_, eye := hall(t)
	glass := thinLens(t)
	var none *thin.Thin
	for name, each := range map[string]struct {
		lens film.Sensor
		want bool
	}{
		"a thin lens":     {glass, true},
		"a pointer":       {&glass, true},
		"a pinhole":       {shotLens, true},
		"no lens":         {nil, true},
		"a nil pointer":   {none, false},
		"a lens unknown":  {rayGiven{glass}, false},
		"a nil in a lens": {rayGiven{}, false},
	} {
		if got := kernelMade(each.lens); got != each.want {
			t.Errorf("%s: the kernel's to make is %v", name, got)
		}
	}
	eye.Scale = 2
	view := viewValues(eye, glass, shutter{}, openWidth, openHeight)
	reach, focus, across, up := glass.Opening()
	want := []float64{reach * eye.Scale, focus, across, up}
	for index, value := range want {
		if view["optics"][index] != value {
			t.Errorf("optics[%d] is %v, want %v", index,
				view["optics"][index], value)
		}
	}
	if view["exposure"][1] != 1 || view["exposure"][0] != 0 {
		t.Errorf("a thin lens's exposure is %v", view["exposure"])
	}
}

// a picture through a thin lens sends no rays: its inputs are a
// pinhole's, and a lens the kernel does not know sends 24 bytes a sample
// more
func TestAThinLensSendsNoRays(t *testing.T) {
	world, eye := hall(t)
	glass := thinLens(t)
	bytes := func(lens film.Sensor) int64 {
		_, spent, err := SampledMeasured(world, eye, lens, 0, openWidth,
			openHeight, openSide)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatal(err)
		}
		return spent.InputBytes
	}
	pinhole, made, given := bytes(shotLens), bytes(glass),
		bytes(rayGiven{glass})
	rays := int64(openWidth * openSide * openHeight * openSide * rayStride)
	if made != pinhole || given != pinhole+rays {
		t.Errorf("inputs: a pinhole %d bytes, a thin lens %d, a lens "+
			"given as rays %d; want the first two equal and the last "+
			"%d more", pinhole, made, given, rays)
	}
}

// two ways of making the same picture: the thin lens's rays made in the
// kernel, and the same lens's rays made on the cpu and sent. They agree
// far inside the exact tolerance, which says the kernel's rays are right
// before the tracer is asked.
func TestAThinLensInTheKernelIsTheSameLensGivenAsRays(t *testing.T) {
	world, eye := hall(t)
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	glass := thinLens(t)
	made, err := Sampled(world, eye, glass, 0, openWidth, openHeight,
		openSide)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	given, err := Sampled(world, eye, rayGiven{glass}, 0, openWidth,
		openHeight, openSide)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(given, made, openWidth, openHeight)
	t.Logf("apart: %+v", found)
	for _, problem := range findings(found, exact) {
		t.Error(problem)
	}
}

// the ray input still draws as slayer's tracer does, for a lens the
// kernel does not make
func TestTheGPUDrawsThroughALensGivenAsRaysAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	withinThrough(t, world, eye, rayGiven{thinLens(t)}, openWidth,
		openHeight, wholeTwo, exact)
}

// openDrawn is the hall drawn by a drawer with the shutter open from eye
// to closing, with nothing baked.
func openDrawn(t *testing.T, world *thing.World, eye, closing maths.Subspace,
	lens film.Sensor) *frame.Frame {
	t.Helper()
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	drawn, _, err := frames.DrawOpen(world, 1, eye, closing, lens, openAt,
		openWidth, openHeight, openSide)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// the kernel's picture through an open shutter is slayer's tracer's of
// the same rays, each sample from the eye at its own time, at the exact
// tolerance: through a pinhole, and through a thin lens with it
func TestTheGPUDrawsAnOpenShutterAsSlayerDoesTheSameRays(t *testing.T) {
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	for name, lens := range map[string]film.Sensor{
		"a pinhole": shotLens, "a thin lens": thinLens(t)} {
		world, eye := hall(t)
		closing := slid(eye)
		world.Pixel = wideOf(world, eye, shotLens, openWidth*openSide,
			openHeight*openSide)
		drawn := openDrawn(t, world, eye, closing, lens)
		reference := shuttered{open: eye, closing: closing, lens: lens,
			side: openSide, seed: seedOf(openAt)}
		truth, err := slayer.Sampled(world, maths.Subspace{}, reference,
			openAt, openWidth, openHeight, openSide)
		if err != nil {
			t.Fatal(err)
		}
		found := compare(truth, drawn, openWidth, openHeight)
		t.Logf("%s, apart: %+v", name, found)
		for _, problem := range findings(found, exact) {
			t.Errorf("%s: %s", name, problem)
		}
	}
}

// and it is not the picture with the shutter closed: the move blurs it
func TestAnOpenShutterIsNotTheClosedPicture(t *testing.T) {
	world, eye := hall(t)
	open := openDrawn(t, world, eye, slid(eye), shotLens)
	closed := openDrawn(t, world, eye, eye, shotLens)
	if apart := differs(open, closed); apart < openWidth*openHeight/wholeFour {
		t.Errorf("an open shutter changed %d pixels of %d", apart,
			openWidth*openHeight)
	}
}

// with the shutter closed and no aperture the picture is the per-call
// one to the bit, however the shutter is said to be closed: not asked
// for, or held open over an eye that does not move
func TestAClosedShutterAndNoApertureIsThePictureItWas(t *testing.T) {
	world, eye := hall(t)
	want, _, err := SampledMeasured(world, eye, shotLens, openAt, openWidth,
		openHeight, openSide)
	mustDevice(t, err)
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	plain, _, err := frames.Draw(world, 1, eye, shotLens, openAt, openWidth,
		openHeight, openSide)
	mustDevice(t, err)
	sameBits(t, "no shutter", want, plain)
	sameBits(t, "a shutter open over a still eye", want,
		openDrawn(t, world, eye, eye, shotLens))
}

// a film through a thin lens is retained as a pinhole's is: each frame
// is the per-call picture to the bit, and a frame whose eye and focus
// alone changed is drawn by sending the header
func TestFramesKeepAThinLensOnTheDevice(t *testing.T) {
	world, eye := movingHall(t)
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	full, err := thin.FrameOf(36, 24)
	if err != nil {
		t.Fatal(err)
	}
	for index, focus := range []float64{13, 9} {
		glass, err := thin.NewThin(full, 50, 1.4, focus)
		if err != nil {
			t.Fatal(err)
		}
		from := eye
		from.Origin = from.Origin.Add(maths.V(float64(index), 0, 0))
		got, spent, err := frames.Draw(world, 1, from, glass, 0, shotWidth,
			shotHeight, shotSide)
		mustDevice(t, err)
		want, _, err := SampledMeasured(world, from, glass, 0, shotWidth,
			shotHeight, shotSide)
		mustDevice(t, err)
		sameBits(t, "a thin lens kept on the device", want, got)
		if moved := index > 0; spent.Retained.Patched != moved {
			t.Errorf("frame %d: patched %v, want %v", index,
				spent.Retained.Patched, moved)
		}
	}
}

// a shutter open over a lens whose rays the cpu makes is refused by
// name; over a still eye the same lens draws
func TestFramesRefuseAnOpenShutterOverALensGivenAsRays(t *testing.T) {
	world, eye := hall(t)
	lens := rayGiven{thinLens(t)}
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	if _, _, err := frames.DrawOpen(world, 1, eye, eye, lens, 0, shotWidth,
		shotHeight, shotSide); err != nil {
		mustDevice(t, err)
	}
	_, _, err := frames.DrawOpen(world, 1, eye, slid(eye), lens, 0,
		shotWidth, shotHeight, shotSide)
	if !errors.Is(err, errShutterOverRays) {
		t.Fatalf("an open shutter over rays gave %v", err)
	}
}

// focusNear and focusFar are how far off the two lit balls of the focus
// check hang, and focusLens its lens, picked 2026-11-12: 200 mm at f/1
// on a full frame, focused on the near ball, blurs the far one over a
// disc sixteen pixels across where each ball alone is eleven.
const (
	focusNear   = 10.0
	focusFar    = 40.0
	focusFocal  = 200.0
	focusStop   = 1.0
	focusWidth  = 192
	focusHeight = 128
	focusSide   = 4
	// focusLit is the light above which a pixel is counted as lit, a
	// thousandth of a ball's glow of one
	focusLit = 1e-3
)

// twoBalls is two glowing balls of one size as the eye sees them, one at
// the focus on the left and one four times as far on the right, in the
// dark, and the eye at the origin looking down minus z.
func twoBalls() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	glow := &thing.Material{Name: "glow", Glow: maths.RGB(1, 1, 1)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.05},
		Material: glow, Motion: at(-0.5, 0, -focusNear)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.2},
		Material: glow, Motion: at(2, 0, -focusFar)})
	return world, maths.Subspace{Orientation: maths.Identity}
}

// litHalves is how many pixels of each half of a picture are lit.
func litHalves(picture *frame.Frame) (left, right int) {
	for row := range picture.H {
		for column := range picture.W {
			pixel := picture.At(column, row)
			if pixel.R+pixel.G+pixel.B <= focusLit {
				continue
			}
			if column < picture.W/wholeTwo {
				left++
			} else {
				right++
			}
		}
	}
	return left, right
}

// a ball at the focus distance is as sharp through the thin lens as
// through a pinhole, and one four times as far is spread over several
// times the pixels: the kernel's lens focuses where it says
func TestAThinLensIsSharpAtItsFocusAndSoftOffIt(t *testing.T) {
	world, eye := twoBalls()
	full, err := thin.FrameOf(36, 24)
	if err != nil {
		t.Fatal(err)
	}
	glass, err := thin.NewThin(full, focusFocal, focusStop, focusNear)
	if err != nil {
		t.Fatal(err)
	}
	lit := func(lens film.Sensor) (int, int) {
		drawn, err := Sampled(world, eye, lens, 0, focusWidth, focusHeight,
			focusSide)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatal(err)
		}
		return litHalves(drawn)
	}
	nearSharp, farSharp := lit(film.Pinhole{Wide: glass.Wide()})
	near, far := lit(glass)
	t.Logf("lit pixels: pinhole %d and %d, thin lens %d and %d", nearSharp,
		farSharp, near, far)
	if nearSharp == 0 || farSharp == 0 {
		t.Fatal("the pinhole sees no ball")
	}
	if near > nearSharp*wholeThree/wholeTwo {
		t.Errorf("the ball at the focus lights %d pixels, %d through a "+
			"pinhole: it is not sharp", near, nearSharp)
	}
	if far < farSharp*wholeThree {
		t.Errorf("the ball off the focus lights %d pixels, %d through a "+
			"pinhole: it is not soft", far, farSharp)
	}
}
