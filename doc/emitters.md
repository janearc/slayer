# Emitters

An emitter describes a light source independently of a visible surface:
a star's illumination, a magnetar's beams, or a jet. Keeping that description
as data lets the CPU evaluate its motion and send the result to the GPU,
which cannot call Go methods.

This document describes the emitter design. Check the exported APIs and GPU
validation for the kinds currently supported by a rendering path.

## Three forms of emission

- **Point:** a position, a Celestial spectrum, and a time-dependent strength.
  It shines equally in every direction and can carry child emitters.
- **Polar:** two beams with independent directions, widths, and falloffs.
  The second pole can be offset from the first pole's opposite; magnetic
  poles need not form a centred, symmetric pair.
- **Linear:** a jet extending along an axis, with a length and longitudinal
  falloff. Knots are a possible later extension.

## Spin and precession

Spin rotates an emitter about its own axis. Precession moves that axis
around a cone, specified by angle and period. A magnetar's beams sweep as
it spins; a precessing jet traces a cone.

## Parent motion

A child inherits its parent's position, spin, and precession, then applies
its own motion. Coupling controls how much rotation it inherits, from zero
to one; lag controls how far behind it follows. Full coupling is rigid
attachment. Lower coupling allows a beam to slip relative to its parent.

A magnetar can combine a hot point emitter for the surface with a polar
child for the beams, slightly offset from opposition and carried by the
star's spin.

## Wander

Seeded noise can vary direction and strength over time, on top of spin,
precession, and parent motion. A beam can flicker or stray while remaining
reproducible for the same seed.

Wander is a callback receiving time and parent state and returning the
emitter's deviation from inherited motion. Seeded noise is the supplied
implementation; callers can provide another.

Evaluate the callback once per frame on the CPU and upload the resulting
state. It must not depend on the current ray: that would require a Go
callback inside the GPU's per-sample work.
