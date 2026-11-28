# Emitters

an emitter supplies illumination to surfaces and media. use
[surface emission](light.md#lights-and-things-that-glow) to make
the source itself visible. Slayer evaluates time-dependent lamp
state for the frame before sending supported sources to Metal.

| Go type | Light model |
|---|---|
| `emit.Point` | inverse-square falloff from a position. optional cone and time-dependent strength. |
| `emit.Polar` | two beams from a position, with a shared width and falloff. the second pole can be offset from the first pole’s opposite. |
| `emit.Beam` | finite Gaussian column with a source radius, reach and divergence. surfaces and media make it visible. |
| `emit.Volume` | sphere with integrated scalar falloff. visibility uses a ray to its centre. |
| `emit.Distant` | constant direction and incident strength across the scene. |

## Spin and precession

in `emit.Polar`, `Spin` is the period of the poles’ rotation about
the spin axis. `Tilt` offsets the magnetic axis from that spin
axis. `Cone` and `Precess` set the angle and period of the spin
axis’s motion about `Axis`.
