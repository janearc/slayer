# Keep the readback bounded

This is a historical design and accounting record. For the shipped
streaming contract and later focused evidence, read
[streamed readback](streamed-readback.md); for current feature status,
read [optimizations](optimizations.md). Proposal wording below retains
the state at the time of this record.

The GPU already dispatches bands. The sampled picture no longer keeps every
raw sample: `metal.Kernel.GridBands` returns one reusable band at a time, and
`gpu/sampled.go` averages each band into the frame as it arrives. The
full-grid `metal.Kernel.Grid` remains; its only non-test caller is
`gpu/flame_bake.go`. This is separate from scene caching and from the cost of
marching through mist.

## What `Grid` does

[`metal.Kernel.Grid`](../metal/grid.go) allocates `width * height * per` float32
values. These dimensions are the supersampled grid. The Objective-C bridge
uses one GPU output band, then copies each band into that full Go slice.

Its only non-test caller, `bakeFlame` in
[`gpu/flame_bake.go`](../gpu/flame_bake.go), copies the samples into density
and shine arrays. The sampled picture's own path,
[`gpu.sampledLightingGrid`](../gpu/sampled.go), no longer calls it.

This paragraph is the former sampled path, which `Grid` served before
`GridBands`. With no CPU texture records, each sample has 12 floats. A 3840 by
2160 picture at 16 samples per pixel therefore asked for 6,370,099,200 raw
bytes before its final frame and other scene storage. This is a size
calculation, not a measured allocation from a new 4K run.

The committed spacecraft studies provide a smaller check: their 960 by
720 pictures at nine samples report 298,598,400 sample-output bytes, matching
the same calculation. Those studies are not a dense-forest performance test.

## The change, as shipped for the sampled picture

The first paragraph below is shipped. The second is shipped in part and the
third is still a proposal; each says so in its first words. The evidence
section further down is still the list of what remains to be measured.

Shipped: let a prepared Metal grid return one bounded band to a consumer. Keep
inputs and its GPU output buffer alive for the dispatch sequence, then release
them with that grid. Average each band into the final frame before reusing its
sample storage. Keep the existing full-grid API for current callers.

Shipped in part: the sampled path times dispatch and CPU finishing apart
(`gpu/sampled.go`, `spent.dispatch` and `spent.average`). Still a proposal:
align renderer bands to complete supersampling rows where possible, which
the fixed 64-row `rows` does not claim; and preserve the per-pixel addition
order, nearest depth, coverage and native counters, which the evidence
section lists as unmeasured.

Proposal, not shipped: the actual byte limit needs an explicit authored budget.
The band is a fixed 64 rows and no budget or refusal exists. If one complete
output row exceeds it, refuse before allocation or implement partial-row
accumulation with its own compatibility tests. Do not call an unbounded minimum
allocation a bounded implementation.

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
