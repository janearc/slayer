# Full still rendering

Profile.Full freezes explicit authored film width, height and rays from a
job or set, including a legacy set. Slayer receives --quality exact and
no width, height or rays overrides. Frozen YAML retains the other film
settings. Native StartFullRender explicitly selects CPU and shares the
existing generation, cancellation, observation and verified-result pipeline.
It runs the configured shared Slayer through the Game tests lane.

A full request refuses catalogue plans, movie jobs, and film settings that
require presets or inherited defaults. Those cases have no full adapter yet;
they never fall back to preview. Queue transport retains its existing
300 second timeout. Native direct shared-Slayer transport has no elapsed
time cutoff, so a still can remain running overnight until completion or
user cancellation.

Receipts measure PNG dimensions and record requested rays per axis and exact
quality beside the producer invocation. Rays per axis means rays squared per
pixel. Sampling and quality are invocation evidence, not measurements inferred
from the PNG. An accepted image requires successful producer completion and
matching authored dimensions. Process success is not artistic approval.
