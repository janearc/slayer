package hillside

import (
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/internal/observability"
)

// recordGPUProfile records one glimpse. Device timings sum command-buffer
// durations, which can overlap; they must not be added to wall timings.
func recordGPUProfile(profile gpu.Profile, observe, add, set func(string, float64)) {
	if profile.Device == "" {
		return
	}
	for _, stage := range []struct {
		name  string
		value float64
	}{
		{"slayer_gpu_total_wall_seconds", profile.Total.Seconds()},
		{"slayer_gpu_encode_wall_seconds", profile.Encode.Seconds()},
		{"slayer_gpu_boxed_cpu_wall_seconds", profile.BoxedCPU.Wall.Seconds()},
		{"slayer_gpu_compile_wall_seconds", profile.Compile.Seconds()},
		{"slayer_gpu_environment_wall_seconds", profile.EnvironmentWall.Seconds()},
		{"slayer_gpu_environment_device_seconds", profile.EnvironmentGPU.Seconds()},
		{"slayer_gpu_bake_wall_seconds", profile.BakeWall.Seconds()},
		{"slayer_gpu_bake_device_seconds", profile.BakeGPU.Seconds()},
		{"slayer_gpu_lighting_wall_seconds", profile.LightingWall.Seconds()},
		{"slayer_gpu_lighting_device_seconds", profile.LightingGPU.Seconds()},
		{"slayer_gpu_acceleration_wall_seconds", profile.AccelerationWall.Seconds()},
		{"slayer_gpu_acceleration_device_seconds", profile.AccelerationGPU.Seconds()},
		{"slayer_gpu_dispatch_wall_seconds", profile.DispatchWall.Seconds()},
		{"slayer_gpu_trace_device_seconds", profile.TraceGPU.Seconds()},
		{"slayer_gpu_finish_wall_seconds", profile.Finish.Seconds()},
	} {
		observe(stage.name, stage.value)
	}
	add("slayer_gpu_glimpses_total", 1)
	add("slayer_gpu_logical_input_bytes_total", float64(profile.InputBytes))
	add("slayer_gpu_retained_sent_bytes_total", float64(profile.Retained.SentBytes))
	add("slayer_gpu_sample_bytes_total", float64(profile.SampleBytes))
	add("slayer_gpu_boxed_cells_total", float64(profile.BoxedCPU.Cells))
	add("slayer_gpu_native_queries_total", float64(profile.Native.Queries))
	add("slayer_gpu_native_candidates_total", float64(profile.Native.Candidates))
	add("slayer_gpu_native_refined_total", float64(profile.Native.Refined))
	add("slayer_gpu_native_fallbacks_total", float64(profile.Native.Fallbacks))
	add("slayer_gpu_retained_inputs_sent_total", float64(profile.Retained.Sent))
	set("slayer_gpu_last_glimpse_peak_sample_readback_bytes", float64(profile.PeakSampleBytes))
	set("slayer_gpu_last_glimpse_lamp_store_bytes", float64(profile.LampStoreBytes))
	set("slayer_gpu_last_glimpse_retained_inputs", float64(profile.Retained.Held))
	set("slayer_gpu_last_glimpse_environment_nodes", float64(profile.EnvironmentNodes))
	set("slayer_gpu_last_glimpse_volume_sky_samples", float64(profile.VolumeSkySamples))
	set("slayer_gpu_last_glimpse_shell_volumes", float64(profile.ShellVolumes))
}

// observeGPUProfile records a GPU profile and its Metal memory.
func observeGPUProfile(profile gpu.Profile) {
	recordGPUProfile(profile, observability.Observe, observability.Add, observability.Set)
	recordGPUMemory(profile, metalMemorySnapshot, observability.Set)
}

// No device was opened for CPU glimpses; do not open one just to sample it.
func recordGPUMemory(profile gpu.Profile, read func() (uint64, uint64, bool), set func(string, float64)) {
	if profile.Device != "" {
		recordMetalMemory(read, set)
	}
}

// recordMetalMemory marks failure separately: the last successful values
// are meaningful only while the availability gauge is one.
func recordMetalMemory(read func() (uint64, uint64, bool), set func(string, float64)) {
	allocated, recommended, available := read()
	if !available {
		set("slayer_gpu_after_glimpse_device_memory_available", 0)
		return
	}
	set("slayer_gpu_after_glimpse_device_allocated_bytes", float64(allocated))
	set("slayer_gpu_device_recommended_working_set_bytes", float64(recommended))
	set("slayer_gpu_after_glimpse_device_memory_available", 1)
}
