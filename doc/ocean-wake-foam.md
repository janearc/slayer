# Aerated surface wake response

`world.ocean.surface-wake.foam` adds a bounded surface appearance mixture.
Each hit combines the original clear-water response with an opaque diffuse
and rough-highlight response. The uncovered branch retains its original IOR,
medium transitions, refraction and total internal reflection. Foam contributes
no emission. It is not a fluid solve, resolved bubbles or propeller cavitation.

```yaml
foam:
  coverage: 0.8
  band: 0.2
  reflectance: [0.9, 0.92, 0.94]
  satin: 0.8
  pattern-wavelength: {value: 3, unit: m}
  width-scale: 0.35
```

All values are authored appearance choices. Coverage is the maximum mixture
fraction; zero or an absent block preserves the original material. `band`
controls the ramp across the shared coherent noise's zero contour; it does
not alter the maximum mixture fraction. One noise octave uses the low 32 bits
of the ocean seed. Reflectance is linear RGB in [0,1]. `satin` is
the tracer's roughness parameter, not a measured bubble distribution.

The optional pattern wavelength defaults to the displacement wavelength;
the optional width scale defaults to one. They change only the appearance
mask, so finer breakup need not increase the wave mesh or alter its shape.

The mask uses the displacement wake's origin, heading, length and speed.
It fades to zero at the aft ends and twice the scaled half-width. Coordinates
are converted to local metres once at staging. Snapshot oceans freeze both
geometry and material at the authored ocean time. Scene-clock oceans evaluate
both at the render's absolute scene time.

CPU and Metal evaluate the same bounded analytic mask. The linear mixture
weights every clear contribution, including reflected/refracted light and
highlights; the opaque share survives TIR. Textured, emissive and shining
base materials are refused. Mean-plane directional and participating-source
approximations are also refused: their incoming source integrator does not
sample this mask's opaque coverage. Surface-only moonlit scenes are supported.

This is a local optical appearance approximation. It does not establish
foam production, bubble sizes, hydrodynamic energy or propagation. The surface
wake and any underwater propeller cavitation remain independently authored.
