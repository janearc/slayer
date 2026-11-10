## v0.1.1 09 november

This is the consolidated change inventory since `2a4d86d`, on the working
branch `dist/2026-11-09`. It includes work also summarized
in the dated release notes below. Drafts and missing integration are called
out where they belong; this inventory is not a fresh test or release verdict.

### Camera travel, sequences and demos

- Added an island-sequence draft with geographic camera/target keyframes,
  angular zoom and separate scene/playback clocks. Source rasters, native
  height indices and bounded prototype geometry remain open across frames;
  view meshes and GPU resources are rebuilt. Each frame records its exact
  effective scene, hashes and costs. This is authored geographic travel,
  not an orbital solution or certified obstacle-avoidance path.
- Verified a first native 60-frame, 320 by 180 ascent through a one-hour
  scene clock to 300 km viewing height, encoded as ten seconds at six fps.
  Frame/source hashes and repeated tree identities were checked. Visible
  terrain faceting, hard authored colour bands, wave water and instrument
  lookdown remain open; this is a choreography study, not the final shot.
- Added the `camera` package: checked constant-speed travel, timed moves,
  linear or smooth easing, holds, continuous joins and frame-time sampling.
  Paths refuse invalid coordinates, timing and crossings of the look target.
- Added a look target per move through `Move.Looking`, including moving
  targets and pans during holds. Position and look joins are both checked;
  these targets are scene coordinates, not object-relative flight planning.
- Added optional angular zoom, independent of travel, with linear or smooth
  easing and exact held endpoints. Existing plans keep their fixed lens.
- Added hyperlapse `scene-duration`, separate from playback `duration`.
  Camera speeds, fire and prescribed weather use scene time; frame records
  retain their actual scene timestamps. This does not animate fixed Sun or
  Moon directions from an ephemeris. See [camera motion](doc/camera-motion.md).
- Added Hillside's Pzm `sequence` command and bundled JSON Schema. It retains
  scene construction between frames, accepts terrain scenes or numbered
  smokes, and supports camera plans, dry runs and resumable numbered PNGs.
- Sequence manifests identify scene, plan, executable and referenced input
  bytes, including terrain and smoke assets. Resume verifies real files,
  frame hashes and timestamps; changed or missing inputs cannot silently
  reuse an earlier frame. Failed preparation does not publish completion.
- Added fast lossless `SequencePNG` and `SequencePNGHash` output. Frames are
  encoded before atomic replacement; the digest describes the bytes written.
  This trades larger files for cheaper compression without changing pixels.
- Added `game run demos` and `game run demo N`, with stable smoke selectors
  and orbital, swarm, ring, hillside, moonrise, house and camera previews.
  Defaults and sample budgets live in a checked menu. Optional terminal
  output keeps the PNG and uses the existing sextant renderer.
- Demo commands build their actual scene executable and report outputs under
  `out/demos`. Missing data is a refusal, not a successful blank picture.
  Argument forwarding needs a suitable Game build; private review dependency
  pins still limit the fresh-checkout experience. A broader preview/data/cost
  guide remains open. See [demos](doc/demos.md).

### The manufactory and its specimen book

- Added `hull`, `clumps`, `quick` and `demo` complexity controls, named YAML
  construction recipes and shared geometry. Detail changes representation;
  instrument apertures and declared physical envelopes stay fixed.
- Added a native-Metal specimen catalogue with local search, family/detail
  filters, comparison views, observatory cutaways and source notes. Each
  view records its binary identity, inputs, bounds, stored faces, GPU timings
  and native query counts. Pending and failed views stay visible in a partial
  book. See [the specimen book](doc/browsable-catalogue.md).
- Catalogue runs refuse unavailable GPUs, unsupported details, excessive
  sample/geometry budgets and existing output destinations. A native image
  records that view's execution; it does not certify every kind or establish
  CPU agreement. The book covers named kinds, not all scene recipes.
- Corrected loose-shoot catalogue orientation and improved specimen sampling
  and the uniform backdrop. Added viewing headings/elevations for useful
  instrument views without changing the manufactured object.
- Embedded the supplied light and dark Miami artwork. Grissom selects its
  native application icon by system appearance; the catalogue includes both
  favicons. Miami's scenery, typesetting and HUD remain design references,
  with their unimplemented work tracked separately.
- Added opaque `manufacture.Kind` and `AttachedLight` aliases, owned local
  light descriptions and `PlaceLights`. Placement caches can retain lights
  without keeping a specimen's construction geometry alive.
- **Go API change:** specimen `Lights(placement)` now returns a light list
  and an error. Callers must handle invalid placements. `Things` and assembly
  paths also propagate placement/construction failures without partial goods.
- Hillside's old count flags retain their original kinds. New kinds enter
  through recipes and YAML; this does not expand the legacy flag catalogue.

### Trees, brush and ecosystems

- Added an optional distant crown representation derived from each seeded clump
  specimen's material and occupied height/quadrant bounds. The woodland example
  keeps geometry farther away. Optional overhead-area quadrature and a
  documented overlap approximation replace the assumption of full canopy cover;
  both the sample work and estimated coverage enter render records. Candidate
  bounds reuse region height envelopes before individual height reads.

  Moving appearance and directional transport remain under review.
- Added optional hierarchical region assessment before root enumeration in
  island forests. Distant supports use authored canopy appearance; measured
  prototype envelopes, source-height bounds and shared work limits govern
  nearer cells. Added stable density/colour thinning at region edges and a
  wider woodland example. Moving-view and resource inspection remain open.
- Added an island-site draft with source-grounded equipment and seeded trees,
  projected-size detail selection, authored aggregate landcover and regional
  prescribed mist. A bounded native height index tightens refinement priority.
  Camera-relative placement shares the island frame; terrain datum conversion,
  temporal residency and inspected moving transitions remain open. See
  [the sourced site study](doc/island-scene.md).
- Indexed selected terrain edges by lattice line instead of scanning every
  leaf at every vertex. Explicit island command ceilings allow denser studies;
  their cell-span targets still report when quality is limited by the budget.
- Added optional authored elliptical mist-bank edge fades after an overview
  exposed rectangular clipping. Zero fade retains the original deck support.
- Added an optional checked secondary-ray reference length. Its zero value
  preserves the geometry-derived default; explicit planetary scenes can pick
  a local base without changing the shared CPU/GPU offset multiplier or the
  point-magnitude contribution. This is a heuristic, not an error certificate.
- Added coast redwoods, giant sequoias and their separate shoot specimens.
  Recursive foliage uses physical needles or scales and bounded shared shoot
  prototypes on CPU and GPU. Construction notes retain botanical dimensions,
  algorithm references and authored choices. See [conifer foliage](doc/conifer-foliage.md).
- Drafted physical recursive shoots for the named spruce and Norway spruce
  entries while retaining their legacy scaffold/reference constructors.
  Foliage carriers are identified by role, including spruce without hanging
  branchlets; physical leaf size is independent of tree scale.
- Drafted `valley-oak`, `alder`, `poplar`, `manzanita`, `chaparral-brush`
  and `joshua-tree`, with distinct forking/leader/multistem/rosette habits
  and closed leaf meshes. Saguaro has separate pleated columns, curved arms,
  rounded caps and shared spine geometry. These are authored family recipes.
- Added named summer, autumn and winter recipes through `WithSeason` and
  YAML. Deciduous winter removes blades while retaining wood and the leaf-on
  reserved envelope; evergreen foliage remains. These are discrete authored
  states, not simulated phenology. See [vegetation families](doc/vegetation-families.md).
- Added bounded spatial populations in `scene/biome`, separate from the
  older top-level `biome`. Region/seed/cell/ordinal identities and independent
  draws keep candidate positions, recipe, geometry seed and heading stable
  across overlapping queries. Query boundaries use half-open ownership.
- Added local `scene.ecosystems` mixtures, density and elevation filters,
  exclusions, prototype variety, detail and optional shared atmosphere.
  Regions share one preparation work allowance; exhaustion returns an error
  rather than dropping later regions. See [regional ecosystems](doc/regional-ecosystems.md).
- Hillside mixed populations still use the cropped planar terrain stage. Final ground
  placement and filtering can change with the crop. Whole-world residency,
  aggregate distant canopy, projected-size vegetation detail and inspected
  mixed-biome scenes remain open; the new plant drafts are not all GPU-accepted.

### Observatory instruments and working sites

- Added a configurable optical observatory: slit dome, mount, truss and
  segmented or monolithic primary. Added Palomar, Keck I/II and Subaru
  scenery families with separate source dimensions and authored supports.
- Added Rubin's survey optical arrangement, shared primary/tertiary backing,
  annular secondary and camera housing. Its facility now has a faceted
  enclosure, base and tiered service building, doors and a separation joint.
- Added ELT scenery with a nominal 39 m primary, 798 installed panels and
  five mirror roles. The thin-support CPU/Metal cutaway discrepancy was
  resolved by the ordinary triangle-intersection repair below. Added draft
  OWL open-frame scenery and an independently placed roll-away shelter, with
  shared component instances and connected structural attachments.
- Added Griffith's rotunda, wings, rear planetarium and east refractor with
  a source-sized 12-inch objective. The west solar instrument is a closed
  roof proxy; none of these buildings is an engineering replica.
- Added Spitzer, Hubble and Webb spacecraft templates and checked custom
  apertures. Webb has 18 primary panels and five shield layers. These are
  ordinary scenery meshes, not compound optical prescriptions or orbits.
  See [space observatories](doc/space-observatories.md).
- Added Maunakea families `cfht`, `gemini-north`, `irtf`, `uh-88`, `ukirt`,
  `jcmt`, `sma-antenna`, `sma-array` and `vlba-maunakea`, alongside Keck and
  Subaru. Shared dish/array components preserve prototypes without expansion.
- Added Haleakala families `panstarrs-1`, `panstarrs-2`, `faulkes-north`,
  `dkist`, `atlas-haleakala`, `tlrs-4`, `solarc`, `mees-solar`, `aeos`,
  `mssc-1-6m`, `mssc-twin-1-2m`, `geodss` and `geodss-site`. Source notes
  distinguish current, ended and removed facilities, nominal apertures and
  picked instrument proxies. See [Hawaiian observatories](doc/hawaiian-observatories.md).
- Added the personal telescopes `dobsonian-36`, `apo-refractor-5` and `sct-24`, with their
  mounts, bases and characteristic structure at the requested apertures.
  Their opaque objectives remain display proxies.
- Added fifteen reusable yard kinds: mobile crane, mirror gantry, scaffold
  tower, service truck, utility pickup, equipment trailer, shipping crate,
  equipment cabinet, generator, cooling unit, access stairs, work platform,
  handrail, cable tray and weather mast. Their dimensions and parked poses
  are authored. See [yard equipment](doc/yard-equipment.md).
- Added optional height above terrain and explicit placement headings,
  retained by relative camera and named launch placements. Added golden-hour
  working-site YAML and summit/yard/Moon study inputs. These include access
  lanes, equipment, personal telescopes, stones and vegetation; a study input
  alone is not a completed render.
- Broader optical/OWL acceptance, general ALMA/SKA recipes, radomes,
  CORONA reconnaissance scenery, gravitational-wave instruments and the flyby
  remain on the [feature backlog](doc/feature-backlog.md).

### Guide-star light, mist and the night sky

- Added finite Gaussian outgoing beams to CPU and Metal: launch radius,
  geometric divergence, axial reach, wavelength and display strength.
  Spectral coordinates survive observer conversion. Beams appear through
  scattering, without an emissive skin in vacuum.
- Added named common boresights and gnomonic sky-tangent offsets in arcseconds.
  Existing explicit pointing still works. The four-beam examples replace
  degree-scale splay with a cited GALACSI pattern; launcher positions remain
  distinct while their axes share the same pointing frame.
- Added visible guide-star launch tubes aligned with resolved beam axes and
  exits. Sodium-layer focus, finite-distance parallax, fluorescence and an
  adaptive-optics loop are not supplied by these outgoing beams.
- Added bounded beam-aware volume sampling on CPU and GPU. Samples refine
  near conservative beam intervals while Gaussian tails retain ordinary
  integration; the bounds do not cut the light off. See [guide stars](doc/guide-stars.md).
- Added prescribed grove mist over canopy footprints and regional mist over
  clearings. Overlapping atmosphere layers can combine terrain-following
  weather, level cloud decks and thinner upper air, clipped by the ground.
  Extinction, scattering, wind, depth and coverage remain explicit controls.
- Retained redwood transpiration and fog research beside the atmosphere
  recipes, with model limits. Tree count does not produce a measured water
  flux; these fields do not simulate condensation or transported moisture.
- Fixed the visible Moon blocking its own light. Distant nonreflecting
  emitters no longer enlarge local secondary-ray offsets, and pure GPU
  emission skips diffuse lighting queries.
- Corrected three Moon-only night inputs that also supplied warm solar
  twilight. Added independent surface and occluded-mist direction references
  plus scene-intent guards. **Light-1 remains open:** surfaces sample sky
  illumination, while painted mist scatters explicit lamps. See
  [lighting regressions](doc/lighting-regression.md).

### Fire, smoke, sparks and fuel

- Added composite `flame.Field` sources with independent burn windows,
  fades and additive transport. A checked `manufacture.Burn` recipe creates
  single sources, seeded clusters or lines independently of fuel geometry.
- Added unit-bearing YAML fire controls for layout, temperature, composition,
  smoke, sparks and finite light cells. Spectral emission combines continuum
  and lines; sodium references and authored powers remain distinct. Hot
  tongues and soot detail are authored appearance controls.
- Added repeatable `embers` firebrands with seeded birth IDs, prescribed
  wind/loft, linear drag, gravity and cooling. Living-particle and aggregate
  spark budgets bound work, including distant-time release queries.
- Added `firefront` stand recipes for prescribed ignition ages in bounded
  spatial groups, smooth burnout and shared smoke. These schedule effects;
  they do not predict spread or consume tree geometry.
- Added the CPU/Metal `fuelbed` surface heat/fuel model after Mandel et al.,
  with explicit timestep/stability checks, retained state, rewind and owned
  snapshots. Woodland/Moon and overhead study scenes distinguish this model
  from the earlier prescribed-front film.
- Added optional `NewAdvected`/`NewMetalAdvected` heat transport and matching
  `heat-transport` YAML. Joint diffusion/cooling/advection bounds and encoded
  weight checks refuse invalid steps. Ambient inflow and fuel staying put
  are explicit; the supplied velocity is not coupled atmospheric wind.
  Reacting CPU/Metal acceptance and spread studies remain open. See
  [heat transport](doc/fire-heat-transport.md).
- Added finite isotropic volume lights in `emit`, with ball-averaged falloff
  held to independent ray quadrature. Composite flame fields and their
  finite lights prepare through Metal; shared thermal palettes avoid
  rebuilding identical source tables.
- Added explicit cached GPU smoke lighting, recomputed for changing frame
  lights. Custom paint/emission retain direct evaluation; unsupported cached
  rig use is refused. This is an approximation with recorded image budgets,
  not exact transmission reuse or a persistent scene cache.
- Crown ignition, moisture, slope feedback, firebrand spotting, transported
  burn history and coupled smoke remain open. See [fire recipes](doc/fire-recipes.md)
  and the separate [surface-fuel model](doc/fire-fuel-design.md).

### GPU traversal, preparation and readback

- Track the originating analytic sphere through outgoing shadow, skylight
  and mirror queries. A convex sphere cannot intersect its own outward ray;
  excluding that primitive avoids false planetary self-hits without raising
  every object's offset. Inward rays and other occluders retain their tests.
  Matched native ocean and negative-sun-altitude controls support the repair;
  general planet-scale intersection accuracy remains unproved.
- Checked pinhole pointers now use the GPU-generated camera-ray path, like
  pinhole values. The `film.PinholeOf` constructor no longer accidentally
  selects full CPU ray generation and upload through its pointer result.
- Added explicit `gpu-native-shadows` and `gpu-native` modes alongside
  software traversal. Nested Metal acceleration structures retain shared
  meshes/groups, instance/material identities and clear/opaque masks.
  Native ray tracing requires supported macOS/Metal hardware; unavailable
  backends and unsupported scenes are reported rather than relabelled.
- Native primary candidates are refined with Slayer's predicate; refused
  candidates use software traversal and counted fallbacks. Native shadow
  candidates now receive the ordinary-mesh check too, with a full software
  shadow query after rejection rather than assuming the ray is clear.
- Stabilized thin-triangle intersections by translating/shearing into the
  ray frame and using compensated edge determinants. This fixes the ELT
  cutaway and scaffold self-shadow witnesses without a larger ray bias or
  looser image tolerance. It does not claim the paper's full double-fallback
  watertight guarantee. See [triangle precision](doc/triangle-precision.md).
- Added CPU/GPU shared mesh-group hierarchies and direct hit normals, avoiding
  expanded foliage and discarded normal searches. Scaled/group intersections
  retain world ray windows. Degenerate triangles and coincident dome-pole
  corners are rejected before their edges or rounded area can create faces.
- Added `gpu.Profile` stage timings, logical sample/input bytes, peak host
  readback-band bytes and native query/fallback counts. Failed bake/query
  paths preserve spent work and distinguish failure from a successful miss.
- Added `Kernel.GridBands` and incremental pixel finishing to the tracer.
  One reusable host sample band replaces the full raw-sample result array;
  global row order, per-pixel summation, coverage, nearest depth and counters
  survive band boundaries. Finishing respects the Go CPU allowance. The
  full-array `Grid` API remains available.
- Streamed flame density/emission, wispy density and smoke-light bands now
  write into reserved packed scene ranges. Writers check lengths, ordering
  and reservations; incomplete output is not a completed field.
- Replaced per-record named-field packing with ABI-aware row plans and final
  capacity reservations. Dropped duplicate construction arrays after their
  consumers finish and avoided whole-input temporary C copies at synchronous
  Metal copying calls. Kernel function switches retain existing resources.
- Added host-only `render/emptyspace` occupancy masks, fingerprints, packet
  bounds and certified sample prefixes. Interpolation halos and independent
  emission count; exterior queries and exhausted proof budgets are unknown.
  **No renderer uses these skips yet.** A pilot ray's miss proves nothing
  about its neighbours. See [the packet contract](doc/mist-packet-contract.md).
- Partial stream cleanup, native size narrowing and preallocation limits are
  explicit. Host finishing has focused evidence; matched device fields,
  lifetimes, moving views and whole-process memory/timing still need checks.
  These preparation changes do not establish lower RSS or an animation speedup.

### Terrain sources, projected detail and world coordinates

- Added optional source-height shading-normal stencils with explicit angular
  steps, bounded normal records, source-halo refusals and query counts. Added
  authored smooth transitions between landcover height colours. Defaults keep
  flat shading and stepped colours; these are appearance approximations, not
  terrain geometry or ecological accuracy guarantees.
- Added a geographic bilinear sampler, stripped float32 TIFF reading and
  separate angular row/column scales for the delivered NOAA coastal raster.
  Registration and source no-data remain explicit; legacy projected rasters
  still require square sampling. Source errors cannot become flat ocean.
- Added `scene/island` view-dependent spherical terrain preparation, bounded
  source queries, world-space matching of unequal-detail patch edges and
  camera-relative meshes. Added a Pzm/Game `island render` study command with a
  checked YAML schema, source hashes and render provenance. Its cell-span metric
  is not a certified relief-error bound.

  The first scene uses NOAA topobathymetry, an explicit radial-height
  approximation and smooth ocean proxy. Regional trees/mist and a first ascent
  sequence are now connected; wave water, richer appearance and CORONA/Keyhole
  instrument views remain open. See [the island scene](doc/island-scene.md).
- Added explicit raster budgets for decoded tile bytes/count, individual
  tiles, TIFF tags and returned grids. Bounded decoding avoids retaining the
  compressed payload and a second decoded array. LRU eviction, cache discard
  and close behavior are explicit; oversized dimensions fail before allocation.
- Added source coverage, availability-returning elevation/grid reads and
  bounded legacy windows. No-data and out-of-coverage samples remain absent;
  corrupt data and exhausted decode budgets remain errors, not sea-level ground.
- Added native source-patch IDs, deterministic subdivision, bounded sampled
  grids and meshes with an explicit source-to-world mapper. Missing sampled
  corners suppress triangles. Sampling fewer points does not bound unseen peaks.
- Added byte/entry-bounded sampled-patch LRU caching, independent of camera
  or render crop. Equivalent resolved grids share entries; stricter caller
  budgets still apply on hits. Consumer-held grids outlive cache eviction
  and count separately from retained cache payloads.
- Added budgeted `SelectPatches` refinement and `SelectProjectedPatches` for
  the actual pinhole pose, lens and raster. Caller-supplied complete bounds
  and representation displacement determine projected pixel error. Budget
  exhaustion keeps coarse coverage and reports `Limited`.
- Added bounded `PatchGrid.StitchTo` source-space edge matching for affine
  mappings. Fine heights follow coarse edge segments, missing samples remain
  missing, source caches stay immutable and added displacement is reported.
  Curved-world seams and resident patch-set management remain open.
- Added tiled float32 TIFF Predictor 3 support, with scanline scratch counted
  in the decode allowance. Unknown predictors and unsupported compressed bit
  order are refused. Fixtures include independent libtiff little/big-endian
  files and hand-derived byte planes.
- Added `SourceReference` GeoTIFF codes and strict GeoKey validation, plus
  `SampleLocation` and native-patch registration for PixelIsArea/PixelIsPoint.
  Point postings avoid an incorrect half-cell shift. Legacy metre APIs refuse
  geographic/geocentric and declared non-metre sources; untagged legacy inputs
  retain their prior assumed-metre contract. No EPSG/datum conversion is added.
- Added float64 spherical anchors, radial/tangent coordinates and render-origin
  helpers in `scene/frame`, including antimeridian normalization. Added
  conservative off-axis pinhole/frustum assessments in `scene/footprint` and
  checked view/lens construction, plus `maths.Local` and `NewSubspace`
  construction helpers. See [world frames](doc/world-frame.md) and
  [terrain patches](doc/terrain-patches.md).
- These are source and geometry foundations. Hillside still stages its static
  terrain crop; global source/datum adapters, camera-relative staging of every
  scene component, fallback coverage, overviews and moving residency are open.
  Ground-to-orbit rendering is not delivered by this merge.

### Phone terrain and ocean references

- Checked the phone JONSWAP/TMA references against retained original pages.
  Labelled the rounded peak fit as model glue while preserving numeric values.
  Donelan/Banner and the complete TMA piecewise-cap attribution remain pending;
  see [the source check](doc/phone-source-checks.md).
- Imported `titan` and `oceantex` as CPU references, preserving original file
  hashes in `doc/phone-import-manifest.json`. Titan provides a geographic
  height-provider boundary, seeded finer detail and measured/generated
  reporting; it is not a resident terrain renderer.
- Oceantex provides spectra, Fourier wave fields, sea states, optics, colour,
  shore/tide sampling and a standalone CPU camera/render reference. It reads
  its source ledger through Pzm, clones exposed value slices, bounds wave,
  image and shore allocation products and respects GOMAXPROCS.
- Standalone phone commands were omitted because they bypassed Pzm. Neither
  package is connected to Hillside or Metal. Whole-cascade wave filtering,
  unresolved slope variance, wind-height conventions, ray-march exhaustion,
  planetary curvature and source verification need work before runtime use.
  Passing package tests does not establish physical or GPU acceptance. See
  [the integration status](doc/phone-ocean-integration.md).

### Scene schemas, Grissom and development

- Added bundled opt-in ecosystem v1 validation through Pzm's YAML reader,
  recipes, stands, exclusions, fire/fuel/weather, lasers, spacecraft and
  renderer choices. Expanded schema key parity to the existing film, rig,
  camera eye, bounds, assets, text, HUD, world and planet reader vocabulary.
  Legacy documents keep their independent reader checks.
- Added ecosystem v2 as a **specification without a reader**: date, anchors,
  globe/space scenes, orbit/attitude, instrument eyes, relative cameras,
  extended sky/weather and edited shots/music timelines. Example YAMLs are
  written ahead of implementation, including Keyhole, solar corona, Webb,
  orbital and lantern scenes. The backlog records gaps in regional ecosystems,
  factory enums and hyperlapse/zoom parity; v2 is not enabled rendering.
- Added Grissom's Cocoa/Metal viewer under `cmd/grissom`: survey mosaics on
  sky triangles, Earth hexes and a standing view. It is a separate graphics
  viewer, not a Slayer ray-traced scene. Added data/setup and compatibility
  layout instructions using Celestial-data's Grissom fetch set.
- Moved build/run documentation to Game, added named command/demo/catalogue
  runs and small check scopes, and documented source-shelf/review dependency
  setup. Added feature, model, source, validation and dated render records;
  prose revisions retain source and model limitations.
- Updated Puzzlemouth to v0.1.4 and Go to 1.27.3; pinned Celestial and
  Celestial-data review candidates and added JSON Schema support. YAML uses
  `gopkg.in/yaml.v3` consistently with Pzm. Dependency releases remain needed
  before a standalone Slayer release; no phone import module upgrade was needed.

### Azazel foundations

- Added the design and private primitives for a future frame/film/check queue.
  **There is no runnable queue, shared admission transaction or supervisor.**
  See [Azazel](azazel/README.md).
- Added overflow-safe memory assessment, identified reservation proposals,
  supplied host-condition checks and their ordinary admission composition.
  Low disk takes precedence, unavailable observations hold work and a
  favorable sample cannot clear a latched stop.
- Added generation-checked stop transitions with explicit eligible resume,
  stale-resume refusal and generation exhaustion that cannot wrap clear.
  These are value proposals, not persistence or permission to launch a child.
- Added ordered snapshot replacement, owned temporary cleanup, immutable
  exclusive result installation and prepared intent/state/completion steps.
  Publication failures preserve original causes and uncertain effects for
  reconciliation; cleanup touches only its owned name.
- Added Darwin temporary-file and nonblocking lock acquisition/release
  primitives, filesystem exclusion/storage capability witnesses, and native
  process start/exit timestamp observations through the public SDK. These
  do not prove child-tree ownership, crash durability or safe recovery.
- Retained paired examples, Rapid properties/counterexamples, native witnesses
  and scoped fault evidence beside the designs. Final independent acceptance,
  root protection, recovery, supervision and engine integration remain open.

### Repairs and checks across the range

- Merged `fix/unit-tests-2026-11-09`: grouped-foliage contracts and geometry
  fixtures now follow the manufactured instances, ecosystem v2 retains the
  v1 keys, and Hillside accepts `-alight` on the GPU. Legacy window infinity
  refusal is attributed to mesh construction; geographic reads and height
  indices retain their own finite-value checks. Game configuration declares
  v0.5.0 and the branch's updated prose/lint conventions.
- Source-export cache tests now accept unavailable Git history and compare
  available provenance with the fixture's actual HEAD. Downloaded source
  metadata is nonexecutable; source commands report shelf and output failures.
- Weather refuses invalid/non-finite durations and intervals; wind summaries
  advance on grids below 32 rows. Ring hull faces wind outward, and planetary
  cooking refuses an empty atmospheric column instead of indexing it.
- Stabilized disc-span fused arithmetic under coverage instrumentation.
  Refused zero-length spherical medium spans, unrepresentable grove bucket
  coordinates and non-finite geometry before native packing.
- Added early host/device byte, smooth-index, group-member, fire/front,
  particle and prototype-budget checks. Replaced a multi-gigabyte GPU test
  allocation with bounded boundary witnesses. Actual fallback lamp reporting
  now counts the scene lamps it submits.
- Expanded real-command coverage using child binaries with matching coverage
  mode/package scope; tests decode PNG/raw output and capture terminal/viewer
  paths without opening a window. Coverage notes distinguish ordinary package
  measurement, per-file gates, generated helpers and unmeasured shader branches.
- Expanded independent mesh/group harness comparisons, committed 4K producer
  identity/input checks, source and numeric fixtures, refusal/cleanup tests,
  native ABI/layout checks and function-scoped mutation evidence. Authored
  appearance records retain their producer and provenance; non-finite output
  cannot pass a closeness check. History-rule changes remain separately listed.
- Dated results include the 2,880-frame, two-minute 4K prescribed-fire film
  and a 180-frame, thirty-second 120x camera hyperlapse. They have different
  workloads and quality settings. Neither proves the dense sixteen-sample
  forest's eight-hour target, a whole-world shot or the current tip's gates.
- Full current-branch tests, coverage, lint/sweep/Shuriken, wider mutations
  and visual/performance acceptance remain separate work. The focused
  [verification record](doc/verification-2026-11-09.md) names earlier scopes
  and limits; this changelog update runs no new unit tests.

## v0.1.0 - v0.1.1 06 november

added:

- YAML camera paths with speeds, timed easing and holds for repeatable
  sequences; numbered smoke scenes can use the same frame and resume writer
- a checked demo catalogue for numbered smokes and named scenes, with PNG
  output and optional terminal previews; selectors need Game forwarding

- fire: a flame field of many seeded patches in one medium, each with
  its own burn window and fade (flame.Field); emission from a sourced
  continuum and lines (flame.Composition); hot tongues as an authored
  temperature profile
- fire recipes in a set's yaml, every quantity with its unit: flame,
  layout (a cluster or a line), smoke, sparks and light cells, checked
  by manufacture.Burn before a scene is built
- embers: firebrands born by a seeded id, carried by wind under linear
  drag and gravity, cooling over their life; budgets bound the living
- firefront: prescribed ignition fronts over a stand's trees, in bounded
  spatial groups, their smoke following what burns. it schedules; it
  does not predict spread
- fuelbed: a surface fire from heat transfer and remaining fuel, after
  Mandel et al. (2007), explicit, on the cpu or on Metal, its stability
  bound refused when broken
- emit: finite isotropic volume lights, their falloff the mean over a
  ball, held to independent ray quadrature
- manufacture: coast redwoods and giant sequoias, their foliage grown
  recursively from shared shoot prototypes, the numbers in
  needled-shoot.yml and the research ledger with their sources
- manufacture: an optical observatory, a dome with a real slit, a mount,
  a truss and a segmented or monolithic mirror; and complexity levels
  for every kind: hull, clumps, quick and demo
- scene/shape: mesh groups, one prototype placed many times with its
  own hierarchy, drawn by the cpu tracer and the gpu without expanding
- scene/medium: grove mist over a stand's canopies, prescribed in yaml,
  drifting with the wind
- gpu: native Metal ray tracing, for shadows (renderer:
  gpu-native-shadows) or for mesh queries too (gpu-native); a cached
  smoke lighting grid (volume-lighting: cached); each stage's cost
  (gpu.Profile)
- metal: a kernel switches entry points and keeps its resources;
  acceleration structures, built from a checked plan, macOS 14 and on
- hillside sequence: timed png frames from a checked yaml plan, the
  scenery kept between frames, each frame written whole, and a manifest
  that refuses a resume when scene, plan, renderer or terrain changed
- set: an opt-in schema for ecosystem documents (ecosystem-v1); recipes,
  stands that exclude around an object and share an atmosphere, fire
  fronts, fuel beds, wind, seed and renderer
- cmd/grissom: survey mosaics laid on the sky's triangles, the earth as
  hexes, and a place to stand under the sky; a work in progress
- the harness compares mesh groups through independent reference
  meshes; committed 4k comparisons are recorded in it

changed:

- hillside's count flags are frozen at the kinds they had; new kinds
  come through recipes and the yaml
- the visible moon no longer blocks its own light; distant emitters
  stay out of local shadow offsets; pure gpu emission skips diffuse
  light queries
- sequence frames are written at fast png compression
- yaml is read through gopkg.in/yaml.v3, as puzzlemouth's yamlfile reads
  it
- the harness rule keeps a second list, read whole, for commits that
  changed the harness and what it judges and are not mechanical; every
  refusal of the rule says why

fixed:

- render/smokecache's provenance tests accept unavailable checkout history
  in source exports and check available revisions against Git's exact HEAD
- a buffer or grid larger than the device or host holds is refused
  before anything is allocated
- degenerate triangles are checked before their edges are assigned
- cmd/sources reports its output and shelf failures
- cmd/weather refuses a time that is not a number, and no longer loops
  on a grid under 32 rows (in 51931c8, with tests)
- cmd/ring's faces wind with the strip's outward direction, and planet's
  Cooked refuses an empty column where it panicked (both in 51931c8)

## v0.1.0 01 november

- the first release
