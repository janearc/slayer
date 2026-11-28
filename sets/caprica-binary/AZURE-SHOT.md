# Azure local-shot plan

This is a construction plan, not a rendered scene. The brief calls for
roughly a dozen Hatzegopteryx soaring within Azure's atmosphere with both
suns in the distance. The flying animals, their placement, cloud shapes and the
fictional ability to soar here are artistic choices. The more detailed
Hatzegopteryx model is being adapted separately; no completed flock is asserted.

## Local frame and star directions

For the artistic inputs in `working-system.yaml`, `system-anchor` reports
Azure's generated radius as 46,726,027.1872 m. At the fixed system epoch,
set a local shot's camera at map [0, 0, 0] metres and Azure's centre at
[0, 0, -46725997.1872] metres: the eye is about 30 m inside the generated
gas volume's outer radius. Positions are east, north, up. The translated
primary and companion positions are approximately
[-2991957414000, 0, -46725997.1872] and
[-3438324706130, -1340101720235, -46725997.1872] metres, respectively.
Their angular separation at Azure is about 21.29 degrees, so both geometric
directions fit within a 50-degree view aimed between them. Their actual
visibility and exposure are unrendered, particularly the fainter white
dwarf. Fresno's translated `drop_at` is approximately
[3830375473414, -2214876735866, -46725997.1872] metres. These positions
are a translation and change of unit of the selected system snapshot, not
new star orbits or invented lights.

Give individual animals metre-scale positions around the origin, not
global coordinates near Azure's authored 20-AU position. This protects the
animals' local coordinates from that particular global float32 magnitude,
but does not establish that all surrounding geometry and lighting retain
the needed precision. The pair, its far star lamps and the camera must be
derived from the same source snapshot. Any omitted distant objects or
artistic local fills must be declared; do not silently substitute a
different system.

## Atmosphere and numerical boundaries

The generated giant is a `medium.Giant` volume with no opaque surface:
it is clear outside the radius and density rises smoothly over its depth
below the radius (`scene/bodies/bodies.go` and `scene/medium/medium.go`).
The first 30 m of its roughly 3,738 km density ramp are almost transparent;
being inside this nominal volume does not produce visible close clouds by
itself. Any local blue wisps or haze introduced for composition must be
declared an artistic atmosphere proxy, not asserted to be resolved fluid
weather or a physical simulation. Keep the actual gas medium and avoid
unsupported renderer combinations; whether the eye-inside-volume path and
both stars render correctly remains to be observed.

Azure's centre is still about 46.7 million metres from this eye, where a
global float32 coordinate has roughly four-metre granularity. The sphere's
crossings and the near-edge density still need numerical observation.
Further, the tracer's default secondary-ray offset scales with the combined
reflective scene extent (`trace/stage.go`); keeping distant reflective
Caprica while staging metre-sized animals can make that offset inappropriate.
`thing.World.RayOriginReference` can override the reference internally, but
no set authoring key is presently established. A safe implementation needs a
declared reference or another evidenced local-origin approach; simply adding
the flock to the old global set does not solve the problem. Native/GPU
numeric limits, medium shadows, culling and source exposure must remain
explicit. Do not bypass a renderer refusal to make the picture run.

Candidate shots and visual proxies should be recorded and chosen with the creative lead
and the system draft's author before rendering. No frame, animation, physical
flight or end-to-end precision guarantee is claimed by this plan.
