//go:build darwin

package gpu

import (
	"bytes"
	"errors"
	"math"
	"time"
	"unsafe"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// restirSettings are how restir finds light (doc/restir.md). The zero
// value is not drawn; use defaultRestir.
type restirSettings struct {
	// Candidates is the lamps each sample takes from the light tree.
	Candidates int
	// Temporal is whether the last frame's reservoirs are joined.
	Temporal bool
	// Taps is how many neighbours' reservoirs are joined, and Radius how
	// many samples away they may be.
	Taps   int
	Radius float32
	// Visibility is whether the lamp kept from the initial candidates is shot
	// at, and dropped if hidden, before it is reused (the paper's way).
	// The final shadow ray is cast either way.
	Visibility bool
	// Cap is the most times a reservoir's count may be the one it joins.
	Cap float32
}

// defaultRestir is four candidates, temporal reuse, three neighbours
// within sixteen samples, counts capped at twenty times.
func defaultRestir() restirSettings {
	return restirSettings{Candidates: 4, Temporal: true, Taps: 3,
		Radius: 16, Cap: 20}
}

// errRestirSettings is settings that cannot be drawn.
var errRestirSettings = errors.New("gpu: restir needs at least one " +
	"candidate, no negative taps, and a positive cap and radius")

// valid is nil if the settings can be drawn, and errRestirSettings if not.
func (set restirSettings) valid() error {
	if set.Candidates < 1 || set.Taps < 0 || !(set.Cap > 0) ||
		!(set.Radius > 0) {
		return errRestirSettings
	}
	return nil
}

// Restir draws pictures lit by restir and keeps what the next frame's
// temporal reuse reads: the last frame's G-buffer and reservoirs, its
// eye, and the light tree for the lamps it saw. One Restir draws one film
// at one size; a frame of another size starts over, with no temporal
// reuse. It is not safe for concurrent use.
type Restir struct {
	Settings restirSettings

	frame        int
	size         [2]int
	lamps        []byte
	tree         lightTree
	treeFloats   []float32
	state        []float32
	prevEye      [16]float32
	prev         int // which of the two G-buffers holds the last frame's
	prevRes      int // which of the three reservoir regions does
	havePrevious bool
}

// NewRestir is a drawer with the default settings.
func NewRestir() *Restir { return &Restir{Settings: defaultRestir()} }

// layout says where each part is among the state's floats.
type restirLayout struct {
	tree           int
	gbuf           [2]int
	res            [3]int
	total, samples int
}

// layoutFor is where each part sits among the state's floats, for a number
// of samples and of tree nodes.
func layoutFor(samples, nodeCount int) restirLayout {
	var at restirLayout
	at.samples = samples
	at.tree = restirCtrlFloats
	next := at.tree + nodeCount*nodeFloats
	for i := range at.gbuf {
		at.gbuf[i] = next
		next += samples * gbufferFloats
	}
	for i := range at.res {
		at.res[i] = next
		next += samples * reservoirFloats
	}
	at.total = next
	return at
}

// restirCtrlFloats is the control block's floats, restir_ctrl in the
// kernel; its slots are the rc_ constants there.
const restirCtrlFloats = 64

// the control block's slots, as the kernel names them
const (
	rcFrame = iota
	rcCandidates
	rcTemporal
	rcSpatial
	rcRadius
	rcCap
	rcNodes
	rcSkipSky
	rcWidth
	rcHeight
	rcTree
	rcGbuf
	rcPrevGbuf
	rcResA
	rcResB
	rcPrevRes
	rcEye
)

// rcVisible is slot 40, past the eye
const rcVisible = 40

// (rcEye above is the last slot of the iota run)
var _ = rcEye

// restirBits puts a whole number in a float's place, as its bits: the kernel
// reads such a slot with as_type, since a float holds a whole number
// exactly only to 2^24 and a picture's layout runs past it.
func restirBits(n int) float32 { return math.Float32frombits(uint32(int32(n))) }

// Draw is the picture of the world at time at, width by height pixels of
// side by side samples, lit by restir. It refuses what restir does not
// draw by name: a lens that is not a pinhole, gravity, a flood, sun, beam
// or polar lamp, and boxed or optical media.
func (drawer *Restir) Draw(world *thing.World, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	started := time.Now()
	picture, spent, err := drawer.draw(world, eye, lens, at, width, height, side)
	return picture, spent.profile(time.Since(started)), err
}

// draw is the work of Draw: the picture, and where the time went in each
// stage.
func (drawer *Restir) draw(world *thing.World, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, side int,
) (*frame.Frame, timing, error) {
	var spent timing
	if err := drawer.Settings.valid(); err != nil {
		return nil, spent, err
	}
	if world == nil {
		return nil, spent, errNoWorld
	}
	if _, pinhole := pinholeOf(lens); !pinhole && lens != nil {
		return nil, spent, unsupported{"restir through a lens that is not a pinhole"}
	}
	if len(world.Fields) > 0 {
		return nil, spent, unsupported{"restir with gravity"}
	}
	side = max(1, side)
	fineWidth, fineHeight, err := checkedFineSize(width, height, side)
	if err != nil {
		return nil, spent, err
	}
	encoded, firsts, err := prepared(world, nil, eye, lens, at,
		[2]int{fineWidth, fineHeight}, restirTraversal, 0, nil, &spent)
	if err != nil {
		return nil, spent, err
	}
	encoded.sideFor(lens, side)
	if encoded.looked || encoded.through != nil || encoded.opticalState ||
		encoded.counts.volumes > 0 {
		return nil, spent, unsupported{"restir with media, optical state, " +
			"or rays made through a lens"}
	}
	start := time.Now()
	if err := drawer.treeFor(encoded); err != nil {
		return nil, spent, err
	}
	spent.restirTree = time.Since(start)

	text, err := restirKernel(encoded.source(restirTraversal))
	if err != nil {
		return nil, spent, err
	}
	start = time.Now()
	kernel, err := metal.Open(text, "restir_gbuffer")
	spent.compile = time.Since(start)
	if err != nil {
		return nil, spent, err
	}
	defer kernel.Close()
	kernel.FixBands(bandRows)
	start = time.Now()
	plan, root, err := encoded.accelerationPlan()
	if err == nil {
		encoded.discardBuildHierarchy()
		err = kernel.Accelerate(plan, root)
	}
	spent.acceleration = time.Since(start)
	spent.accelerationGPU = kernel.GPUTime()
	if err != nil {
		return nil, spent, err
	}
	spent.device = kernel.Device()
	inputs := encoded.inputs()
	inputs = append(inputs, firsts)
	for _, input := range inputs {
		spent.inputBytes += int64(len(input))
	}

	samples := fineWidth * fineHeight
	at2 := layoutFor(samples, len(drawer.tree.nodes))
	fresh := drawer.size != [2]int{fineWidth, fineHeight} ||
		len(drawer.state) != at2.total || !drawer.havePrevious
	if fresh {
		drawer.state = make([]float32, at2.total)
		drawer.frame, drawer.prev, drawer.prevRes = 0, 0, 0
		drawer.size = [2]int{fineWidth, fineHeight}
	}
	state := drawer.state
	copy(state[at2.tree:], drawer.treeFloats)
	cur := 1 - drawer.prev
	resA := (drawer.prevRes + 1) % 3
	resB := (drawer.prevRes + 2) % 3
	set := drawer.Settings
	ctrl := state[:restirCtrlFloats]
	ctrl[rcFrame] = restirBits(drawer.frame)
	ctrl[rcVisible] = restirBits(flag(set.Visibility))
	ctrl[rcCandidates] = restirBits(set.Candidates)
	ctrl[rcTemporal] = restirBits(flag(set.Temporal))
	ctrl[rcSpatial] = restirBits(set.Taps)
	ctrl[rcRadius] = set.Radius
	ctrl[rcCap] = set.Cap
	ctrl[rcNodes] = restirBits(len(drawer.tree.nodes))
	ctrl[rcSkipSky] = restirBits(flag(allZero(encoded.sky.bytes)))
	ctrl[rcWidth] = restirBits(fineWidth)
	ctrl[rcHeight] = restirBits(fineHeight)
	ctrl[rcTree] = restirBits(at2.tree)
	ctrl[rcGbuf] = restirBits(at2.gbuf[cur])
	ctrl[rcPrevGbuf] = restirBits(at2.gbuf[drawer.prev])
	ctrl[rcResA] = restirBits(at2.res[resA])
	ctrl[rcResB] = restirBits(at2.res[resB])
	ctrl[rcPrevRes] = restirBits(at2.res[drawer.prevRes])
	copy(ctrl[rcEye:rcEye+16], drawer.prevEye[:])

	// the inputs' rays slot is the restir state, copied to the device for
	// each pass as every per-call input is
	inputs[raysAt] = restirBytes(state)
	spent.inputBytes += int64(len(inputs[raysAt]))

	pass := func(name string, per int, into []float32) error {
		if err := kernel.Function(name); err != nil {
			return err
		}
		return kernel.GridBands(fineWidth, fineHeight, rows, per, inputs,
			func(first, down int, bandSamples []float32) error {
				copy(into[first*fineWidth*per:], bandSamples[:down*fineWidth*per])
				return nil
			})
	}
	start = time.Now()
	before := kernel.GPUTime()
	if err = pass("restir_gbuffer", gbufferFloats,
		state[at2.gbuf[cur]:at2.gbuf[cur]+samples*gbufferFloats]); err != nil {
		return nil, spent, err
	}
	if err = pass("restir_resample", reservoirFloats,
		state[at2.res[resA]:at2.res[resA]+samples*reservoirFloats]); err != nil {
		return nil, spent, err
	}
	if err = pass("restir_spatial", reservoirFloats,
		state[at2.res[resB]:at2.res[resB]+samples*reservoirFloats]); err != nil {
		return nil, spent, err
	}
	spent.restirWall = time.Since(start)
	spent.restirGPU = kernel.GPUTime() - before
	if err = kernel.Function("trace"); err != nil {
		return nil, spent, err
	}

	start = time.Now()
	band := newBand(world, encoded.paints, encoded.records(), eye, lens,
		[2]int{fineWidth, fineHeight}, width, height, side)
	setup := time.Since(start)
	start = time.Now()
	made, err := encoded.drawn(kernel, inputs, fineWidth, fineHeight, &band)
	spent.inputBytes += made
	spent.banded(&band, time.Since(start), setup)
	spent.trace = traceDeviceTime(kernel.GPUTime(), before+spent.restirGPU,
		0, 0)
	if err != nil {
		return nil, spent, err
	}

	// what the next frame reads: this frame's eye, G-buffer and reservoirs
	copy(drawer.prevEye[:], eyeFloats(encoded.header.bytes))
	drawer.prev, drawer.prevRes = cur, resB
	drawer.frame++
	drawer.havePrevious = true
	spent.restir = true
	spent.restirLamps = len(encoded.lamps.bytes) / (lampFloats * 4)
	spent.restirNodes = len(drawer.tree.nodes)
	return band.picture, spent, nil
}

// treeFor builds the light tree of the scene's lamps, or keeps the last
// one when the lamps are the same bytes.
func (drawer *Restir) treeFor(encoded *scene) error {
	if drawer.tree.nodes != nil && bytes.Equal(drawer.lamps, encoded.lamps.bytes) {
		return nil
	}
	places, powers, err := crowdOf(restirFloats(encoded.lamps.bytes))
	if err != nil {
		return err
	}
	tree, err := buildLightTree(places, powers)
	if err != nil {
		return err
	}
	drawer.tree = tree
	drawer.treeFloats = tree.floats()
	drawer.lamps = append(drawer.lamps[:0], encoded.lamps.bytes...)
	// a different crowd is a different history
	drawer.havePrevious = false
	return nil
}

// eyeFloats is the header's eye as restir keeps it for the next frame:
// the place, the three axes and the lens, each three numbers but the lens's
// four, out of the header's first five rows.
func eyeFloats(header []byte) []float32 {
	row := restirFloats(header[:5*16])
	out := make([]float32, 0, 16)
	for _, at := range []int{0, 4, 8, 12} {
		out = append(out, row[at:at+3]...)
	}
	return append(out, row[16:20]...)
}

// allZero is whether every byte of data is nought.
func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// restirBytes and restirFloats view floats and bytes as each other.
func restirBytes(floats []float32) []byte {
	if len(floats) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&floats[0])), len(floats)*4)
}

// restirFloats is the bytes read as float32 values, four bytes to each; a
// last part of fewer than four bytes is dropped.
func restirFloats(data []byte) []float32 {
	floats := make([]float32, len(data)/4)
	copy(restirBytes(floats), data)
	return floats
}
