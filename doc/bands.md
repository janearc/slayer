# Rendering a spectral band

A band render represents what a sensor restricted to a wavelength would
record: an X-ray camera, radio receiver, or narrow visible-light filter.
The implemented CPU entry point is `trace.RenderIn`.

The later sections keep the parts of the original note's design that are
not built yet: lamps lighting volumes in a band, and the GPU path.

## Current CPU path

The first surface hit contributes its material’s `Radiant.Emits` value.
It also transmits light from behind it according to `Through`, raised to
the path length in local coordinates. Surfaces without `Radiant` are dark
and opaque in this path.

Volumes accumulate emission and attenuation along the ray. A `thing.Lit`
material can scale line emission by excitation, and radar beams produce
surface echoes according to `Scatters`.

A lamp that carries a spectrum, an `emit.Banded` such as `emit.Lit`, lights
surfaces in the band. The surface sends back its `Scatters` fraction as a
matte surface does in the visible: `Scatters / π` times the band's power
landing there, times the cosine of incidence.

The power landing is `Lands`: the inverse square and the lamp's cone. A
surface in the way shadows it, and the volumes between take their share in
the same band. A lamp without a spectrum, such as `thing.Sun`, lights
nothing in a band.

This is direct light only. Nothing bounces, and lamps do not light volumes
in a band.

Gas media without that emission path only
attenuate the band; their stored lines are not connected to emission.

Gravitational fields are omitted and the sky is dark. Output repeats the
band’s strength across RGB. The zero band selects ordinary visible-light
rendering. Validation follows `Render`, also rejecting negative or
nonfinite wavelengths.

## Proposed: lamps lighting volumes

A magnetar’s X-rays should illuminate a surrounding swarm, rather than
only appear where the source itself is visible. Surfaces in the swarm are
lit now, as above. Volumes are not: the design has them scatter
isotropically by their band-specific `Scatters` value.

The older audited `render/trace` has no lamp term in a band at all. Before
a GPU implementation, add and test the volume term on the CPU, then
consolidate the tracer paths.

## Proposed GPU representation

The current `gpu.Sampled` path renders visible RGB, not arbitrary bands.
For a future band path, evaluate each material’s `Emits`, `Through`, and
`Scatters`, along with emitter band power, once on the CPU and upload the
numbers. A Metal kernel cannot call Go material interfaces.

libtheme’s `bands.Light` owns spectra and band boundaries. The design’s
initial exclusions are line-excited gas (`thing.Lit`), textured roughness,
and radar beams. Unsupported cases should return named errors before
kernel execution rather than silently dropping those effects.
