# Rendering a spectral band

A band render represents what a sensor restricted to a wavelength would
record: an X-ray camera, radio receiver, or narrow visible-light filter.
use `trace.RenderIn` with a `maths.Band` wavelength in nanometres.

## Current CPU path

The first surface hit contributes its material's `Radiant.Emits` value.
It also transmits light from behind it according to `Through`, raised to
the path length in local coordinates. Surfaces without `Radiant` are dark
and opaque in this path.

Volumes accumulate emission and attenuation along the ray. A `thing.Lit`
material can scale line emission by excitation, and radar beams produce
surface echoes according to `Scatters`.

A lamp that carries a spectrum, an `emit.Banded` such as `emit.Lit`, lights
surfaces in the band. The surface sends back its `Scatters` fraction as a
matte surface does in the visible: `Scatters / pi` times the band's power
landing there, times the cosine of incidence.

The power landing is `Lands`: the inverse square and the lamp's cone. A
surface in the way shadows it, and the volumes between take their share in
the same band. only lamps carrying a band spectrum contribute here.

This is direct light only. Nothing bounces, and lamps do not light volumes
in a band.

gas media without the supported emission path only attenuate the band.

Gravitational fields are omitted and the sky is dark. Output repeats the
band's strength across RGB. The zero band selects ordinary visible-light
rendering. Validation follows `Render`, also rejecting negative or
nonfinite wavelengths.

## GPU

band rendering uses the CPU. Metal’s `gpu.Sampled` path renders visible
RGB. Celestial’s `bands.Light` owns spectra and band boundaries.
