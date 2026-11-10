# The orbital scene

Two Culture orbitals surround a star inside a Dyson swarm, all suspended
in a nebula. This note records the scene on 2026-10-03: its appearance,
development history, and the proposed path from Go scene construction to
a data-driven film queue. One scene unit is 1,000 km.

The dimensions and capability limits below describe that historical
snapshot. [EXAMPLES.md](../EXAMPLES.md) describes the commands, and
[the lantern](the-lantern.md) documents the central swarm.

## Scene components

- **Orbitals:** two adjacent bands sharing an axis and rotating in opposite
  directions. The snapshot specifies a radius of 1.84 million km, a 24-hour
  day for approximately one g, width 32,000 km, and walls 1,000 km high.
- **Floors:** hexagonal biomes painted by `biome.Make`: seas, coasts,
  deserts, forests, downhill rivers, lakes, peak volcanoes, and cities
  beside water. City emission can be warm or neon.
- **Inner walls:** a black-marble pattern of city lights facing the gap.
- **Lantern:** a 9,000 km star at 9,000 K surrounded by twelve panel rings.
  Rings are spaced 1.2 stellar radii apart, from six to about nineteen radii.
- **Ring motion:** widths narrow from nine radii inward to two outward;
  orbital periods shorten from 60 to 15 seconds. End-over-end tumble
  periods shorten from 40 to 10 seconds.
- **Panels:** worn edges, holes, and dents, with blue-black glass on both sides.
- **Beams:** two poles offset 15 degrees from opposition, inclined 50 degrees
  to an axis spinning every 1.5 seconds. The axis precesses around a
  35-degree cone every five seconds. Axial light is 30 times the star's.
- **Nebula:** a custom wake medium, cleared along the orbital path with
  denser material outside and a ridge at the channel edge; density `4.5e-5`.
- **Camera:** stars behind, a 90-degree pinhole, and the documented close
  far-view setup 600,000 km away at 25 degrees above the orbital plane.

## Development history

On 2026-10-02, the paired orbitals surrounded a small magnetar in a neon-lit
nebula (`418637d`). A far view, black marble in the gap, and wider bands
followed (`7e8064d`). Before point lamps existed, a glowing sphere and
four back-to-back spotlights supplied the central illumination.

On the morning of 2026-10-03, `slayer/emit` introduced a point emitter
carrying a libtheme spectrum. `slayer/lantern` then replaced the glowing
sphere and spotlights (`9129407`).

The first swarm used sixty rotating rings with golden-spiral poles
(`0aaeae4`). A ten-second 720p animation was stopped after 75 frames
because of its sustained full load. The subsequent working constraint
was to avoid long unrestricted full-load runs.

Inspection of that film found dimming at frame 13 and a later halo.
The magnetar had remained beneath the lantern and flared after one
second. At the time, volume shadow integration also continued beyond
the light; that was recorded as a tracer bug, not a lighting choice.

A persistently lit patch of floor had a different cause: rings rotating
only about their own poles left the bands, and therefore some gaps,
stationary. This motivated the later tumble motion.

Fewer, wider bands made motion legible. Wear evolved from edge cuts to
smooth contours (`fd0f188`, `22ec4d1`, `c194dc2`). Rings were separated
radially, made faster and narrower outward, and given glass exteriors
with matte interiors (`a28fe6d`).

The magnetar was removed and panel count doubled (`a18a502`). Twelve rings,
blue-black glass on both sides, and a hotter but dimmer star followed
(`52f393f`). Polar beams (`7aad166`), band tumbling, and a thicker nebula
completed this snapshot (`00e508f`).

## How the snapshot is configured

`cmd/orbital/main.go` constructs the world through `scene` and `farView`.
The lantern comes from `slayer/lantern`, emitters from `slayer/emit`, and
biomes from `slayer/biome`. Wake, marble, and city-light textures are
custom Go types in the command.

The recorded version renders on the CPU. Its tiled textures, wake medium,
and polar beams were outside the GPU path's support. The command sets
the beam clock before each frame because `thing.Lamp` has no time argument.

A film is a run such as `orbital -far -from 600 -frames N -fps R -first K`:
numbered PNGs joined manually with ffmpeg. `-first` selects where to resume;
there was no persistent queue or finer scheduling abstraction.

## Proposed scene-data primitives

Express reusable components rather than adding another large scene struct:

- Bodies with sphere or mesh geometry and mass, with gravitational motion
  unless explicitly declared as driven.
- Point, polar, and linear emitters, with spectra and parent motion.
- Elements, deformation, per-face materials, and arrangements for panels
  bound around a body. A lantern is a composition of those primitives.
- Media described by fields: reach, channel, ridge, and density for the wake.
- Texture rules for biomes, features, cities, and marble.
- Explicit time-dependent lamps, media, emitters, spin, tumble, and precession.

## Proposed film queue

A shot file would import a scene and specify camera, time range, and look.
A film would contain ordered shots. Ranges should subdivide from minutes
to seconds to individual frames without changing their timing.

The proposed `direct -plan` command would turn the film into queue tasks,
each owning a frame or frame range and its expected PNGs. Recorded task
completion would allow resumption and rendering ranges out of order.
This is a plan, not an implemented command example.

For interactive or faster rendering, this scene also needs GPU support for
its textures, a baked wake medium, polar emitters, and lamp timing. Those
capabilities belong in the rendering engine rather than in scene-specific
copies of the algorithms.
