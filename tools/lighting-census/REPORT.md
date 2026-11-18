# Reading a lighting census

To print a saved run's report as JSON, use Game's CPU lane from the Slayer
checkout:

```sh
game lane tests -- go run ./tools/lighting-census/cmd/report read --root /absolute/census/output --reference cpu
```

`census.Summarize(outputDirectory, referenceMode)` reads the saved manifest,
case records and linear frames. It returns a Go `Report`; callers can encode
it as JSON. This operation reads saved files and does not start a renderer.
It loads a reference and candidate frame into memory at the same time; memory
use grows with image dimensions and can be multiple GiB for the largest
permitted frames. Use Game's CPU lane and coordinate with active renders.

The accepted reference modes are `cpu` and `gpu-native-meshes`. The latter is
direct for lighting but still uses production density baking; neither is a
promise of bit-identical results. In the current build both CPU and GPU
sample when a scene has more than sixteen lamps; selection and numerical
differences still need measurement. Approximate lighting modes cannot be
selected as references.

The report has one entry for each manifest case. `not-run`, `missing`,
`running`, `unsupported`, `failed`, `partial`, `invalid`, `missing-reference`
and `inconclusive` remain distinct. `reference` names a present reference, and
`measured` means a comparison was possible. Neither means pass: no global
quality threshold is imposed.

A dark reference has no relative error and is inconclusive, even if its
absolute difference is zero. Nonfinite RGB or coverage, coverage outside
[0,1], negative or invalid depth and damaged linear artifacts are errors;
positive infinity is valid as the renderer's no-hit depth sentinel.

PNG previews are checked for a readable header and matching dimensions; this
is not a full PNG integrity check, so a damaged preview may still need to be
regenerated.

The comparator requires the full recorded case metadata, the manifest's
scene time, phase, frame index and dimensions, matching executable, camera
and input hashes, and both the linear RGB PFM and its coverage/depth
sidecar.

The reported RGB differences are linear HDR values: total absolute
difference, the reference's absolute energy, their ratio, RMS channel error,
worst channel and its coordinates, and the worst error divided by the
reference peak. Coverage and depth differences use the existing GPU test
tolerances of 0.001 absolute coverage and 0.001 relative depth. Error
coordinates are zero-based from the top left.

Each measured frame includes paired reference and candidate wall times and
paths to both preview PNGs, beside the HDR difference and its worst pixel.

The previews use unit-exposure development; they are diagnostic and do not
replace the full authored tone, gauge or spectral presentation. Device names
are recorded when available; timing involving a GPU is marked not comparable
if its device is unknown, and GPU-to-GPU timing also requires matching device
names. `timing_comparable` describes the recorded
metadata, not statistical certainty or a speedup claim.

Report exact and approximate winners separately; do not rank across those
classes or select a winner from pending, partial, missing or unsupported
cases. This report does not select winners.

GPU direct-light modes still use production density baking and must retain
that qualification. A later winner decision needs repeated valid paired
timings, stated quality criteria and visual inspection of approximation
differences. The saved previews make side-by-side inspection possible; a
separate difference heatmap is not yet produced.

The images' spectral bands are not compared here. Where a registered smoke
has extra bands and either side is a GPU mode, the case remains `partial`.

The reader uses the census artifact dialect: unit-scale, little-endian PFM
with bottom row first; the `.cd32` sidecar has little-endian coverage and
depth float32 pairs in top-row-first order. Both must have exact lengths. It
opens files beneath the supplied output directory and refuses invalid paths
or inconsistent identities.

Matching metadata does not prove that separately built worlds resolved
identical scene content: reads can be dynamic or undeclared, and source inputs
might change while a frame is drawn. Use immutable or independently verified
complete inputs before drawing a stronger conclusion.

A report only compares cases in the same output directory. Each invocation
creates a new directory, so selecting one GPU mode alone leaves its reference
missing. Selecting all modes for a scene includes its CPU and GPU direct-light
references, but the current runner holds the GPU lane while drawing the CPU
reference too. Cross-directory aggregation is not implemented.
