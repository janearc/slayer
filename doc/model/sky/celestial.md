# Celestial: positions, observers, and time

The celestial design starts with one question: where does an object appear
from an observer's location, at a given instant, in a given frame?
The Sun over a hillside, Kepler-16b through the Sun's gravitational lens,
and a merger timed at two detectors all need that same foundation.

The celestial layer depends on measure and remains independent of rendering.

## Core concepts

**Instant:** a Julian date split into whole day and fraction. A single
float64 date resolves only about 40 microseconds at contemporary dates;
a two-part representation supports the much finer intervals needed for
inter-detector timing.

**Time scale:** the clock convention attached to an instant.

- **UTC:** Civil time with leap seconds

- **TAI:** Atomic time; the design records TAI-UTC as 37 s since 2017

- **TT:** TAI plus exactly 32.184 s

- **TDB:** Barycentric dynamical time, differing periodically from TT by under
  about 2 ms

- **UT1:** Earth rotation time, measured by IERS


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
coordinates, or a site's local azimuth and altitude. Azimuth starts north.

**Accuracy:** angular and distance error estimates carried with state
and sighting results. Initial implementations use Meeus's published
accuracy estimates; more precise sources can improve them later.

## Derived locations

Lagrange points derive from two orbiting objects. The Sun's gravitational
lens derives from its mass and radius: grazing light meets the axis beyond
roughly 548 AU, at `R^2c^2 / (4GM)`.

For a target, the focal line extends away from the Sun opposite the target's
direction. These locations are calculated rather than stored as fixed
positions.
