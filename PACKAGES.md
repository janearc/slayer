# Slayer's packages

For the current pipeline and capability boundaries, read
[the renderer guide](doc/renderer.md). The [optimization catalogue](doc/optimizations.md)
connects these packages to their performance and invalidation contracts.

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
- **bioforms/framing:** whole-figure camera suggestions and tracking from
  posed part bounds. See [whole-figure cameras](doc/bioform-framing.md).
- **scene/thing:** worlds, surfaces, volumes, fields, lamps, sky, and haze.
  Motions place objects over time and compose through nested transforms.
  A look is made of a material and a finish; `Stopping` is what a material
  takes from a beam beyond the visible; and one clear thing may stand inside
  another, the higher `Rank` deciding what a point inside both is made of.
- **scene/shape:** local geometry: spheres, planes, meshes, and ring tubes.
- **scene/texture:** spatial appearance, including checkers, hexes, planetary
  bands, storms, aurorae, city lights, image maps, wireframes, and the rings
  of sawn wood. Procedural
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
- **scene/named:** scenes built in Go, by name, with typed parameters: the
  swarm, zappy, the orbital, the ring, moonrise and the magnetar, and the
  weather, a second kind drawn as maps with no world. A job names one with
  `scene: {named, with}`. Zappy has channels that music can play. It is a
  bridge: each is owed a set file.
- **scene/biome:** recipe candidates distributed on a fixed spatial lattice;
  authored scenery populations, not a model of plant ecology.
- **scene/frame:** an anchored float64 world and render-relative positions.
  It supplies geometry, not a projection, ephemeris or GPU transport adapter.
- **scene/footprint:** bounded geometry assessed through an explicit
  perspective view. It does not infer terrain extrema or lens optics.
- **scene/island:** sourced geographic terrain prepared in one spherical
  world; a view selects geometry while source coverage and frame stay fixed.
- **titan:** the ground of a world: heights from a map as far as the map goes
  and made up below that, on grissom's mesh.
- **titan/over:** a height map laid finer as a mesh, one stretch of it or
  every cell as finely as an eye resolves it, its survey kept where it was
  and detail made along the surface's normal from the map's own fall line,
  drainage, slope and roughness. It imports `mesh` and `maths`, and is
  imported by scenes and by `session` for a set's made ground.

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
- **embers:** prescribed firebrand trajectories and cooling; a rendering
  model, not a combustion or secondary-ignition solver.
- **firefront:** prescribed ignition scheduled over fuel locations; it does
  not predict wildfire spread.
- **plume:** fire-plume rise and spreading from measured correlations,
  with advecting eddies around a smooth mean column.
- **moonlight:** full-moon illumination derived from solar radiation,
  lunar size, distance, albedo, and atmospheric transmission.
- **skytable:** sampled sky illumination shared by CPU and GPU shading.
- **skyglow:** the glow a flat layer of air gathers from a city's own
  light, worked by orders of scattering on a grid; both tracers read it.
- **lantern:** a star inside orbiting reflective panels: a Dyson swarm
  casting shafts of light through its gaps.

## Procedural objects

- **manufacture:** objects grown from a kind and seed. Rows define ranges;
  the seed chooses reproducible values. Kinds include conifers, ferns,
  stones, logs, cans, fires, lamps, and a cliff house. Their dimensions and
  colours are fictional design choices unless explicitly identified otherwise.
  `MaturityReport` counts what each kind is made of and which of its numbers
  are measured or picked; `tools/maturity` prints it (`game run maturity`).
- **weathering:** what happens to a place, laid over every surface of a
  set's world by agents (rain streaks, soot, dust, moss, ivy, graffiti,
  rust, spalled brick, vacancy, a mineral source, glitter, nanomachines, a
  festival), each with a rule for where it can land and an age that only
  lays more. Colour only: it reads the CPU tracer's face direction through
  `texture.Oriented`. A set says it in `world.weathering`.
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
- **materials/specimen:** current solid phases and pore water, with SI mass,
  volume and density accounting. It supplies no hydration or optical model;
  see [Porous material specimens](doc/model/material/specimens.md).
- **materials:** elements, formulas, bulk materials, mixtures, and particles,
  with constructor validation. A material may carry optical constants per
  band and x-ray mass attenuation per energy, each reading with the row it
  was read from; `Passes` is the share of a beam through a thickness.

## Sound

- **beats:** editable audio grids, explicitly estimated detection and
  original/stem provenance. `cuesheet.Measured` makes a grid into a clock;
  the visual metronome remains the check before a long film.

- **cheeptune:** music from a score and a voicing, both JSON, on two
  machines: the four channels of a Game Boy and a studio (FM, analogue,
  plucked, struck and drum channels, a reverb and a delay). Rendered to
  samples and a WAV, with a playhead that gives the beat and the bar at
  any moment. It came from `tune` at commit 5a0d353.

  It holds a library of 33 tunes for videos, each with a cut sheet.
  `cheeptune/SCORING.md` teaches a small model to write a score; see
  [doc/music.md](doc/music.md).
- **cheeptune/audition:** a film of a tune's piano roll, and a page, served
  on this machine only, to keep or toss each tune.
- **cuesheet:** plays a scene's channels from music: tempo steps, steps
  that count, follows that breathe.

## Planets and weather

- **planet:** single-scattering transfer through layered gas, haze, and cloud
  above a reflecting deck. Multiple scattering is omitted, so thick white
  haze appears too dark.
- **weather:** a rotating atmospheric fluid driven by storms and damped by
  friction. Jets and vortices emerge from the flow. CPU and Metal paths
  implement the model.
- **occultation:** ring optical-depth profiles from published Voyager
  stellar-occultation measurements.
- **arid:** slayer's water in one package, gathered a piece at a time. It
  holds the tracer's wave surfaces: a sampled height field, the flat plane
  beyond it and a surface wake (from `scene/ocean`), and the cove that
  stages a shore with its water (from `scene/cove`).

  Still to come in:
  `scene/seaside`, `simulation/waves` and `oceantex`; the
  lake show in `improv/moonlake` and the water inside `session`, `trace`,
  `gpu` and `set` are left where they are. See [its README](arid/README.md).
- **oceantex:** the sea drawn from any point of view, a beach to orbit, from
  published formulas and measurements. It imports `titan` for its shore.

## Tracing and frames

- **trace:** the CPU tracer, used by `slayer`, `ring`, and
  `swarm`. It handles surfaces, volumes, sky, gravitational fields, sampled
  pixels, and band rendering. A look bends through as many as eight clear
  surfaces and knows what is on each side of every one, so water may stand
  in a glass and a bubble in the water. It mirrors once.
- **render/frame:** linear RGB, coverage, and first-hit depth in a pixel grid.
- **render/colour:** the darkroom: exposure, bloom, highlight compression,
  sRGB, false colour, letterboxing, and grain.
- **render/cells:** scene frames supplied as a stream for terminal display.
- **render/compare:** checks terminal output against a raster render of
  the same scene at coarser resolution.
- **render/shortcut:** approximations with explicit error budgets. `Apply`
  implements baked noise, but no package outside `render/shortcut` calls it;
  other declared settings await tracer hooks.
- **render/beamstep:** refines volume integration where a ray may cross a
  narrow light beam.
- **render/emptyspace:** conservative occupancy for interpolated grids; it
  does not alter a renderer's sample positions or transport.
- **render/smokecache:** render and comparison caching keyed by executable
  bytes, declared inputs, and rendering parameters.
- **render/smokejudge:** draws smokes as the terminal and as a real picture,
  by group, at full or small size, and judges the terminal's honest. A smoke
  with bands is drawn in them, through its tone and palette.
- **render/carry:** adapters between Slayer and ninjafilm frame types.
- **render/finish:** the tail every frame goes through after it is drawn:
  development by a plain tone or a ninjafilm stock, the mean of a frame's
  glimpses, titles over the picture for a span of the film, frames read
  from a directory, and the cut that fits them to a film's shape. It
  traces nothing and writes nothing.
- **render/live:** the view behind `slayer watch`: a job's frame drawn small
  at once and then finer, in a terminal, again when the job changes or a
  key asks. It keeps a status folder a graphical front end can read.
- **render/progress:** what a renderer reports while a frame is in flight:
  bands of finished rows, and a meter that gives the share done and an
  honest time remaining as a plain line or a human one. It has no terminal
  code and does not import a tracer.
- **render/workload:** bbolt observations of scene costs, keyed by complete
  source/build/device/render/probe identity. Preserves independent memory
  peaks and attempt/sample retry integrity; starts no work and grants no
  admission. Runtime measurement and pressure decisions belong to callers.
- **raw:** linear RGB stored in PFM files, for later development or summing
  into a long exposure. Coverage and depth are not stored.
- **fermat:** deterministic replay of 64 pixels selected by a raw-image hash.
  It requires bitwise CPU agreement; GPU comparisons use tolerances.

## GPU support

- **metal:** a cgo bridge to Apple's Metal API. Kernels compile from source
  when opened and are checked against Go reference implementations.
- **gpu:** scene packing, field baking, software/native Metal tracing,
  retained `Frames`, lighting shortcuts and surface/volume passes.
  `Renderer` adds a per-instance volume-grid policy and gravity seed cache;
  see [the renderer guide](doc/renderer.md) for capability boundaries.
  Unsupported scene features return named errors. CPU results remain the
  reference, with declared tolerances for GPU approximations.

## Scene input and camera timing

- **door:** the `slayer` tool and its verbs (draw, list, tunes, assess,
  develop, cache, watch, edit, schema); it joins session, finish and movie
  and draws nothing itself. `cmd/slayer` is a thin `main`. See
  [door/README.md](door/README.md).
- **diskcache:** the one place exact results are kept between runs, a
  file an entry, with a cap a kind. See [doc/caches.md](doc/caches.md).
- **treebolt:** the GPU's tree file, `trees.bolt`: its last-use record,
  its listing, its cap and the rewrite that gives the disc back. See
  [doc/caches.md](doc/caches.md).
- **contrib/nvim:** `slayer.lua`, the job editing setup `slayer edit`
  loads.
- **job:** reads and checks a job, the document that says how to render a
  scene: its schema, film presets, canonical form and key, and its flags
  and shortcuts. `jobs/` holds examples. It imports no renderer.
- **set:** YAML scene settings delegated to their owning packages, with
  file/line/column errors. Example files live in `sets/`. See
  [doc/sets.md](doc/sets.md).
- **session:** stages a set, a smoke or the procedural hillside and
  draws it. `Run` is its command line; `Open(ctx, job, stamp)` is its
  library door: a session that stages a job's scene once and draws its
  frames, at any size asked, as undeveloped light.
- **stage:** a set's world built for the tracer: boxes, spheres, lamps,
  spots, floods, sky, bodies, suns, wisps of gas and fizzes, in the
  order the file gives them.
- **take:** camera paths, frame timing, and samples during shutter openings.
- **take/mocks:** generated motion mocks for take tests.
- **typography/mocks:** generated mocks for typography tests.

## Improv

Each `improv/moonlake` program is its own command, written for one film;
[its README](improv/moonlake/README.md) is the worked example.

- **improv/walk:** short vertical films that walk a stranger through a set
  or a scene: a picture, labels at real places, scale bars and a panel
  with numbers and a sentence. Any set can use it.
- **improv/host:** a host laid over any film, in time with its music.
- **improv/visualizer:** a tune's instruments, a lane each, scrolling
  past a playhead, with the place written large as bar.beat.sixteenth;
  rgb24 frames for ffmpeg, like the metronome.
- **improv/searching:** the heard. shoot: a checked song edit,
  retained observatory and radio sets, and authored blazar views. Its
  [README](improv/searching/README.md) covers the small native preview.
- **improv/moonlake/omnipositor:** the Moonlake pass compositor, plus a
  [set adapter](improv/moonlake/omnipositor/SCENE.md) that combines scenery
  plates with actual 3D foreground transport. Plate reuse and reduced
  transport resolution are explicit preview choices.
- **improv/pan:** a film of stills moved under a window, cut on a tune's
  bars; `improv/pan/cmd/pan write SCRIPT --out FILM.mp4`.
- **improv/cloudsea:** the cloud sea with its guide stars corrected, a
  walkthrough script and its shorts.
- **improv/magnetar:** the short about a real magnetar.
- **improv/quasar:** the arithmetic behind the photon ring shorts:
  `rings`, `facts`, a diagram and the pull-back; see its `NEXT.md`.
- **improv/moonlake/untraced:** the light show in four panes in step with
  the song.
- **improv/moonlake/rig:** the light show's geometry and pass sum, shared
  by `lightshow` and the drawings.
- **improv/moonlake/show**, **explain**, **explainer:** moonlake's score
  and cue sheet reading, the explainer frame and its command.

- **improv/moonlake/cueshow:** compiles a score and a cue sheet into one
  moment's light state as key=value lines.
- **improv/moonlake/lightshow:** the lights for one moment: strobes, haze
  shafts and club lasers over a plate and its mask.
- **improv/moonlake/metronome:** raw frames for ffmpeg that flash each
  measured beat, for checking the score by eye against the song.
- **improv/moonlake/mirrorlake:** lays a raw's picture mirrored below the far
  shore, with a ripple.
- **improv/moonlake/mistlake:** moonlit mist over a lake, laid on a raw.
- **improv/moonlake/passes:** writes one set per stage fixture, each lit by
  that fixture alone.
- **improv/moonlake/pfmstat:** a pfm's largest value, its mean, and the mean
  of each tenth of its rows.
- **improv/moonlake/scalepfm:** multiplies a raw's light by two to the power
  of some stops.
- **improv/moonlake/seamatte:** turns a raw below the far shore into
  oceantex's water, in the raw's units.
- **improv/moonlake/seawater:** the flooded crater's water as oceantex sees
  it, lit by the moon, written as a pfm.

## Azazel and internal packages

- **azazel:** a foreground command queue under validation, with agent submission,
  ordered dispatch and durable results. `azazel/internal` holds `queue`,
  `budget`, `locking`, `processinfo` and `publication`, for host capacity,
  locks, process
  observation and file publication. See [azazel/README.md](azazel/README.md).
- **internal/appicon:** the artwork Slayer's viewers use.
- **internal/commandtest:** builds real commands for their integration tests.

## Output and overlays

- **interfaces/file:** PNG output.
- **interfaces/movie:** H.264 and AAC mp4 writing on macOS through
  AVFoundation, with no ffmpeg. It reads a file's facts back (`Probe`),
  sound files (`ReadSound`) and a film's pictures (`OpenFilm`).
- **interfaces/terminal:** images represented by two-colour cells with
  six subpixels, plus terminal sizing and streaming.
  `Live` gives a caller the screen's size, resize events, decoded keys, a
  clipped present and a status row, and gives the terminal back on a
  signal. Byte 0x1d is a host program's key and is never bound.
- **interfaces/terminal/mocks:** generated test doubles.
- **interfaces/sheet:** labelled contact sheets.
- **interfaces/learn:** raster approximations of terminal cells for model
  inspection. Actual terminal fonts can produce different glyph shapes.
- **hud:** text, frames, bars, and labels over the rendered image, laid out
  in cells. World-anchored labels can request occlusion checks. An
  element may be given a span of time to be drawn in, and there is a
  default theme for a hud that names none.
- **hud/mocks:** generated test doubles.

## Sources

- **sources:** published works and datasets fetched to an external shelf
  and checked against manifest hashes.
- **research:** named values read from research data with recorded sources.
- **observer:** the boundary from celestial spectral light to libtheme
  colour through the CIE 1931 observer.

## ninjafilm

ninjafilm develops rendered light with photographic sensitivity, curves,
and grain. Jobs expose its stocks through `stock`, used by `slayer draw` and
`slayer develop`, alongside its lenses, frames and numeric helpers.

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
- **animation:** a committed-executable 4K comparison of two scenes; see its
  [README](burning-trousers/animation/README.md).

## Commands and supporting files

Commands live in `cmd/`; [EXAMPLES.md](EXAMPLES.md) shows their use.

- **`demo`:** List and preview numbered smokes and named scenes

- **`slayer`:** The front door: draw a job, list, assess, develop, cache,
  watch, edit and schema; `slayer render` draws a set file. Judging the
  smokes is `slayer assess --smokes`.

- **`swarm`:** Render the lantern

- **`orbital`:** Render paired orbitals around the lantern

- **`ring`:** Render a mirror ring around a flaring magnetar

- **`hillside`:** Render procedural objects or a set file

- **`moonrise`:** Render the Moon behind a tree

- **`house`:** Render the shoebox from its set file

- **`weather`:** Simulate and visualise atmospheric flow; thin over the
  named scene `weather`

- **`sources`:** Fetch and verify published sources

- **`catalogue`:** List the manufactory and render a local specimen book,
  and compose it as a PDF

- **`grissom`:** The survey viewer: images on the sky's triangles, the earth
  as hexes

- **`island`:** Render sourced geographic terrain in a declared sphere

- **`quorum`:** Display the newest picture in a watched folder in a terminal
  pane

- **`show`:** Draw a PNG in a terminal that speaks kitty's graphics protocol


`tools/gate` tracks completed style migrations; `tools/mutate` exercises
mutations; `tools/breakdown` counts the files git tracks by kind, with
each kind's lines and share (`game run code-breakdown`). `sets/` holds
scene data. `smokes/` holds uncommitted
render records and images. `doc/` contains longer design notes and guides.
