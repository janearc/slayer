# Single-fibre optics

`fibre.New` prepares a circular dielectric shaft surrounded by air. One
`Model.At` event uses the direction toward the camera and the signed
offset across that shaft. `Event.Evaluate` returns four RGB angular
densities, each per steradian. `Event.Energy` gives the corresponding
fractions integrated over directions.

- **R** is uncoloured Fresnel reflection at the cuticle.
- **TT** transmits through the shaft once, with absorption along the
  refracted chord.
- **TRT** reflects once inside the shaft before leaving it. Its longer
  path absorbs more light.
- **Higher** sums the energies of all further internal reflections and
  spreads them uniformly in azimuth, with a longitudinal lobe.

The longitudinal density follows d'Eon's expression in *Physically Based
Rendering*, §9.9. The first three lobes use Chiang et al.'s finite-period
logistic azimuthal distribution. Cuticle tilt shifts the longitudinal
peaks. A separate primary width can make the outside reflection narrower
or broader relative to the transmission lobes. Absorption is optical
depth along one diameter. For a coefficient in inverse metres, multiply
by the physical diameter in metres. The model uses RGB absorption, a
constant index in all three channels, and a circular cross-section. The
near-field approximation is nonreciprocal and approximates thicker fur
with lobe widths. It has no medulla, undercoat, or between-fibre transport.

## Hit and light contract

The hit must identify the shaft and provide the signed root-to-tip unit
tangent `T`, an outward radial direction `N`, and the direction `V` from
the hit toward the camera. Use the normal plane of `T` for projections
`Np` and `Vp`, made unit length. The view elevation is `asin(T · V)`,
and the offset is `T · (Np × Vp)`. For a source direction `L` pointing
away from the fibre, elevation is `asin(T · L)`. Azimuth is the signed
angle from `Vp` to the projected source direction, measured about `T`.
An exactly axial direction has no projected angle. Treat it as a limit
of the fibre geometry, with a consistent frame for the singular case.

The API supplies the factor already weighted for integration over solid
angle `d(sin elevation) d(azimuth)`. Multiplying it by a Lambert cosine
would apply an extra projection and discard backlighting. For sampled
area lights and skies, use the source radiance and the sampling density
in the same solid-angle measure. A point source uses its incident
direction and its existing distance falloff.

## CPU rendering

Use the model as the material on a shaft-frame mesh. In a groom layer,
set `Material: &thing.Material{Name: "hair", Fibre: model}`. The groom
then retains centres, tangents, and the face range of each shaft. The CPU
tracer reads a barycentric frame at each hit and evaluates direct light
from point, spot, beam and distant sources, including the existing spread
of a disc sun. It samples the sky and hemispherical fills eight times per
lobe, from the same distributions used by the evaluator.

The light path is a thin-fibre approximation. All four lobes share the
near-field hit point for external transport, with an oriented offset for
numerical roundoff. A ray beside the fibre starts on the outside of that
face. A ray through it starts just within the face, and consumes only the
origin shaft’s first outward crossing. Other shafts in the same mesh,
later hits on the original shaft, skin, and the rest of the scene can
still occlude it. Existing volume and gas attenuation applies. The
transport does not resolve each lobe’s separate surface exit location
or light refracted through intervening solid boundaries.

The CPU path accepts visible RGB and fibre materials with independent
shaft optics and ordinary air-side lighting. It refuses fibre materials
combined with diffuse, clear, metallic, emissive, or subsurface response.
Narrow-band rendering, explicitly seeded clear camera media and the
ocean-specific light paths are refused. The shaft model assumes air and
should be staged outside any solid or liquid medium. Fibre hits sample
lamps and sky only. Scattering between shafts, transparency to a scene
background, and traced glossy bounce paths from a fibre remain outside
this implementation. The polygonal shaft supplies coverage, so fine
strands still require adequate pixel sampling.

Metal refuses explicit fibre materials. Supporting them requires shaft
frames transported with the mesh, material packing, conditional lobe
sampling, and the same shadow-exit rule in the kernel. The groom can also
use ordinary surface materials on both renderers.

The analytic comparisons for later verification are normal-incidence
Fresnel, a zero-absorption white fibre, and increasing chord absorption
that attenuates transmission while leaving R unchanged. Independently
integrate the directional densities over a sphere and compare each lobe
against its energy fraction. A backlit isolated shaft and a two-shaft
occlusion fixture distinguish transmission from a light leak. These
comparisons have not been run.

Sources

- [Chiang, Bitterli, Tappan and Burley, *A Practical and Controllable Hair and Fur Model for Production Path Tracing*](https://www.disneyanimation.com/publications/a-practical-and-controllable-hair-and-fur-model-for-production-path-tracing/)
- [Pharr, Jakob and Humphreys, *Physically Based Rendering*, 4th ed., §9.9](https://www.pbr-book.org/4ed/Reflection_Models/Scattering_from_Hair)
- [NIST DLMF 10.25.2](https://dlmf.nist.gov/10.25.E2) and [10.40.1](https://dlmf.nist.gov/10.40.E1) for the modified Bessel function
