# The beam should leave the tube

A laser can say `launcher: true`. Slayer then places a shared factory
launch tube with its open exit at that beam's origin. The tube's positive
y axis follows the resolved beam direction, including any arcsecond offset.
The adapter retains the mesh and pose with the static scene.

The `guide-star-launcher` kind is also available in the catalogue. Its
3 m tube, collars and attachment shoe are authored scenery. The housing
is an ordinary closed mesh with an open bore, using the same GPU route
as the other manufactured equipment.

Beam radius, divergence and display strength remain separate controls.
A wide display beam does not establish an optical aperture or laser power.
There is no sodium return, adaptive-optics correction or finite-beacon
parallax in this outgoing beam model.

## Keep the instrument facing its target

A named thing may supply `heading: {value: 0, unit: deg}`. This overrides
its seeded placement turn; it is clockwise from the model's north pose.
Omitting it preserves existing seeded placement. Stand instances retain
their existing distribution of turns.

An optical recipe already bakes its azimuth and altitude into the assembly.
An explicit heading rotates that whole assembly as an additional placement
turn. Existing named ground instruments retain their authored 55-degree
altitude and north-facing local pose; heading changes their azimuth.

Laser origin offsets remain world east, up and south from the named base.
They do not acquire a local rotation when heading is supplied. A site must
place its launch tubes with the telescope structure and give the shared
sky pointing the matching direction. This keeps old scene offsets stable.

The golden-hour study uses the ELT family at heading zero and a common
northward 55-degree boresight. Its four launch exits sit just outside the
front truss rim, rather than at unrelated roof corners. This is an authored
mount arrangement, not an ESO engineering layout.

The [guide-star notes](guide-stars.md) and
[site expansion](observatory-site-expansion.md) retain the primary references
for the outgoing beam and the optional 64-arcsecond GALACSI pattern.

## Working-site study

Run the [golden-hour scene](golden-hour-site.yaml) through Game:

    game build camera-binary
    game run hillside -- -set doc/golden-hour-site.yaml -out out/golden-hour.png

Its main optical instrument is north-facing at 55 degrees. The nearby
Rubin family supplies a service wing, trucks, a crane, a mirror gantry and
utility equipment. The three personal telescopes stand in the visitor area.

The optional sun `tint: [1, 0.55, 0.22]` is a picked linear RGB filter on
the direct daylight lamp. Every channel lies between zero and one.
Omitting it retains white daylight; fill light is unchanged. This gives
the study a warm sun without claiming calibrated atmospheric absorption.
