# Finite-layer optics for bioforms

Skin can return diffuse light on the illuminated side and pass some
light to the other side of a thin feature. Slayer's surface diffusion
models the first effect along a mesh. A finite-thickness model provides
a way to budget light between reflection, absorption and transmission.
The multipole work of Donner and Jensen includes lateral and transmitted
profiles and treats diffusion through a stack of finite slabs.
[Light Diffusion in Multi-Layered Translucent Materials](https://graphics.ucsd.edu/~henrik/papers/layered/)

## Two diffuse fluxes through a slab

`materials/tissue.SolveSlab` accepts ordered homogeneous layers. Each
layer specifies thickness in metres and effective absorption and
backscatter coefficients per metre for three linear RGB bands. The
two-flux model represents diffuse energy travelling toward and away from
the front of a locally plane-parallel slab. It solves each layer and
sums the repeated diffuse reflections between layers. Front and back
reflectance may differ with layer order. Transmission is reciprocal for
the model's matched indices. The solver is linear in layer count with
constant working storage. It computes integrated flux fractions and
does not provide a spatial or angular BSSRDF.
[Hébert and Becker, Correspondence Between Continuous and Discrete Two-Flux Models](https://lspwww.epfl.ch/publications/colour/cbcad2fmfratodl_08.pdf),
[Hébert et al., Compositional Reflectance and Transmittance Model for Multilayer Specimens](https://pubmed.ncbi.nlm.nih.gov/17767233/).

The exact choices of layer count, thickness and coefficients belong to
the caller. No human or animal data or default coefficients ship with
this model. An artistic input is authored. A calibrated input would
need documented diffuse reflectance and transmission measurements,
specimen geometry and band integration. The effective backscatter
coefficient is the coupling between the two fluxes. A measured
microscopic scattering coefficient or reduced scattering coefficient
cannot be substituted without a conversion or a fit. Existing human
pigment RGB values and Gaussian widths remain illustrative.

The current solver assumes matched refractive indices on both slab
interfaces and between all layers. It omits directional single
scattering, interface Fresnel losses, lateral displacement and
subsurface escape at the edges of the slab. It can evaluate a finite
effective stack at a representative thickness under diffuse
illumination. It does not predict how light travels through an actual
ear or how a material reflects at grazing angles.

## CPU transmittance through a thin feature

Use `thing.NewThinTissue(layers, share)` and assign the result to
`thing.Material.ThinTissue`. The layer thicknesses sum to the maximum
supported depth in scene metres. Their proportions stay fixed at other
depths. For example, a template of 2 mm and 6 mm is solved at 1 mm and
3 mm at a 4 mm chord. A zero share leaves ordinary shading in place.
The material's index supplies the two air–tissue interface losses.

`mesh.Mesh.InteriorChord` starts from a hit on a closed posed Slayer mesh
and finds the next crossing along a supplied inward direction, up to a
strict mesh-local distance limit. It checks face winding at the
crossings and rejects edge or numerically unresolved hits. This assumes
nonintersecting mesh components. Topological closure alone does not
establish that the mesh is free of self-intersection. The query uses up
to two ordinary mesh tree traversals. Its work depends on mesh shape
and traversal, with linear worst-case face visits. It sees only its own
mesh and does not check external shadowing.

At a visible hit, the CPU tracer queries along the inward normal of the
posed mesh. On a smooth mesh, the interpolated vertex normals supply
the optical axis and angles. A flat face or an interpolated normal
that does not point outward uses the geometric normal.

For smooth meshes, four additional sample directions surround the
normal column. A sampled entry can move sideways by up to one tenth of
its normal depth. All five samples have equal weight, including misses
and shadowed samples that contribute zero. Flat meshes use the single
normal column. Each sample must reach the first outward crossing and
pass the independent shadow checks. Winding, edge hits and unresolved
crossings follow the geometric chord query's rules. An unrelated
shadow-casting object that crosses a column suppresses its contribution.

The projected chord length times the placement scale is the estimated
normal thickness in metres. Transmission eases in as the dot product
of the inward axis and entry normal increases from 0.90 to 0.98. It
fades out over the last quarter of the supported thickness. Each column
represents a filled slab and omits enclosed or overlapping tissues.

At the entry the tracer samples ordinary directed lights and sky
through the scene's shadow and volume queries, including self-shadow.
The geometric face sets the ray origin and admits only directions in
its outer hemisphere. The optical normal supplies the incidence
cosine, the cosine-weighted sky frame, and the entry Fresnel loss.
The slab's transmitted fraction converts the entering irradiance to an
outgoing diffuse contribution.
The exit uses the Fresnel factor for the external view direction by
reciprocity, independently capped by the existing mirror remainder.
Ambient floods have no directed visibility and do not enter this path.
The bounded footprint approximates lateral transfer with the same
directions in each colour band. It omits refraction of the columns and
directional forward scattering.

The authored share is removed from the combined local and lateral
diffuse reflection at air-side hits, including where no usable chord
is found. The transmitted term can use at most this share of the entry
illumination per band. Reserved light that does not transmit is unused.
The solver's slab reflectance is not added to the existing surface
reflection. Paint, sheen and emission keep their own inputs, and
backlit colour follows the template's absorption and backscatter.
The transfer is a bounded mixture of renderer approximations. It does
not give a spatial BSSRDF or a calibrated skin model.

This path requires visible RGB, a closed shadow-casting Slayer mesh,
a finite positive placement scale and opaque dielectric material
optics. Metal and narrow-band rendering refuse the material. CPU scenes
with curved fields, declared initial clear media or the ocean
mean-plane or spotlight in-scattering paths also refuse it. A camera
leg currently inside a clear medium uses ordinary surface shading.
The opaque working view reports the omission of thin tissue
transmission.

## Backlit slab study

[`thin-tissue-study.yaml`](../../../sets/bioforms/thin-tissue-study.yaml)
places an opaque 2 mm control and transmitting 2, 4 and 8 mm slabs in
one frame, from left to right. The directional backlight is white and
a weak front light gives the control a lit surface. A small opaque
baffle should shadow a rectangle on the 2 mm slab. The transmitted
slabs should get dimmer with increasing thickness. All four slabs use
the same front colour. The optical coefficients and share in
[`manufacture/thin_tissue_study.go`](../../../manufacture/thin_tissue_study.go)
are authorable study inputs chosen to show the effect. This geometry
tests the normal-column approximation on plane-parallel slabs.
The native human ear is currently a relief on the head shell. A study
of transmission through a pinna needs a separately resolved thin
surface on that form.

```sh
game run slayer -- draw sets/bioforms/thin-tissue-study.yaml \
  --renderer cpu --camera backlit --out thin-tissue-study.png
```
