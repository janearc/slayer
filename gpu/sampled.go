//go:build darwin

package gpu

import (
	_ "embed"
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

//go:embed trace.metal
var traceSource string

//go:embed ring.metal
var ringSource string

//go:embed wake.metal
var wakeSource string

//go:embed propeller_legacy.metal
var propellerLegacySource string

//go:embed propeller.metal
var propellerSource string

//go:embed fizz.metal
var fizzSource string

// probeVolumes sizes the kernel's crossing arrays for source(): the
// probes and the wisp bake read volumes by index and never walk
// most_volumes, so any roomy size serves them alike.
const probeVolumes = 32

// sourceWith builds kernel source sized for the requested number of CPU
// texture records per sample and the scene's own count of volumes.
func sourceWith(records, volumes int) string {
	return sourceWithState(records, volumes, false)
}

// sourceWithOpticalState is an opt-in ABI extension for named extinction and
// explicit initial media. Ordinary source omits these bindings; independent
// rendering features can still extend its source and their own layouts.
func sourceWithOpticalState(records, volumes int) string {
	return sourceWithState(records, volumes, true)
}

// sourceWithState builds the Metal source for the given record and volume
// counts. optical selects the optical variant of the source.
func sourceWithState(records, volumes int, optical bool) string {
	text := diagnosticDefines() + metal.GeodesicLibrary() +
		metal.NoiseLibrary + prelude(records, volumes)
	if optical {
		text += clearMediumStruct.msl() + opticalStateDeclarations +
			ringSource + wakeSource + propellerSource + fizzSource + traceSource
		return "#define SLAYER_OPTICAL_MEDIA 1\n" + text
	}
	return stripOpticalMedia(text + ringSource + wakeSource + propellerLegacySource + traceSource)
}

// sourceWithPropellerDispersal admits standalone advanced density fields
// without changing the named-medium ABI or ordinary legacy source bytes.
func sourceWithPropellerDispersal(records, volumes int) string {
	return propellerDispersalNumbers() + strings.Replace(sourceWith(records, volumes), propellerLegacySource, propellerSource, 1)
}

// stripOpticalMedia removes
// opt-in shader alternatives while preserving the
// source bytes of each #else branch. This keeps optical-state bindings out
// of kernels for scenes without optical media.
func stripOpticalMedia(source string) string {
	lines := strings.SplitAfter(source, "\n")
	var output strings.Builder
	inside, useOptical, nested := false, false, 0
	for _, line := range lines {
		marker := strings.TrimSpace(line)
		switch {
		case marker == "#ifdef SLAYER_OPTICAL_MEDIA" && !inside:
			inside, useOptical, nested = true, false, 0
			continue
		case inside && (strings.HasPrefix(marker, "#if ") ||
			strings.HasPrefix(marker, "#ifdef ") ||
			strings.HasPrefix(marker, "#ifndef ")):
			nested++
			if useOptical {
				output.WriteString(line)
			}
			continue
		case inside && marker == "#else" && nested == 0:
			useOptical = true
			continue
		case inside && marker == "#endif" && nested > 0:
			nested--
			if useOptical {
				output.WriteString(line)
			}
			continue
		case inside && marker == "#endif" && nested == 0:
			inside, useOptical = false, false
			continue
		}
		if !inside || useOptical {
			output.WriteString(line)
		}
	}
	if inside {
		panic("gpu: unclosed SLAYER_OPTICAL_MEDIA shader branch")
	}
	return output.String()
}

const opticalStateDeclarations = `
struct OpticalInitialHeld {
	float index;
	int rank;
	int medium;
	int reserved;
};
struct OpticalInitialState {
	int4 info;
	OpticalInitialHeld held[2 * refractions];
};
`

// rows is the most rows of samples one GPU submission holds, which sizes its
// readback; how many it does hold the time of the one before decides (metal's
// band_heights.go).
const rows = 64

// bandRows is, when not nought, the one height every band of a trace is
// drawn at in place of rows: how the test shows that a picture does not
// depend on its bands. Production leaves it nought.
var bandRows int

// Sampled renders a regular sample grid per pixel on the GPU and returns the
// same frame representation as trace.Sampled. Unsupported scene features return
// a named error rather than disappearing.
//
// Sky rays are evaluated on the CPU. The kernel evaluates every lamp, while the
// CPU tracer samples lamps in scenes with more than eight; those larger scenes
// can differ by lamp-sampling noise.
func Sampled(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, error) {
	return sampledGrown(world, nil, eye, lens, at, width, height, side)
}

// SampledMeasured draws the same frame as Sampled and reports its costs.
// Times describe this call; scene construction by the caller is separate.
func SampledMeasured(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	return measuredBackend(world, eye, lens, at, width, height, side,
		softwareTraversal)
}

// SampledNative uses native mesh shadows and software primary intersections.
// It requires native ray tracing; it never silently changes to another mode.
func SampledNative(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	return measuredBackend(world, eye, lens, at, width, height, side,
		nativeShadowTraversal)
}

// SampledAccelerated uses native closest-hit and shadow queries for meshes.
// Analytic shapes retain their existing intersections. Native numerical
// differences are judged by image quality, not identical hit bits.
func SampledAccelerated(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	return measuredBackend(world, eye, lens, at, width, height, side, nativeTraversal)
}

type traversal uint8

const (
	softwareTraversal traversal = iota
	nativeShadowTraversal
	nativeTraversal
	cachedTraversal
	// fastTraversal is nativeTraversal with the lamp store of reach.go
	fastTraversal
	// fasterTraversal is fastTraversal with the sky's looks shared
	// among a pixel's samples (sky_share.go)
	fasterTraversal
	// restirTraversal is nativeTraversal with restir direct light
	// (restir.go), for a crowd of lamps
	restirTraversal
)

// stored is whether a mode lights boxed media from the lamp store.
func (mode traversal) stored() bool {
	return mode == fastTraversal || mode == fasterTraversal
}

// primary reports modes that query native closest triangles, without relying
// on the ordering of the traversal constants.
func (mode traversal) primary() bool {
	return mode == nativeTraversal || mode == cachedTraversal ||
		mode == restirTraversal || mode.stored()
}

// measuredBackend records disjoint stage costs for the chosen traversal.
func measuredBackend(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int, mode traversal,
) (*frame.Frame, Profile, error) {
	return measuredBackendWithInitialMedia(world, eye, lens, at, width,
		height, side, mode, nil)
}

// measuredBackendWithInitialMedia is the measured variant of the backend
// call that also takes the initial clear media.
func measuredBackendWithInitialMedia(world *thing.World, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, side int,
	mode traversal, media []trace.InitialMedium,
) (*frame.Frame, Profile, error) {
	start := time.Now()
	picture, spent, err := sampledBackendWithInitialMedia(world, nil, eye,
		lens, at, width, height, side, mode, media)
	return picture, spent.profile(time.Since(start)), err
}

// SampledWithInitialMedia draws in the requested traversal mode with an
// explicit ordered camera stack. The mode preserves the caller's independent
// lighting and native-traversal selection. Unsupported combinations fail
// before opening a Metal kernel.
func SampledWithInitialMedia(mode Mode, world *thing.World,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width, height,
	side int, media []trace.InitialMedium) (*frame.Frame, Profile, error) {
	selected, err := traversalFor(mode)
	if err != nil {
		return nil, Profile{}, err
	}
	return measuredBackendWithInitialMedia(world, eye, lens, at, width,
		height, side, selected, media)
}

// traversalFor returns the traversal for a mode, or refuses a mode it does
// not know.
func traversalFor(mode Mode) (traversal, error) {
	if !mode.known() {
		return softwareTraversal, fmt.Errorf("%w: %d", errNoMode, mode)
	}
	return traversals[mode], nil
}

// profile is the timing as a caller reads it, with the call's whole wall
// time.
func (spent timing) profile(total time.Duration) Profile {
	return Profile{
		EnvironmentWall: spent.environmentWall, EnvironmentGPU: spent.environment,
		EnvironmentNodes: spent.environmentNodes, VolumeSkySamples: spent.volumeSky,
		ShellVolumes: spent.shells,
		BoxedCPU:     spent.boxedCPU,
		Native:       spent.native,
		Device:       spent.device, Total: total,
		LightingWall: spent.lightingWall, LightingGPU: spent.lighting,
		Encode: spent.encode, Compile: spent.compile,
		BakeWall: spent.bakeWall, BakeGPU: spent.bake,
		DispatchWall: spent.dispatch, TraceGPU: spent.trace,
		Finish: spent.average, InputBytes: spent.inputBytes,
		SampleBytes:      spent.sampleBytes,
		PeakSampleBytes:  spent.peakSampleBytes,
		AccelerationWall: spent.acceleration, AccelerationGPU: spent.accelerationGPU,
		Retained: spent.retained, LampStoreBytes: spent.storeBytes,
		SkyShared: spent.skyShared,
		Restir:    spent.restir, RestirLamps: spent.restirLamps,
		RestirNodes: spent.restirNodes, RestirWall: spent.restirWall,
		RestirGPU: spent.restirGPU, RestirTreeWall: spent.restirTree,
	}
}

// sampledGrown also includes analytic conifers generated by the kernel using
// the manufacture.Conifer growth rule.
func sampledGrown(world *thing.World, trees []grown, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height,
	side int) (*frame.Frame, error) {
	picture, _, err := sampledTimed(world, trees, eye, lens, at, width,
		height, side)
	return picture, err
}

// timing separates CPU wall time for encoding, compilation, and averaging from
// Metal timestamps for baking and tracing.
type timing struct {
	seedCached                            bool
	environmentWall, environment          time.Duration
	environmentNodes, volumeSky           int
	shells                                int
	boxedCPU                              BoxedPreparation
	native                                NativeCounts
	encode, compile, bake, trace, average time.Duration
	lightingWall, lighting                time.Duration
	bakeWall, dispatch                    time.Duration
	acceleration, accelerationGPU         time.Duration
	device                                string
	inputBytes, sampleBytes               int64
	peakSampleBytes                       int64
	// retained is what a Frames drawer sent; zero for the per-call path
	retained FrameInputs
	// storeBytes is the lamp store's size, nought without fast light
	storeBytes int64
	// skyShared is whether fast light shared the sky's looks
	skyShared             bool
	restir                bool
	restirLamps           int
	restirNodes           int
	restirWall, restirGPU time.Duration
	restirTree            time.Duration
}

// sampledTimed renders analytic conifers and returns stage timings.
func sampledTimed(world *thing.World, trees []grown, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height,
	side int, volumeCells ...int) (*frame.Frame, timing, error) {
	return sampledBackend(world, trees, eye, lens, at, width, height, side,
		softwareTraversal, volumeCells...)
}

// sampledBackend keeps traversal selection explicit during validation.
func sampledBackend(world *thing.World, trees []grown, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, side int,
	mode traversal, volumeCells ...int) (*frame.Frame, timing, error) {
	return sampledBackendWithInitialMedia(world, trees, eye, lens, at,
		width, height, side, mode, nil, volumeCells...)
}

// sampledBackendWithInitialMedia is the sampled variant of the backend
// call that also takes the initial clear media.
func sampledBackendWithInitialMedia(world *thing.World, trees []grown,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width, height,
	side int, mode traversal, media []trace.InitialMedium,
	volumeCells ...int) (*frame.Frame, timing, error) {
	if world != nil && len(world.Fields) > 0 {
		if err := trace.ValidateClearState(world, media); err != nil {
			return nil, timing{}, err
		}
		if len(media) > 0 {
			return nil, timing{}, unsupported{"initial camera media with gravity fields"}
		}
		if mode != softwareTraversal {
			return nil, timing{}, unsupported{"gravity with native traversal"}
		}
		return fieldSampled(world, eye, lens, at, width, height, side, volumeCells...)
	}
	return sampledLightingGridWithInitialMedia(world, trees, eye, lens, at,
		width, height, side, mode, cellsFor(mode), media, volumeCells...)
}

// sampledLightingGridWithInitialMedia is the sampled lighting-grid variant
// of the backend call that also takes the initial clear media.
func sampledLightingGridWithInitialMedia(world *thing.World, trees []grown,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width, height,
	side int, mode traversal, lightingCells int,
	media []trace.InitialMedium, volumeCells ...int,
) (*frame.Frame, timing, error) {
	var spent timing
	side = max(1, side)
	start := time.Now()
	fineWidth, fineHeight, err := checkedFineSize(width, height, side)
	if err != nil {
		spent.encode = time.Since(start)
		return nil, spent, err
	}
	encoded, firsts, err := preparedWithInitialMedia(world, trees, eye,
		lens, at, [2]int{fineWidth, fineHeight}, mode, lightingCells, nil,
		media, &spent, volumeCells...)
	if err != nil {
		return nil, spent, err
	}
	encoded.spreadOver(world, side)
	encoded.shareSkyFor(mode, side)
	encoded.sideFor(lens, side)
	start = time.Now()
	kernel, err := metal.Open(encoded.source(mode), "trace")
	spent.compile = time.Since(start)
	if err != nil {
		return nil, spent, err
	}
	defer kernel.Close()
	kernel.FixBands(bandRows)
	if mode != softwareTraversal {
		memoryStage("per-call-native-plan-before")
		start = time.Now()
		plan, root, makeError := encoded.accelerationPlan()
		memoryStage("per-call-native-plan-after")
		if makeError == nil {
			encoded.discardBuildHierarchy()
			memoryStage("per-call-native-build-before")
			makeError = kernel.Accelerate(plan, root)
			memoryStage("per-call-native-build-after")
		}
		spent.acceleration = time.Since(start)
		spent.accelerationGPU = kernel.GPUTime()
		if makeError != nil {
			return nil, spent, makeError
		}
	}
	spent.device = kernel.Device()
	inputs := encoded.inputs()
	if mode.primary() {
		inputs = append(inputs, firsts)
	}
	if err = encoded.lit(kernel, copied(kernel, inputs), mode,
		&spent); err != nil {
		return nil, spent, err
	}
	for _, input := range inputs {
		spent.inputBytes += int64(len(input))
	}
	start = time.Now()
	band := newBand(world, encoded.paints, encoded.records(), eye, lens,
		[2]int{fineWidth, fineHeight}, width, height, side)
	setup := time.Since(start)
	start = time.Now()
	made, err := encoded.drawn(kernel, inputs, fineWidth, fineHeight, &band)
	spent.inputBytes += made
	spent.banded(&band, time.Since(start), setup)
	spent.trace = traceDeviceTime(kernel.GPUTime(), spent.accelerationGPU,
		spent.lighting, spent.environment)
	if err != nil {
		return nil, spent, err
	}
	return band.picture, spent, nil
}

// prepared is the world encoded for a mode at a size in samples, with
// its deferred fields baked, and the native first faces where the mode
// asks the device for closest triangles. It times the encode and bake.
//
// The retained path encodes by it as the per-call path does, so the two
// cannot drift apart.
func prepared(world *thing.World, trees []grown, eye maths.Subspace,
	lens film.Sensor, at time.Duration, fine [2]int, mode traversal,
	lightingCells int, keeper *keptTrees, spent *timing, volumeCells ...int) (*scene, []byte,
	error) {
	return preparedWithInitialMedia(world, trees, eye, lens, at, fine, mode,
		lightingCells, keeper, nil, spent, volumeCells...)
}

// preparedWithInitialMedia prepares a frame's scene with the take's initial
// clear media and its volume cells.
func preparedWithInitialMedia(world *thing.World, trees []grown,
	eye maths.Subspace, lens film.Sensor, at time.Duration, fine [2]int,
	mode traversal, lightingCells int, keeper *keptTrees,
	media []trace.InitialMedium, spent *timing, volumeCells ...int,
) (*scene, []byte, error) {
	// Retained and native modes have no curved-ray transport yet.
	if world != nil && len(world.Fields) > 0 {
		return nil, nil, unsupported{"gravity in retained or native rendering"}
	}
	start := time.Now()
	sceneReceipt(eye)
	memoryStage("encode-before")
	encoded, err := encodeKeptWithInitialMedia(world, trees, eye, lens, at,
		fine[0], fine[1], keeper, media, volumeCells...)
	memoryStage("encode-after")
	if encoded != nil {
		spent.counted(encoded)
	}
	if err == nil && mode == cachedTraversal {
		encoded.reserveLightingAt(lightingCells)
	}
	if err == nil && mode.stored() {
		err = encoded.reserveReachAt(lightingCells)
	}
	var firsts []byte
	if err == nil && mode.primary() {
		firsts = encoded.nativeFirsts()
	}
	if err == nil {
		encoded.discardFaceValues()
		if mode == softwareTraversal {
			encoded.discardBuildHierarchy()
		}
	}
	memoryStage("encode-after-discard")
	// the encode's working arrays are garbage now, and the collector
	// keeps freed pages resident: on the neon street's nine blocks, 2.8
	// GiB freed and held through the whole draw. Handing them back took
	// 31 ms there (measured 2026-11-20), once an encode.
	debug.FreeOSMemory()
	memoryStage("encode-after-release")
	spent.encode = time.Since(start)
	if err != nil {
		return nil, nil, err
	}
	start = time.Now()
	spent.bake, err = encoded.bakeAll()
	memoryStage("bake-after")
	spent.bakeWall = time.Since(start)
	return encoded, firsts, err
}

// counted records what the encoded scene holds that a profile reports.
func (spent *timing) counted(encoded *scene) {
	spent.boxedCPU = encoded.boxedCPU
	spent.shells = encoded.counts.shells
	spent.volumeSky = encoded.volumeSky
	if grid := encoded.incident; grid != nil {
		spent.environmentNodes = grid.nodes[0] * grid.nodes[1] * grid.nodes[2]
	}
}

// source is the trace kernel's text for this scene in a mode: sized
// for its records and volumes, with the native queries the mode asks.
func (encoded *scene) source(mode traversal) string {
	text := sourceWith(encoded.records(), encoded.counts.volumes)
	if encoded.propellerDispersal {
		text = sourceWithPropellerDispersal(encoded.records(), encoded.counts.volumes)
	}
	if encoded.opticalState {
		text = sourceWithOpticalState(encoded.records(),
			encoded.counts.volumes)
	}
	if mode != softwareTraversal {
		text = "#define SLAYER_NATIVE_SHADOWS 1\n" + text
	}
	if mode.primary() {
		text = "#define SLAYER_NATIVE_MESHES 1\n" + text
	}
	return text
}

// newBand is the finisher for a picture width by height pixels of side
// by side samples each, fine samples across and down, seen from eye
// through lens.
func newBand(world *thing.World, paints []painting, records int,
	eye maths.Subspace, lens film.Sensor, fine [2]int, width, height,
	side int) bandFinisher {
	finish := finishing{world: world, paints: paints, records: records,
		wide:  wideOf(world, eye, lens, fine[0], fine[1]),
		discs: film.Disced(world.Lamps)}
	return bandFinisher{finish: finish, picture: frame.New(width, height),
		side: side}
}

// banded records a drawing's costs: its wall time less the finishing
// inside it, and the finishing with its setup.
func (spent *timing) banded(band *bandFinisher, wall, setup time.Duration) {
	spent.dispatch = wall - band.spent
	spent.average = setup + band.spent
	spent.sampleBytes, spent.peakSampleBytes = band.bytes, band.peak
	spent.native = band.native
}

// wideOf returns the world's explicit angular pixel width or derives it from
// neighbouring central camera rays, matching the CPU tracer.
func wideOf(world *thing.World, eye maths.Subspace, lens film.Sensor,
	width, height int) float64 {
	if world.Pixel > 0 {
		return world.Pixel
	}
	middle := film.Look(eye, lens, width/2, height/2, width, height,
		maths.Band{})
	beside := film.Look(eye, lens, width/2+1, height/2, width, height,
		maths.Band{})
	cosine := math.Max(-1, math.Min(1, middle.Dir.Dot(beside.Dir)))
	return math.Acos(cosine)
}

// finishing supplies CPU sky evaluation and spatial material colour and
// emission for GPU hit records. wide is angular sample width in radians, used
// to select the texture footprint.
type finishing struct {
	world   *thing.World
	paints  []painting
	records int
	wide    float64
	// discs is the world's suns drawn as discs, found once
	discs []film.Sun
}

// averaged finishes sample light on the CPU, then combines each pixel's RGB and
// coverage by averaging and retains its nearest sample depth.
func averaged(finish finishing, samples []float32, width, height,
	side int) *frame.Frame {
	picture := frame.New(width, height)
	rowsLeft := make(chan int, height)
	for row := range height {
		rowsLeft <- row
	}
	close(rowsLeft)
	var working sync.WaitGroup
	for range runtime.NumCPU() {
		working.Add(1)
		go func() {
			defer working.Done()
			for row := range rowsLeft {
				drawRow(finish, picture, samples, row, width,
					side)
			}
		}()
	}
	working.Wait()
	return picture
}

// drawRow finishes and averages every sample block in one image row.
func drawRow(finish finishing, picture *frame.Frame, samples []float32,
	row, width, side int) {
	for column := range width {
		picture.Set(column, row,
			pixel(finish, samples, column, row, width*side, side))
	}
}

// pixel combines one output pixel's finished samples.
func pixel(finish finishing, samples []float32, column, row,
	fineWidth, side int) frame.Pixel {
	sum := frame.Pixel{Depth: float32(math.Inf(1))}
	share := 1 / float32(side*side)
	each := per + perRecord*finish.records
	for down := range side {
		for across := range side {
			fine := (row*side+down)*fineWidth + column*side + across
			sample := finish.sample(samples[fine*each:][:each])
			sum.R += sample.R * share
			sum.G += sample.G * share
			sum.B += sample.B * share
			sum.Cover += sample.Cover * share
			sum.Depth = min(sum.Depth, sample.Depth)
		}
	}
	return sum
}

// sample adds weighted CPU sky light and evaluates its texture records.
func (finish finishing) sample(sample []float32) frame.Pixel {
	light := lightOf(sample[0:3])
	weight := lightOf(sample[4:7])
	sky := finish.world.Sky
	if weight != (maths.Light{}) {
		way := maths.V(float64(sample[8]), float64(sample[9]),
			float64(sample[10]))
		var beyond maths.Light
		if sky != nil {
			beyond = sky.Light(way.Unit())
		}
		// and the sun, where it is drawn as a disc and the ray finds it,
		// as slayer's tracer adds it
		if disc := film.Discs(finish.discs, way); disc != (maths.Light{}) {
			beyond = beyond.Add(disc)
		}
		if sky != nil || beyond != (maths.Light{}) {
			light = light.Add(beyond.Times(weight))
		}
	}
	for record := range finish.records {
		at := per + perRecord*record
		light = light.Add(finish.painted(sample[at : at+perRecord]))
	}
	depth := sample[7]
	if depth >= nowhere {
		depth = float32(math.Inf(1))
	}
	return frame.Pixel{R: float32(light.R), G: float32(light.G),
		B: float32(light.B), Cover: sample[3], Depth: depth}
}

// painted combines local material reflectance with incident light and adds
// emitted texture light attenuated toward the camera. Both textures use the CPU
// tracer's footprint convention. Material index -1 means there is no texture
// location to evaluate.
func (finish finishing) painted(record []float32) maths.Light {
	material := int(record[0])
	if material < 0 {
		return maths.Light{}
	}
	textures := finish.paints[material]
	local := maths.V(float64(record[1]), float64(record[2]),
		float64(record[3]))
	footprint := float64(record[4]) * finish.wide
	var light maths.Light
	if textures.paint != nil {
		colour := texture.Sample(textures.paint, local, footprint)
		light = light.Add(colour.Times(lightOf(record[5:8])))
	}
	if textures.shine != nil {
		shine := texture.Sample(textures.shine, local, footprint)
		light = light.Add(shine.Times(lightOf(record[8:11])))
	}
	return light
}

// lightOf converts three kernel values to linear RGB.
func lightOf(three []float32) maths.Light {
	return maths.RGB(float64(three[0]), float64(three[1]),
		float64(three[2]))
}
