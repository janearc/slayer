# Committed 4K comparison

This benchmark uses one clean, committed Slayer executable for both
5000-redwood scenes. Terrain, mist, camera, time, exposure and all sixteen
samples per pixel stay fixed. Only the renderer changes.

The benchmark in `measurement_test.go` expects a prepared executable at
`out/review-4k/hillside` and checks it against the clean source commit.
The benchmark draws two 4K frames. Coordinate any run with the render
queue operator. The normal test suite does not execute the benchmark.

Artifacts stay in out/review-4k: matched scene YAML, complete command logs,
raw RGB, PNGs, and measurement.json. The producer records the clean Git
revision and SHA-256 of the executable, source catalogue, terrain, scenes,
PNGs and raws, plus external command wall time and actual native outcomes.

The comparison visits every developed PNG pixel. It retains the existing
two-percent budget beyond one OKLab JND, reports the worst pixel, and requires
alpha agreement. PFM preserves RGB, not depth or coverage; the separate GPU
tests hold those quantities. A still comparison establishes no motion budget.
