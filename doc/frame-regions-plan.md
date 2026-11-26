# Drawing a region and judging a picture

A region is a rectangle in the pixel coordinates of the complete picture.
Its camera, pixel footprint, sampling and exposure belong to that complete
picture. The result is the rectangle alone, at the same resolution as the
complete picture.

## Region

The job film and the draw command will accept a rectangle, with the command
standing over the job. Validate positive size and containment before staging.
Keep the full dimensions for the camera, sampling, detail and lighting grids;
carry the rectangle separately to rendering, output and retained-row keys.

Use the existing CPU section renderer where its contract applies. On Metal,
dispatch only the region while retaining global sample coordinates and the
full-frame header. Keep sample accumulation order unchanged.

The CPU section entry point presently validates the full frame's sample
allocation and does not accept initial media or cancellation. A small region
must not allocate a full fine frame, but relaxing that check requires separate
overflow and region-work limits. Preserve cancellation and initial media or
refuse them explicitly; do not claim that the existing entry point covers
those cases. Kept rows need a region-aware implementation or a clear refusal.

Development is part of the answer. Meter the full view as a whole draw does.
Effects which need neighbouring pixels require a sufficient halo and a crop
after development, or an explicit refusal. Coordinate-dependent stock,
titles, HUD, wire, composite and other paths need the same treatment; do not
silently return an image with different semantics.

Refuse frame-wide ReSTIR
unless its dependencies can be preserved. Check each other shortcut rather
than assuming it is local. Region identity must distinguish kept rows from
both the full frame and other rectangles.

First draw a small frame twice on each backend to establish repeatability.
Compare raw pixels and final output for several regions, including frame edges
and a nonzero origin, against the same build's full frame. Exercise invalid
rectangles, cache separation, sampling, effects and named refusals. Native
Metal equality is conditional on the control: report any observed variation.

Compare the street region to its original only if the binary and effective
inputs are established; otherwise compare to a fresh matching whole frame.

## Torture set

A YAML set names a job and regions with names and observations. One command
draws each at the requested sampling and compares it to a retained higher
sampling reference. Reference identity includes the actual build, resolved
inputs, renderer, settings and rectangle; incomplete or mismatched references
are never reused. A reference is published only after it is complete.

Compare linear light before clipping with a per-channel absolute error and
report signed light loss as well as a separately defined local bright-pixel
count. Verify with synthetic cases that discarding a bright sample incurs an
error; keep the count diagnostic rather than treating every isolated bright
pixel as a defect.

A finite noisy reference cannot guarantee every clamp
scores worse: measure reference uncertainty and keep that limitation visible.
Keep the chosen reference identity fixed across candidate changes, and report
both identities. Reuse the existing labelled sheet. Record the reference,
settings, timings and variation alongside the sheet, and inspect it before
adding it to the review page.

## Evidence and limits

Measure staging separately from drawing: each region can pay the full scene
setup. Run the five street regions at 1, 4 and 16 rays, compare their scores
per side, and also at 2 per side so the sequence includes 1, 4 and 16 total
samples per pixel. The recorded street image used 4 per side, or 16 total.

Report both units and compare scores and appearance with the observed
reduction in speckle. Reference convergence and repeated-run variation must
be measured, not assumed.

Permit up to 64 rays per side for whole pictures and bounded regions; do not
label 16 as a higher reference for a 16-ray trial. The supplied set's film defaults differ from
the recorded picture dimensions.

The separate 2k set also fixes
exposure at 13381, unlike the other portrait set; recover the actual invocation
and build before claiming reproduction.

The street set itself fixes exposure; another supplied set leaves it to the
meter. Both paths need coverage.

Keep code in small units: documentation, types and signatures, tests with
anti-tests, then implementation. Use Game for execution and the project's
review gates. The branch is ready for independent review only after its
evidence and remaining limitations are recorded.
