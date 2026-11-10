# Slayer's packages

A guide to responsibilities and boundaries. [OPERATION.md](OPERATION.md)
follows the rendering pipeline; [EXAMPLES.md](EXAMPLES.md) describes the
scenes. Import paths below are relative to `github.com/janearc/slayer/`.

## Geometry and arithmetic

- **maths:** vectors, rays, boxes, rotations, subspaces, and linear RGB.
  Light may exceed one. The original Lamb implementation was reviewed
  before being retained; `maths/README.md` records its provenance.
- **maths/hexes:** hexagonal tilings of spheres, cylinder interiors, and
  planes, with point-to-tile lookup. Refinement preserves existing centres.
- **maths/noise:** stateless seeded hashes and smooth noise. No clock or
  shared random generator affects reproducibility.
- **geom:** simple types retained for older triangle-generation code,
  including the original conifers and GPU adapters.
- **radiance:** spectral radiance per square metre, steradian, and nanometre,
  including black-body emission from Planck's law.

## Scene descriptions

- **camera:** checked moves, speeds and holds, sampled from elapsed time
  without a mutable playhead. The path keeps its look target separate.
- **scene/thing:** worlds, surfaces, volumes, fields, lamps, sky, and haze.
  Motions place objects over time and compose through nested transforms.
- **scene/shape:** local geometry: spheres, planes, meshes, and ring tubes.
- **scene/texture:** spatial appearance, including checkers, hexes, planetary
  bands, storms, aurorae, city lights, image maps, and wireframes. Procedural
  textures use local 3D coordinates rather than only wrapped UVs.
- **scene/medium:** density fields for smoke, clouds, atmospheres, gas giants,
  discs, jets, plumes, pillars, and tails. Materials supply optical behaviour.
- **scene/gas:** band-dependent absorption, scattering, and phase functions,
  with air, haze, Venus, and Mars models. Gas emission is not implemented.
- **scene/sky:** distant illumination indexed by direction: gradients,
  stars, or hexagonal patterns.
- **scene/world:** plate-based terrain on hexagonal tiles, including
  mountains, rifts, sea level, climate, volcanoes, and canyons.
- **scene/bodies:** planets, stars, and black holes assembled from ground,
  atmosphere, rings, moons, discs, clouds, and other parts. Original sizes
  and masses were entered from memory and still need source verification.
- **scene/bodies/template:** named planets and moons, Pluto and Charon, the
  Sun, quasars, pulsars, magnetars, a hot Jupiter, and comets.
- **scene/swatches:** the named paint box used by the original scenes.
- **scene/smoke:** numbered scenes preserving rendering milestones.

## Light sources

- **film:** emitters and camera projections. Lamps include sun, flood,
  spotlight, laser, and radar; cameras include pinhole, physical focal
  lengths and gauges, and Hubble's camera.
- **emit:** data descriptions of light independent of visible surfaces.
  Time-evaluated emitters supply CPU lamps and GPU-compatible state.
- **lines:** approximate emission colours from NIST line wavelengths.
  Relative strengths are suitable for illustration, not spectroscopy.
- **flame:** soot emission, radiative heat fraction, and flame puffing.
  Comments distinguish measured, modelled, and chosen parameters.
- **fuelbed:** experimental surface heat and remaining fuel, with a CPU
  reference and a retained Metal stencil; the grass proxy cites Mandel et al.
- **plume:** fire-plume rise and spreading from measured correlations,
  with advecting eddies around a smooth mean column.
- **moonlight:** full-moon illumination derived from solar radiation,
  lunar size, distance, albedo, and atmospheric transmission.
- **skytable:** sampled sky illumination shared by CPU and GPU shading.
- **lantern:** a star inside orbiting reflective panels: a Dyson swarm
  casting shafts of light through its gaps.

## Procedural objects

- **manufacture:** objects grown from a kind and seed. Rows define ranges;
  the seed chooses reproducible values. Kinds include conifers, ferns,
  stones, logs, cans, fires, lamps, and a cliff house. Their dimensions and
  colours are fictional design choices unless explicitly identified otherwise.
- **forest:** ground geometry and seeded placements.
- **mesh:** validated triangle surfaces, bounding hierarchies, and
  intersection designed to avoid cracks at shared edges.
- **cast:** the dual of a hexagonal tiling, joining adjacent tile centres
  into triangles so geometry and colour share a lattice.
- **biome:** tile colours from height and wetness fields, producing
  clustered seas, coasts, forests, and deserts.
- **globe:** stable sphere intersection at planetary scales, tested against
  high-precision arithmetic.
- **terrain:** measured elevation models, including USGS Kilauea lidar.
  Tile-based reading avoids loading an entire multi-gigabyte raster.
- **typography:** font outlines triangulated and extruded with bevels.
  Repeated letters share mesh geometry.
- **materials:** elements, formulas, bulk materials, mixtures, and particles,
  with constructor validation.

## Planets and weather

- **planet:** single-scattering transfer through layered gas, haze, and cloud
  above a reflecting deck. Multiple scattering is omitted, so thick white
  haze appears too dark.
- **weather:** a rotating atmospheric fluid driven by storms and damped by
  friction. Jets and vortices emerge from the flow. CPU and Metal paths
  implement the model.
- **occultation:** ring optical-depth profiles from published Voyager
  stellar-occultation measurements.

## Tracing and frames

- **trace:** the newer CPU tracer and migration target. It handles surfaces,
  volumes, sky, gravitational fields, sampled pixels, and band rendering.
- **render/trace:** the older audited tracer, still used by `cmd/slayer`,
  `smokecheck`, `ring`, and `swarm`.
- **render/frame:** linear RGB, coverage, and first-hit depth in a pixel grid.
- **render/colour:** the darkroom: exposure, bloom, highlight compression,
  sRGB, false colour, letterboxing, and grain.
- **render/cells:** scene frames supplied as a stream for terminal display.
- **render/compare:** checks terminal output against a raster render of
  the same scene at coarser resolution.
- **render/shortcut:** approximations with explicit error budgets. Baked
  noise is active; other declared settings await tracer hooks.
- **render/smokecache:** render and comparison caching keyed by executable
  bytes, declared inputs, and rendering parameters.
- **render/carry:** adapters between Slayer and ninjafilm frame types.
- **raw:** linear RGB stored in PFM files, for later development or summing
  into a long exposure. Coverage and depth are not stored.
- **fermat:** deterministic replay of 64 pixels selected by a raw-image hash.
  It requires bitwise CPU agreement; GPU comparisons use tolerances.

## GPU support

- **metal:** a cgo bridge to Apple's Metal API. Kernels compile from source
  when opened and are checked against Go reference implementations.
- **gpu:** scene packing, haze baking, and GPU tracing through `gpu.Sampled`.
  Unsupported scene features return named errors. CPU results remain the
  reference, with declared tolerances for GPU approximations.

## Scene input and camera timing

- **set:** YAML scene settings delegated to their owning packages, with
  file/line/column errors. Example files live in `sets/`. See
  [doc/sets.md](doc/sets.md).
- **stage:** a set's world built for the tracer: boxes, spheres, lamps,
  spots, floods, sky, bodies, suns and wisps of gas, in the order the
  file gives them.
- **take:** camera paths, frame timing, and samples during shutter openings.
- **take/mocks:** generated motion mocks for take tests.
- **typography/mocks:** generated mocks for typography tests.

## Output and overlays

- **interfaces/file:** PNG output.
- **interfaces/terminal:** images represented by two-colour cells with
  six subpixels, plus terminal sizing and streaming.
- **interfaces/terminal/mocks:** generated test doubles.
- **interfaces/sheet:** labelled contact sheets.
- **interfaces/learn:** raster approximations of terminal cells for model
  inspection. Actual terminal fonts can produce different glyph shapes.
- **hud:** text, frames, bars, and labels over the rendered image, laid out
  in cells. World-anchored labels can request occlusion checks.
- **hud/mocks:** generated test doubles.

## Sources

- **sources:** published works and datasets fetched to an external shelf
  and checked against manifest hashes.
- **research:** named values read from research data with recorded sources.
- **observer:** the boundary from celestial spectral light to libtheme
  colour through the CIE 1931 observer.

## ninjafilm

ninjafilm develops rendered light with photographic sensitivity, curves,
and grain. Commands currently use its lenses, frames, and numeric helpers;
stock development is built and tested but not yet exposed by a command.

Six stocks evoke different film processes: `soot` sees blue; `ash` sees
blue and green; `smoke` records all RGB channels in monochrome; `dusk`
uses two dyes; `pollen` uses a dyed mosaic; `lantern` uses stacked dyes.
Stock data selects the response without a separate algorithm per name.

- **ninjafilm:** `Stocks` and `Roll`, the client entry points.
- **ninjafilm/roll:** exposure integration and grain-based development.
- **ninjafilm/record:** a photosensitive layer and its developed absorption.
- **ninjafilm/crystal:** grain area, expected counts, and variation.
- **ninjafilm/curve:** exposure-to-density response.
- **ninjafilm/lay:** stacked layers or mosaic sites.
- **ninjafilm/span:** whole-nanometre intervals and RGB membership.
- **ninjafilm/shelf:** stock assembly from the embedded tables.
- **ninjafilm/data:** CSV tables, chosen parameters, and parsing.
- **ninjafilm/lens:** thin-lens projection, focus, aperture, corner falloff,
  and integration of glimpses into raw exposure.
- **ninjafilm/film:** gauges from Super 8 to IMAX and Hubble, projections,
  cameras, and emitters.
- **ninjafilm/frame:** grids of linear light, coverage, and depth.
- **ninjafilm/lights:** physical spectral power, from gamma rays to radio.
- **ninjafilm/maths:** typed power, temperature, and area, plus Planck's law.
- **ninjafilm/sensor:** light collectors; camera and telescope extensions
  remain planned.
- **ninjafilm/colour:** development, sRGB, false colour, masks, and grain.
- **ninjafilm/grain:** stateless seeded gradient noise.
- **ninjafilm/hash:** deterministic site and frame randomness.
- **ninjafilm/number:** numeric and RGB helpers.
- **ninjafilm/mocks:** generated test doubles.

See [ninjafilm's guides](ninjafilm/README.md) for use, design, and sources.

## burning-trousers

The harness computes expectations independently and uses only exported
Slayer APIs. Its changes must be committed separately from renderer changes.
[INSIDE.md](burning-trousers/INSIDE.md) defines the exact boundaries.

- **fixture:** closed-form scenes for lighting, gas, mirrors, the white
  furnace, and CIE 1931 colour checks.
- **torture:** 200 random scenes checked for finite, nonnegative light,
  energy constraints, reproducibility, and scale invariance.
- **exact:** bitwise regression frames for behaviour-preserving changes.
- **kernel:** CPU and GPU evaluated against the same physical fixtures.
- **divergence:** perceptual image comparison using just-noticeable
  differences. The baseline allows at most 2% of pixels beyond one JND.
- **mesh:** comparisons between Slayer's mesh and the older shape mesh,
  plus per-face rays to detect missed geometry.
- **manufactory:** external checks of seeded output and closed,
  outward-facing skins.
- **rules:** import and commit-boundary enforcement.
- **scenes:** seeded random scenes for invariant and regression checks.
- **fake:** analytic lamps, geometry, sky, slabs, and sensors.
- **mocks:** generated implementations of renderer dependencies.
- **bench:** renderer timings using the harness's own fakes.
- **cmd/apart:** pixelwise comparison of raw images.

## Commands and supporting files

Commands live in `cmd/`; [EXAMPLES.md](EXAMPLES.md) shows their use.

- **`demo`:** List and preview numbered smokes and named scenes

- **`slayer`:** Render a set file and check its terminal representation

- **`smokecheck`:** Render and compare numbered smoke scenes

- **`swarm`:** Render the lantern

- **`orbital`:** Render paired orbitals around the lantern

- **`ring`:** Render a mirror ring around a flaring magnetar

- **`hillside`:** Render procedural objects or a set file

- **`moonrise`:** Render the Moon behind a tree

- **`house`:** Render the shoebox from its set file

- **`weather`:** Simulate and visualise atmospheric flow

- **`develop`:** Develop or sum raw frames

- **`sources`:** Fetch and verify published sources

- **`lightbox`:** Serve the smoke gallery


`tools/gate` tracks completed style migrations; `tools/mutate` exercises
mutations. `sets/` holds scene data. `smokes/` holds uncommitted
render records and images. `doc/` contains longer design notes and guides.
