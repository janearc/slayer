//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the pictures these checks draw: small, so each draw is quick, with
// two by two samples a pixel so a pixel averages several; picked
// 2026-11-12 for these checks
const (
	shotWidth, shotHeight, shotSide = 48, 32, 2
	// filmFrames is how many frames of a film are drawn in a row
	filmFrames = 4
	// frameStep is the time between them, a film's frame at 24 a second
	frameStep = time.Second / 24
)

// perCalls is each traversal's per-call function, the reference a
// drawer's picture is held to.
var perCalls = map[traversal]func(*thing.World, maths.Subspace,
	film.Sensor, time.Duration, int, int, int) (*frame.Frame, Profile,
	error){
	softwareTraversal: SampledMeasured, nativeShadowTraversal: SampledNative,
	nativeTraversal: SampledAccelerated, cachedTraversal: SampledCached,
	fastTraversal: SampledFast, fasterTraversal: SampledFaster}

// rebakes is whether a mode bakes light into the boxes each frame it
// encodes: the cached mode's source grids and fast light's lamp store.
func rebakes(mode traversal) bool {
	return mode == cachedTraversal || mode.stored()
}

// mustDevice fails, and does not skip, where there is no Metal device:
// a check that cannot run fails. The package's older checks skip there;
// these hold a claim about the device and cannot make it without one.
func mustDevice(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, metal.ErrNoDevice) {
		t.Fatal("this check needs a Metal device and there is none:", err)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// sameBits fails unless two frames are one, every pixel's light, cover
// and depth compared as bits.
func sameBits(t *testing.T, what string, want, got *frame.Frame) {
	t.Helper()
	if want == nil || got == nil || want.W != got.W || want.H != got.H {
		t.Fatalf("%s: frames of different sizes or none", what)
	}
	apart, first := 0, -1
	for index, one := range want.Pix {
		if !pixelBits(one, got.Pix[index]) {
			apart++
			first = max(first, index)
		}
	}
	if apart > 0 {
		t.Fatalf("%s: %d of %d pixels differ in their bits, the last at "+
			"%d", what, apart, len(want.Pix), first)
	}
}

// pixelBits is whether two pixels are the same to the bit.
func pixelBits(one, other frame.Pixel) bool {
	bits := math.Float32bits
	return bits(one.R) == bits(other.R) && bits(one.G) == bits(other.G) &&
		bits(one.B) == bits(other.B) &&
		bits(one.Cover) == bits(other.Cover) &&
		bits(one.Depth) == bits(other.Depth)
}

// drift is a motion along a line at a steady speed, so a film's frames
// differ.
type drift struct{ from, way maths.Vec }

// At is where it is after a time.
func (moving drift) At(at time.Duration) maths.Subspace {
	return maths.Subspace{Origin: moving.from.Add(moving.way.Scale(
		at.Seconds()))}
}

// movingHall is the hall with a ball rolling across it.
func movingHall(t *testing.T) (*thing.World, maths.Subspace) {
	world, eye := hall(t)
	ball := &thing.Material{Name: "rolling", Colour: maths.RGB(0.8, 0.3, 0.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Material: ball,
		Motion: drift{from: maths.V(-2, -1, 2), way: maths.V(6, 0, 0)}})
	return world, eye
}

// sceneFor is the scene each traversal is checked on: the fire, with
// its smoke lighting grid, in the cached mode; the hall otherwise.
func sceneFor(t *testing.T, mode traversal) (*thing.World, maths.Subspace) {
	if rebakes(mode) {
		return fire(t)
	}
	return movingHall(t)
}

// traversalsChecked is every traversal, in a fixed order.
var traversalsChecked = []traversal{softwareTraversal,
	nativeShadowTraversal, nativeTraversal, cachedTraversal,
	fastTraversal, fasterTraversal}

// shotLens is the pinhole these checks look through.
var shotLens = film.Pinhole{Wide: 50 * maths.Degree}

// perCall is the reference picture of a frame.
func perCall(t *testing.T, mode traversal, world *thing.World,
	eye maths.Subspace, at time.Duration) *frame.Frame {
	t.Helper()
	picture, _, err := perCalls[mode](world, eye, shotLens, at, shotWidth,
		shotHeight, shotSide)
	mustDevice(t, err)
	return picture
}

// drawnBy is a drawer's picture of a frame, and its profile.
func drawnBy(t *testing.T, frames *Frames, mode traversal, world *thing.World,
	generation uint64, eye maths.Subspace, at time.Duration) (*frame.Frame,
	Profile) {
	t.Helper()
	take := shot{world: world, generation: generation, eye: eye,
		lens: shotLens, at: at, width: shotWidth, height: shotHeight,
		side: shotSide}
	start := time.Now()
	picture, spent, err := frames.draw(take, mode)
	mustDevice(t, err)
	return picture, spent.profile(time.Since(start))
}

// TestFramesColdAndWarmAreThePerCallPicture draws a frame twice in each
// mode, cold and then warm with nothing changed: both are the per-call
// picture, the second kept on the device.
//
// In every mode but the cached one, whose light bakes rewrite the header
// and the boxes each frame, a cold frame sends each input once and a
// warm frame sends nothing.
func TestFramesColdAndWarmAreThePerCallPicture(t *testing.T) {
	for _, mode := range traversalsChecked {
		world, eye := sceneFor(t, mode)
		frames := newFrames(mode)
		want := perCall(t, mode, world, eye, 0)
		cold, coldSpent := drawnBy(t, frames, mode, world, 1, eye, 0)
		sameBits(t, "cold", want, cold)
		// every input once, and in the cached mode the header and the
		// boxes again around each light bake
		sent := coldSpent.Retained
		if frames.why != noneHeld || sent.Sent < sent.Held ||
			(!rebakes(mode) && sent.Sent != sent.Held) {
			t.Errorf("mode %d: the cold frame opened for %q and sent %+v",
				mode, frames.why, coldSpent.Retained)
		}
		warm, warmSpent := drawnBy(t, frames, mode, world, 2, eye, 0)
		sameBits(t, "warm", want, warm)
		if frames.why != "" || warmSpent.Retained.Patched {
			t.Errorf("mode %d: the warm frame opened for %q, patched %v",
				mode, frames.why, warmSpent.Retained.Patched)
		}
		if !rebakes(mode) && warmSpent.Retained.Sent != 0 {
			t.Errorf("mode %d: an unchanged frame sent %+v", mode,
				warmSpent.Retained)
		}
		frames.Close()
	}
}

// TestFramesDrawAFilmFrameByFrameAsThePerCallPath draws frames 0 to N-1
// of a film whose time advances, in each mode: frame N after the rest
// is the per-call picture of frame N, and the frames do differ, so the
// check is not of one picture over and over.
func TestFramesDrawAFilmFrameByFrameAsThePerCallPath(t *testing.T) {
	for _, mode := range traversalsChecked {
		world, eye := sceneFor(t, mode)
		frames := newFrames(mode)
		var first, got *frame.Frame
		for index := range filmFrames {
			at := time.Duration(index) * frameStep
			want := perCall(t, mode, world, eye, at)
			got, _ = drawnBy(t, frames, mode, world, uint64(index), eye, at)
			sameBits(t, "a film's frame", want, got)
			if index > 0 && frames.why != "" {
				t.Errorf("mode %d frame %d opened again for %q", mode,
					index, frames.why)
			}
			if first == nil {
				first = got
			}
		}
		frames.Close()
		if differs(first, got) == 0 {
			t.Errorf("mode %d: the film's last frame is its first", mode)
		}
	}
}

// differs is how many pixels of two frames are not the same to the bit.
func differs(one, other *frame.Frame) int {
	apart := 0
	for index, pixel := range one.Pix {
		if !pixelBits(pixel, other.Pix[index]) {
			apart++
		}
	}
	return apart
}

// TestFramesMoveTheCameraBySendingOnlyTheHeader moves the eye with the
// generation, the world and the time kept: the world is not encoded,
// the header alone is sent, and the picture is the per-call one from the
// new eye, in every mode.
func TestFramesMoveTheCameraBySendingOnlyTheHeader(t *testing.T) {
	for _, mode := range traversalsChecked {
		world, eye := sceneFor(t, mode)
		frames := newFrames(mode)
		drawnBy(t, frames, mode, world, 1, eye, frameStep)
		header := len(frames.last.header)
		for _, step := range []maths.Vec{maths.V(0.5, 0, 0),
			maths.V(0.5, 0.25, -1)} {
			eye.Origin = eye.Origin.Add(step)
			eye.Orientation = maths.LookAt(eye.Origin, maths.Vec{})
			got, spent := drawnBy(t, frames, mode, world, 1, eye, frameStep)
			sameBits(t, "a camera move",
				perCall(t, mode, world, eye, frameStep), got)
			moved := spent.Retained
			if !moved.Patched || moved.Sent != 1 ||
				moved.SentBytes != int64(header) || spent.Encode != 0 {
				t.Errorf("mode %d: a camera move sent %+v, encoded %v",
					mode, moved, spent.Encode)
			}
		}
		frames.Close()
	}
}

// TestFramesEncodeAgainWhenMoreThanTheCameraMoved is the near misses of
// a camera move: another generation, another time, another world, and
// another size, each with the eye moved; none is patched, and each is
// the per-call picture.
func TestFramesEncodeAgainWhenMoreThanTheCameraMoved(t *testing.T) {
	world, eye := movingHall(t)
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	twin, _ := movingHall(t)
	for name, next := range map[string]shot{
		"another generation": {world: world, generation: 2},
		"another time":       {world: world, generation: 1, at: frameStep},
		"another world":      {world: twin, generation: 1},
	} {
		drawnBy(t, frames, softwareTraversal, world, 1, eye, 0)
		moved := eye
		moved.Origin = moved.Origin.Add(maths.V(1, 0, 0))
		got, spent := drawnBy(t, frames, softwareTraversal, next.world,
			next.generation, moved, next.at)
		if spent.Retained.Patched {
			t.Errorf("%s: patched", name)
		}
		sameBits(t, name, perCall(t, softwareTraversal, next.world, moved,
			next.at), got)
	}
	take := shot{world: world, generation: 1, eye: eye, lens: shotLens,
		width: shotWidth / 2, height: shotHeight, side: shotSide}
	if _, spent, err := frames.draw(take, softwareTraversal); err != nil ||
		spent.retained.Patched || frames.why != "another size" {
		t.Errorf("a smaller frame: patched %v, opened for %q (%v)",
			spent.retained.Patched, frames.why, err)
	}
}

// TestFramesGiveAStalePictureToACallerThatLies keeps the generation
// across a lamp moved in place: the drawer cannot see it, and draws the
// old lamp, as its doc says. A new generation draws the new one.
func TestFramesGiveAStalePictureToACallerThatLies(t *testing.T) {
	world, eye := movingHall(t)
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	old, _ := drawnBy(t, frames, softwareTraversal, world, 1, eye, 0)
	world.Lamps[0] = emit.Lit{At: maths.V(3, 2, 2), Beyond: 0.1,
		Strength: maths.RGB(30, 30, 30)}
	stale, _ := drawnBy(t, frames, softwareTraversal, world, 1, eye, 0)
	sameBits(t, "a lie", old, stale)
	told, _ := drawnBy(t, frames, softwareTraversal, world, 2, eye, 0)
	sameBits(t, "told", perCall(t, softwareTraversal, world, eye, 0), told)
	if differs(old, told) == 0 {
		t.Error("the moved lamp changed nothing, so this proves nothing")
	}
}

// change is a change between two frames, the mode the first is drawn
// in, why the drawer should open its device state again for it, or
// empty where it keeps it, and why it should build its hierarchy again,
// or empty.
type change struct {
	name, why, rebuilt string
	first              traversal
	make               func(t *testing.T,
		world *thing.World) (traversal, int)
}

// the hall's things that the changes move or reshape: the solid awning,
// a mesh of thirty-two faces, by the order hall adds them
const awningAt = 4

// changes are the changes a drawer follows, by an update, by opening
// again or by building its hierarchy again: a lamp moved, a volume more,
// a picture larger, another mode, a mesh moved and a mesh reshaped.
var changes = []change{
	{"a lamp moved", "", "", softwareTraversal,
		func(_ *testing.T, world *thing.World) (traversal, int) {
			world.Lamps[0] = emit.Lit{At: maths.V(-3, 2, 2), Beyond: 0.1,
				Strength: maths.RGB(20, 20, 30)}
			return softwareTraversal, shotWidth
		}},
	{"a volume more", "another kernel source", "", softwareTraversal,
		func(_ *testing.T, world *thing.World) (traversal, int) {
			haze := &thing.Material{Name: "more", Colour: maths.Grey(0.6)}
			world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1,
				Thick: 0.3}, Material: haze, Motion: at(2, 0, 2)})
			return softwareTraversal, shotWidth
		}},
	{"a larger picture", "another size", "", softwareTraversal,
		func(*testing.T, *thing.World) (traversal, int) {
			return softwareTraversal, shotWidth + shotSide
		}},
	{"another mode", "another mode", "", softwareTraversal,
		func(*testing.T, *thing.World) (traversal, int) {
			return nativeTraversal, shotWidth
		}},
	{"a mesh moved", "", "another hierarchy", nativeTraversal,
		func(_ *testing.T, world *thing.World) (traversal, int) {
			world.Things[awningAt].Motion = at(0.5, -0.8, -3)
			return nativeTraversal, shotWidth
		}},
	{"a mesh reshaped", "", "other triangles under the hierarchy",
		nativeTraversal,
		func(t *testing.T, world *thing.World) (traversal, int) {
			flat := floor(t, 4)
			taller := make([]maths.Vec, len(flat.Vertices))
			for index, corner := range flat.Vertices {
				taller[index] = corner.Add(maths.V(0, corner.X/4, 0))
			}
			reshaped, err := shape.NewMesh(taller, flat.Faces, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			world.Things[awningAt].Shape = reshaped
			return nativeTraversal, shotWidth
		}},
}

// TestFramesFollowEveryChangeToThePerCallPicture draws the hall, makes
// one change, and draws again: the picture is the per-call one, kept or
// opened again as the change asks. An unchanged frame is kept.
func TestFramesFollowEveryChangeToThePerCallPicture(t *testing.T) {
	for _, each := range append(changes, change{name: "nothing",
		first: softwareTraversal,
		make: func(*testing.T, *thing.World) (traversal, int) {
			return softwareTraversal, shotWidth
		}}) {
		world, eye := movingHall(t)
		frames := newFrames(each.first)
		drawnBy(t, frames, each.first, world, 1, eye, 0)
		mode, width := each.make(t, world)
		take := shot{world: world, generation: 2, eye: eye, lens: shotLens,
			width: width, height: shotHeight, side: shotSide}
		got, spent, err := frames.draw(take, mode)
		mustDevice(t, err)
		want, _, err := perCalls[mode](world, eye, shotLens, 0, width,
			shotHeight, shotSide)
		mustDevice(t, err)
		sameBits(t, each.name, want, got)
		if frames.why != each.why || frames.rebuilt != each.rebuilt {
			t.Errorf("%s: opened for %q, rebuilt for %q; want %q, %q",
				each.name, frames.why, frames.rebuilt, each.why,
				each.rebuilt)
		}
		if each.name == "a lamp moved" && spent.retained.Sent != 1 {
			t.Errorf("a lamp moved: sent %+v, want the lamps alone",
				spent.retained)
		}
		frames.Close()
	}
}

// TestFramesDrawALensGivenAsRaysPerCall draws through a lens whose rays
// the cpu makes: the per-call picture, by the per-call path, leaving the
// held state as it was. A thin lens is the kernel's to make, and kept
// (exposure_test.go).
func TestFramesDrawALensGivenAsRaysPerCall(t *testing.T) {
	world, eye := movingHall(t)
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	drawnBy(t, frames, softwareTraversal, world, 1, eye, 0)
	kept := frames.held
	glass := rayGiven{thinLens(t)}
	got, profile, err := frames.Draw(world, 1, eye, glass, 0, shotWidth,
		shotHeight, shotSide)
	mustDevice(t, err)
	want, _, err := SampledMeasured(world, eye, glass, 0, shotWidth,
		shotHeight, shotSide)
	mustDevice(t, err)
	sameBits(t, "a lens given as rays", want, got)
	if frames.held != kept || profile.Retained != (FrameInputs{}) ||
		frames.last != nil {
		t.Errorf("a lens given as rays touched the held state: %+v",
			profile.Retained)
	}
}

// TestFramesRefuseAfterAnErrorAndAfterClose draws a world the kernel
// cannot draw: the error is returned, the device state let go, and the
// next draw of a good world refused by name. A closed drawer refuses by
// its own name, and closes twice harmlessly.
func TestFramesRefuseAfterAnErrorAndAfterClose(t *testing.T) {
	world, eye := movingHall(t)
	frames, err := NewFrames(Software)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = frames.Draw(world, 1, eye, shotLens, 0, shotWidth,
		shotHeight, shotSide); err != nil {
		mustDevice(t, err)
	}
	hazy, _ := movingHall(t)
	hazy.Haze.Density = 0.1
	_, _, err = frames.Draw(hazy, 2, eye, shotLens, 0, shotWidth, shotHeight,
		shotSide)
	if err == nil || frames.held != nil {
		t.Fatalf("a hazy world drew (%v), or left the device held", err)
	}
	_, _, again := frames.Draw(world, 3, eye, shotLens, 0, shotWidth,
		shotHeight, shotSide)
	if !errors.Is(again, errFramesFailed) || !errors.Is(again, err) {
		t.Errorf("a draw after an error says %v", again)
	}
	frames.Close()
	frames.Close()
	if _, _, err = frames.Draw(world, 4, eye, shotLens, 0, shotWidth,
		shotHeight, shotSide); !errors.Is(err, ErrFramesClosed) {
		t.Errorf("a draw after Close says %v", err)
	}
}

// TestFramesRefuseNoWorldAndNoModeWithoutBreaking refuses a draw of no
// world and leaves the drawer drawing; refuses modes that are none of
// the four; and makes a drawer for each that is.
func TestFramesRefuseNoWorldAndNoModeWithoutBreaking(t *testing.T) {
	world, eye := movingHall(t)
	frames, err := NewFrames(Software)
	if err != nil {
		t.Fatal(err)
	}
	defer frames.Close()
	if _, _, err = frames.Draw(nil, 1, eye, shotLens, 0, shotWidth, shotHeight,
		shotSide); !errors.Is(err, errNoWorld) {
		t.Errorf("no world says %v", err)
	}
	got, _, err := frames.Draw(world, 1, eye, shotLens, 0, shotWidth,
		shotHeight, shotSide)
	mustDevice(t, err)
	sameBits(t, "after no world", perCall(t, softwareTraversal, world, eye,
		0), got)
	for _, mode := range []Mode{noMode, modes, modes + 1} {
		if _, err := NewFrames(mode); !errors.Is(err, errNoMode) {
			t.Errorf("mode %d says %v", mode, err)
		}
	}
	for mode := Software; mode < modes; mode++ {
		made, err := NewFrames(mode)
		if err != nil || made.mode != traversals[mode] {
			t.Errorf("mode %d: %v", mode, err)
		}
	}
	var none *Frames
	if _, _, err := none.Draw(world, 1, eye, shotLens, 0, shotWidth,
		shotHeight, shotSide); !errors.Is(err, ErrFramesClosed) {
		t.Errorf("no drawer says %v", err)
	}
}

// TestFramesDrawAnIncidentSkyGridAsThePerCallPath draws a short film of
// the hall with its sky gathered into an incident grid, whose bake runs
// over the held inputs: every frame is the per-call picture.
func TestFramesDrawAnIncidentSkyGridAsThePerCallPath(t *testing.T) {
	world, eye := movingHall(t)
	light, err := thing.NewVolumeIllumination(incidentSamples)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := thing.NewIncidentGrid(maths.Box{Min: maths.V(-6, -2, -6),
		Max: maths.V(6, 6, 6)}, [3]int{incidentNodes, incidentNodes,
		incidentNodes})
	if err != nil {
		t.Fatal(err)
	}
	if world.VolumeIllumination, err = light.WithGrid(grid); err != nil {
		t.Fatal(err)
	}
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	for index := range filmFrames {
		at := time.Duration(index) * frameStep
		got, spent := drawnBy(t, frames, softwareTraversal, world,
			uint64(index), eye, at)
		sameBits(t, "an incident grid's frame",
			perCall(t, softwareTraversal, world, eye, at), got)
		if spent.EnvironmentNodes == 0 || spent.EnvironmentGPU <= 0 {
			t.Fatalf("frame %d gathered no incident grid: %+v", index, spent)
		}
	}
}

// incidentSamples and incidentNodes are a small incident sky grid for
// a check, picked 2026-11-12: few directions, few nodes.
const (
	incidentSamples = 8
	incidentNodes   = 4
)

// TestTheHeaderPatchIsTheEncodersCamera encodes the hall from one eye,
// patches its header to another, and the bytes are the header encoded
// from the other eye; the fields that are not the camera's are left as
// they were.
func TestTheHeaderPatchIsTheEncodersCamera(t *testing.T) {
	world, eye := movingHall(t)
	other := eye
	other.Origin = other.Origin.Add(maths.V(1, -2, 3))
	other.Orientation = maths.LookAt(other.Origin, maths.V(1, 0, 0))
	wider := film.Pinhole{Wide: 70 * maths.Degree, Pixel: 1.25}
	from, err := encode(world, eye, shotLens, frameStep, shotWidth,
		shotHeight)
	if err != nil {
		t.Fatal(err)
	}
	to, err := encode(world, other, wider, frameStep, shotWidth+1,
		shotHeight)
	if err != nil {
		t.Fatal(err)
	}
	patched := append([]byte(nil), from.header.bytes...)
	headerStruct.patch(patched, cameraValues(other, wider, shotWidth+1,
		shotHeight))
	if string(patched) != string(to.header.bytes) {
		t.Fatal("the patched header is not the one encoded from the eye")
	}
	camera := map[string]bool{}
	for name := range cameraValues(eye, shotLens, 1, 1) {
		camera[name] = true
	}
	for index, member := range headerStruct.fields {
		at := index * fieldBytes
		same := string(patched[at:at+fieldBytes]) ==
			string(from.header.bytes[at:at+fieldBytes])
		if same == camera[member.name] {
			t.Errorf("field %s: unchanged %v, a camera field %v",
				member.name, same, camera[member.name])
		}
	}
}

// TestTheHeaderPatchRefusesWhatIsNotAField panics on a name the header
// lacks and on a field given five numbers, as row does.
func TestTheHeaderPatchRefusesWhatIsNotAField(t *testing.T) {
	row := make([]byte, headerStruct.size())
	for name, given := range map[string]values{
		"no such field": {"elsewhere": {1}},
		"five numbers":  {"origin": {1, 2, 3, 4, 5}},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s was patched", name)
				}
			}()
			headerStruct.patch(row, given)
		}()
	}
}

// TestEveryStaleRuleFindsItsOwnChange holds a key and changes one thing
// at a time: each change is found by its own rule, and an unchanged
// candidate by none.
func TestEveryStaleRuleFindsItsOwnChange(t *testing.T) {
	key := heldKey{mode: nativeTraversal, text: "kernel", width: 4,
		height: 3, per: per}
	state := &held{key: key}
	same := candidate{key: key, triangles: true}
	if why := state.stale(same); why != "" {
		t.Errorf("an unchanged frame is stale: %q", why)
	}
	var none *held
	if why := none.stale(same); why != noneHeld {
		t.Errorf("nothing held says %q", why)
	}
	for _, each := range []struct {
		why  string
		next func(candidate) candidate
	}{
		{"another mode", func(next candidate) candidate {
			next.key.mode = cachedTraversal
			return next
		}},
		{"another kernel source", func(next candidate) candidate {
			next.key.text += " "
			return next
		}},
		{"another size", func(next candidate) candidate {
			next.key.height++
			return next
		}},
		{"another sample layout", func(next candidate) candidate {
			next.key.per += perRecord
			return next
		}},
	} {
		if why := state.stale(each.next(same)); why != each.why {
			t.Errorf("a change found as %q, want %q", why, each.why)
		}
		if why := state.first(hierarchyRules, each.next(same)); why != "" {
			t.Errorf("%s rebuilds the hierarchy for %q", each.why, why)
		}
	}
	if why := state.first(hierarchyRules, same); why != "" {
		t.Errorf("an unchanged hierarchy is built again for %q", why)
	}
	moved := same
	moved.triangles = false
	if why := state.first(hierarchyRules, moved); why !=
		"other triangles under the hierarchy" || state.stale(moved) != "" {
		t.Errorf("other triangles: rebuilt for %q, stale %q", why,
			state.stale(moved))
	}
	software := &held{key: heldKey{mode: softwareTraversal}}
	if why := software.first(hierarchyRules,
		candidate{key: software.key}); why != "" {
		t.Errorf("software triangles, never compared, rebuild for %q", why)
	}
}
