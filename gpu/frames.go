//go:build darwin

package gpu

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// Frames draws the pictures of a film, or of a live view, keeping on the
// device between pictures what the per-call functions make for each one:
// the compiled kernel, the scene's inputs and, in the native modes, the
// hierarchy over its meshes.
//
// Its pictures are the per-call function's of the same mode, bit for bit;
// frames_test.go holds it to that. A frame is drawn by the first rule in
// frameRules that applies, or retained.
//
// A lens whose rays the cpu makes, one that is neither a pinhole nor a
// thin lens, is drawn by the per-call path, for that frame: its rays are
// a per-frame input, made in stripes when large.
//
// DrawOpen holds the shutter open while the eye moves, each sample at
// its own time, in the kernel (exposure.go). A thin lens's rays are made
// there too, so a film through one is retained as a pinhole's is.
//
// A camera move is drawn without encoding the world: the caller passed
// the last frame's generation, and the world, the time, the size and the
// mode are the last frame's. Only the header's camera fields are
// rewritten, and that one input sent.
//
// A caller that keeps its generation across a change to its world gets
// the old world's picture. The drawer cannot see the change and does not
// guess; the generation is the caller's word that nothing changed.
//
// Any other frame is retained: encoded by the per-call path's own
// encoder, each input compared where the device holds it, and only the
// parts that differ sent. Nothing is copied or hashed on the host.
//
// The held state is kept while staleRules finds nothing changed: the
// mode, the kernel source, the size in samples and the numbers a
// sample. When one changed it is let go and opened again for the frame,
// as the per-call path opens it.
//
// A kept kernel's hierarchy is built again, in place, where
// hierarchyRules finds its layout or its triangles changed: moving
// meshes, sparks among them, change it every frame, and the per-call
// path builds one every frame.
//
// After any error the drawer lets its device state go and refuses every
// later draw with ErrFramesFailed: metal says a kernel that timed out is
// not used again, and the drawer cannot tell a timeout from another
// failure. After Close it refuses with ErrFramesClosed.
//
// Held until Close or an error: the kernel and its hierarchy, the
// device's copy of every input (593 MB for doc/fire-night.yaml, by the
// baseline of 12 November 2026) and two output bands. Calls are
// sequential.
//
// On the host it keeps the last frame's header, a few hundred bytes, its
// paints, and the world it drew, to know a camera move when it sees one;
// and the face trees of the meshes the last frame drew, by their bytes,
// with its packed faces, so a mesh that only moves is neither grown nor
// packed again (see keptTrees).
type Frames struct {
	mode         traversal
	pass         CompositePass
	transmission *frame.Frame
	held         *held
	last         *lastFrame
	// failed is the first error, after which every draw is refused
	failed error
	closed bool
	// why is why the last retained frame opened the device state again,
	// or empty where it kept it; rebuilt is why a kept frame built its
	// hierarchy again, or empty where it kept that too
	why, rebuilt string
	// trees is the face trees and packed faces of the meshes the last
	// frame drew, kept for the next; see keptTrees
	trees *keptTrees
}

// shot is one frame asked of a drawer.
type shot struct {
	transmission        **frame.Frame
	world               *thing.World
	generation          uint64
	eye                 maths.Subspace
	lens                film.Sensor
	at                  time.Duration
	width, height, side int
	// closing is the eye where the shutter closes, or nil for a shutter
	// that is closed
	closing      *maths.Subspace
	initialMedia []trace.InitialMedium
}

// view is the header's fields the shot's eye, lens and shutter make, for
// a picture of so many samples.
func (take shot) view(fine [2]int) values {
	return viewValues(take.eye, take.lens, take.shutter(), fine[0], fine[1])
}

// shutter is the shot's shutter: closed where it has no closing eye.
func (take shot) shutter() shutter {
	return shutter{closing: take.closing, side: take.side, at: take.at}
}

// frameKey is what must be the last frame's for only the camera to have
// moved.
type frameKey struct {
	world               *thing.World
	generation          uint64
	at                  time.Duration
	width, height, side int
	mode                traversal
	initialMedia        [32]byte
}

// lastFrame is what a camera-only frame draws from: the last retained
// frame's key, its header as the device holds it, its paints and records
// for the finish, and its scene's counts for the profile.
type lastFrame struct {
	key     frameKey
	header  []byte
	paints  []painting
	records int
	counted timing
}

// traversals is each known Mode's traversal.
var traversals = [modes]traversal{Software: softwareTraversal,
	NativeShadows: nativeShadowTraversal, NativeMeshes: nativeTraversal,
	CachedLighting: cachedTraversal, FastLighting: fastTraversal,
	FasterLighting: fasterTraversal}

// NewFrames is a drawer for pictures in one mode. It opens nothing on the
// device until its first draw.
func NewFrames(mode Mode) (*Frames, error) {
	if !mode.known() {
		return nil, fmt.Errorf("%w: %d", errNoMode, mode)
	}
	return newFrames(traversals[mode]), nil
}

// NewCompositeFrames retains a drawer for one contribution to a composite.
// Only pinhole views without an open shutter are admitted for these passes.
func NewCompositeFrames(mode Mode, pass CompositePass) (*Frames, error) {
	if err := validPass(mode, pass); err != nil {
		return nil, err
	}
	frames, err := NewFrames(mode)
	if err == nil {
		frames.pass = pass
	}
	return frames, err
}

// Transmission is the last VolumeTransport draw's RGB transmission. Like
// Draw's frame, it is caller-readable and must not be changed during a draw.
// It is nil for other passes and after an error or Close.
func (frames *Frames) Transmission() *frame.Frame { return frames.transmission }

// newFrames is a drawer for a traversal already known.
func newFrames(mode traversal) *Frames {
	return &Frames{mode: mode, trees: newKeptTrees()}
}

// Draw draws the world at a time from eye through lens, width by height
// pixels of side by side samples, as the per-call function of the
// drawer's mode draws it, and reports its costs.
//
// generation is the caller's word for the world's content: the same
// generation with the same world and time says nothing in it changed.
func (frames *Frames) Draw(world *thing.World, generation uint64,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width,
	height, side int) (*frame.Frame, Profile, error) {
	return frames.DrawWithInitialMedia(world, generation, eye, lens, at,
		width, height, side, nil)
}

// DrawWithInitialMedia is Draw with an explicit ordered stack of clear media
// containing the camera. The state participates in retained-frame identity.
func (frames *Frames) DrawWithInitialMedia(world *thing.World,
	generation uint64, eye maths.Subspace, lens film.Sensor, at time.Duration,
	width, height, side int, media []trace.InitialMedium,
) (*frame.Frame, Profile, error) {
	take := shot{world: world, generation: generation, eye: eye,
		lens: lens, at: at, width: width, height: height,
		side: max(1, side), initialMedia: slices.Clone(media)}
	return frames.measured(take)
}

// DrawOpen is Draw with the shutter held open while the eye moves from
// eye to closing: each sample looks from the eye at its own time between
// the two, so the picture is blurred by the move at the cost of one draw.
//
// Only the eye moves; the world is the one at the time given. A pixel's
// samples share the shutter between them, so the blur wants several
// samples a side. An eye that does not move draws as Draw does.
func (frames *Frames) DrawOpen(world *thing.World, generation uint64,
	eye, closing maths.Subspace, lens film.Sensor, at time.Duration,
	width, height, side int) (*frame.Frame, Profile, error) {
	return frames.DrawOpenWithInitialMedia(world, generation, eye, closing,
		lens, at, width, height, side, nil)
}

// DrawOpenWithInitialMedia is DrawOpen with a fixed explicit camera stack.
// Moving a camera through or across an interface needs a time-varying stack,
// which this API does not infer or support.
func (frames *Frames) DrawOpenWithInitialMedia(world *thing.World,
	generation uint64, eye, closing maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
	media []trace.InitialMedium,
) (*frame.Frame, Profile, error) {
	take := shot{world: world, generation: generation, eye: eye,
		lens: lens, at: at, width: width, height: height,
		side: max(1, side), closing: &closing,
		initialMedia: slices.Clone(media)}
	return frames.measured(take)
}

// measured draws a shot where the drawer is usable, and reports its
// costs; a draw that fails ends the drawer.
func (frames *Frames) measured(take shot) (*frame.Frame, Profile, error) {
	if err := frames.usable(take.world); err != nil {
		return nil, Profile{}, err
	}
	if len(take.initialMedia) > 0 || frames.pass != CompletePicture {
		hasNamedOptics, err := trace.PrepareClearState(take.world,
			take.initialMedia)
		if err != nil {
			return nil, Profile{}, err
		}
		if len(take.initialMedia) > 0 && take.closing != nil {
			return nil, Profile{}, unsupported{"moving shutter with an explicit camera medium stack"}
		}
		if frames.pass != CompletePicture &&
			(len(take.initialMedia) > 0 || hasNamedOptics) {
			return nil, Profile{}, unsupported{"clear-medium transport in a composite pass"}
		}
	}
	frames.transmission = nil
	if frames.pass != CompletePicture {
		if !compositePinhole(take.lens) || take.closing != nil {
			return nil, Profile{}, fmt.Errorf("gpu: composite passes require a closed pinhole shutter")
		}
		if frames.pass == VolumeTransport {
			take.transmission = &frames.transmission
		}
	}
	start := time.Now()
	picture, spent, err := frames.draw(take, frames.mode)
	if err != nil {
		frames.fail(err)
		frames.transmission = nil
	}
	return picture, spent.profile(time.Since(start)), err
}

// Close lets the device state go. Closing twice is harmless.
func (frames *Frames) Close() {
	frames.letGo()
	frames.trees = newKeptTrees()
	frames.closed = true
	frames.transmission = nil
}

// usable refuses a draw on a closed or failed drawer, or of no world.
func (frames *Frames) usable(world *thing.World) error {
	switch {
	case frames == nil || frames.closed:
		return ErrFramesClosed
	case frames.failed != nil:
		return fmt.Errorf("%w: %w", ErrFramesFailed, frames.failed)
	case world == nil:
		return errNoWorld
	}
	return nil
}

// fail lets the device state go and keeps the error that ended it.
func (frames *Frames) fail(err error) {
	frames.letGo()
	frames.trees = newKeptTrees()
	frames.failed = err
}

// letGo closes whatever is held on the device and forgets the last frame.
func (frames *Frames) letGo() {
	if frames.held != nil {
		frames.held.close()
		frames.held = nil
	}
	frames.last = nil
}

// frameRule is a way of drawing a frame and when it applies.
type frameRule struct {
	why     string
	applies func(frames *Frames, take shot, mode traversal) bool
	draw    func(frames *Frames, take shot,
		mode traversal) (*frame.Frame, timing, error)
}

// frameRules are the ways other than retained, in the order they are
// asked; the first that applies draws the frame.
var frameRules = []frameRule{
	{"a lens whose rays the cpu makes: they are a per-frame input",
		raysGiven, (*Frames).perCall},
	{"the camera alone moved", cameraOnly, (*Frames).moved},
}

// draw draws a frame by the first rule that applies, or retained.
func (frames *Frames) draw(take shot, mode traversal) (*frame.Frame,
	timing, error) {
	for _, rule := range frameRules {
		if rule.applies(frames, take, mode) {
			return rule.draw(frames, take, mode)
		}
	}
	return frames.retained(take, mode)
}

// raysGiven is whether the frame's lens is one whose rays the cpu makes.
func raysGiven(_ *Frames, take shot, _ traversal) bool {
	return !kernelMade(take.lens)
}

// cameraOnly is whether only the camera can have moved since the last
// retained frame, which is still on the device.
func cameraOnly(frames *Frames, take shot, mode traversal) bool {
	return frames.pass == CompletePicture && frames.held != nil && frames.last != nil &&
		frames.last.key == keyOf(take, mode)
}

// keyOf is a frame's key in a mode.
func keyOf(take shot, mode traversal) frameKey {
	return frameKey{world: take.world, generation: take.generation,
		at: take.at, width: take.width, height: take.height,
		side: take.side, mode: mode,
		initialMedia: initialMediaKey(take.initialMedia)}
}

// initialMediaKey hashes a take's initial clear media into a key, so
// takes with different media never share one.
func initialMediaKey(media []trace.InitialMedium) [32]byte {
	if len(media) == 0 {
		return [32]byte{}
	}
	hash := sha256.New()
	var word [8]byte
	put := func(value uint64) {
		binary.LittleEndian.PutUint64(word[:], value)
		_, _ = hash.Write(word[:])
	}
	put(uint64(len(media)))
	for _, each := range media {
		put(math.Float64bits(each.Index))
		put(uint64(int64(each.Rank)))
		name := each.ClearMedium.Name()
		put(uint64(len(name)))
		_, _ = hash.Write([]byte(name))
		coefficients := each.ClearMedium.Extinction()
		put(math.Float64bits(coefficients.R))
		put(math.Float64bits(coefficients.G))
		put(math.Float64bits(coefficients.B))
		put(math.Float64bits(each.ClearMedium.MetresPerUnit()))
		scattering := each.ClearMedium.Scattering()
		put(math.Float64bits(scattering.R))
		put(math.Float64bits(scattering.G))
		put(math.Float64bits(scattering.B))
		put(math.Float64bits(each.ClearMedium.Phase()))
		put(uint64(each.ClearMedium.InscatterSteps()))
		put(math.Float64bits(each.ClearMedium.MaxInscatterDistanceMetres()))
	}
	var key [32]byte
	copy(key[:], hash.Sum(nil))
	return key
}

// perCall draws the frame as the per-call function does, touching
// nothing held. The next frame is not a camera move from this one. It
// refuses a shutter open over a moving eye: its rays have one eye.
func (frames *Frames) perCall(take shot, mode traversal) (*frame.Frame,
	timing, error) {
	frames.last = nil
	if take.shutter().over(take.eye) {
		return nil, timing{}, errShutterOverRays
	}
	return sampledLightingGridWithInitialMedia(take.world, nil, take.eye,
		take.lens, take.at, take.width, take.height, take.side, mode,
		cellsFor(mode), take.initialMedia)
}

// remember keeps what a camera-only frame after this one draws from.
func (frames *Frames) remember(take shot, mode traversal, encoded *scene,
	spent timing) {
	var counted timing
	counted.shells, counted.volumeSky = spent.shells, spent.volumeSky
	counted.environmentNodes = spent.environmentNodes
	counted.inputBytes = spent.inputBytes
	frames.last = &lastFrame{key: keyOf(take, mode),
		header: slices.Clone(encoded.header.bytes),
		paints: encoded.paints, records: encoded.records(),
		counted: counted}
}
