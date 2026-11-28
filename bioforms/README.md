# Bioform paint and tissue

CPU textures on a Slayer mesh can use rest-space coordinates that the
mesh interpolates at posed hits. Give each corner its corresponding rest
point with `mesh.NewWithRestPoints`, or supply a `bioforms/assembly.Part`
with `Rest` and optional `Paint` coordinates. Use the same chart for
paint created separately from the mesh. Metal currently refuses painted
meshes carrying rest points.

`pigment.NewPaint` takes an ordered list of layers in that chart.
`pigment.Spots` gives a seeded point-sampled field. `NewScatter` gives a
footprint-filtered field for freckles, blemishes or speckles. `NewPatch`
gives a shallow ellipsoid for a blemish. `NewStroke` makes lines for scars and tattoo
outlines. Combine strokes for a design or supply a custom `Mask` for a
filled shape or an anatomy-specific region. Keep custom masks read-only
during rendering. Layer colours are linear RGB. Freckles and tattoos
filter reflectance, while scars and blemishes mix their own diffuse
reflectance into it. The marks change colour. Surface relief for scars
still needs geometry.

Apply the paint to an existing texture's colour using `Paint.At` and
forward the footprint if the texture implements `texture.Footprinted`.
`primates.Adult().WithPaint(paint)` connects the stack to the existing
native torso material. `primates.AdultHuman().WithPaint(paint)` applies the
same layers to the [native human exterior](../doc/model/bioform/human/README.md#skin-tone-and-marks)
through the constructor's default posed material. An empty stack
preserves the unmarked texture. Strokes and patches soften under a wider
footprint.

## Scattered marks

Give `pigment.NewScatter` a seed, a cell spacing, a maximum radius, a
density and a feather share. It generates mark centres and individual
sizes. An optional `Region` mask restricts coverage in the same chart.
Set each field's pigment through a `Paint` layer. The adult detail study
uses separate scatter fields for freckles and blemishes. Layers with
different seeds and sizes can also describe speckled skin on an animal.

Scatter approximates pixel coverage by widening and attenuating small
marks, then fading unresolved fields away. Choose the base palette for
the wider view. Footprint filtering uses a scalar width in the chart's
units.

## Healed scratches

`pigment.NewScarRake` generates a group of roughly parallel curved
scratches with tapered tips. Place `Centre` in the anatomy's rest chart
and give it two local surface directions. `Along` orients the marks and `Across` orders
them. Choose `Count`, `Length`, `Spacing` and `Radius` for the group. The
seed varies each stroke's path, length and width within those dimensions.
`Depth` limits the projection onto the surface and softens its edge.
Choose it for the local curvature and distance to any nearby skin.

Apply the mask in a `pigment.Scar` layer with the desired colour and
opacity. The mark moves with the chart when the mesh deforms and changes
diffuse colour only. The [adult cheek study](../sets/bioforms/adult-human-cheek-scar.yaml)
pairs a three-stroke group with the same freckled skin without the scars.

## Mottling

`pigment.Mottle` gives a signed local pigment shift. Its seed selects the
field, its scale sets the coarsest lattice spacing, and its strength sets
the maximum magnitude. Supply an unwrapped rest point in the same units
as the scale. Multiply the shift by a smooth anatomical mask to place it
on the face, the sides of a trunk, or the back of a flipper. The finer
octaves fade as the footprint approaches their spacing, an approximation
to averaging. At a wide enough footprint the shift is zero. A posed
mesh uses the posed footprint as an estimate of chart width.

The native adult uses separate seeded shifts for melanin and vascular
absorption. The python profiles use the same field to vary their authored
saddle or net palettes before adding the pale underside and facial marks.
Snake books can set `pattern.mottle` with `seed`, `scale` in metres and
`strength` from zero to one. `pattern.warp` accepts the same controls for
irregular dorsal contours. Its field shifts the longitudinal and angular
coordinates independently and tapers before the flank islands. Each
block is optional. The placed kind mixes its specimen seed with each
field's seed.

A marine reptile texture can use the same field in its own body chart,
layered under a dorsal or ventral colour mask. Mosasaur fossils preserve
evidence of eumelanin. The choice and placement of mottling for a
mosasaur reconstruction are illustrative.
[Fossil skin pigmentation](https://www.nature.com/articles/nature12899).

## Surface finish

Use `relief.Pores` and `relief.Furrows` for shallow pits and grooves on a
bioform. Their dimensions are in the paint chart's units. Both return
height gradients and fade their fine slopes as the pixel footprint widens.
Combine separately seeded fields or vary their depth with anatomy masks.

The CPU renderer reads `thing.Material.Finish` in the paint chart, carries
the slopes to the posed surface and uses the returned Satin for lamp
highlights. This changes the shading normal and leaves the underlying
mesh, silhouette and shadow intersections intact. Diffuse transport
still uses the material's separate subsurface and thin-tissue settings.
Metal refuses a procedural finish.

The native adult puts stronger relief on the nose, forehead and cheeks.
Its nose and forehead carry tighter highlights. Independent broad and
fine melanin and vascular fields set skin colour. The palette and relief
are illustrative and share the rest chart across poses.

## Diffuse transport in the CPU renderer

Supply a `thing.NewSubsurface` profile to a material on a smooth Slayer
mesh. Each radius is the Gaussian standard deviation for one linear RGB
channel, in the mesh's own units. The share is the fraction of diffuse
reflectance that can be supplied by neighbouring illumination. Choose
both for the material and scale of the specimen. The zero value keeps the
existing local material response.

```go
profile, err := thing.NewSubsurface(thing.SubsurfaceProfile{
    Radius: [3]float64{0.0024, 0.00108, 0.00052},
    Share:  0.24,
})
if err != nil {
    return err
}
material := &thing.Material{
    Name:       "illustrative skin",
    Colour:     maths.RGB(0.49, 0.32, 0.26),
    Gloss:      0.025,
    Index:      1.40,
    Satin:      0.72,
    Subsurface: profile,
}
```

The RGB widths above are illustrative millimetre-scale values for a mesh
built in metres. At a new render time and placement, CPU staging samples
unit-albedo diffuse illumination at each posed vertex. The ordinary lamp
and sky routines account for scene shadows, self-shadowing, and volume
attenuation. The hit retains its exact local lighting apart from the
fraction admitted to the mesh diffusion profile. The renderer multiplies
the mixed illumination by the hit's diffuse texture, so fine paint
details keep their rest-space registration. Emission and the specular
response are handled separately.

`diffusion.SurfaceProbes` computes each channel's weighted contribution
from other vertices and returns the remaining weight for the local hit.
`SurfaceDelta` expresses that operation as a change from the input vertex
lighting for callers such as the torso study. The shortest path follows
connected posed mesh edges. Distances beyond three Gaussian widths carry
no weight. Triangle area weights the samples. The available area at an
open mesh boundary is renormalized. Diffusion does not cross to an
unconnected sheet or another object. Other objects influence it through
the normal lamp and sky shadow paths. Vertex samples interpolate across
triangles, so narrow shadows and radii around or below the local vertex
spacing can be under-resolved. With no contributing neighbour, that
channel retains its local lighting.

Staging reads the ordinary lamp set and, with sky lighting present,
traces the existing sixteen sky directions per sampled vertex. Lamp
sampling and shadow-ray cost depend on the scene. The filter uses space
linear in the indexed mesh, plus the largest search queue, and runs a
local graph search from each vertex. A vertex can be revisited when a
shorter path is found. A broad profile can reach most of a connected
component per search and cost at least quadratic work in its vertex
count. No vertex count or performance envelope has been measured.

The CPU path applies the profile to the front side of an opaque,
nonconducting, smooth Slayer mesh with positive placement scale. It
refuses the mean-plane ocean light path and unsupported shapes. Narrow
spectral renders use the existing band material response. Metal refuses
the profile explicitly. This is a lateral approximation with one
Gaussian per channel. It omits single-scattered transmission through
thin features, multiple optical layers, wavelength-calibrated
coefficients and light bounced between objects.

The distinction between incident and outgoing surface locations follows
[Jensen et al., A Practical Model for Subsurface Light Transport](https://graphics.stanford.edu/~srm/publications/SG01-subsurf-abstract.html).
The RGB Gaussian approach follows the irradiance filtering principle in
[d'Eon and Luebke, Advanced Techniques for Realistic Real-Time Skin Rendering](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-14-advanced-techniques-realistic-real-time-skin).

## Finite layers and thin regions

The [finite-layer optics](../doc/model/bioform/tissue-layers.md) package
computes diffuse reflectance and transmission through a material stack.
`thing.Material.ThinTissue` opts a closed mesh into visible RGB CPU
transmission along a normal chord. This response reserves a share of
the existing local and lateral diffuse reflection and uses illumination
at the paired point across the mesh. See the same page for its geometry,
lighting and material limits and the backlit slab study.
