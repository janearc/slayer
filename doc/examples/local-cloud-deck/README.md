# Local cloud deck study

This scene explores the existing `medium.Cloud` prescribed field through a
new `world.cloud-decks` constructor. The field cuts seeded 3-D fractal noise
at a height-dependent threshold, with rounded tops and clear gaps; it is not
simulated condensation or weather dynamics. Its dimensions, coverage, density,
scale, colour, camera and lights are artistic choices. The first render does
not establish GPU/CPU equivalence.

Job 585 produced a visible pale bank with a lumpy upper edge, but it was
soft, low contrast and filled most of the frame. The independent visual
assessment also saw broad flat grey patches. The image alone cannot separate
scene optics and noise from the GPU bake. This 260×260×60 m box is sampled
at 96 cells along its longest sides (about 2.7 m cells, ~23 vertically),
with trilinear reads; the noise scale is 35 m. `study-cpu-job.yaml` asks
the same scene at the same resolution and one ray on CPU, which samples the
field directly. It is a matched visual comparison, not a claim of backend
parity or an isolated measure of all causes of softness.

Job 588, the CPU comparison, was also soft, with broadly similar pale lobes.
That makes scene field, lighting or framing plausible contributors, though a
single CPU/GPU pair does not identify a sole cause. `billows-v2.yaml` is an
unrendered artistic attempt at greater separation and depth: coverage drops
from 0.65 to 0.4, the deck grows from 60 to 85 m high, the camera moves back,
and a stronger side key replaces the flatter light while fill is reduced.
These changes are concurrent and do not isolate individual effects. It is a
standalone study before applying the approach to Azure; no billow quality is
claimed until it is seen.

`base` is the centre of the bottom of the bounded box in map coordinates;
`size` is east width, north width and vertical depth. World units must be
metres, extinction is inverse metres and scale is metres. The underlying
generic GPU volume grid can lose small structure, and the finite box and
single-scattering lighting can be visible. The existing flat-air cloud-deck
refusal remains unchanged. No time-varying wind is exposed by this slice.

An optional `feather` width in metres smoothly fades density to zero at all
six box faces. Zero or omission retains the original field; a positive width
must fit within half of the shortest box side. This is an artistic boundary
control applied to density, not a screen composite or atmospheric simulation.
The GPU samples the resulting field onto its ordinary finite grid.
