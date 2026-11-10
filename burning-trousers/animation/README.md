# Committed 4K comparison

This benchmark uses one clean, committed Slayer executable for both
5000-redwood scenes. Terrain, mist, camera, time, exposure and all sixteen
samples per pixel stay fixed. Only the renderer changes.

From the Slayer root, with the source shelf and dependencies configured:

    game build
    game build review-4k-binary
    game run review-4k

Follow the machine's render policy and run serially: this draws two 4K frames.
`game check` only compiles the benchmark. The named build uses its expected
path; the producer checks that executable against the harness's clean commit.

Artifacts stay in out/review-4k: matched scene YAML, complete command logs,
raw RGB, PNGs, and measurement.json. The producer records the clean Git
revision and SHA-256 of the executable, source catalogue, terrain, scenes,
PNGs and raws, plus external command wall time and actual native outcomes.

The comparison visits every developed PNG pixel. It retains the existing
two-percent budget beyond one OKLab JND, reports the worst pixel, and requires
alpha agreement. PFM preserves RGB, not depth or coverage; the separate GPU
tests hold those quantities. A still comparison establishes no motion budget.
