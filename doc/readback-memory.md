# Keep the readback bounded

The GPU already dispatches bands. The host still keeps every raw sample
until the whole picture has returned. This is separate from scene caching
and from the cost of marching through mist.

## What the current code does

[`metal.Kernel.Grid`](../metal/grid.go) allocates `width * height * per` float32
values. These dimensions are the supersampled grid. The Objective-C bridge
uses one GPU output band, then copies each band into that full Go slice.
[`gpu.sampledLightingGrid`](../gpu/sampled.go) averages it afterward.

With no CPU texture records, each sample has 12 floats. A 3840 by 2160
picture at 16 samples per pixel therefore asks for 6,370,099,200 raw bytes
before its final frame and other scene storage. This is a size calculation,
not a measured allocation from a new 4K run.

The committed spacecraft studies provide a smaller check: their 960 by
720 pictures at nine samples report 298,598,400 sample-output bytes, matching
the same calculation. Those studies are not a dense-forest performance test.

## Proposed change

Let a prepared Metal grid return one bounded band to a consumer. Keep inputs
and its GPU output buffer alive for the dispatch sequence, then release them
with that grid. Average each band into the final frame before reusing its
sample storage. Keep the existing full-grid API for current callers.

Align renderer bands to complete supersampling rows where possible. Preserve
the per-pixel addition order, nearest depth, coverage and native counters.
Account for CPU finishing separately from dispatch wall time; the existing
profile promises disjoint stages.

The actual byte limit needs an explicit authored budget. If one complete
output row exceeds it, refuse before allocation or implement partial-row
accumulation with its own compatibility tests. Do not call an unbounded
minimum allocation a bounded implementation.

## Evidence needed

Compare the existing and streamed sample paths on small real Metal grids,
including short final bands and sample sides that do not divide the usual
band size. Compare finished surfaces, reflections, fields, misses and native
counts. Exercise consumer failure and resource cleanup without giant fixtures.

Measure a manageable matched scene cold and warm under the same memory
watchdog. Only then run the streamed 4K case. Record both total sample bytes
and peak batch bytes; streaming reduces retained storage even though the
same amount of sample data crosses the boundary.

This is enabling work for the animation and preparation-memory budgets.
Packet traversal, cached atmospheric transmission and retained frame resources
remain separate work, each needing its own timing and image-acceptance result.
