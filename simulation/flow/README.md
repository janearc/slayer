# Periodic two-dimensional flow

This package provides a CPU reference and a Metal backend for
constant-density, incompressible Navier–Stokes flow on a periodic
rectangle. It models horizontal velocity, without a free surface or
full three-dimensional flow.

`Config` supplies a `spectral.Shape`, domain width and height in metres,
kinematic viscosity in square metres per second, and the largest
accepted step in seconds. Construction copies the input, projects the
field to a solenoidal periodic field, and preserves both velocity means.

`Velocity` returns independent row-major component slices in metres per
second. `Diagnostics` returns elapsed time, divergence RMS and mean
kinetic energy per unit mass.

`Step` accepts a positive interval up to the configured maximum. It also
requires `dt*(max|u|*kxMax + max|v|*kyMax + nu*(kxMax^2 + kyMax^2)) <= 0.5`,
where the wavenumbers are the highest retained modes in inverse metres.
This is a picked conservative guard, not a stability theorem. A refused
step preserves state and diagnostics. Nil or zero flows refuse steps
and return zero snapshots.

The CPU solver uses the dealiased rotational equation and classical RK4.
The rectangular two-thirds rule retains integer modes through
`|k| <= floor(N/3)` on each axis, including the zero mode. Real Nyquist
components have zero derivative. Tests use public spectral transforms
to measure divergence independently of the projection implementation.

The Metal backend retains projected spectral velocity. All four RK
stages share one session batch: vorticity, physical fields, rotational
products, Fourier transforms and projected accelerations. Intermediate
stages stay on-device. A device copy preserves FFT results when shared
scratch would otherwise replace them during a later transform.

`OpenMetal` uses the same model and guard. It also refuses spectra,
coefficients and RK weights outside its checked float32 range. It
requires a usable device and never substitutes CPU execution. Each
instance has one caller and an idempotent `Close`; accepted snapshots
remain available afterward.

A native failure closes the backend, preserves the accepted snapshots
and refuses further steps. A candidate becomes accepted only after
finite velocity and diagnostics pass. Metal arithmetic is float32;
CPU arithmetic is float64. Rounding does not preserve exact momentum
or energy.

Tests cover uniform flow, viscous decay, translating Taylor–Green
vortices and interactions between different Fourier modes. They also
check mean velocity, energy, independent divergence, both FFT stage
parities, copies and transactional refusals. Controlled frozen-equation
and omitted-copy faults fail the numerical assertions.
