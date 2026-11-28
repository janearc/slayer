# Comparing regions of a picture

A region comparison draws selected rectangles at the full picture's pixel
scale. It puts each result beside a higher-sample reference, so that a change
can be judged before drawing the whole picture. The set names the job and,
for each rectangle, a short name and what the rectangle is meant to show.
The rectangles are in the complete frame's coordinates.

## Running a set

use `slayer compare-regions SET` with a frozen input snapshot.
the comparison set names a job relative to the snapshot and each
rectangle in full-frame pixels. supply:

- `--snapshot`, `--inputs` and `--complete-inputs` for the snapshot
  and its complete list of stable input paths.
- `--width`, `--height`, `--rays` and `--reference-rays` for the full
  frame and both per-side sample counts. the reference count must
  exceed the candidate count and be at most 64.
- `--references` for an existing absolute directory and `--out`
  for a new absolute output directory, both outside the snapshot.

set `SLAYER_SHELF` inside the snapshot and keep `SLAYER_CACHE` stable
and outside it. the command depends on the supplied input list.
`--dry-run` checks the request without drawing. use the
[region limits](frame-regions-plan.md#supported-draws) and
[shared queue](azazel.md#submit-work) for the render.

the result directory contains `report.json` with raw-light
comparisons and draw timings, `sheet.png` with candidate and
reference pairs, and `references.json` with the reference plan.
use `--reference-plan` to pin a previous plan and `--repeat` to
compare up to eight candidate draws with the first.

## What the numbers say

Compare candidate and reference RGB before display clipping. Report the mean
absolute difference and the signed candidate-minus-reference mean for each
channel. The signed value makes systematic loss of light visible. Separately
count a pixel when its positive luminance residual is greater than a stated
threshold and no one of its eight neighbours exceeds that threshold. The
outside edge is excluded because its neighbourhood is incomplete.

The count
is a diagnostic, not a verdict: a real point of light can be isolated.

Report the threshold in linear-light units and keep it the same when comparing
runs. A finite reference contains noise too. A clamp can happen to reduce
error against a noisy reference, so neither the count nor one distance can
guarantee that all light loss is penalized. Include explicit darkening and
clipping cases in the checks, and show the signed differences alongside the
absolute errors.

## Reference identity and storage

A reference has its own declared identity, independent of the candidate
build. Keep that identity fixed while comparing candidate changes, and report
both identities. To change the reference, explicitly select or draw a new
one; do not silently replace the target when the candidate changes.

The reference identity covers the actual rendering build, backend and mode,
full frame dimensions, rectangle, camera and time, seed, sampling and every
setting that affects its linear light. It also covers the job, set and all
transitively read inputs by content, along with the device or runtime identity
where it can affect the result.

Include development settings in the identity
of any stored developed picture. If a dependency cannot be enumerated and
identified, do not claim a reusable cache hit. Inputs must remain the ones
identified throughout the draw; otherwise discard the result and say why.

Store the reference and its manifest together. Validate the manifest, content
digest, dimensions and finite RGB before reuse. A missing, incomplete,
corrupt or mismatched entry is a miss or a named error, never a reference
accepted on its filename alone. Draw and validate in a temporary location,
then publish the complete entry atomically. A failed or interrupted draw must
not replace a valid entry.

Keep the reference's build and input record with
the report so a reader can tell exactly what was compared.

## Sampling and evidence

The reference uses more rays per side than the candidate. The job schema
permits up to 64 rays per side; a 64-ray candidate has no higher reference
within that limit. More rays cost quadratically in samples per pixel;
measure the actual cost, including staging, rather than promising a time
from the ray count alone.

Draw the reference at increasing sampling and repeat it to measure both
convergence and run-to-run variation. Record the samples, differences and
identities; a higher count alone does not prove convergence.

see [the street region measurements](region-work-in-progress.md)
for cases where different measures did and did not follow sampling
changes.
