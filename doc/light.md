# How light works in Slayer

Slayer follows rays backwards from the camera to discover what light
arrives at each pixel. Geometry decides where a ray goes; materials and
media decide what is emitted, reflected, scattered, absorbed or transmitted.
The renderer keeps the result linear until photographic development.

## Lights and things that glow

A visible glowing surface and a lamp have different jobs. Surface emission
makes the surface bright when seen. An explicit lamp supplies illumination
to other surfaces and scattering media. Emission from a volume appears
along camera rays, but the tracer does not solve general indirect lighting
from every glowing object onto everything nearby. Scene construction must
supply the corresponding lamp when that effect is needed.

The default surface mixture weights emission by the portion retained by
the surface after reflection and transmission. A CPU material can opt into
`IndependentEmission` to add its own surface emission separately, for
example a luminous wake on clear water. Metal encoding refuses this option.

`emit` supplies point/conical, polar, finite Gaussian beam, finite volume
and distant sources. Their geometry, falloff and units differ. A finite
disc source gives soft shadows through multiple source directions. A beam
is a spatial radiance distribution, not a luminous cylinder pasted into
the image. [Emitter notes](emitters.md) distinguish the implemented kinds
from proposals such as child emitters and wander.

A sky supplies directional background radiance and surface environment
lighting. Surface sky lighting uses hemisphere samples with cosine weighting
and checks visibility. It can illuminate shade even when no explicit lamp
points at that surface.

An afloat `setting: space` scene may choose `light: moon` and give a moon
azimuth and altitude. Slayer stages its modeled full-moon distant source and
the corresponding computed moonlit atmosphere sky; it does not infer lunar
phase or position from a date and place.

Participating ocean diffuse-sky transport samples that computed sky through
the authored mean-plane interface. The direct moon lamp still illuminates
visible surfaces through the ordinary surface path; it is not itself included
in the ocean sky-scatter integral.

## Surfaces and shadows

At the first surface, shading combines colour, emission, explicit lamps,
visible sky and supported reflected/refracted light. Opaque geometry blocks
shadow rays. Volumes attenuate light on those paths too. Shadow work asks
whether a lamp is visible; it does not make the space between lamp and wall
visible to the camera.

Mirrors have one followed bounce. Refraction uses Snell's law and an
unpolarized Fresnel share, with up to eight clear-surface crossings. The
tracer starts rays in air; it does not infer that a camera is underwater.
After a refracted path has already bent, some reflected branches are paid
for but not followed. These are material modeling limits even with every
optional shortcut off.

Measured n and k data can drive supported metal
finishes; [materials](materials.md) separates that implementation from the
larger composition design.

## Mist, smoke and a visible laser

For a ray segment, transmission T is the fraction of background light
that survives extinction. Source light S is emission plus light scattered
toward the camera. The segment contributes `S + T * background`. RGB
transmission can tint the background; transmission is not just an alpha
chosen for appearance.

The ordinary volume march uses 48 base strides and 10 shadow samples
toward a sampled lamp. Narrow-beam refinement splits strides near beam
envelopes so a thin beam is less likely to fall between samples. Its
envelope allocates effort; it does not clip the beam's tails.

Painted media use their colour as a scattering control and add emission
separately. Gas optics use their own band-dependent scattering and
absorption coefficients and do not currently emit. Dedicated procedural
media need their own encoding and support checks. Global haze is a
separate camera-path treatment; its model does not attenuate lamp light.

Matter is needed to see a beam from the side. Increase the relevant
scattering medium, check the beam crosses it, and check exposure before
raising beam power. Very dense matter can hide the distant beam through
extinction. A bright emissive background can remain visible through a
small transmission, so the march's existing faint-transmission stopping
threshold is a modeling limitation, not an absolute energy bound.

## Single scattering and its normalization

The legacy painted-medium lamp response uses an authored factor of 0.5.
The opt-in `isotropic-single` model uses phase density `1/(4*pi)` and
adds sky scattering. This deliberately changes brightness; turning it on
is a model choice, not just a faster evaluation of the old scene.

For ecosystem sets, the block is under `scene.volume-illumination`:

```yaml
scene:
  volume-illumination:
    model: isotropic-single
    sky-samples: 16
```

Merge that block into an appropriate existing set; it is not a complete
set. The island scene's input instead calls its block `volume-lighting`.
Do not confuse that with the ecosystem's `scene.volume-lighting:
direct|cached`, which selects a smoke source-lighting backend.

Sky directions cover the sphere in fixed antipodal pairs, with no surface
cosine factor. Counts are even, from 2 through 64. Every direction checks
geometry and medium transmission. This is single scattering: clouds do
not repeatedly illuminate one another, and isotropic painted mist does
not model measured droplet asymmetry or polarization.

See
[volume environment lighting](model/sky/volume-environment-lighting.md) for equations,
controls and the optional incident-sky interpolation grid.

## The three kinds of stored light

- A **density bake** stores matter, leaving illumination to the trace.
- A **source-light grid** stores an approximation to incident light in
  space. Smoke grids, the optional incident-sky grid and `lamp-store`
  have different inputs, domains and refresh rules.
- A **render cache** stores a completed deterministic result so unchanged
  work can be reused. It should preserve that path's answer exactly.

These are not interchangeable. The fast lamp store can soften a shadow
inside mist by a cell. Surface shading does not read that grid. Shared
`sky-looks` changes how the surface sky directions are distributed among
pixel rays. Read [fast light](fast-light.md) before using either in a
measurement; [modes](modes.md) gives practical choices.

## Colour, bands and the darkroom

Visible rendering carries RGB light; that is not a continuous spectral
transport solution. The tracer transports radiance, not coherent complex
amplitude: it does not compute interference fringes, polarization or
wavefront diffraction through gaps between objects. Spectral-band renders are separate requests with
backend-specific support. False colour assigns display colours to those
results; it is not the eye's view. See [bands](bands.md).

Linear RGB can exceed one. A PFM keeps that range. A PNG has already gone
through exposure, tone mapping and display encoding. Bloom spreads bright
image energy in development; it is not multiple scattering in the world.
Film stock adds its own response and seeded grain. Compare raw light when
checking transport and developed images when judging a film; keep the
same development for a visual comparison.

## Diagnose a dark or strange picture

Check camera direction, scene units and time first. Then separate a
surface with no illumination from a bright surface hidden behind mist.
Check whether a glowing object has a lamp, whether the medium receives
sky light, and which scattering normalization the set selected.

Compare
one small direct-light reference before enabling caches or shortcuts.
For beams, inspect continuity and shadows across moving frames as well
as the still: a coarse spatial sample can miss a narrow feature entirely.
