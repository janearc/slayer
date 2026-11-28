# Caprica system: handoff and authored direction

`system.yaml` is an exact snapshot of the uncommitted system draft from
2026-11-27, based on Slayer RC `6753d4163`. It is preserved as a draft,
not a reviewed or rendered result. The draft's comment refers to a README
that was not present alongside the handed-over file. Its geometry and visual
claims still need assessment before shot selection.

In the 2026-11-27 creative brief, Earth-like Caprica orbits the lava drop
**Fresno**. **Amber** is a yellow gas giant close to the larger, hotter
star, and **Azure** is a blue ice giant. Azure is to have
about a dozen Hatzegopteryx soaring in its atmosphere with both suns
shining distantly. This is authored fictional direction, not a claim that
the animals or their flight on an ice giant are scientifically plausible.
The earlier standalone scene named `azure` was a palette experiment that
predates the Azure brief.

The system's fictional history gives Fresno an atmosphere after millions
of years of comet inundation. The draft's `drop-haze-*` wisps and wake
approximate atmosphere and outflow visually. The draft does not simulate
comet deposition or atmospheric evolution. Preserve those wisps while developing the actual history and
appearance, and do not describe the static comet fragments as a procedural
population or an animated breakup.

## What the draft already contains

The original binary geometry and Fresno's drop placement are retained:
9.2 AU separation, 48 AU drop orbit, epoch 4 years, phase 8 degrees and
the canonical `drop_at`. The white dwarf's 7400 K is a new authored choice.
The named `caprica` retains the original generated physical inputs and
`circles: pair`, but the draft adds a new made-up air look and cloud shell;
those appearance overrides are not automatically the same visual weather
as the earlier Caprica look book. Preserve the earlier artwork separately.

The existing `the-hot-giant` is a hot-Jupiter template near the primary,
with its template's escaping tail. Relative to the primary at the authored
epoch, its center is approximately 0.29026 AU away. It is a plausible
candidate for the Amber designation, but the draft currently does not
name it Amber and its cream gas palette has not been shown to achieve the intended
yellow look. Its cyan/violet crown is made of spotlighted scattering wisps,
not an emissive aurora. Preserve those shapes, lights, camera aims and tail
when mapping the designation; do not silently assert a physically simulated
aurora, wind accretion or escape.

Azure is not yet present. An actual location, size, appearance and camera
must be authored without inventing facts about its orbit. The existing
flowing-giant bridge refuses binary stars, while ordinary generated giant
materials and template bodies can be staged with the pair's actual lamps.
Thus the current bridge must not be forced past its guard or presented as a
flowing Azure atmosphere under two suns. The separate Hatzegopteryx
constructors in `manufacture/animals/vertebrates/reptiles/hatzegopteryx`
are useful starting points, but `doc/model/bioform/hatzegopteryx/README.md`
states that no existing study is approved as anatomically convincing. A
dozen distinct soaring individuals and their staging in Azure have not yet
been implemented. Existing fossil and flight analogies are documented in
`doc/model/bioform/hatzegopteryx/SOURCES.md`; exact anatomy, color, poses and
the fictional atmospheric flight must remain explicit artistic choices.

## Coordinate warning before selecting shots

The draft uses map coordinates east/north/up with `world.unit: 1000000`;
one authored unit is one million metres. A read-only `set.Pair.Placed` and
`thing.Orbit.At` calculation gives the primary's map position approximately
[-6822707.02707, 2213848.79561, 0], Fresno [-374.139656, -1027.940256, 0],
and Caprica [-0.0000002456, -0.0000000918, 0] at the still's epoch. The
draft's `caprica` and `drop-from-caprica` eyes have north coordinates near
-2052 and -2090, about 2.05 million km from Caprica, so they are not yet
shown to be near its cloud deck. This is a coordinate calculation, not a
render or formal review. Retain the draft camera values as historical input
and correct them in a separately recorded revision before choosing shots.
The read-only command `game run system-anchor -- locate
sets/caprica-binary/system.yaml` reproduces the binary and drop-orbiting body
positions through the constructor; its `primary_world` components use world
x/up/south axes, while the body line explicitly uses map east/north/up axes.

System-wide rendering at trillions of metres can stress single-precision
geometry and lighting. Camera-relative rebasing, asset scales, visibility,
light balance and stream continuity require evidence; no proposed image is
claimed to show them until a permitted render is inspected. The handoff says
no render had been made and calls for the creative lead and draft author
to participate in shot selection.
