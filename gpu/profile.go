package gpu

import "time"

// Profile records a render call's measured costs, without changing its image.
// Top-level wall stages are disjoint. GPU times sum command-buffer execution
// intervals attributed to the stage; those intervals may overlap and their
// sum need not fit inside its wall time. BoxedCPU.Wall sits inside Encode.
// Total excludes caller construction.
type Profile struct {
	// Environment records current-frame incident-sky lattice preparation.
	// Nodes is zero for direct integration; times are disjoint from TraceGPU.
	EnvironmentWall, EnvironmentGPU time.Duration
	EnvironmentNodes                int
	VolumeSkySamples                int
	// ShellVolumes counts radial weather fields evaluated directly on the GPU.
	ShellVolumes int
	// BoxedCPU measures generic density/texture grid sampling during encoding.
	// It excludes GPU-deferred fields and packing the sampled values into inputs.
	BoxedCPU BoxedPreparation
	// Native counts actual closest-query outcomes, including
	// software retries.
	Native NativeCounts
	// LightingWall includes optional source-grid preparation and readback.
	LightingWall, LightingGPU time.Duration
	Device                    string
	Total, Encode, Compile    time.Duration
	// BakeWall includes opening the bake kernel and reading its grids.
	BakeWall, BakeGPU time.Duration
	// DispatchWall includes input copies, buffers, tracing and readback.
	DispatchWall, TraceGPU time.Duration
	Finish                 time.Duration
	// AccelerationWall includes planning and native hierarchy construction.
	AccelerationWall, AccelerationGPU time.Duration
	// InputBytes excludes Metal's padding for empty buffers.
	InputBytes int64
	// SampleBytes is the logical raw sample payload consumed before
	// averaging. Streaming consumes it in bands rather than retaining one
	// full array.
	SampleBytes int64
	// PeakSampleBytes is the peak Go raw-sample readback buffer. Metal
	// keeps a separate output band; neither value is a whole-process memory
	// measure.
	PeakSampleBytes int64
	// Retained is what a Frames drawer sent to the inputs it keeps on
	// the device. It is zero for the per-call functions, which copy every
	// input for every grid they run.
	Retained FrameInputs
	// LampStoreBytes is the size of fast light's lamp store, baked
	// inside LightingWall. It is nought for a picture lit exactly, so a
	// profile says which a picture was.
	LampStoreBytes int64
	// SkyShared is whether fast light shared the sky's looks among a
	// pixel's samples.
	SkyShared bool
	// Restir is whether the picture was lit by restir (doc/restir.md): the
	// crowd of lamps it chose among and the light tree's nodes, the wall
	// time of its passes (host copies included), their device time, and
	// the tree's build. All nought for a picture lit exactly.
	Restir                bool
	RestirLamps           int
	RestirNodes           int
	RestirWall, RestirGPU time.Duration
	RestirTreeWall        time.Duration
}

// traceDeviceTime isolates trace commands from the kernel's cumulative GPU
// clock. Environment and lighting commands are counted in their own stages.
func traceDeviceTime(after, before, lighting, environment time.Duration) time.Duration {
	return after - before - lighting - environment
}

// FrameInputs counts a retained frame's traffic to the device. Held is
// how many inputs the drawer keeps; Sent how many times one was found
// changed and sent, only its changed parts, and SentBytes the bytes
// copied.
//
// The cached mode's light bakes change the header and the boxes between
// grids, so one frame can send an input more than once. Patched is
// whether the frame was a camera move drawn without encoding the world.
type FrameInputs struct {
	Held, Sent int
	SentBytes  int64
	Patched    bool
}

// BoxedPreparation records CPU grid work without counting it as a new stage.
// Cells counts density sample locations; textures can add queries per cell.
type BoxedPreparation struct {
	Wall    time.Duration
	Cells   int64
	Volumes int
}

// NativeCounts distinguishes native replies from refinement fallbacks.
// Queries without a triangle candidate are misses, not software retries.
type NativeCounts struct {
	Queries    uint64 // closest-mesh query attempts
	Candidates uint64 // native triangle candidates
	Refined    uint64 // candidates answered by native refinement
	Fallbacks  uint64 // candidates retried by software traversal
}

// nativeCounts reads the diagnostic lane, four exact four-bit integers.
// A trace has at most two closest queries: primary and one mirrored bounce.
func nativeCounts(samples []float32, stride int) NativeCounts {
	var count NativeCounts
	for index := 11; index < len(samples); index += stride {
		packed := uint32(samples[index])
		count.Queries += uint64(packed & 15)
		count.Candidates += uint64((packed >> 4) & 15)
		count.Refined += uint64((packed >> 8) & 15)
		count.Fallbacks += uint64((packed >> 12) & 15)
	}
	return count
}
