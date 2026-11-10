package main

import (
	"bytes"
	"testing"

	"github.com/janearc/slayer/gpu"
)

// These are formatter inputs, not claimed device measurements. Keep the
// real-command backend/counter checks too; this test does not replace them.
func TestGPUProfileFormattingKeepsZeroAndSingleUnitBoundaries(t *testing.T) {
	const header = "gpu fixture: encode 2ns, compile 3ns, " +
		"bake wall 7ns (gpu 5ns), dispatch/readback wall 13ns " +
		"(gpu 11ns), finish 17ns; input 19 bytes, sample output 23 bytes\n"
	base := gpu.Profile{Device: "fixture", Encode: 2, Compile: 3,
		BakeWall: 7, BakeGPU: 5, DispatchWall: 13, TraceGPU: 11,
		Finish: 17, InputBytes: 19, SampleBytes: 23}
	one := base
	one.LightingWall, one.LightingGPU = 1, 0
	one.AccelerationWall, one.AccelerationGPU = 1, 0
	one.Native = gpu.NativeCounts{Queries: 1, Candidates: 1, Refined: 1}
	two := base
	two.LightingWall, two.LightingGPU = 2, 1
	two.AccelerationWall, two.AccelerationGPU = 2, 1
	two.Native = gpu.NativeCounts{Queries: 2, Candidates: 2, Refined: 1, Fallbacks: 1}
	distinct := base
	distinct.LightingWall, distinct.LightingGPU = 5, 2
	distinct.AccelerationWall, distinct.AccelerationGPU = 7, 3
	distinct.Native = gpu.NativeCounts{Queries: 7, Candidates: 4, Refined: 3, Fallbacks: 1}
	for _, check := range []struct {
		name string
		data gpu.Profile
		want string
	}{
		{"absent", gpu.Profile{}, ""},
		{"no-optional-stages", base, header},
		{"one-unit", one, header +
			"smoke lighting: wall 1ns (gpu 0s)\n" +
			"native closest: 1 queries, 1 candidates, 1 refined, 0 software fallbacks\n" +
			"native hierarchy: wall 1ns (gpu 0s)\n"},
		{"two-units", two, header +
			"smoke lighting: wall 2ns (gpu 1ns)\n" +
			"native closest: 2 queries, 2 candidates, 1 refined, 1 software fallbacks\n" +
			"native hierarchy: wall 2ns (gpu 1ns)\n"},
		{"distinct-stages", distinct, header +
			"smoke lighting: wall 5ns (gpu 2ns)\n" +
			"native closest: 7 queries, 4 candidates, 3 refined, 1 software fallbacks\n" +
			"native hierarchy: wall 7ns (gpu 3ns)\n"},
	} {
		t.Run(check.name, func(t *testing.T) {
			var said bytes.Buffer
			reportGPUProfile(&said, check.data)
			if got := said.String(); got != check.want {
				t.Fatalf("profile formatting =%q, want %q", got, check.want)
			}
		})
	}
}
