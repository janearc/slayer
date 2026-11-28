# Full still rendering

Profile.Full resolves the authored film dimensions and sampling from the
job or set, including a legacy set. For each dimension, an explicit
job value takes precedence over its preset and then its referenced set.
The estimate uses a byte-checked copy of Slayer's preset table. Slayer
receives `--quality exact` and uses the dimensions in the frozen YAML by
default. The setup sheet can supply explicit width and height overrides
while preserving authored rays. The receipt records the selected preset
and actual output dimensions.

The setup sheet can instead freeze 1–64 total samples per pixel for a CPU or GPU
still; it sends `--spp` and records the requested total in the receipt. The
producer refuses exact totals on physical rigs, regions and other
unsupported modes. The existing per-side rays choice remains available as
the authored default. GPU exact totals support software or native
traversal. Accelerated lighting remains unsupported.

Output-size choices come from the producer's
`job.Presets()` table, and are not physical film gauges.
The sheet can also choose a camera named in an open set; the override is
sent as `--camera` and recorded in the receipt. The sheet also lists cameras
from a job's referenced set.

Opening a referenced set from a supported still job retains its camera path
and still time for the working view, Preview and Full Render. The scene
file has no copy of the path. Choose an explicit scene camera to use its
authored lens and pose at the same still time. Keep the authored aspect ratio to render with
the borrowed path. The editor requires the set's own optics and sources.
Render the original job to retain movie paths, physical rigs, job-specific
sources, film or development overrides, titles, renderer or quality choices,
rendering shortcuts, or render limits.

Frozen YAML retains the other film settings. Native StartFullRender shares
the existing generation, cancellation, observation and verified-result
pipeline. The sheet offers CPU, software GPU and native GPU mesh traversal.
Azazel submits both GPU choices to the GPU lane. Native GPU requires a
supported Metal ray-tracing device and macOS 14 or later. Authored regions
require CPU or software GPU. The default remains CPU for the Azazel
transport. An explicitly selected direct Slayer transport uses the
chosen renderer and configured processor setting.

For a set or still job in a read-only directory, the frozen editor buffer
is placed in the private evidence directory. Relative scene and asset paths
still resolve from the original document's directory. Catalogue documents
and sequence plans do not yet have a relocated-input adapter.

A full still request refuses catalogue plans and jobs that author fps,
duration, start or scene-duration. A preset with a rate leaves a job as a
still unless it authors a movie clock field. Still renders require a
resolved size and sampling count. A physical frame derives its rows from
the resolved width and checks a height supplied by the job, its preset or
the render sheet within Slayer's one-row tolerance.

Azazel previews and full renders both use the queue schema's seven-day
maximum attempt timeout.
Direct shared-Slayer transport has no elapsed time cutoff, so a still can
remain running overnight until completion or user cancellation.

The completed full-render caption shows the frozen output dimensions and
samples per pixel. Receipts measure PNG dimensions and record the same
sampling count as `samples_per_pixel`. An explicit count is kept as given.
For authored rays, the spatial count is the square of the rays per axis
and the receipt also retains `rays_per_axis`. For a physical rig the spatial
count applies to each shutter glimpse. The receipt records the request's
total primary samples, physical frame, lens, focus, shutter and glimpse
count, and a job's authored development stock when present.

Sampling and quality describe the producer invocation. PNG acceptance
requires successful producer completion and matching authored dimensions.
The producer logs the completed sample count for exact GPU draws. Native
mesh queries and their outcome appear in the same log when used.
For a still with an authored region, the size selection controls the full
frame. The estimate counts samples in the crop, PNG acceptance checks
the crop dimensions, and the receipt records its position and full-frame
dimensions.
The setup refuses authored regions with a non-set source, wire mode, a
job aperture or stock development. Slayer still checks the resolved
scene, renderer and development before writing the PNG.

For a direct render of a job with a referenced set, Ferruccio checks that
the same set bytes still exist before the producer starts and after it
finishes. A persistent edit prevents acceptance and a receipt. These
checks do not attest which bytes the producer opened if a file changes
and is restored between the checks.

A completed job receipt records its referenced set's path and SHA-256
under `referenced_set`. The `source_sha256` identifies the job YAML.
Ferruccio checks the set hash against the matching completed Azazel
job record before writing the receipt.
