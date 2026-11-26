# Full still rendering

Profile.Full freezes explicit authored film width, height and rays from a
job or set, including a legacy set. Slayer receives --quality exact. By
default it receives no width, height or rays overrides. A size selected in
the setup sheet is frozen as explicit width and height overrides, while the
authored rays remain unchanged; the receipt records the preset name and
actual output dimensions.

The setup sheet can instead freeze 1–64 total samples per pixel for a CPU or GPU
still; it sends `--spp` and records the requested total in the receipt. The
producer refuses exact totals on physical rigs, regions and other
unsupported modes. The existing per-side rays choice remains available as
the authored default. GPU exact totals use software traversal; native and
accelerated lighting modes remain unsupported.

Output-size choices come from the producer's
`job.Presets()` table, and are not physical film gauges.
The sheet can also choose a camera named in an open set; the override is
sent as `--camera` and recorded in the receipt. Cameras in a set referenced
by a separate job are not yet listed in the sheet.

Frozen YAML retains the other film settings. Native StartFullRender shares
the existing generation, cancellation, observation and verified-result
pipeline. The sheet offers CPU and GPU; Azazel submits the request to the
matching queue lane. The default remains CPU for the Azazel transport. An
explicitly selected direct Slayer transport uses the chosen renderer and
configured processor setting.

For a set or still job in a read-only directory, the frozen editor buffer
is placed in the private evidence directory. Relative scene and asset paths
still resolve from the original document's directory. Catalogue documents
and sequence plans do not yet have a relocated-input adapter.

A full request refuses catalogue plans, movie jobs, and film settings whose
required authored size or sampling cannot be resolved. A selected output
size supplies the dimensions but still requires authored rays. Other cases
have no full adapter yet; they never fall back to preview.

An Azazel full-render request carries the
schema's seven-day maximum timeout; the 300-second limit is for previews.
Direct shared-Slayer transport has no elapsed time cutoff, so a still can
remain running overnight until completion or user cancellation.

Receipts measure PNG dimensions and record requested rays per axis and exact
quality beside the producer invocation. Rays per axis means rays squared per
pixel. Sampling and quality are invocation evidence, not measurements inferred
from the PNG. An accepted image requires successful producer completion and
matching authored dimensions. Process success is not artistic approval.
