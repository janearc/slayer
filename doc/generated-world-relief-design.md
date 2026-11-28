# Generated-world relief: a design for visible ground

Status: proposal, 2026-11-27. This describes work to build, not a feature
already present. It applies to generated rocky worlds in `world.bodies`;
the separate `scene.planet` spectral giant is a different renderer.

First slice: `world.NewRelief` now exposes a fixed-angle reading of the
warped coarse ground plus the Scrooged tree's height detail, scaled from
unitless heights to authored metres. It also estimates a normal by centred
differences. This is only a field: no shape, hit, shadow, water, or camera
placement uses it yet, and it has not been visually verified. Its angular
filter is fixed at construction rather than following each camera.

`Relief.WithRidges` is a separately authored height in metres. When
nonzero, it uses the same fixed-angle crest and rock-mask inputs as the
Scrooged mountain paint and fades the added height in above sea level;
zero preserves the base field. The fade width and height are artistic
choices, not evidence of erosion or a physical sea-shore model. This is
still a field: mesh and camera integration and visible comparisons remain
to be done.

## What exists

`world.World.Smoothed` blends coarse tile height and wetness at a body-space
direction. Its heights are dimensionless, not metres. `world.Scrooged`
builds finer, footprint-dependent height and ridge information to decide
land, water and paint. Its refinement is a visual sampling rule: it does
not currently displace the surface. `bodies.Body.build` puts that material
on `shape.Sphere`; `bodies.Stand` uses the sphere's radius. Thus the terrain
cannot today change a planet's silhouette, cast a mountain's geometric
shadow, or lift a camera above an actual ridge. That is the problem to
solve, not an assertion that its painted height already is geometry.

## One ground shared by every view

Make a deterministic body-space elevation function from the existing
seeded world. It returns a signed height in metres above a stated sea
level, and a normal or surface derivatives when asked. Map the existing
smoothed coarse land and sea values into an authored height range; define
the conversion explicitly, since the existing values are unitless. The
choice of range, ridge strength and any exotic formations is art direction,
declared by the set as made up. A zero-amplitude setting must be a sphere.

Factor the stable height contribution out of Scrooged so both geometry and
paint read the same underlying field. Pixel footprint may filter detail for
a particular representation, but must not redefine the world's geography:
two cameras, a shadow ray, a camera stand and a saved seed must refer to the
same physical surface. Explain any filtered approximation and its limit.
Sea occupies a consistent level against that field: coast, water interface,
land material, object placement and shadow geometry must agree. Do not
move a coast by independently re-rolling its noise for a camera.

## Geometry and shading are separate

Real displacement changes the hit point and horizon and participates in
occlusion, shadows and the local camera/placement surface. Derive its
geometric normal from the same displaced surface. Fine shading normals may
add sub-pixel detail, but they change neither silhouette nor hit distance;
do not report them as mountains. The existing planar `shape.HeightField`
separates geometric and optional shading normals and offers an instructive
contract, but it is not a spherical terrain implementation. Reuse its
ideas where appropriate without treating a tangent plane as a whole planet.

At large distance use a conservative enclosing bound and a representation
whose projected error is below an explicitly chosen pixel threshold. Near
the surface, refine a bounded visible region from the same field, joining
adjacent levels without cracks or steps. A camera moving between levels
must retain the same physical features and a continuous horizon. Keep
world and camera positions in double precision until rebased into a local
frame for the GPU; apply one coherent conversion to primary, shadow and
placement rays. Bounds must include the full displacement, and a missing
high-resolution patch must never silently uncover the old sphere. Limits
on detail and memory must be explicit; do not reduce planet complexity to
hide a failure. Geometry strategy (spherical patches, radial intersection,
or another method) is an implementation decision to measure, not yet a
claim that one is faster or correct in Slayer.

## What the sources support

Michelic, Kenzel, Haubenwallner, Kerbl and Steinberger, *Real-time Rendering
of Procedural Planets at Arbitrary Altitudes* (I3D 2019), describe a
projected persistent terrain/ocean grid adapting smoothly to changing
views and a ground-to-space presentation with atmosphere and clouds:
https://www.cg.tuwien.ac.at/research/publications/2019/kerbl_2019_planet_poster/ .
This motivates continuity across scales, not a claim that Slayer has their
method or its performance. The page is an institutional publication
record; pin a stable retrievable copy in `sources.yaml` before depending on
its implementation details.

Tzathas, Gailleton, Steer and Cordonnier, *Physically-based analytical
erosion for fast terrain generation*, Computer Graphics Forum 43(2), 2024,
author copy: https://www-sop.inria.fr/reves/Basilic/2024/TGSC24/Analytical_Terrains_EG.pdf .
They describe terrain shaped by uplift and stream-power erosion, including
approximations for hillslopes. This suggests a future explicit erosion
model; it does not justify calling our initial procedural ridges eroded or
scientifically realistic. Shelf and hash the paper through Slayer's source
workflow before implementation relies on it.

Schott et al., *Terrain Amplification using Multi-scale Erosion* (2024),
DOI 10.1145/3658200, is already catalogued as
`schott-2024-terrain-amplification` on the integration branch. Its
multiscale approach is relevant to future refinement; examine the pinned
author copy and its assumptions before adapting it. The initial work may
instead use existing authored Scrooged detail and must say so.

## Pictures that must answer the design

Make a small set of distinctive authored worlds, not a Caprica-only
special case. Render the same seed and surface from orbit, a grazing
mountain horizon, and near ground, with a light that reveals relief. Hold
camera and seed fixed for a zero-height versus displaced comparison and a
geometry-only versus shading-normal comparison; the latter must show which
changes silhouette and shadows. Include coast and camera stand views, plus
a continuous approach across detail transitions. Inspect every output at
full size, and record whether mountains, horizon and shadow actually read;
the fact that a draw completes is not acceptance. Do not call a picture
investor-ready until it reads as intended. Tests and code review remain
for the release owner to request.
