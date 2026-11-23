# Low-retention scene encoding trial

Set `SLAYER_LOW_RETENTION=1` to opt into this experiment. With it unset,
the existing retained-tree and packed-pool behavior is unchanged.

The trial grows each unique mesh tree and immediately copies it into the
scene-wide pool, instead of retaining all individual trees until the
pool is built. It does not retain the per-mesh CPU trees or their CPU
pool between frames. A retained Metal drawer still keeps its device
buffers and acceleration structures. Unchanged frames may need their
CPU trees rebuilt; this trades reuse for lower peak memory.

This keeps full geometry. It does not page the scene: the complete pooled
CPU scene, packed GPU input and native acceleration data must still fit.
It does not implement or guarantee a 2 GiB page or process limit.

Focused CPU tests compare every encoded input across moving frames, shared
MeshGroup instances and changed mutable meshes. On macOS, tests also
compare native first-face maps and acceleration-plan data without opening
a Metal device. An overlay anti-test disabling the flag failed as
expected; another overlay corrupting a streamed tree root was detected.

On a levelled balcony city set, one 960 x 540 still at four samples per pixel
produced byte-identical PNG and linear PFM output in both modes. A moving
480 x 270 two-frame Metal run at one sample per pixel executed two draws
and produced matching PNGs. These small tests do not establish parity for
all scenes, light transport, or changing geometry. One run per setting is
not a reliable performance comparison.

A CPU-only synthetic measurement used 128 distinct 2,048-face meshes,
two isolated processes, one run per mode, no GPU draw and the same binary.
The input world remained alive in both. Process lifetime peak fell from
320,390,032 to 241,697,608 bytes (about 24.6%). Timing and GC can vary;
this is not a city or a prediction of GPU-render peak.

Machine-readable
stage checkpoints are enabled separately with `SLAYER_MEMORY_STAGES=1`;
they sample overlapping process and Go memory figures, not allocation
ownership. The sampler and its switch are described in
`memory-stages-trial.md`; this trial only adds checkpoints and the
device-wide counters, which `SLAYER_MEMORY_STAGES=1` also turns on.
