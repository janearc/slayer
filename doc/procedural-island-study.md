# Seeded spherical terrain: regional study

This study connects an invented, deterministic spherical height field to
the existing `scene/island.Heights` interface and its bounded view-selected
patch geometry. The source's seed, base, relief height, wavelength and fine
amplitude are explicit art direction, and the spherical field does not
depend on camera. A 3D field avoids introducing a longitude-chart seam.
The procedural source is exclusive of raster inputs and must declare
`made-up: true`; its authored height envelope must fit the patch bounds.

Schott, Galin, Guérin, Peytavie and Paris, *Terrain Amplification using
Multi-scale Erosion* (2024), DOI 10.1145/3658200, pinned in `sources.yaml`
as `schott-2024-terrain-amplification` and described at
https://h-schott.github.io/p/mserosion/, motivates retaining coarse form
while controlling finer detail. Our ridged-noise formula, spectrum and
colours are artistic choices; this study does not implement their method,
erosion, deposition or hydrology. It is not a scientific terrain model.

The island selector samples and stitches a finite geographic rectangle,
not a whole planet. Its projected cell target is not an omitted-relief
error bound; a budget can limit the result. The first pair uses identical
source parameters, light and material, but different cameras; raw light and
render receipts accompany the images. Seeing a regional patch does not
establish global continuity, a globe-patch join, water consistency or a
stable moving-camera transition. No visual result is claimed until the
rendered pair is inspected.

## Mesh-footprint normal experiment

The optional `detail.normal-footprint: mesh` keeps the authored normal
stencil as a minimum but enlarges it to the coarsest selected cell touching
each shared vertex. Coarse/fine stitched edge values use the existing edge
conformance. It changes shading normals only; it neither changes positions
nor adds a shadow offset. It can change when patch selection changes, needs
a source halo outside the region, and supplies no bound on the angle between
the result and any face normal. Normal transitions and behavior at seams
remain to be assessed visually.

Asirvatham and Hoppe, *Terrain Rendering Using GPU-Based Geometry Clipmaps*,
GPU Gems 2, chapter 2 (2005),
https://developer.nvidia.com/gpugems/gpugems2/part-i-geometric-complexity/chapter-2-terrain-rendering-using-gpu-based-geometry,
compute normal data from terrain levels and blend across level transitions;
their normal map is finer than their geometry. Our coarsest-touching-cell
stencil is a separate engineering heuristic motivated by relating shading
scale to the selected representation, not their method or a proven fix for
shadow terminators. In particular, wider sampling can discard useful detail.
