package main

import (
	"fmt"
	"io"

	"github.com/janearc/slayer/gpu"
)

// reportGPUProfile formats the completed measurement. It neither samples a
// clock nor renders; callers retain responsibility for obtaining real costs.
func reportGPUProfile(said io.Writer, measured gpu.Profile) {
	if measured.Device != "" {
		fmt.Fprintf(said, "gpu %s: encode %v, compile %v, "+
			"bake wall %v (gpu %v), dispatch/readback wall %v "+
			"(gpu %v), finish %v; input %d bytes, "+
			"sample output %d bytes\n",
			measured.Device, measured.Encode, measured.Compile,
			measured.BakeWall, measured.BakeGPU,
			measured.DispatchWall,
			measured.TraceGPU, measured.Finish,
			measured.InputBytes, measured.SampleBytes)
	}
	if measured.LightingWall > 0 {
		fmt.Fprintf(said, "smoke lighting: wall %v (gpu %v)\n", measured.LightingWall, measured.LightingGPU)
	}
	if measured.PeakSampleBytes > 0 {
		fmt.Fprintf(said, "sample readback: peak Go band %d bytes; "+
			"Metal keeps a separate band\n",
			measured.PeakSampleBytes)
	}

	if measured.Native.Queries > 0 {
		counts := measured.Native
		fmt.Fprintf(said, "native closest: %d queries, %d candidates, "+
			"%d refined, %d software fallbacks\n", counts.Queries,
			counts.Candidates, counts.Refined, counts.Fallbacks)
	}

	if measured.AccelerationWall > 0 {
		fmt.Fprintf(said, "native hierarchy: wall %v (gpu %v)\n",
			measured.AccelerationWall, measured.AccelerationGPU)
	}
}
