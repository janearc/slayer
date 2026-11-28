# Generated terrain: art-directed multiscale detail

2026-11-27. Proposed bounded slice; no implementation or render is claimed.

## Purpose and evidence

Make rocky worlds visibly different through the same system constructor,
with landforms that retain their broad identity as the camera approaches.
The immediate objective is a compelling orbital image, not a physically
realistic geological simulation.

Schott, Galin, Guérin, Paris and Peytavie, *Terrain Amplification using
Multi-scale Erosion* (2024, DOI 10.1145/3658200), amplify low-resolution
terrain with details at several scales, controlled by the artist. Their
method includes erosion and deposition approximations and aims at
hydrological consistency. The paper is pinned in `sources.yaml` as
`schott-2024-terrain-amplification`, SHA-256
`85a42371d704166f92ec6497cef98c0d458072285b75a7e3a13616344551203a`.
Its authors describe their approach at
https://h-schott.github.io/p/mserosion/.

The inference for this slice is modest: keep the broad terrain independent
of its finer visual detail, and let the constructor control their relative
strength. The proposal below does **not** implement that paper's erosion,
hydrology or deposition, and must not be described as doing so. Its invented
values, colour and any exaggeration are artistic choices, declared `made_up`
in the set and printed when drawn.

## Current path and limitation

`scene/generate/make.go` builds `world.Terrain` from the planet's seed,
water-derived ocean fraction, size-derived plate count and temperature-derived
cold; it supplies a fixed Earth or Mercury `world.Look` and fixed fineness.
`scene/world/world.go` builds coarse plate heights and biomes once.
`scene/world/scrooged.go` derives a continuous, multilevel *paint* from
coarse terrain and a tree, choosing detail by pixel footprint, with fixed
height, relief and ridge gains. `scene/bodies/bodies.go` draws a rocky body
as a sphere with that ground material. The apparent relief of this path is
paint: it does not displace the sphere, cast mountain shadows, or produce
true topography at a landing camera. Do not call it geometric terrain.

## First builder slice

Expose an optional, made-up `ground_look` on generated rocky planets. Its
first version should give artists independent controls for existing coarse
land coverage (`ocean`, share 0–1), fine land variation (`variation`, share
0–1), and ridge contrast (`ridge_contrast`, share 0–1). A nil block keeps
the exact old behavior. A present block must say every control and
`made_up: true`; zero is an explicit value, not omission. It is refused for
a giant. The body's existing seed remains the world seed and is printed with
the choices. The constructor, v2 schema, draw provenance and body-level
validation must agree; use existing Pzm and set machinery.

Implement ridge contrast by scaling the existing Scrooged ridge contribution
to the *paint decision*, not by changing planet size, geometry or the broad
coarse plate generation. Define the default so omitted input reproduces the
current constants exactly. At zero, preserve the base height/biome and other
fine detail while suppressing the extra ridge contribution; at one, use a
documented, artist-picked stronger ridge contribution. Keep interpolation
continuous, world-space and deterministic, so changing camera does not
reroll a world. Preserve existing coast and pixel-footprint handling. The
precise mapping from the authored share to contrast is to be chosen by
visual iterations and recorded as an artistic choice, not derived from the
paper. `ocean` changes the generated coarse world, so it must be applied
before `world.New`; it does not represent actual water volume. `variation`
sets the existing `Vary` paint parameter, not geometric roughness.

Before implementation, trace how `Ground.Fined`, `groundMaterial` and
Scrooged pass values to the GPU and check whether the requested controls
survive both renderers. Do not assume that a Go field automatically reaches
Metal. If ridge contrast cannot be carried without changing unrelated paths,
land coverage plus variation can be the first smaller slice and ridge
contrast must remain explicitly unimplemented. Do not broaden into a new
erosion simulator to ship a control.

## Visual work and assessment

Use two generated rocky worlds with the same physical inputs but different
art-directed blocks: one mostly ocean with restrained ridges; one mostly
land with stronger ridges and varied paint. Also make a matched pair changing
only ridge contrast. For each, render the same globe and nearer oblique
camera, plus a second longitude. Keep seeds, camera, lighting and exposure
fixed within comparisons; list exact settings and backend in each receipt.
The researcher inspects every frame at full size for visibly distinct,
coherent landforms, continuous coastlines and stable major features across
views. Check for stripes, square/hex seams and speckle as well as overblown
contrast. A painted planet that looks good from orbit is not evidence that
surface-level mountain geometry exists. Iterate visual failures and label
unfinished frames as studies.

No automated test or code review is requested in this slice. A render is
an observation of the chosen settings and backend, not proof of every
possible seed or a substitute for later integration work.

## Implemented controls, pending visual assessment

`ground_look: {ocean: 0.65, variation: 0.3, ridge_contrast: 0.5,
made_up: true}` is accepted on generated rocks. One half ridge contrast
retains the old multiplier; one doubles it and zero suppresses the ridge
contribution to the paint. This scale is invented for art direction. A
positive authored ocean uses the existing Earth water/land palette even on
a world whose derived water is dry; this does not add an atmosphere or
clouds. The implementation is pending rendering and visual assessment.
