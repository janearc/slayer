package gpu

import "time"

// Profile records a render call's measured costs, without changing its image.
// Wall stages are disjoint. GPU times sit inside their corresponding wall
// stages and must not be added to them. Total excludes caller construction.
type Profile struct {
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
