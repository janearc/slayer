# Emitters

An emitter describes a light source independently of a visible surface:
a star's illumination, a magnetar's beams, or a jet. Keeping that description
as data lets the CPU evaluate its motion and send the result to the GPU,
which cannot call Go methods.

## Forms of emission

- **Point:** a position, a Celestial spectrum, and a time-dependent strength.
  It shines equally in every direction, or in a cone when `Width` is set.
  A `Pulse` function gives its strength over time.
- **Polar:** two beams with independent directions, widths, and falloffs.
  The second pole can be offset from the first pole's opposite; magnetic
  poles need not form a centred, symmetric pair.

The kinds `emit` has are `Point`, `Polar`, `Beam` (a finite Gaussian
column), `Volume` (a sphere of light) and `Distant`.

## Spin and precession

For `emit.Polar` only: `Spin` turns the poles about the spin axis, and
`Cone` and `Precess` move the spin axis around a cone, by angle and period.
A magnetar's beams sweep as it spins; a precessing jet traces a cone.
