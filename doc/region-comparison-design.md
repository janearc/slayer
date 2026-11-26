# Comparing regions of a picture

A region comparison draws selected rectangles at the full picture's pixel
scale. It puts each result beside a higher-sample reference, so that a change
can be judged before drawing the whole picture. The set names the job and,
for each rectangle, a short name and what the rectangle is meant to show.
The rectangles are in the complete frame's coordinates.

## Running a set

One command reads the set, validates every rectangle and requested setting,
then draws the regions and prints a report. Its verb and flags are declared
through puzzlemouth.

The command records the effective frame dimensions,
camera, time, renderer, sampling, threshold and development settings. It
reports scene staging and each draw separately, including elapsed wall time;
the total includes staging and sheet creation. A failed draw or missing
reference is an error, not a partial success presented as a complete set.

The output contains the candidate and reference pictures, their measurements,
and one labelled sheet made with `interfaces/sheet`. Each region is placed
beside its reference. The sheet is for visual judgment; the numbers do not
decide whether the picture is good. Inspect the sheet before placing it and
its explanatory note on the review page.

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

Run the set at
1, 4 and 16 rays and compare the numbers and sheet against the corresponding
whole-frame appearances. If the figures do not follow the visible change,
say so and investigate the measure rather than declaring a successful
prediction. A final visual judgment belongs to the person reviewing the
sheet.
