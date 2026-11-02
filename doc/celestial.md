# Celestial: positions, observers, and time

The celestial design starts with one question: where does an object appear
from an observer’s location, at a given instant, in a given frame?
The Sun over a hillside, Kepler-16b through the Sun’s gravitational lens,
and a merger timed at two detectors all need that same foundation.

This note records the 2026-10-14 package plan and its intended sequence.
It is not an inventory of the currently imported celestial library.
The planned layer depends on measure and remains independent of rendering.

## Core concepts

**Instant:** a Julian date split into whole day and fraction. A single
float64 date resolves only about 40 microseconds at contemporary dates;
a two-part representation supports the much finer intervals needed for
inter-detector timing.

**Time scale:** the clock convention attached to an instant.

| Scale | Convention |
|---|---|
| UTC | Civil time with leap seconds |
| TAI | Atomic time; the design records TAI−UTC as 37 s since 2017 |
| TT | TAI plus exactly 32.184 s |
| TDB | Barycentric dynamical time, differing periodically from TT by under about 2 ms |
| UT1 | Earth rotation time, measured by IERS |

Leap-second offsets belong to dated tables, not an assumption that the
recorded offset remains valid forever.

**Object:** anything providing position and velocity at an instant, from
stars and planets to a fixed point, an Earth site, or a spacecraft.
The common state uses the solar-system barycentre and ICRS axes.

**Sighting:** one object observed from another, returning direction,
distance, and emission time. Callers explicitly choose light-time,
aberration, solar-deflection, and refraction corrections. Refraction
applies to Earth-based observing sites.

**Frame:** conversion to sky coordinates, ecliptic coordinates, galactic
coordinates, or a site’s local azimuth and altitude. Azimuth starts north.

**Accuracy:** angular and distance error estimates carried with state
and sighting results. Initial implementations use Meeus’s published
accuracy estimates; more precise sources can improve them later.

## Derived locations

Lagrange points derive from two orbiting objects. The Sun’s gravitational
lens derives from its mass and radius: grazing light meets the axis beyond
roughly 548 AU, at `R²c² / (4GM)`.

For a target, the focal line extends away from the Sun opposite the target’s
direction. These locations are calculated rather than stored as fixed
positions.

## Gravitational waves

A merger’s light and waves share a sky location and distance. The proposed
wave layer adds:

- A merger object with two waveform polarizations and their orientation.
- A ground-based interferometer with a location and two arm directions,
  using a closed-form antenna pattern.
- Arrival delays from the detector separation projected along propagation,
  divided by c, supporting sky localization.

Build this after solar and lunar positions. Celestial supplies geometry
and timing; the renderer owns visual representation.

## Implementation sequence

Each stage uses independently published test answers with citations:

1. Calendar and two-part Julian dates: Meeus examples 7.a and 7.b;
   then time scales using IERS leap-second tables.
2. Sidereal time, Earth sites, and local coordinates: Meeus 12.a,
   12.b, and 13.b.
3. Low-accuracy solar position, about 0.01 degrees: Meeus 25.a.
4. Solar-lens focal distance from IAU constants and target direction.

Later additions retain the same interfaces: the Moon (Meeus 47.a), planets,
stars with proper motion, Keplerian and circumbinary orbits, Lagrange
points, detectors, and JPL ephemerides where the initial accuracy is
insufficient.

Keep rendering dependencies out. Tests must use external expectations,
and every empirical or chosen value must identify its origin.
