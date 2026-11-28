# How light works in Slayer

Slayer traces rays from the camera through geometry, materials and
media. each path contributes emission, reflected light and light
transmitted or scattered through the scene. the result stays linear
until development.

## Lights and things that glow

surface and volume emission make the emitter visible along a camera
ray. add an explicit lamp when that emitter should also illuminate
nearby surfaces or scattering media.

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

surface shading combines material colour, emission, lamps, sky and
supported reflections and refractions. opaque geometry blocks shadow
rays. volumes attenuate the light that reaches the surface.

mirrors have one followed bounce. refraction uses Snell’s law and an
unpolarized Fresnel share, with up to eight clear-surface crossings.
some reflected branches after a refraction are left untraced. rays
start in air by default. set the camera’s `initial-media` stack
explicitly to start in a clear medium such as water. see
[camera support](renderer.md#cameras-and-time) for the backend limits.

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

see [volume environment lighting](model/sky/volume-environment-lighting.md)
for the equations and incident-sky interpolation grid.

## The three kinds of stored light

- A **density bake** stores matter, leaving illumination to the trace.
- A **source-light grid** stores an approximation to incident light in
  space. Smoke grids, the optional incident-sky grid and `lamp-store`
  have different inputs, domains and refresh rules.
- A **render cache** stores a completed deterministic result so unchanged
  work can be reused. It should preserve that path's answer exactly.

the fast lamp store can soften a shadow inside mist by a cell. surface
shading uses its own light queries. the [lighting flags](flags.md)
cover other approximations, including shared sky samples and ReSTIR.

## Colour, bands and the darkroom

visible rendering carries RGB radiance. [spectral bands](bands.md)
are separate requests with backend-specific support and false-colour
display. the transport model omits interference, polarization and
wavefront diffraction.

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

compare a small direct-light reference before enabling shortcuts.
inspect narrow beams across moving frames for gaps or unstable shadows.
