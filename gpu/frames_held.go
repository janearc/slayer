//go:build darwin

package gpu

import (
	"slices"
	"time"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// held is what a Frames drawer keeps on the device between frames: the
// trace kernel, with its hierarchy in the native modes, and a stream
// holding every input. key and layout say what they were opened for.
type held struct {
	kernel *metal.Kernel
	stream *metal.GridStream
	key    heldKey
	layout metal.Layout
	inputs int
}

// heldKey is what a kernel and its stream are opened for: the mode, the
// kernel's source, the picture's size in samples and the numbers each
// sample gives back.
type heldKey struct {
	mode               traversal
	text               string
	width, height, per int
}

// trianglesAt is the triangles' place among the scene's inputs, as
// inputs gives them and as trace.metal binds them: buffer(2). The native
// hierarchy is built from these bytes.
const trianglesAt = 2

// nodesAt and bendsAt are the boxes' and the bends' places among the
// inputs: buffer(3) and buffer(12).
const (
	nodesAt = 3
	bendsAt = 12
)

// headerAt is the header's place among the inputs: buffer(0).
const headerAt = 0

// candidate is a frame's encoded scene as the stale rules see it: its
// key, its hierarchy's layout, and whether the device's triangles are
// already its own.
type candidate struct {
	key       heldKey
	layout    metal.Layout
	triangles bool
}

// staleRule is a reason the held state cannot draw a frame, or, for a
// hierarchy rule, a reason its hierarchy must be built again.
type staleRule struct {
	why   string
	stale func(state *held, next candidate) bool
}

// staleRules are every reason to open the device state again, in the
// one place they are written down.
var staleRules = []staleRule{
	{"another mode", func(state *held, next candidate) bool {
		return state.key.mode != next.key.mode
	}},
	{"another kernel source", func(state *held, next candidate) bool {
		return state.key.text != next.key.text
	}},
	{"another size", func(state *held, next candidate) bool {
		return state.key.width != next.key.width ||
			state.key.height != next.key.height
	}},
	{"another sample layout", func(state *held, next candidate) bool {
		return state.key.per != next.key.per
	}},
}

// hierarchyRules are every reason to build a kept kernel's hierarchy
// again, on that kernel, before the frame: as the per-call path builds
// one for every frame, from the frame's own plan and triangles.
var hierarchyRules = []staleRule{
	{"another hierarchy", func(state *held, next candidate) bool {
		return !state.layout.Same(next.layout)
	}},
	{"other triangles under the hierarchy",
		func(state *held, next candidate) bool {
			return state.key.mode != softwareTraversal && !next.triangles
		}},
}

// noneHeld is why a frame opens the device state when there is none.
const noneHeld = "nothing held"

// stale is why the held state cannot draw the next frame, or empty.
func (state *held) stale(next candidate) string {
	if state == nil {
		return noneHeld
	}
	return state.first(staleRules, next)
}

// first is the first of the rules that holds for the next frame, or
// empty.
func (state *held) first(rules []staleRule, next candidate) string {
	for _, rule := range rules {
		if rule.stale(state, next) {
			return rule.why
		}
	}
	return ""
}

// retained draws a frame over the held state, opening it again first
// where a stale rule says it cannot draw this frame.
func (frames *Frames) retained(take shot, mode traversal) (*frame.Frame,
	timing, error) {
	var spent timing
	if lowRetentionEnabled() {
		frames.trees = newKeptTrees()
	}
	frames.last = nil
	start := time.Now()
	width, height, err := checkedFineSize(take.width, take.height,
		take.side)
	if err != nil {
		spent.encode = time.Since(start)
		return nil, spent, err
	}
	fine := [2]int{width, height}
	encoded, firsts, err := preparedWithInitialMedia(take.world, nil,
		take.eye, take.lens, take.at, fine, mode, cellsFor(mode),
		frames.trees, take.initialMedia, &spent)
	if err != nil {
		return nil, spent, err
	}
	// the encoder's shutter is closed; the shot's is written over it
	headerStruct.patch(encoded.header.bytes, take.view(fine))
	encoded.spreadOver(take.world, take.side)
	encoded.shareSkyFor(mode, take.side)
	inputs := encoded.inputs()
	if mode.primary() {
		inputs = append(inputs, firsts)
	}
	state, err := frames.hold(encoded, inputs, mode, fine, &spent)
	if err != nil {
		return nil, spent, err
	}
	picture, err := state.traced(take, encoded, inputs, fine, &spent)
	if err != nil {
		return nil, spent, err
	}
	frames.remember(take, mode, encoded, spent)
	frames.trees.turn()
	return picture, spent, nil
}

// traced runs the frame's light bakes and its trace over the held
// inputs, and finishes the picture.
func (state *held) traced(take shot, encoded *scene, inputs [][]byte,
	fine [2]int, spent *timing) (*frame.Frame, error) {
	spent.device = state.kernel.Device()
	base := state.kernel.GPUTime()
	run := state.grids(inputs, spent)
	memoryStage("traced-lit-before")
	if err := encoded.lit(state.kernel, run, state.key.mode,
		spent); err != nil {
		return nil, err
	}
	memoryStage("traced-lit-after")
	for _, input := range inputs {
		spent.inputBytes += int64(len(input))
	}
	start := time.Now()
	band := newBand(take.world, encoded.paints, encoded.records(),
		take.eye, take.lens, fine, take.width, take.height, take.side)
	memoryStage("traced-band-created")
	consume := band.consume
	if take.transmission != nil {
		// Transport has no CPU-painted surfaces or sky. Its otherwise
		// unused sky-weight slots carry transmission in the same dispatch.
		band.finish.world = &thing.World{}
		band.finish.discs = nil
		transmitted := newBand(&thing.World{}, nil, encoded.records(),
			take.eye, take.lens, fine, take.width, take.height, take.side)
		*take.transmission = transmitted.picture
		consume = func(first, down int, samples []float32) error {
			if err := band.consume(first, down, samples); err != nil {
				return err
			}
			for offset := 0; offset < len(samples); offset += encoded.perSample() {
				copy(samples[offset:offset+3], samples[offset+4:offset+7])
				clear(samples[offset+4 : offset+7])
			}
			return transmitted.consume(first, down, samples)
		}
	}
	setup := time.Since(start)
	start = time.Now()
	err := run(fine[0], fine[1], encoded.perSample(), consume)
	sending := spent.dispatch
	spent.banded(&band, time.Since(start), setup)
	spent.dispatch += sending
	spent.trace = traceDeviceTime(state.kernel.GPUTime(), base,
		spent.lighting, spent.environment)
	return band.picture, err
}

// hold is the held state for this frame: kept, its inputs sent by the
// grids as each runs, or opened again with all of them. It times the
// kernel source as compiling, the hierarchy's plan and build as
// acceleration, and the sending as dispatch.
func (frames *Frames) hold(encoded *scene, inputs [][]byte, mode traversal,
	fine [2]int, spent *timing) (*held, error) {
	start := time.Now()
	key := heldKey{mode: mode, text: passSource(encoded.source(mode), frames.pass), width: fine[0],
		height: fine[1], per: encoded.perSample()}
	spent.compile = time.Since(start)
	start = time.Now()
	memoryStage("native-plan-before")
	plan, root, err := planned(encoded, mode)
	memoryStage("native-plan-after")
	spent.acceleration = time.Since(start)
	if err != nil {
		return nil, err
	}
	next := candidate{key: key, layout: plan.Layout()}
	if frames.held != nil && mode != softwareTraversal {
		next.triangles, err = frames.held.stream.Same(trianglesAt,
			inputs[trianglesAt])
		if err != nil {
			return nil, err
		}
	}
	frames.why, frames.rebuilt = frames.held.stale(next), ""
	if frames.why == "" {
		// the inputs are sent by the grids, each before it runs
		spent.retained.Held = frames.held.inputs
		frames.rebuilt = frames.held.first(hierarchyRules, next)
		if frames.rebuilt == "" {
			memoryStage("native-build-reused")
			return frames.held, nil
		}
		return frames.held, frames.held.rebuild(next, plan, root, spent)
	}
	frames.letGo()
	frames.held, err = openHeld(next, plan, root, inputs, spent)
	return frames.held, err
}

// rebuild builds the kernel's hierarchy again from the frame's plan, in
// the place of the one it has, and keeps its layout. The build reads the
// triangles from the stream's input, which it first makes hold them, so
// the device holds them once; the sync that follows finds them sent.
func (state *held) rebuild(next candidate, plan *metal.Acceleration,
	root int, spent *timing) error {
	memoryStage("native-rebuild-before")
	start, before := time.Now(), state.kernel.GPUTime()
	sent, err := state.kernel.RebuildOver(plan, root, state.stream,
		trianglesAt)
	if sent > 0 {
		spent.retained.Sent++
		spent.retained.SentBytes += sent
	}
	memoryStage("native-rebuild-after")
	spent.acceleration += time.Since(start)
	spent.accelerationGPU = state.kernel.GPUTime() - before
	if err == nil {
		state.layout = next.layout
	}
	return err
}

// planned is the frame's native hierarchy plan and root in a native
// mode, with the build hierarchy let go after it, as the per-call path
// does; none in the software mode.
func planned(encoded *scene, mode traversal) (*metal.Acceleration, int,
	error) {
	if mode == softwareTraversal {
		return nil, 0, nil
	}
	plan, root, err := encoded.accelerationPlan()
	if err == nil {
		encoded.discardBuildHierarchy()
	}
	return plan, root, err
}

// openHeld compiles the kernel, sends every input to a new stream, and
// builds the hierarchy where there is a plan, over the stream's triangles
// so the device holds them once. The stream comes first for that.
func openHeld(next candidate, plan *metal.Acceleration, root int,
	inputs [][]byte, spent *timing) (*held, error) {
	memoryStage("metal-open-before")
	start := time.Now()
	kernel, err := metal.Open(next.key.text, "trace")
	spent.compile += time.Since(start)
	if err != nil {
		return nil, err
	}
	kernel.FixBands(bandRows)
	start = time.Now()
	metal.ReportDeviceMemoryStage("metal-grid-before")
	stream, err := kernel.OpenGrid(next.key.width, next.key.height, rows,
		next.key.per, inputs)
	memoryStage("metal-grid-after")
	metal.ReportDeviceMemoryStage("metal-grid-after")
	spent.dispatch += time.Since(start)
	if err != nil {
		kernel.Close()
		return nil, err
	}
	spent.retained.Held = len(inputs)
	for _, input := range inputs {
		spent.retained.Sent++
		spent.retained.SentBytes += int64(len(input))
	}
	if err := accelerateOver(kernel, stream, plan, root, spent); err != nil {
		stream.Close()
		kernel.Close()
		return nil, err
	}
	return &held{kernel: kernel, stream: stream, key: next.key,
		layout: next.layout, inputs: len(inputs)}, nil
}

// accelerateOver builds the kernel's first hierarchy from the frame's
// plan over the stream's triangles, which the stream already holds; no
// plan, in the software mode, builds nothing.
func accelerateOver(kernel *metal.Kernel, stream *metal.GridStream,
	plan *metal.Acceleration, root int, spent *timing) error {
	if plan == nil {
		return nil
	}
	memoryStage("native-build-before")
	start := time.Now()
	_, err := kernel.AccelerateOver(plan, root, stream, trianglesAt)
	memoryStage("native-build-after")
	spent.acceleration += time.Since(start)
	spent.accelerationGPU = kernel.GPUTime()
	return err
}

// sync sends each input whose bytes the device does not already hold.
func (state *held) sync(inputs [][]byte, spent *timing) error {
	spent.retained.Held = state.inputs
	for index, input := range inputs {
		if err := state.send(index, input, spent); err != nil {
			return err
		}
	}
	return nil
}

// send makes the device's copy of one input the host's, copying only
// the parts that differ, and counts it where it sent any.
func (state *held) send(index int, input []byte, spent *timing) error {
	sent, err := state.stream.Refresh(index, input)
	if sent > 0 {
		spent.retained.Sent++
		spent.retained.SentBytes += sent
	}
	return err
}

// grids runs the kernel's selected entry point over the held inputs,
// first making them what the host holds now: a light bake changes the
// header and the boxes between its grids, as the per-call path's copies
// see them.
func (state *held) grids(inputs [][]byte, spent *timing) grids {
	return func(width, height, per int,
		consume func(first, down int, samples []float32) error) error {
		memoryStage("grid-sync-before")
		if err := state.sync(inputs, spent); err != nil {
			return err
		}
		memoryStage("grid-sync-after")
		if err := state.stream.Reshape(width, height, rows,
			per); err != nil {
			return err
		}
		memoryStage("grid-reshape-after")
		memoryStage("grid-bands-before")
		err := state.stream.DrawBands(consume)
		memoryStage("grid-bands-after")
		return err
	}
}

// close lets the stream and then its kernel go.
func (state *held) close() {
	state.stream.Close()
	state.kernel.Close()
}

// moved draws a camera move over the last frame's scene, still on the
// device: the header's fields of the eye, the lens and the shutter
// rewritten and sent, nothing encoded.
func (frames *Frames) moved(take shot, _ traversal) (*frame.Frame, timing,
	error) {
	state, last := frames.held, frames.last
	spent := last.counted
	spent.device = state.kernel.Device()
	spent.retained = FrameInputs{Held: state.inputs, Patched: true}
	fine := [2]int{state.key.width, state.key.height}
	header := slices.Clone(last.header)
	headerStruct.patch(header, take.view(fine))
	start := time.Now()
	band := newBand(take.world, last.paints, last.records, take.eye,
		take.lens, fine, take.width, take.height, take.side)
	setup := time.Since(start)
	before := state.kernel.GPUTime()
	start = time.Now()
	err := state.send(headerAt, header, &spent)
	if err == nil {
		err = state.stream.Reshape(fine[0], fine[1], rows, state.key.per)
	}
	if err == nil {
		err = state.stream.DrawBands(band.consume)
	}
	spent.banded(&band, time.Since(start), setup)
	spent.trace = state.kernel.GPUTime() - before
	if err != nil {
		return nil, spent, err
	}
	last.header = header
	return band.picture, spent, nil
}
